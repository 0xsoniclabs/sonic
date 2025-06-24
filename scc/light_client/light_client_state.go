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

package light_client

import (
	"fmt"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
)

// state holds the current state of the light client.
type state struct {
	committee  scc.Committee
	period     scc.Period
	headNumber idx.Block
	headHash   common.Hash
	headRoot   common.Hash
	hasSynced  bool
}

// newState creates a new State with the given committee.
// The given committee must be a valid committee and it is expected to be the
// committee for the genesis period.
func newState(committee scc.Committee) *state {
	return &state{
		committee: committee,
	}
}

// head returns the block number of the latest known block and a bool which
// is only true if the state has already been successfully synced.
func (s *state) head() (idx.Block, bool) {
	return s.headNumber, s.hasSynced
}

// stateRoot returns the state root of the latest known block and a bool which
// is only true if the state has already been successfully synced.
func (s *state) stateRoot() (common.Hash, bool) {
	return s.headRoot, s.hasSynced
}

// sync updates the light client state using certificates from the provider.
// This serves as the primary method for synchronizing the light client state
// with the network.
// If successful, the most recent block number is returned.
// If an error occurs, the returned block number is 0 with the corresponding error.
func (s *state) sync(p provider) (idx.Block, error) {
	if p == nil {
		return 0, fmt.Errorf("cannot update with nil provider")
	}

	// Get the latest block number from the provider.
	blockCerts, err := p.getBlockCertificates(LatestBlock, uint64(1))
	if err != nil {
		return 0, fmt.Errorf("failed to get block certificates: %w", err)
	}
	if len(blockCerts) == 0 {
		return 0, fmt.Errorf("provider returned zero block certificates")
	}

	// get period for the latest block
	headCert := blockCerts[0]
	headPeriod := scc.GetPeriod(headCert.Subject().Number)

	// if the latest block is not newer than the current head, return the current head
	if headCert.Subject().Number == s.headNumber {
		return s.headNumber, nil
	}
	if headCert.Subject().Number < s.headNumber {
		return 0, fmt.Errorf("provider returned old block head %d, expected > %d",
			headCert.Subject().Number, s.headNumber)
	}

	// sync from current period to latest.
	// this process will update the committee and period of the state.
	if err := s.syncToPeriod(p, headPeriod); err != nil {
		return 0, fmt.Errorf("failed to sync to period %d: %w", headPeriod, err)
	}

	// verify latest block certificate with latest committee
	if err := headCert.Verify(s.committee); err != nil {
		return 0,
			fmt.Errorf("failed to authenticate block certificate for block %d: %w",
				headCert.Subject().Number, err)
	}

	// update the state with the latest block
	s.headNumber = headCert.Subject().Number
	s.headHash = headCert.Subject().Hash
	s.headRoot = headCert.Subject().StateRoot
	s.hasSynced = true

	// return the latest block number
	return s.headNumber, nil
}

// syncToPeriod is a helper function to updates the light client state
// to the given period using the given provider
func (s *state) syncToPeriod(p provider, target scc.Period) error {
	if s.period == target {
		return nil
	}
	if s.period > target {
		return fmt.Errorf("cannot sync to a previous period. current: %d, target: %d",
			s.period, target)
	}

	// get all the committee certificates from the current period to the target.
	committeeCerts, err := p.getCommitteeCertificates(s.period+1, uint64(target-s.period))
	if err != nil {
		return err
	}

	for _, c := range committeeCerts {
		// update the state with the committee certificate
		if err = s.updateCommittee(c); err != nil {
			return err
		}
	}

	return nil
}

// updateCommittee is a helper function to update the light client state
// to the next period with the given certificate.
func (s *state) updateCommittee(c cert.CommitteeCertificate) error {
	// verify the period
	target := s.period + 1
	if c.Subject().Period != target {
		return fmt.Errorf("unexpected committee certificate period: %d. expected: %d",
			c.Subject().Period, target)
	}

	// verify the committee certificate
	if err := c.Subject().Committee.Validate(); err != nil {
		return fmt.Errorf("invalid committee for period %d failed, %w",
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
