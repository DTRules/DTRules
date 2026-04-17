// Copyright 2024 Paul Snow
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package operators

import (
	"crypto/sha256"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"golang.org/x/crypto/ripemd160" //nolint:staticcheck
	"golang.org/x/crypto/sha3"
)

func init() {
	Register("sha256", opSha256)
	Register("keccak256", opKeccak256)
	Register("ripemd160", opRipemd160)
	Register("sha3", opSha3)
}

// opSha256 computes SHA-256: ( bytes -- bytes(32) )
func opSha256(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	b, err := a.RBytesValue()
	if err != nil {
		return err
	}
	h := sha256.Sum256(b.Bytes())
	return state.DataPush(dtrules.NewRBytes(h[:]))
}

// opKeccak256 computes Keccak-256 (Ethereum): ( bytes -- bytes(32) )
func opKeccak256(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	b, err := a.RBytesValue()
	if err != nil {
		return err
	}
	h := sha3.NewLegacyKeccak256()
	h.Write(b.Bytes())
	return state.DataPush(dtrules.NewRBytes(h.Sum(nil)))
}

// opRipemd160 computes RIPEMD-160: ( bytes -- bytes(20) )
func opRipemd160(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	b, err := a.RBytesValue()
	if err != nil {
		return err
	}
	h := ripemd160.New()
	h.Write(b.Bytes())
	return state.DataPush(dtrules.NewRBytes(h.Sum(nil)))
}

// opSha3 computes SHA3-256: ( bytes -- bytes(32) )
func opSha3(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	b, err := a.RBytesValue()
	if err != nil {
		return err
	}
	h := sha3.New256()
	h.Write(b.Bytes())
	return state.DataPush(dtrules.NewRBytes(h.Sum(nil)))
}
