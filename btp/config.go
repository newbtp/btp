// Copyright 2017 The go-btpereum Authors
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
	"math/big"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	"github.com/btpereum/go-btpereum/common"
	"github.com/btpereum/go-btpereum/consensus/btpash"
	"github.com/btpereum/go-btpereum/core"
	"github.com/btpereum/go-btpereum/btp/downloader"
	"github.com/btpereum/go-btpereum/btp/gasprice"
	"github.com/btpereum/go-btpereum/miner"
	"github.com/btpereum/go-btpereum/params"
)

// DefaultConfig contains default settings for use on the btpereum main net.
var DefaultConfig = Config{
	SyncMode: downloader.FastSync,
	btpash: btpash.Config{
		CacheDir:       "btpash",
		CachesInMem:    2,
		CachesOnDisk:   3,
		DatasetsInMem:  1,
		DatasetsOnDisk: 2,
	},
	NetworkId:      1,
	LightPeers:     100,
	DatabaseCache:  512,
	TrieCleanCache: 256,
	TrieDirtyCache: 256,
	TrieTimeout:    60 * time.Minute,
	Miner: miner.Config{
		GasFloor: 8000000,
		GasCeil:  8000000,
		GasPrice: big.NewInt(params.GWei),
		Recommit: 3 * time.Second,
	},
	TxPool: core.DefaultTxPoolConfig,
	GPO: gasprice.Config{
		Blocks:     20,
		Percentile: 60,
	},
}

func init() {
	home := os.Getenv("HOME")
	if home == "" {
		if user, err := user.Current(); err == nil {
			home = user.HomeDir
		}
	}
	if runtime.GOOS == "darwin" {
		DefaultConfig.btpash.DatasetDir = filepath.Join(home, "Library", "btpash")
	} else if runtime.GOOS == "windows" {
		localappdata := os.Getenv("LOCALAPPDATA")
		if localappdata != "" {
			DefaultConfig.btpash.DatasetDir = filepath.Join(localappdata, "btpash")
		} else {
			DefaultConfig.btpash.DatasetDir = filepath.Join(home, "AppData", "Local", "btpash")
		}
	} else {
		DefaultConfig.btpash.DatasetDir = filepath.Join(home, ".btpash")
	}
}

//go:generate gencodec -type Config -formats toml -out gen_config.go

type Config struct {
	// The genesis block, which is inserted if the database is empty.
	// If nil, the btpereum main net block is used.
	Genesis *core.Genesis `toml:",omitempty"`

	// Protocol options
	NetworkId uint64 // Network ID to use for selecting peers to connect to
	SyncMode  downloader.SyncMode

	NoPruning  bool // Whbtper to disable pruning and flush everything to disk
	NoPrefetch bool // Whbtper to disable prefetching and only load state on demand

	// Whitelist of required block number -> hash values to accept
	Whitelist map[uint64]common.Hash `toml:"-"`

	// Light client options
	LightServ         int  `toml:",omitempty"` // Maximum percentage of time allowed for serving LES requests
	LightBandwidthIn  int  `toml:",omitempty"` // Incoming bandwidth limit for light servers
	LightBandwidthOut int  `toml:",omitempty"` // Outgoing bandwidth limit for light servers
	LightPeers        int  `toml:",omitempty"` // Maximum number of LES client peers
	OnlyAnnounce      bool // Maximum number of LES client peers

	// Ultra Light client options
	ULC *ULCConfig `toml:",omitempty"`

	// Database options
	SkipBcVersionCheck bool `toml:"-"`
	DatabaseHandles    int  `toml:"-"`
	DatabaseCache      int
	DatabaseFreezer    string

	TrieCleanCache int
	TrieDirtyCache int
	TrieTimeout    time.Duration

	// Mining options
	Miner miner.Config

	// btpash options
	btpash btpash.Config

	// Transaction pool options
	TxPool core.TxPoolConfig

	// Gas Price Oracle options
	GPO gasprice.Config

	// Enables tracking of SHA3 preimages in the VM
	EnablePreimageRecording bool

	// Miscellaneous options
	DocRoot string `toml:"-"`

	// Type of the EWASM interpreter ("" for default)
	EWASMInterpreter string

	// Type of the EVM interpreter ("" for default)
	EVMInterpreter string

	// RPCGasCap is the global gas cap for btp-call variants.
	RPCGasCap *big.Int `toml:",omitempty"`

	// Checkpoint is a hardcoded checkpoint which can be nil.
	Checkpoint *params.TrustedCheckpoint

	// CheckpointOracle is the configuration for checkpoint oracle.
	CheckpointOracle *params.CheckpointOracleConfig
}
