// Copyright 2019 The go-btpereum Authors
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
	"github.com/btpereum/go-btpereum/core"
	"github.com/btpereum/go-btpereum/core/forkid"
	"github.com/btpereum/go-btpereum/p2p/enode"
	"github.com/btpereum/go-btpereum/rlp"
)

// btpEntry is the "btp" ENR entry which advertises btp protocol
// on the discovery network.
type btpEntry struct {
	ForkID forkid.ID // Fork identifier per EIP-2124

	// Ignore additional fields (for forward compatibility).
	Rest []rlp.RawValue `rlp:"tail"`
}

// ENRKey implements enr.Entry.
func (e btpEntry) ENRKey() string {
	return "btp"
}

func (btp *btpereum) startbtpEntryUpdate(ln *enode.LocalNode) {
	var newHead = make(chan core.ChainHeadEvent, 10)
	sub := btp.blockchain.SubscribeChainHeadEvent(newHead)

	go func() {
		defer sub.Unsubscribe()
		for {
			select {
			case <-newHead:
				ln.Set(btp.currentbtpEntry())
			case <-sub.Err():
				// Would be nice to sync with btp.Stop, but there is no
				// good way to do that.
				return
			}
		}
	}()
}

func (btp *btpereum) currentbtpEntry() *btpEntry {
	return &btpEntry{ForkID: forkid.NewID(btp.blockchain)}
}
