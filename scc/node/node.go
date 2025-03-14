package node

import (
	"fmt"
	"sync"

	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert"
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

	// the next blocks and states to be signed by the local node.
	nextCommitteeToSign scc.Period
	nextBlockToSign     idx.Block

	// mu to protect the state of the node.
	mu sync.Mutex
}

// NewNode creates a new node with the given store.
func NewNode(store Store) (*Node, error) {
	state, err := store.GetLatestCertificationChainState()
	if err != nil {
		return nil, fmt.Errorf("failed to load SCC state, %w", err)
	}
	return &Node{store: store, state: state}, nil
}

func (n *Node) SetKey(key *bls.PrivateKey) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Find the next block and period to sign with the given key. This
	// implicitly checks that the key is known on the network.
	nextCommittee, nextBlock, err := n.state.GetSigningStateOf(key.PublicKey())
	if err != nil {
		return fmt.Errorf("failed to get signing state, %w", err)
	}

	n.key = key
	n.nextCommitteeToSign = nextCommittee
	n.nextBlockToSign = nextBlock
	return nil
}

// ProcessNewBlock should be called after a new block is added to the Sonic
// chain. It starts the creation of a corresponding block certificate and, if
// the block is the last one of the period, a new committee certificate for the
// following period. If this node is an active member of a committee, it will
// sign the certificates and return signatures to be broadcasted. If not, these
// signatures will be nil.
func (n *Node) ProcessNewBlock(stmt cert.BlockStatement) (
	[]cert.Attestation[cert.CommitteeStatement],
	[]cert.Attestation[cert.BlockStatement],
	error,
) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Update certificates in local store.
	blockCert := cert.NewCertificate(stmt)
	if err := n.store.UpdateBlockCertificate(blockCert); err != nil {
		return nil, nil, fmt.Errorf("failed to create block certificate, %w", err)
	}

	if scc.IsLastBlockOfPeriod(stmt.Number) {
		committeeStmt := cert.CommitteeStatement{
			Period:    scc.GetPeriod(stmt.Number) + 1,
			Committee: n.state.GetCurrentCommittee(),
		}
		committeeCert := cert.NewCertificate(committeeStmt)
		if err := n.store.UpdateCommitteeCertificate(committeeCert); err != nil {
			return nil, nil, fmt.Errorf("failed to create committee certificate, %w", err)
		}
	}

	// Without signing authority, there are no attestations to be made.
	if n.key == nil {
		return nil, nil, nil
	}

	// Collect out-standing signatures to be broadcasted.
	var committeeAttestations []cert.Attestation[cert.CommitteeStatement]
	curPeriod := scc.GetPeriod(stmt.Number)
	if scc.IsFirstBlockOfPeriod(stmt.Number) {
		curPeriod++
	}
	for ; n.nextCommitteeToSign <= curPeriod; n.nextCommitteeToSign++ {
		certificate, err := n.store.GetCommitteeCertificate(n.nextCommitteeToSign)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get committee certificate, %w", err)
		}
		committeeAttestations = append(committeeAttestations, cert.Attest(certificate.Subject(), *n.key))
	}

	var blockAttestations []cert.Attestation[cert.BlockStatement]
	for ; n.nextBlockToSign <= stmt.Number; n.nextBlockToSign++ {
		certificate, err := n.store.GetBlockCertificate(n.nextBlockToSign)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get block certificate, %w", err)
		}
		blockAttestations = append(blockAttestations, cert.Attest(certificate.Subject(), *n.key))
	}

	return committeeAttestations, blockAttestations, nil
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

	// TODO: clean this up
	info, found := n.state.validators[validatorId]
	if !found {
		return fmt.Errorf("validator %d not found in certification chain state", validatorId)
	}

	// TODO: check that the signature is correct
	if !signature.Verify(info.Key, cert.Subject()) {
		// TODO: add logging
		return fmt.Errorf("signature for block %d of validator %d is invalid", block, validatorId)
	}

	// TODO: do not use the current but the active committee
	// the current is the one that would become active if this is the last block of the period
	committee := n.state.GetCurrentCommittee()
	signerId, found := committee.GetMemberId(info.Key)
	if !found {
		return fmt.Errorf("validator not found in the committee")
	}

	// TODO: check the validity of the signature using the known public key of
	// the validator; needs key tracking;
	if err := cert.Add(signerId, signature); err != nil {
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
