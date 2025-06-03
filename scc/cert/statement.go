package cert

import (
	"encoding/binary"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
)

const (
	StatementEncodingVersion = byte(0x00)
)

// Statement is a claim that can be signed, attested, and/or certified.
type Statement interface {
	// GetDataToSign returns the data that should be signed by the issuer of the
	// statement. The data should be a deterministic serialization of the
	// statement that is unique to the statement.
	GetDataToSign() []byte
}

// statement is a common base for all statements. It contains a chain ID to
// prevent replay attacks.
type statement struct {
	ChainId uint64
}

// getDataToSign returns the data that should be signed by the issuer of the
// statement. This function is used by the GetDataToSign method of the
// implementing types to create a common prefix for the data to sign. It covers
// a encoding version number, a chain ID, and a document ID.
func (s statement) getDataToSign(documentId string) []byte {
	// Start with a version number for potential future updates.
	res := []byte{StatementEncodingVersion}

	// Continue with a application specific constant, to avoid reuse of the same
	// statement for different purposes.
	res = append(res, "scc_"...)
	res = append(res, []byte(documentId)...)

	// Add the chain ID to the statement to prevent replay attacks.
	res = binary.BigEndian.AppendUint64(res, s.ChainId)
	return res
}

// BlockStatement is a statement that a block on a given chain with a certain
// number has a certain hash and state root.
type BlockStatement struct {
	statement
	Number    idx.Block
	Hash      common.Hash
	StateRoot common.Hash
}

// NewBlockStatement creates a new block statement with the given chain ID,
// block number, hash, and state root.
func NewBlockStatement(chainID uint64, number idx.Block, hash, stateRoot common.Hash) BlockStatement {
	return BlockStatement{
		statement: statement{ChainId: chainID},
		Number:    number,
		Hash:      hash,
		StateRoot: stateRoot,
	}
}

// GetDataToSign returns the data that should be signed by the issuer of the
// statement. It follows the following format:
//   - 1 byte encoding version
//   - 6 bytes "scc_bs" constant
//   - 8 bytes chain ID in big-endian
//   - 8 bytes block number
//   - 32 bytes block hash
//   - 32 bytes state root
func (s BlockStatement) GetDataToSign() []byte {
	res := s.getDataToSign("bs")
	res = binary.BigEndian.AppendUint64(res, uint64(s.Number))
	res = append(res, s.Hash.Bytes()...)
	res = append(res, s.StateRoot.Bytes()...)
	return res
}

// CommitteeStatement is a statement that a committee was elected for a certain
// period.
type CommitteeStatement struct {
	statement
	Period    scc.Period
	Committee scc.Committee
}

func NewCommitteeStatement(chainID uint64, period scc.Period, committee scc.Committee) CommitteeStatement {
	return CommitteeStatement{
		statement: statement{ChainId: chainID},
		Period:    period,
		Committee: committee,
	}
}

// GetDataToSign returns the data that should be signed by the issuer of the
// statement. It follows the following format:
//   - 1 byte encoding version
//   - 6 bytes "scc_cs" constant
//   - 8 bytes chain ID in big-endian
//   - 8 bytes period
//   - variable length committee serialization
func (s CommitteeStatement) GetDataToSign() []byte {
	res := s.getDataToSign("cs")
	res = binary.BigEndian.AppendUint64(res, uint64(s.Period))
	res = append(res, s.Committee.Serialize()...)
	return res
}
