// Copyright 2014 The go-btpereum Authors
// This file is part of the go-btpereum library.
//
// The go-btpereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-btpereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-btpereum library. If not, see <http://www.gnu.org/licenses/>.

// Package btp implements the btpereum protocol.
package btp

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/btpereum/go-btpereum/accounts"
	"github.com/btpereum/go-btpereum/accounts/abi/bind"
	"github.com/btpereum/go-btpereum/common"
	"github.com/btpereum/go-btpereum/common/hexutil"
	"github.com/btpereum/go-btpereum/consensus"
	"github.com/btpereum/go-btpereum/consensus/clique"
	"github.com/btpereum/go-btpereum/consensus/btpash"
	"github.com/btpereum/go-btpereum/core"
	"github.com/btpereum/go-btpereum/core/bloombits"
	"github.com/btpereum/go-btpereum/core/rawdb"
	"github.com/btpereum/go-btpereum/core/types"
	"github.com/btpereum/go-btpereum/core/vm"
	"github.com/btpereum/go-btpereum/btp/downloader"
	"github.com/btpereum/go-btpereum/btp/filters"
	"github.com/btpereum/go-btpereum/btp/gasprice"
	"github.com/btpereum/go-btpereum/btpdb"
	"github.com/btpereum/go-btpereum/event"
	"github.com/btpereum/go-btpereum/internal/btpapi"
	"github.com/btpereum/go-btpereum/log"
	"github.com/btpereum/go-btpereum/miner"
	"github.com/btpereum/go-btpereum/node"
	"github.com/btpereum/go-btpereum/p2p"
	"github.com/btpereum/go-btpereum/p2p/enr"
	"github.com/btpereum/go-btpereum/params"
	"github.com/btpereum/go-btpereum/rlp"
	"github.com/btpereum/go-btpereum/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	APIs() []rpc.API
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
	SetContractBackend(bind.ContractBackend)
}

// btpereum implements the btpereum full node service.
type btpereum struct {
	config *Config

	// Channel for shutting down the service
	shutdownChan chan bool

	server *p2p.Server

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

	// DB interfaces
	chainDb btpdb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	APIBackend *btpAPIBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	btperbase common.Address

	networkID     uint64
	netRPCService *btpapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and btperbase)
}

func (s *btpereum) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

// SetClient sets a rpc client which connecting to our local node.
func (s *btpereum) SetContractBackend(backend bind.ContractBackend) {
	// Pass the rpc client to les server if it is enabled.
	if s.lesServer != nil {
		s.lesServer.SetContractBackend(backend)
	}
}

// New creates a new btpereum object (including the
// initialisation of the common btpereum object)
func New(ctx *node.ServiceContext, config *Config) (*btpereum, error) {
	// Ensure configuration values are compatible and sane
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run btp.btpereum in light sync mode, use les.Lightbtpereum")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	if config.Miner.GasPrice == nil || config.Miner.GasPrice.Cmp(common.Big0) <= 0 {
		log.Warn("Sanitizing invalid miner gas price", "provided", config.Miner.GasPrice, "updated", DefaultConfig.Miner.GasPrice)
		config.Miner.GasPrice = new(big.Int).Set(DefaultConfig.Miner.GasPrice)
	}
	if config.NoPruning && config.TrieDirtyCache > 0 {
		config.TrieCleanCache += config.TrieDirtyCache
		config.TrieDirtyCache = 0
	}
	log.Info("Allocated trie memory caches", "clean", common.StorageSize(config.TrieCleanCache)*1024*1024, "dirty", common.StorageSize(config.TrieDirtyCache)*1024*1024)

	// Assemble the btpereum object
	chainDb, err := ctx.OpenDatabaseWithFreezer("chaindata", config.DatabaseCache, config.DatabaseHandles, config.DatabaseFreezer, "btp/db/chaindata/")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	btp := &btpereum{
		config:         config,
		chainDb:        chainDb,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, chainConfig, &config.btpash, config.Miner.Notify, config.Miner.Noverify, chainDb),
		shutdownChan:   make(chan bool),
		networkID:      config.NetworkId,
		gasPrice:       config.Miner.GasPrice,
		btperbase:      config.Miner.btperbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks, params.BloomConfirms),
	}

	bcVersion := rawdb.ReadDatabaseVersion(chainDb)
	var dbVer = "<nil>"
	if bcVersion != nil {
		dbVer = fmt.Sprintf("%d", *bcVersion)
	}
	log.Info("Initialising btpereum protocol", "versions", ProtocolVersions, "network", config.NetworkId, "dbversion", dbVer)

	if !config.SkipBcVersionCheck {
		if bcVersion != nil && *bcVersion > core.BlockChainVersion {
			return nil, fmt.Errorf("database version is v%d, Gbtp %s only supports v%d", *bcVersion, params.VersionWithMeta, core.BlockChainVersion)
		} else if bcVersion == nil || *bcVersion < core.BlockChainVersion {
			log.Warn("Upgrade blockchain database version", "from", dbVer, "to", core.BlockChainVersion)
			rawdb.WriteDatabaseVersion(chainDb, core.BlockChainVersion)
		}
	}
	var (
		vmConfig = vm.Config{
			EnablePreimageRecording: config.EnablePreimageRecording,
			EWASMInterpreter:        config.EWASMInterpreter,
			EVMInterpreter:          config.EVMInterpreter,
		}
		cacheConfig = &core.CacheConfig{
			TrieCleanLimit:      config.TrieCleanCache,
			TrieCleanNoPrefetch: config.NoPrefetch,
			TrieDirtyLimit:      config.TrieDirtyCache,
			TrieDirtyDisabled:   config.NoPruning,
			TrieTimeLimit:       config.TrieTimeout,
		}
	)
	btp.blockchain, err = core.NewBlockChain(chainDb, cacheConfig, chainConfig, btp.engine, vmConfig, btp.shouldPreserve)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		btp.blockchain.Sbtpead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	btp.bloomIndexer.Start(btp.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	btp.txPool = core.NewTxPool(config.TxPool, chainConfig, btp.blockchain)

	// Permit the downloader to use the trie cache allowance during fast sync
	cacheLimit := cacheConfig.TrieCleanLimit + cacheConfig.TrieDirtyLimit
	checkpoint := config.Checkpoint
	if checkpoint == nil {
		checkpoint = params.TrustedCheckpoints[genesisHash]
	}
	if btp.protocolManager, err = NewProtocolManager(chainConfig, checkpoint, config.SyncMode, config.NetworkId, btp.eventMux, btp.txPool, btp.engine, btp.blockchain, chainDb, cacheLimit, config.Whitelist); err != nil {
		return nil, err
	}
	btp.miner = miner.New(btp, &config.Miner, chainConfig, btp.EventMux(), btp.engine, btp.isLocalBlock)
	btp.miner.SetExtra(makeExtraData(config.Miner.ExtraData))

	btp.APIBackend = &btpAPIBackend{ctx.ExtRPCEnabled(), btp, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.Miner.GasPrice
	}
	btp.APIBackend.gpo = gasprice.NewOracle(btp.APIBackend, gpoParams)

	return btp, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"gbtp",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// CreateConsensusEngine creates the required type of consensus engine instance for an btpereum service
func CreateConsensusEngine(ctx *node.ServiceContext, chainConfig *params.ChainConfig, config *btpash.Config, notify []string, noverify bool, db btpdb.Database) consensus.Engine {
	// If proof-of-authority is requested, set it up
	if chainConfig.Clique != nil {
		return clique.New(chainConfig.Clique, db)
	}
	// Otherwise assume proof-of-work
	switch config.PowMode {
	case btpash.ModeFake:
		log.Warn("btpash used in fake mode")
		return btpash.NewFaker()
	case btpash.ModeTest:
		log.Warn("btpash used in test mode")
		return btpash.NewTester(nil, noverify)
	case btpash.ModeShared:
		log.Warn("btpash used in shared mode")
		return btpash.NewShared()
	default:
		engine := btpash.New(btpash.Config{
			CacheDir:       ctx.ResolvePath(config.CacheDir),
			CachesInMem:    config.CachesInMem,
			CachesOnDisk:   config.CachesOnDisk,
			DatasetDir:     config.DatasetDir,
			DatasetsInMem:  config.DatasetsInMem,
			DatasetsOnDisk: config.DatasetsOnDisk,
		}, notify, noverify)
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}

// APIs return the collection of RPC services the btpereum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *btpereum) APIs() []rpc.API {
	apis := btpapi.GetAPIs(s.APIBackend)

	// Append any APIs exposed explicitly by the les server
	if s.lesServer != nil {
		apis = append(apis, s.lesServer.APIs()...)
	}
	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append any APIs exposed explicitly by the les server
	if s.lesServer != nil {
		apis = append(apis, s.lesServer.APIs()...)
	}

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "btp",
			Version:   "1.0",
			Service:   NewPublicbtpereumAPI(s),
			Public:    true,
		}, {
			Namespace: "btp",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "btp",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "btp",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.APIBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *btpereum) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *btpereum) btperbase() (eb common.Address, err error) {
	s.lock.RLock()
	btperbase := s.btperbase
	s.lock.RUnlock()

	if btperbase != (common.Address{}) {
		return btperbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			btperbase := accounts[0].Address

			s.lock.Lock()
			s.btperbase = btperbase
			s.lock.Unlock()

			log.Info("btperbase automatically configured", "address", btperbase)
			return btperbase, nil
		}
	}
	return common.Address{}, fmt.Errorf("btperbase must be explicitly specified")
}

// isLocalBlock checks whbtper the specified block is mined
// by local miner accounts.
//
// We regard two types of accounts as local miner account: btperbase
// and accounts specified via `txpool.locals` flag.
func (s *btpereum) isLocalBlock(block *types.Block) bool {
	author, err := s.engine.Author(block.Header())
	if err != nil {
		log.Warn("Failed to retrieve block author", "number", block.NumberU64(), "hash", block.Hash(), "err", err)
		return false
	}
	// Check whbtper the given address is btperbase.
	s.lock.RLock()
	btperbase := s.btperbase
	s.lock.RUnlock()
	if author == btperbase {
		return true
	}
	// Check whbtper the given address is specified by `txpool.local`
	// CLI flag.
	for _, account := range s.config.TxPool.Locals {
		if account == author {
			return true
		}
	}
	return false
}

// shouldPreserve checks whbtper we should preserve the given block
// during the chain reorg depending on whbtper the author of block
// is a local account.
func (s *btpereum) shouldPreserve(block *types.Block) bool {
	// The reason we need to disable the self-reorg preserving for clique
	// is it can be probable to introduce a deadlock.
	//
	// e.g. If there are 7 available signers
	//
	// r1   A
	// r2     B
	// r3       C
	// r4         D
	// r5   A      [X] F G
	// r6    [X]
	//
	// In the round5, the inturn signer E is offline, so the worst case
	// is A, F and G sign the block of round5 and reject the block of opponents
	// and in the round6, the last available signer B is offline, the whole
	// network is stuck.
	if _, ok := s.engine.(*clique.Clique); ok {
		return false
	}
	return s.isLocalBlock(block)
}

// Setbtperbase sets the mining reward address.
func (s *btpereum) Setbtperbase(btperbase common.Address) {
	s.lock.Lock()
	s.btperbase = btperbase
	s.lock.Unlock()

	s.miner.Setbtperbase(btperbase)
}

// StartMining starts the miner with the given number of CPU threads. If mining
// is already running, this mbtpod adjust the number of threads allowed to use
// and updates the minimum price required by the transaction pool.
func (s *btpereum) StartMining(threads int) error {
	// Update the thread count within the consensus engine
	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := s.engine.(threaded); ok {
		log.Info("Updated mining threads", "threads", threads)
		if threads == 0 {
			threads = -1 // Disable the miner from within
		}
		th.SetThreads(threads)
	}
	// If the miner was not running, initialize it
	if !s.IsMining() {
		// Propagate the initial price point to the transaction pool
		s.lock.RLock()
		price := s.gasPrice
		s.lock.RUnlock()
		s.txPool.SetGasPrice(price)

		// Configure the local mining address
		eb, err := s.btperbase()
		if err != nil {
			log.Error("Cannot start mining without btperbase", "err", err)
			return fmt.Errorf("btperbase missing: %v", err)
		}
		if clique, ok := s.engine.(*clique.Clique); ok {
			wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
			if wallet == nil || err != nil {
				log.Error("btperbase account unavailable locally", "err", err)
				return fmt.Errorf("signer missing: %v", err)
			}
			clique.Authorize(eb, wallet.SignData)
		}
		// If mining is started, we can disable the transaction rejection mechanism
		// introduced to speed sync times.
		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)

		go s.miner.Start(eb)
	}
	return nil
}

// StopMining terminates the miner, both at the consensus engine level as well as
// at the block creation level.
func (s *btpereum) StopMining() {
	// Update the thread count within the consensus engine
	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := s.engine.(threaded); ok {
		th.SetThreads(-1)
	}
	// Stop the block creating itself
	s.miner.Stop()
}

func (s *btpereum) IsMining() bool      { return s.miner.Mining() }
func (s *btpereum) Miner() *miner.Miner { return s.miner }

func (s *btpereum) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *btpereum) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *btpereum) TxPool() *core.TxPool               { return s.txPool }
func (s *btpereum) EventMux() *event.TypeMux           { return s.eventMux }
func (s *btpereum) Engine() consensus.Engine           { return s.engine }
func (s *btpereum) ChainDb() btpdb.Database            { return s.chainDb }
func (s *btpereum) IsListening() bool                  { return true } // Always listening
func (s *btpereum) btpVersion() int                    { return int(ProtocolVersions[0]) }
func (s *btpereum) NetVersion() uint64                 { return s.networkID }
func (s *btpereum) Downloader() *downloader.Downloader { return s.protocolManager.downloader }
func (s *btpereum) Synced() bool                       { return atomic.LoadUint32(&s.protocolManager.acceptTxs) == 1 }
func (s *btpereum) ArchiveMode() bool                  { return s.config.NoPruning }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *btpereum) Protocols() []p2p.Protocol {
	protos := make([]p2p.Protocol, len(ProtocolVersions))
	for i, vsn := range ProtocolVersions {
		protos[i] = s.protocolManager.makeProtocol(vsn)
		protos[i].Attributes = []enr.Entry{s.currentbtpEntry()}
	}
	if s.lesServer != nil {
		protos = append(protos, s.lesServer.Protocols()...)
	}
	return protos
}

// Start implements node.Service, starting all internal goroutines needed by the
// btpereum protocol implementation.
func (s *btpereum) Start(srvr *p2p.Server) error {
	s.startbtpEntryUpdate(srvr.LocalNode())

	// Start the bloom bits servicing goroutines
	s.startBloomHandlers(params.BloomBitsBlocks)

	// Start the RPC service
	s.netRPCService = btpapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		if s.config.LightPeers >= srvr.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", s.config.LightPeers, srvr.MaxPeers)
		}
		maxPeers -= s.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// btpereum protocol.
func (s *btpereum) Stop() error {
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	s.engine.Close()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)
	return nil
}
