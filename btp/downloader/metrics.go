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

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/btpereum/go-btpereum/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("btp/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("btp/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("btp/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("btp/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("btp/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("btp/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("btp/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("btp/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("btp/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("btp/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("btp/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("btp/downloader/receipts/timeout", nil)

	stateInMeter   = metrics.NewRegisteredMeter("btp/downloader/states/in", nil)
	stateDropMeter = metrics.NewRegisteredMeter("btp/downloader/states/drop", nil)
)
