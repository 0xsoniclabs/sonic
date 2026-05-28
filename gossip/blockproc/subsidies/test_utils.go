// Copyright 2026 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

package subsidies

// This file contains utilities for testing post-subsidy transactions in this
// and other packages. It includes a builder for creating post-subsidy
// transactions with different configurations for testing purposes and provides
// access to internal constants.

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// GetMaxPostTxSizeForTests returns the maximum size of post-subsidy
// transactions for testing purposes.
func GetMaxPostTxSizeForTests() uint64 {
	return (Sponsorship{
		mode: sponsorshipModeFundBacked,
	}).Overhead().Size
}

// PostTxBuilder is a helper for building post-subsidies transactions for testing.
type PostTxBuilder struct {
	mode       *sponsorshipMode
	overhead   *uint64
	gasLimit   *uint64
	identifier *Identifier
	usedGas    *uint64
	gasPrice   *big.Int
	nonce      *uint64
}

func NewPostTxBuilder() *PostTxBuilder {
	return &PostTxBuilder{}
}

func (b *PostTxBuilder) ForFundBacked() *PostTxBuilder {
	mode := sponsorshipModeFundBacked
	b.mode = &mode
	return b
}

func (b *PostTxBuilder) ForNetworkSponsoredWithTracking() *PostTxBuilder {
	mode := sponsorshipModeNetworkWithTracking
	b.mode = &mode
	return b
}

func (b *PostTxBuilder) WithId(id Identifier) *PostTxBuilder {
	b.identifier = &id
	return b
}

func (b *PostTxBuilder) WithOverhead(overhead uint64) *PostTxBuilder {
	b.overhead = &overhead
	return b
}

func (b *PostTxBuilder) WithGasLimit(gasLimit uint64) *PostTxBuilder {
	b.gasLimit = &gasLimit
	return b
}

func (b *PostTxBuilder) WithUsedGas(usedGas uint64) *PostTxBuilder {
	b.usedGas = &usedGas
	return b
}

func (b *PostTxBuilder) WithGasPrice(gasPrice *big.Int) *PostTxBuilder {
	b.gasPrice = gasPrice
	return b
}

func (b *PostTxBuilder) WithNonce(nonce uint64) *PostTxBuilder {
	b.nonce = &nonce
	return b
}

func (b *PostTxBuilder) BuildForTesting() *types.Transaction {

	mode := sponsorshipModeFundBacked
	if b.mode != nil {
		mode = *b.mode
	}

	id := Identifier{0x01, 0x02, 0x03}
	if b.identifier != nil {
		id = *b.identifier
	}

	config := gasConfig{
		gasLimitForDeductFees: 100_000,
		gasLimitForTrack:      150_000,
	}
	if b.overhead != nil {
		config.overheadFundBackedSponsoring = *b.overhead
		config.overheadNetworkTrackedSponsoring = *b.overhead
	}

	if b.gasLimit != nil {
		config.gasLimitForDeductFees = *b.gasLimit
		config.gasLimitForTrack = *b.gasLimit
	}

	gasUsed := uint64(123)
	if b.usedGas != nil {
		gasUsed = *b.usedGas
	}

	gasPrice := big.NewInt(456)
	if b.gasPrice != nil {
		gasPrice = b.gasPrice
	}

	nonce := uint64(12)
	if b.nonce != nil {
		nonce = *b.nonce
	}

	nonceSource := &fakeNonceSource{
		nonce: nonce,
	}

	sponsorship := Sponsorship{
		mode:   mode,
		id:     id,
		config: config,
	}
	tx, err := sponsorship.GetPostTransactions(nonceSource, gasUsed, gasPrice)
	if err != nil || len(tx) == 0 {
		panic(fmt.Sprintf("failed to create post-sponsorship transaction for testing: %v", err))
	}
	return tx[0]
}

type fakeNonceSource struct {
	nonce uint64
}

func (f *fakeNonceSource) GetNonce(addr common.Address) uint64 {
	return f.nonce
}
