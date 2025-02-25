package ethapi

import (
	"context"
	"errors"
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

// PublicSccAPI provides an API to access certificates of the Sonic
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
) ([]committeeCertificateJson, error) {
	return getCertificates(
		ctx,
		s.backend.EnumerateCommitteeCertificates(first.Period()),
		func(cert cert.CommitteeCertificate) committeeCertificateJson {
			return committeeCertificateJson{cert}
		},
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
) ([]blockCertificateJson, error) {
	return getCertificates(
		ctx,
		s.backend.EnumerateBlockCertificates(first.Block()),
		func(cert cert.BlockCertificate) blockCertificateJson {
			return blockCertificateJson{cert}
		},
		maxResults,
		s.maxResults,
	)
}

type SccApiBackend interface {
	EnumerateCommitteeCertificates(first scc.Period) iter.Seq[result.T[cert.CommitteeCertificate]]
	EnumerateBlockCertificates(first idx.Block) iter.Seq[result.T[cert.BlockCertificate]]
}

// TODO: replace with actual JSON serialization

type committeeCertificateJson struct {
	cert.CommitteeCertificate
}

type blockCertificateJson struct {
	cert.BlockCertificate
}

// --- PeriodNumber -----------------------------------------------------------

// PeriodNumber is an JSON RPC argument type for period numbers. It can be
// either a non-negative integer or the special value "latest". The integer
// can be in decimal, hex (0x prefix), octal (0 prefix) or binary (0b prefix).
type PeriodNumber int64

const (
	HeadPeriod = PeriodNumber(-1)
)

// UnmarshalJSON parses the given JSON fragment into a Period. It supports:
// - "latest" as string arguments
// - the period number in hex (0x prefix), octal (0 prefix), binary (0b prefix) or decimal
// Returned errors:
// - if the given argument isn't a known strings
// - if the period number is negative
func (p *PeriodNumber) UnmarshalJSON(data []byte) error {
	res, err := unmarshalPositiveIntegerJsonString(data, "latest")
	if err != nil {
		return err
	}
	*p = PeriodNumber(res)
	return nil
}

func (p PeriodNumber) IsHead() bool {
	return p == HeadPeriod
}

// Period returns the period number as a period.
func (p PeriodNumber) Period() scc.Period {
	return scc.Period(p)
}

// --- BlockNumber ------------------------------------------------------------

// BlockNumber is an JSON RPC argument type for block numbers. It can be
// either a non-negative integer or the special value "latest". The integer
// can be in decimal, hex (0x prefix), octal (0 prefix) or binary (0b prefix).
type BlockNumber int64

const (
	HeadBlock = BlockNumber(-1)
)

// UnmarshalJSON parses the given JSON fragment into a BlockNumber. It supports:
// - "latest" as string arguments
// - the block number in hex (0x prefix), octal (0 prefix), binary (0b prefix) or decimal
// Returned errors:
// - if the given argument isn't a known strings
// - if the period number is negative
func (p *BlockNumber) UnmarshalJSON(data []byte) error {
	res, err := unmarshalPositiveIntegerJsonString(data, "latest")
	if err != nil {
		return err
	}
	*p = BlockNumber(res)
	return nil
}

func (p BlockNumber) IsHead() bool {
	return p == HeadBlock
}

// Block returns the block number as a block.
func (p BlockNumber) Block() idx.Block {
	return idx.Block(p)
}

// --- Number -----------------------------------------------------------------

// Number is an JSON RPC argument type for an integer parameter. It can be
// either a non-negative integer or the special value "max". The integer
// can be in decimal, hex (0x prefix), octal (0 prefix) or binary (0b prefix).
type Number int64

// UnmarshalJSON parses the given JSON fragment into a Period. It supports:
// - "max" as string arguments
// - the period number in hex (0x prefix), octal (0 prefix), binary (0b prefix) or decimal
// Returned errors:
// - if the given argument isn't a known strings
// - if the period number is negative
func (p *Number) UnmarshalJSON(data []byte) error {
	res, err := unmarshalPositiveIntegerJsonString(data, "max")
	if err != nil {
		return err
	}
	if res < 0 {
		res = math.MaxInt64
	}
	*p = Number(res)
	return nil
}

// Period returns the period number as a period.
func (p Number) Int64() int64 {
	return int64(p)
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
	if got := requestedNumber.Int64(); int64(limit) > got {
		limit = int(got)
	}
	if limit == 0 {
		return nil, nil
	}
	// TODO: add support for the latest block!

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

// unmarshalPositiveIntegerJsonString parses the given JSON fragment into a
// positive integer or -1 if the string is equal to the given name.
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
func unmarshalPositiveIntegerJsonString(data []byte, nameOfMax string) (int64, error) {
	input := strings.TrimSpace(string(data))
	if len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"' {
		input = input[1 : len(input)-1]
	}

	if input == nameOfMax {
		return -1, nil
	}

	// Parse the integer based on its prefix.
	res, err := strconv.ParseInt(input, 0, 64)
	if err != nil {
		return 0, err
	}
	if res < 0 {
		return 0, errors.New("number of elements cannot be negative")
	}
	return res, err
}
