// Copyright 2015 The go-btpereum Authors
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

package btp

import (
	"context"
	"errors"
	"math/big"

	"github.com/btpereum/go-btpereum/accounts"
	"github.com/btpereum/go-btpereum/common"
	"github.com/btpereum/go-btpereum/common/math"
	"github.com/btpereum/go-btpereum/core"
	"github.com/btpereum/go-btpereum/core/bloombits"
	"github.com/btpereum/go-btpereum/core/rawdb"
	"github.com/btpereum/go-btpereum/core/state"
	"github.com/btpereum/go-btpereum/core/types"
	"github.com/btpereum/go-btpereum/core/vm"
	"github.com/btpereum/go-btpereum/btp/downloader"
	"github.com/btpereum/go-btpereum/btp/gasprice"
	"github.com/btpereum/go-btpereum/btpdb"
	"github.com/btpereum/go-btpereum/event"
	"github.com/btpereum/go-btpereum/params"
	"github.com/btpereum/go-btpereum/rpc"
)

// btpAPIBackend implements btpapi.Backend for full nodes
type btpAPIBackend struct {
	extRPCEnabled bool
	btp           *btpereum
	gpo           *gasprice.Oracle
}

// ChainConfig returns the active chain configuration.
func (b *btpAPIBackend) ChainConfig() *params.ChainConfig {
	return b.btp.blockchain.Config()
}

func (b *btpAPIBackend) CurrentBlock() *types.Block {
	return b.btp.blockchain.CurrentBlock()
}

func (b *btpAPIBackend) Sbtpead(number uint64) {
	b.btp.protocolManager.downloader.Cancel()
	b.btp.blockchain.Sbtpead(number)
}

func (b *btpAPIBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.btp.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.btp.blockchain.CurrentBlock().Header(), nil
	}
	return b.btp.blockchain.GbtpeaderByNumber(uint64(blockNr)), nil
}

func (b *btpAPIBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.btp.blockchain.GbtpeaderByHash(hash), nil
}

func (b *btpAPIBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.btp.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.btp.blockchain.CurrentBlock(), nil
	}
	return b.btp.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (b *btpAPIBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block, state := b.btp.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, blockNr)
	if err != nil {
		return nil, nil, err
	}
	if header == nil {
		return nil, nil, errors.New("header not found")
	}
	stateDb, err := b.btp.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *btpAPIBackend) GetBlock(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.btp.blockchain.GetBlockByHash(hash), nil
}

func (b *btpAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return b.btp.blockchain.GetReceiptsByHash(hash), nil
}

func (b *btpAPIBackend) GetLogs(ctx context.Context, hash common.Hash) ([][]*types.Log, error) {
	receipts := b.btp.blockchain.GetReceiptsByHash(hash)
	if receipts == nil {
		return nil, nil
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

func (b *btpAPIBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.btp.blockchain.GetTdByHash(blockHash)
}

func (b *btpAPIBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.btp.BlockChain(), nil)
	return vm.NewEVM(context, state, b.btp.blockchain.Config(), *b.btp.blockchain.GetVMConfig()), vmError, nil
}

func (b *btpAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.btp.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *btpAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.btp.BlockChain().SubscribeChainEvent(ch)
}

func (b *btpAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.btp.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *btpAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.btp.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *btpAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.btp.BlockChain().SubscribeLogsEvent(ch)
}

func (b *btpAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.btp.txPool.AddLocal(signedTx)
}

func (b *btpAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.btp.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *btpAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.btp.txPool.Get(hash)
}

func (b *btpAPIBackend) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	tx, blockHash, blockNumber, index := rawdb.ReadTransaction(b.btp.ChainDb(), txHash)
	return tx, blockHash, blockNumber, index, nil
}

func (b *btpAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.btp.txPool.Nonce(addr), nil
}

func (b *btpAPIBackend) Stats() (pending int, queued int) {
	return b.btp.txPool.Stats()
}

func (b *btpAPIBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.btp.TxPool().Content()
}

func (b *btpAPIBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.btp.TxPool().SubscribeNewTxsEvent(ch)
}

func (b *btpAPIBackend) Downloader() *downloader.Downloader {
	return b.btp.Downloader()
}

func (b *btpAPIBackend) ProtocolVersion() int {
	return b.btp.btpVersion()
}

func (b *btpAPIBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *btpAPIBackend) ChainDb() btpdb.Database {
	return b.btp.ChainDb()
}

func (b *btpAPIBackend) EventMux() *event.TypeMux {
	return b.btp.EventMux()
}

func (b *btpAPIBackend) AccountManager() *accounts.Manager {
	return b.btp.AccountManager()
}

func (b *btpAPIBackend) ExtRPCEnabled() bool {
	return b.extRPCEnabled
}

func (b *btpAPIBackend) RPCGasCap() *big.Int {
	return b.btp.config.RPCGasCap
}

func (b *btpAPIBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.btp.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *btpAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.btp.bloomRequests)
	}
}
