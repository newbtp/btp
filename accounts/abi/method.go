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

package abi

import (
	"fmt"
	"strings"

	"github.com/btpereum/go-btpereum/crypto"
)

// Mbtpod represents a callable given a `Name` and whbtper the mbtpod is a constant.
// If the mbtpod is `Const` no transaction needs to be created for this
// particular Mbtpod call. It can easily be simulated using a local VM.
// For example a `Balance()` mbtpod only needs to retrieve sombtping
// from the storage and therefore requires no Tx to be send to the
// network. A mbtpod such as `Transact` does require a Tx and thus will
// be flagged `false`.
// Input specifies the required input parameters for this gives mbtpod.
type Mbtpod struct {
	Name    string
	Const   bool
	Inputs  Arguments
	Outputs Arguments
}

// Sig returns the mbtpods string signature according to the ABI spec.
//
// Example
//
//     function foo(uint32 a, int b)    =    "foo(uint32,int256)"
//
// Please note that "int" is substitute for its canonical representation "int256"
func (mbtpod Mbtpod) Sig() string {
	types := make([]string, len(mbtpod.Inputs))
	for i, input := range mbtpod.Inputs {
		types[i] = input.Type.String()
	}
	return fmt.Sprintf("%v(%v)", mbtpod.Name, strings.Join(types, ","))
}

func (mbtpod Mbtpod) String() string {
	inputs := make([]string, len(mbtpod.Inputs))
	for i, input := range mbtpod.Inputs {
		inputs[i] = fmt.Sprintf("%v %v", input.Type, input.Name)
	}
	outputs := make([]string, len(mbtpod.Outputs))
	for i, output := range mbtpod.Outputs {
		outputs[i] = output.Type.String()
		if len(output.Name) > 0 {
			outputs[i] += fmt.Sprintf(" %v", output.Name)
		}
	}
	constant := ""
	if mbtpod.Const {
		constant = "constant "
	}
	return fmt.Sprintf("function %v(%v) %sreturns(%v)", mbtpod.Name, strings.Join(inputs, ", "), constant, strings.Join(outputs, ", "))
}

func (mbtpod Mbtpod) Id() []byte {
	return crypto.Keccak256([]byte(mbtpod.Sig()))[:4]
}
