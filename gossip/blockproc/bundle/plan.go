package bundle

import (
	"bytes"
	"errors"
	"io"
	"maps"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

type TransactionBundle2 struct {
	Transactions map[TxReference]*types.Transaction
	Plan         ExecutionPlan2
}

const (
	bundleEncodingVersion2 byte = 2
)

type bundleEncodingV2 struct {
	Bundle types.Transactions
	Plan   []byte
}

func (b *TransactionBundle2) Encode() []byte {

	// Create canonical form of list of included transactions.
	transactions := slices.Collect(maps.Values(b.Transactions))
	slices.SortFunc(transactions, func(a, b *types.Transaction) int {
		hashA := a.Hash()
		hashB := b.Hash()
		return bytes.Compare(hashA[:], hashB[:])
	})

	// TODO: check error handling
	encodedPlan := bytes.NewBuffer(nil)
	_ = b.Plan.encode(encodedPlan)

	buffer := bytes.Buffer{}
	// encode into a buffer can only fail due to OOM
	// since we are encoding a struct with fixed fields, we can ignore the error
	_ = rlp.Encode(&buffer, bundleEncodingVersion2)
	_ = rlp.Encode(&buffer, bundleEncodingV2{
		transactions,
		encodedPlan.Bytes(),
	})
	return buffer.Bytes()
}

func (b *TransactionBundle2) Decode(data []byte) error {
	var version byte
	if err := rlp.DecodeBytes(data, &version); err != nil {
		return err
	}
	if version != bundleEncodingVersion2 {
		return errors.New("unsupported bundle encoding version")
	}

	var decoded bundleEncodingV2
	if err := rlp.DecodeBytes(data, &decoded); err != nil {
		return err
	}

	b.Transactions = make(map[TxReference]*types.Transaction)
	for _, tx := range decoded.Bundle {
		if !tx.Protected() {
			return errors.New("unsupported transaction type in bundle")
		}
		signer := types.LatestSignerForChainID(tx.ChainId())
		sender, err := types.Sender(signer, tx)
		if err != nil {
			return err
		}
		txData := TxReference{
			From: sender,
			Hash: tx.Hash(),
		}
		b.Transactions[txData] = tx
	}

	return b.Plan.decode(bytes.NewReader(decoded.Plan))
}

type ExecutionPlan2 struct {
	Group Group
	Range BlockRange
}

func (p *ExecutionPlan2) Hash() common.Hash {
	hasher := crypto.NewKeccakState()
	_ = p.encode(hasher)
	return common.BytesToHash(hasher.Sum(nil))
}

func (p *ExecutionPlan2) encode(writer io.Writer) error {
	return errors.Join(
		p.Group.encode(writer),
		p.Range.encode(writer),
	)
}

func (p *ExecutionPlan2) decode(reader io.Reader) error {
	return errors.Join(
		p.Group.decode(reader),
		p.Range.decode(reader),
	)
}

type Group struct {
	Flags ExecutionFlags
	Steps []GroupOrTransaction
}

const (
	groupEncodingMarker       byte = 0x00
	txReferenceEncodingMarker byte = 0x01
)

func (g *Group) encode(writer io.Writer) error {
	_, err := writer.Write([]byte{
		byte(g.Flags),
		byte(len(g.Steps)), // TODO: limit number of steps to 255
	})
	if err != nil {
		return err
	}
	for _, step := range g.Steps {
		var mark byte
		switch step.(type) {
		case *Group:
			mark = groupEncodingMarker
		case *TxReference:
			mark = txReferenceEncodingMarker
		}
		_, err = writer.Write([]byte{mark})
		if err != nil {
			return err
		}
		if err := step.encode(writer); err != nil {
			return err
		}
	}
	return err
}

func (g *Group) decode(reader io.Reader) error {
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		return err
	}
	g.Flags = ExecutionFlags(header[0])
	nSteps := int(header[1])
	g.Steps = make([]GroupOrTransaction, 0, nSteps)
	for range nSteps {
		marker := make([]byte, 1)
		if _, err := io.ReadFull(reader, marker); err != nil {
			return err
		}
		var step GroupOrTransaction
		switch marker[0] {
		case groupEncodingMarker:
			step = &Group{}
		case txReferenceEncodingMarker:
			step = &TxReference{}
		default:
			return errors.New("unknown step marker")
		}
		if err := step.decode(reader); err != nil {
			return err
		}
		g.Steps = append(g.Steps, step)
	}
	return nil
}

// TxReference represents a single step in an execution plan, referencing a
// transaction to be processed at this point of the plan.
type TxReference struct {
	// From is the sender of the transaction.
	From common.Address
	// Hash is the transaction hash to be signed (not the hash of the
	// transaction including its signature) where the bundle-only marker has
	// been removed.
	Hash common.Hash
}

func (t *TxReference) encode(writer io.Writer) error {
	_, err1 := writer.Write(t.From.Bytes())
	_, err2 := writer.Write(t.Hash.Bytes())
	return errors.Join(err1, err2)
}

func (t *TxReference) decode(reader io.Reader) error {
	from := make([]byte, common.AddressLength)
	if _, err := io.ReadFull(reader, from); err != nil {
		return err
	}
	hash := make([]byte, common.HashLength)
	if _, err := io.ReadFull(reader, hash); err != nil {
		return err
	}
	t.From = common.BytesToAddress(from)
	t.Hash = common.BytesToHash(hash)
	return nil
}

type GroupOrTransaction interface {
	encode(writer io.Writer) error
	decode(reader io.Reader) error
}

var _ GroupOrTransaction = (*Group)(nil)
var _ GroupOrTransaction = (*TxReference)(nil)

// TODO:
// - implement plan serialization
// - implement plan hashing
// - implement plan debugging
// - implement bundle validation
// - implement bundle execution
// - implement bundle builder for new format
