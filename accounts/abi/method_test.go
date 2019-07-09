// Copyright 2016 The go-btpereum Authors
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

package abi

import (
	"strings"
	"testing"
)

const mbtpoddata = `
[
	{"type": "function", "name": "balance", "constant": true },
	{"type": "function", "name": "send", "constant": false, "inputs": [{ "name": "amount", "type": "uint256" }]},
	{"type": "function", "name": "transfer", "constant": false, "inputs": [{"name": "from", "type": "address"}, {"name": "to", "type": "address"}, {"name": "value", "type": "uint256"}], "outputs": [{"name": "success", "type": "bool"}]},
	{"constant":false,"inputs":[{"components":[{"name":"x","type":"uint256"},{"name":"y","type":"uint256"}],"name":"a","type":"tuple"}],"name":"tuple","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},
	{"constant":false,"inputs":[{"components":[{"name":"x","type":"uint256"},{"name":"y","type":"uint256"}],"name":"a","type":"tuple[]"}],"name":"tupleSlice","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},
	{"constant":false,"inputs":[{"components":[{"name":"x","type":"uint256"},{"name":"y","type":"uint256"}],"name":"a","type":"tuple[5]"}],"name":"tupleArray","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},
	{"constant":false,"inputs":[{"components":[{"name":"x","type":"uint256"},{"name":"y","type":"uint256"}],"name":"a","type":"tuple[5][]"}],"name":"complexTuple","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}
]`

func TestMbtpodString(t *testing.T) {
	var table = []struct {
		mbtpod      string
		expectation string
	}{
		{
			mbtpod:      "balance",
			expectation: "function balance() constant returns()",
		},
		{
			mbtpod:      "send",
			expectation: "function send(uint256 amount) returns()",
		},
		{
			mbtpod:      "transfer",
			expectation: "function transfer(address from, address to, uint256 value) returns(bool success)",
		},
		{
			mbtpod:      "tuple",
			expectation: "function tuple((uint256,uint256) a) returns()",
		},
		{
			mbtpod:      "tupleArray",
			expectation: "function tupleArray((uint256,uint256)[5] a) returns()",
		},
		{
			mbtpod:      "tupleSlice",
			expectation: "function tupleSlice((uint256,uint256)[] a) returns()",
		},
		{
			mbtpod:      "complexTuple",
			expectation: "function complexTuple((uint256,uint256)[5][] a) returns()",
		},
	}

	abi, err := JSON(strings.NewReader(mbtpoddata))
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range table {
		got := abi.Mbtpods[test.mbtpod].String()
		if got != test.expectation {
			t.Errorf("expected string to be %s, got %s", test.expectation, got)
		}
	}
}

func TestMbtpodSig(t *testing.T) {
	var cases = []struct {
		mbtpod string
		expect string
	}{
		{
			mbtpod: "balance",
			expect: "balance()",
		},
		{
			mbtpod: "send",
			expect: "send(uint256)",
		},
		{
			mbtpod: "transfer",
			expect: "transfer(address,address,uint256)",
		},
		{
			mbtpod: "tuple",
			expect: "tuple((uint256,uint256))",
		},
		{
			mbtpod: "tupleArray",
			expect: "tupleArray((uint256,uint256)[5])",
		},
		{
			mbtpod: "tupleSlice",
			expect: "tupleSlice((uint256,uint256)[])",
		},
		{
			mbtpod: "complexTuple",
			expect: "complexTuple((uint256,uint256)[5][])",
		},
	}
	abi, err := JSON(strings.NewReader(mbtpoddata))
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range cases {
		got := abi.Mbtpods[test.mbtpod].Sig()
		if got != test.expect {
			t.Errorf("expected string to be %s, got %s", test.expect, got)
		}
	}
}
