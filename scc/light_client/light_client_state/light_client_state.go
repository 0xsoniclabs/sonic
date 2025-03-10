package lc_state

import (
	"fmt"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/0xsoniclabs/sonic/scc/light_client/provider"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
)

// State  holds the current state of the light client.
type State struct {
	committee  scc.Committee
	period     scc.Period
	headNumber idx.Block
	headHash   common.Hash
}

// NewState creates a new State with the given committee.
func NewState(committee scc.Committee) *State {
	return &State{
		committee: committee,
	}
}

// Head returns the block number of the latest known block.
func (s *State) Head() idx.Block {
	return s.headNumber
}

// Sync updates the light client state using data from the provider.
// This serves as the primary process for synchronizing the light client state
// with the network.
// If success, the local client will reflect the most recent block and
// its corresponding certification committee.
func (s *State) Sync(p provider.Provider) (idx.Block, error) {

	// Get the latest block number from the provider.
	blockCerts, err := p.GetBlockCertificates(provider.LatestBlock, 1)
	if err != nil {
		return 0, err
	}
	if len(blockCerts) == 0 {
		return 0, nil
	}

	// get period for the latest block
	headCert := blockCerts[0]
	headPeriod := scc.GetPeriod(headCert.Subject().Number)

	// sync from current to latest
	if err := s.syncToPeriod(p, headPeriod); err != nil {
		return 0, err
	}

	// verify latest block certificate with latest committee
	if err := headCert.Verify(s.committee); err != nil {
		return 0, err
	}

	// update the state with the latest block
	s.headNumber = headCert.Subject().Number
	s.headHash = headCert.Subject().Hash

	// return the latest block number
	return s.headNumber, nil
}

// syncToPeriod is a helper function to updates the light client state
// to the given period using the given provider
func (s *State) syncToPeriod(p provider.Provider, target scc.Period) error {
	if s.period == target {
		return nil
	}
	if s.period > target {
		return fmt.Errorf("cannot sync to a previous period. current: %d, target: %d",
			s.period, target)
	}

	// get all committee certificates from the provider
	committeeCerts, err := p.GetCommitteeCertificates(s.period+1, uint64(target-s.period))
	if err != nil {
		return err
	}

	// update the state with the committee certificates
	for _, c := range committeeCerts {
		// update the state with the committee certificate
		if err = s.updateOnePeriod(c); err != nil {
			return err
		}
	}

	return nil
}

func (s *State) updateOnePeriod(c cert.CommitteeCertificate) error {
	// verify the period
	target := s.period + 1
	if c.Subject().Period != target {
		return fmt.Errorf("unexpected committee certificate period: %d. expected: %d",
			c.Subject().Period, target)
	}

	// verify the committee certificate
	if err := c.Subject().Committee.Validate(); err != nil {
		return fmt.Errorf("committee certificate verification for period %d failed, %w",
			target, err)
	}

	// verify the committee certificate with the current committee
	if err := c.Verify(s.committee); err != nil {
		return fmt.Errorf("committee certificate verification for period %d failed, %w",
			target, err)
	}

	// update the state with the committee certificate
	s.committee = c.Subject().Committee
	s.period = target

	return nil
}
