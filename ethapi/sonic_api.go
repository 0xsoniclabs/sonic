// Copyright 2025 Sonic Operations Ltd
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

package ethapi

import (
	"context"
	"iter"
	"math"
	"strconv"
	"strings"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/0xsoniclabs/sonic/utils/result"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

//go:generate mockgen -source=sonic_api.go -package=ethapi -destination=sonic_api_mock.go

// PublicSccApi provides an API to access certificates of the Sonic
// Certification Chain.
type PublicSccApi struct {
	backend    SccApiBackend
	maxResults int
}

func NewPublicSccApi(backend SccApiBackend) *PublicSccApi {
	return &PublicSccApi{
		backend:    backend,
		maxResults: 128, // TODO: make this a configuration option
	}
}

// GetCommitteeCertificates returns a list of certificates starting from the
// given period. The number of returned certificates is limited by the minimum
// of the requested number and the configured maximum number of results.
func (s *PublicSccApi) GetCommitteeCertificates(
	ctx context.Context,
	first PeriodNumber,
	maxResults Number,
) ([]CommitteeCertificate, error) {
	if first.IsLatest() {
		cert, err := s.backend.GetLatestCommitteeCertificate()
		return []CommitteeCertificate{
			toJsonCommitteeCertificate(cert),
		}, err
	}
	return getCertificates(
		ctx,
		s.backend.EnumerateCommitteeCertificates(first.Index()),
		toJsonCommitteeCertificate,
		maxResults,
		s.maxResults,
	)
}

// GetBlockCertificates returns a list of certificates starting from the
// given block number. The number of returned certificates is limited by the
// minimum of the requested number and the configured maximum number of results.
func (s *PublicSccApi) GetBlockCertificates(
	ctx context.Context,
	first BlockNumber,
	maxResults Number,
) ([]BlockCertificate, error) {
	if first.IsLatest() {
		cert, err := s.backend.GetLatestBlockCertificate()
		return []BlockCertificate{
			toJsonBlockCertificate(cert),
		}, err
	}
	return getCertificates(
		ctx,
		s.backend.EnumerateBlockCertificates(first.Index()),
		toJsonBlockCertificate,
		maxResults,
		s.maxResults,
	)
}

// SccApiBackend is the backend interface for the Sonic Certification Chain API.
// An implementation thereof provides access to the Sonic Certification Chain.
type SccApiBackend interface {
	GetLatestCommitteeCertificate() (cert.CommitteeCertificate, error)
	EnumerateCommitteeCertificates(first scc.Period) iter.Seq[result.T[cert.CommitteeCertificate]]

	GetLatestBlockCertificate() (cert.BlockCertificate, error)
	EnumerateBlockCertificates(first idx.Block) iter.Seq[result.T[cert.BlockCertificate]]
}

// --- Period and Block Numbers -----------------------------------------------

type PeriodNumber = index[scc.Period]
type BlockNumber = index[idx.Block]

// index is an JSON RPC argument type for uint64 numbers. It can be
// either a non-negative integer or the special value "latest". The integer
// can be in decimal, hex (0x prefix), octal (0 prefix) or binary (0b prefix).
type index[T ~uint64] struct {
	index  T
	latest bool
}

func NewIndex[T ~uint64](value T) index[T] {
	return index[T]{
		index:  value,
		latest: false,
	}
}

func NewLatest[T ~uint64]() index[T] {
	return index[T]{
		latest: true,
	}
}

// UnmarshalJSON parses the given JSON fragment into a Period. It supports:
// - "latest" as string arguments
// - the period number in hex (0x prefix), octal (0 prefix), binary (0b prefix) or decimal
// Returned errors:
// - if the given argument isn't a known strings
// - if the period number is negative
func (p *index[T]) UnmarshalJSON(data []byte) error {
	res, isMax, err := unmarshalUint64JsonString(data, "latest")
	if err != nil {
		return err
	}
	*p = index[T]{
		index:  T(res),
		latest: isMax,
	}
	return nil
}

func (p index[T]) Index() T {
	return p.index
}

func (p index[T]) IsLatest() bool {
	return p.latest
}

// --- Number -----------------------------------------------------------------

// Number is an JSON RPC argument type for an integer parameter. It can be
// either a non-negative integer or the special value "max". The integer
// can be in decimal, hex (0x prefix), octal (0 prefix) or binary (0b prefix).
type Number uint64

// UnmarshalJSON parses the given JSON fragment into a Period. It supports:
// - "max" as string arguments
// - the period number in hex (0x prefix), octal (0 prefix), binary (0b prefix) or decimal
// Returned errors:
// - if the given argument isn't a known strings
// - if the period number is negative
func (p *Number) UnmarshalJSON(data []byte) error {
	res, isMax, err := unmarshalUint64JsonString(data, "max")
	if err != nil {
		return err
	}
	if isMax {
		res = math.MaxUint64
	}
	*p = Number(res)
	return nil
}

func (p Number) UInt64() uint64 {
	return uint64(p)
}

// --- internal helpers -------------------------------------------------------

// getCertificates obtains a list of certificates from the given source and
// applies the given encoding function to each certificate. The number of
// returned certificates is limited by the minimum of the requested number and
// the configured maximum number of results. The retrieval stops when the
// limit is reached, the context is cancelled, or an error occurs.
func getCertificates[T any, R any](
	ctx context.Context,
	source iter.Seq[result.T[T]],
	encode func(T) R,
	requestedNumber Number,
	configuredLimit int,
) ([]R, error) {
	// Determine the effective limit.
	limit := configuredLimit
	if got := uint64(requestedNumber); uint64(limit) > got {
		limit = int(got)
	}
	if limit == 0 {
		return nil, nil
	}

	res := make([]R, 0, limit)
	for entry := range source {

		// Process the next certificate.
		cert, err := entry.Unwrap()
		if err != nil {
			return nil, err
		}
		res = append(res, encode(cert))
		if len(res) >= limit {
			break
		}

		// Check the context whether the client has cancelled the request.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}
	return res, nil
}

// unmarshalUint64JsonString parses the given JSON fragment into a uint64. It
// supports the special value "max" as a maximum value. The function returns
// a boolean flag indicating whether the value is the maximum value.
//
// The function accepts the following formats:
//   - decimal numbers
//   - hex numbers (0x prefix)
//   - octal numbers (0 prefix)
//   - binary numbers (0b prefix)
//   - the given name as a special value
//
// Returned errors:
//   - if the given argument isn't a known strings
//   - if the data encodes a negative number
//   - if the data encodes a number larger than math.MaxInt64
func unmarshalUint64JsonString(
	data []byte,
	nameOfMax string,
) (value uint64, isMax bool, err error) {
	input := strings.TrimSpace(string(data))
	if len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"' {
		input = input[1 : len(input)-1]
	}

	if input == nameOfMax {
		return 0, true, nil
	}

	// Parse the integer based on its prefix.
	res, err := strconv.ParseUint(input, 0, 64)
	if err != nil {
		return 0, false, err
	}
	return res, false, err
}
