package node

import (
	"fmt"
	"sync"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

// Node is a node in the Sonic Certification Chain. It is responsible for
// handling the progression of the chain by responding to new block statements
// and creating new certificates.
type Node struct {
	store Store
	state State

	// key is the private key used by this node to sign certificates. It is nil
	// if the node is not part of a committee.
	key *bls.PrivateKey

	// mu to protect the state of the node.
	mu sync.Mutex
}

// NewNode creates a new node with the given store and committee. The provided
// committee is expected to be the active committee of the current block.
func NewNode(store Store, committee scc.Committee) *Node {
	return &Node{
		store: store,
		state: newInMemoryState(committee),
	}
}

func (n *Node) SetKey(key *bls.PrivateKey) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.key = key
}

// ProcessNewBlock should be called after a new block is added to the Sonic
// chain. It starts the creation of a corresponding block certificate and, if
// the block is the last one of the period, a new committee certificate for the
// following period. If this node is an active member of a committee, it will
// sign the certificates and return signatures to be broadcasted. If not, these
// signatures will be nil.
func (n *Node) ProcessNewBlock(stmt cert.BlockStatement) (
	*cert.Signature[cert.BlockStatement],
	*cert.Signature[cert.CommitteeStatement],
	error,
) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Register a new block certificate and sign it if this is a active node.
	blockCert := cert.NewCertificate(stmt)
	if err := n.store.UpdateBlockCertificate(blockCert); err != nil {
		return nil, nil, fmt.Errorf("failed to create block certificate, %w", err)
	}
	var blockSignature *cert.Signature[cert.BlockStatement]
	if n.key != nil {
		tmp := cert.Sign(stmt, *n.key)
		blockSignature = &tmp
	}
	if !scc.IsLastBlockOfPeriod(stmt.Number) {
		return blockSignature, nil, nil
	}

	// At the end of the block, create a new committee certificate and sign it.
	committeeStmt := cert.CommitteeStatement{
		Period:    scc.GetPeriod(stmt.Number) + 1,
		Committee: n.state.GetCurrentCommittee(),
	}
	committeeCert := cert.NewCertificate(committeeStmt)
	if err := n.store.UpdateCommitteeCertificate(committeeCert); err != nil {
		return nil, nil, fmt.Errorf("failed to create committee certificate, %w", err)
	}
	var committeeSignature *cert.Signature[cert.CommitteeStatement]
	if n.key != nil {
		tmp := cert.Sign(committeeStmt, *n.key)
		committeeSignature = &tmp
	}

	return blockSignature, committeeSignature, nil
}

// ProcessIncomingBlockSignature should be called when a new block signature is
// received over the network. It adds the signature to the block certificate
// store in the local DB if the block is known, the signature is valid, and no
// signature for the given validator is already present.
func (n *Node) ProcessIncomingBlockSignature(
	validatorId idx.ValidatorID,
	block idx.Block,
	signature cert.Signature[cert.BlockStatement],
) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	cert, err := n.store.GetBlockCertificate(block)
	if err != nil {
		return fmt.Errorf("failed to get block certificate, %w", err)
	}
	// TODO: check the validity of the signature using the known public key of
	// the validator; needs key tracking;
	if err := cert.Add(0, signature); err != nil {
		return fmt.Errorf("failed to add signature to block certificate, %w", err)
	}
	return n.store.UpdateBlockCertificate(cert)
}

// ProcessIncomingCommitteeSignature should be called when a new committee
// signature is received over the network. It adds the signature to the
// committee certificate stored in the local DB if the period is known, the
// signature is valid, and no signature for the given validator is already
// present.
func (n *Node) ProcessIncomingCommitteeSignature(
	validatorId idx.ValidatorID,
	period scc.Period,
	signature cert.Signature[cert.CommitteeStatement],
) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	cert, err := n.store.GetCommitteeCertificate(period)
	if err != nil {
		return fmt.Errorf("failed to get committee certificate, %w", err)
	}
	// TODO: check the validity of the signature using the known public key of
	// the validator; needs key tracking;
	if err := cert.Add(0, signature); err != nil {
		return fmt.Errorf("failed to add signature to committee certificate, %w", err)
	}
	return n.store.UpdateCommitteeCertificate(cert)
}
