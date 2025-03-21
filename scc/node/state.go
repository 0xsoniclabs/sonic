package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
)

//go:generate mockgen -source=state.go -destination=state_mock.go -package=node

type State struct {
	blockHeight idx.Block
	validators  map[idx.ValidatorID]validatorInfo
}

type validatorInfo struct {
	Valid                            bool
	Key                              bls.PublicKey
	ProofOfPossession                bls.Signature
	VotingPower                      uint64
	NextExpectedCommitteeAttestation scc.Period
	NextExpectedBlockAttestation     idx.Block
}

type GenesisValidator struct {
	Key               bls.PublicKey
	ProofOfPossession bls.Signature
}

func NewGenesisState(validators map[idx.ValidatorID]GenesisValidator) (State, error) {
	infos := make(map[idx.ValidatorID]validatorInfo, len(validators))
	for id, info := range validators {
		if !info.Key.Validate() {
			return State{}, fmt.Errorf("invalid key for validator %d", id)
		}
		if !info.Key.CheckProofOfPossession(info.ProofOfPossession) {
			return State{}, fmt.Errorf("invalid proof of possession for validator %d", id)
		}
		infos[id] = validatorInfo{
			Valid:             true,
			Key:               info.Key,
			ProofOfPossession: info.ProofOfPossession,
			VotingPower:       1,
		}
	}
	return State{
		blockHeight: 0,
		validators:  infos,
	}, nil
}

func NewFakeGenesisState(numValidators int) (State, error) {
	validators := make(map[idx.ValidatorID]GenesisValidator, numValidators)
	for id := range idx.ValidatorID(numValidators) {
		key := bls.NewPrivateKeyForTests(byte(id))
		validators[id+1] = GenesisValidator{
			Key:               key.PublicKey(),
			ProofOfPossession: key.GetProofOfPossession(),
		}
	}
	return NewGenesisState(validators)
}

func (s *State) GetBlockHeight() idx.Block {
	return s.blockHeight
}

func (s *State) GetCurrentCommittee() scc.Committee {
	// Every validator with a non-zero voting power is a member of the committee.
	members := make([]scc.Member, 0, len(s.validators))
	for _, info := range s.validators {
		if !info.Valid || info.VotingPower == 0 {
			continue
		}
		members = append(members, scc.Member{
			PublicKey:   info.Key,
			VotingPower: info.VotingPower,
		})
	}

	// Committee members are sorted by their public key.
	slices.SortFunc(members, func(a, b scc.Member) int {
		keyA := a.PublicKey.Serialize()
		keyB := b.PublicKey.Serialize()
		return bytes.Compare(keyA[:], keyB[:])
	})

	return scc.NewCommittee(members...)
}

func (s *State) GetSigningStateOf(key bls.PublicKey) (
	scc.Period,
	idx.Block,
	error,
) {
	for _, info := range s.validators {
		if info.Key == key {
			return info.NextExpectedCommitteeAttestation,
				info.NextExpectedBlockAttestation, nil
		}
	}
	return 0, 0, fmt.Errorf("unknown signer")
}

func (s State) Serialize() ([]byte, error) {
	// TODO: consider replacing this with a Protobuf serialization.
	return s.MarshalJSON()
}

func (s *State) Deserialize(data []byte) error {
	return s.UnmarshalJSON(data)
}

func (s *State) MarshalJSON() ([]byte, error) {
	return json.Marshal(storeJson{
		BlockHeight: s.blockHeight,
		Validators:  s.validators,
	})
}

func (s *State) UnmarshalJSON(data []byte) error {
	var store storeJson
	if err := json.Unmarshal(data, &store); err != nil {
		return err
	}
	s.blockHeight = store.BlockHeight
	s.validators = store.Validators
	return nil
}

type storeJson struct {
	BlockHeight idx.Block
	Validators  map[idx.ValidatorID]validatorInfo
}
