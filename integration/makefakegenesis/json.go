package makefakegenesis

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/0xsoniclabs/sonic/integration/makegenesis"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/drivertype"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/ier"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/genesis"
	"github.com/0xsoniclabs/sonic/opera/genesisstore"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/Fantom-foundation/lachesis-base/lachesis"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type GenesisJson struct {
	Rules            opera.Rules
	BlockZeroTime    time.Time
	Accounts         []Account      `json:",omitempty"`
	Txs              []Transaction  `json:",omitempty"`
	GenesisCommittee *scc.Committee `json:",omitempty"`
}

type Account struct {
	Name    string
	Address common.Address
	Balance *big.Int                    `json:",omitempty"`
	Code    VariableLenCode             `json:",omitempty"`
	Nonce   uint64                      `json:",omitempty"`
	Storage map[common.Hash]common.Hash `json:",omitempty"`
}

type Transaction struct {
	Name string
	To   common.Address
	Data VariableLenCode `json:",omitempty"`
}

func LoadGenesisJson(filename string) (*GenesisJson, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read genesis json file; %v", err)
	}
	var decoded GenesisJson
	upgrades := opera.SonicFeatures
	decoded.Rules = opera.FakeNetRules(upgrades) // use fakenet rules as defaults
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal genesis json file; %v", err)
	}
	return &decoded, nil
}

func ApplyGenesisJson(json *GenesisJson) (*genesisstore.Store, error) {
	if json.BlockZeroTime.IsZero() {
		return nil, fmt.Errorf("block zero time must be set")
	}

	builder := makegenesis.NewGenesisBuilder()
	for _, acc := range json.Accounts {
		if acc.Balance != nil {
			builder.AddBalance(acc.Address, acc.Balance)
		}
		if acc.Code != nil {
			builder.SetCode(acc.Address, acc.Code)
		}
		if acc.Nonce != 0 {
			builder.SetNonce(acc.Address, acc.Nonce)
		}
		if acc.Storage != nil {
			for key, val := range acc.Storage {
				builder.SetStorage(acc.Address, key, val)
			}
		}
	}

	genesisTime := inter.Timestamp(json.BlockZeroTime.UnixNano())

	_, genesisStateRoot, err := builder.FinalizeBlockZero(json.Rules, genesisTime)
	if err != nil {
		return nil, err
	}

	builder.SetCurrentEpoch(ier.LlrIdxFullEpochRecord{
		LlrFullEpochRecord: ier.LlrFullEpochRecord{
			BlockState: iblockproc.BlockState{
				LastBlock: iblockproc.BlockCtx{
					Idx:     0,
					Time:    genesisTime,
					Atropos: hash.Event{},
				},
				FinalizedStateRoot:    hash.Hash(genesisStateRoot),
				EpochGas:              0,
				EpochCheaters:         lachesis.Cheaters{},
				CheatersWritten:       0,
				ValidatorStates:       make([]iblockproc.ValidatorBlockState, 0),
				NextValidatorProfiles: make(map[idx.ValidatorID]drivertype.Validator),
				DirtyRules:            nil,
				AdvanceEpochs:         0,
			},
			EpochState: iblockproc.EpochState{
				Epoch:             1,
				EpochStart:        genesisTime + 1,
				PrevEpochStart:    genesisTime,
				EpochStateRoot:    hash.Hash(genesisStateRoot),
				Validators:        pos.NewBuilder().Build(),
				ValidatorStates:   make([]iblockproc.ValidatorEpochState, 0),
				ValidatorProfiles: make(map[idx.ValidatorID]drivertype.Validator),
				Rules:             json.Rules,
			},
		},
		Idx: 1,
	})

	blockProc := makegenesis.DefaultBlockProc()
	buildTx := txBuilder()
	genesisTxs := make(types.Transactions, 0, len(json.Txs))
	for _, tx := range json.Txs {
		genesisTxs = append(genesisTxs, buildTx(tx.Data, tx.To))
	}
	err = builder.ExecuteGenesisTxs(blockProc, genesisTxs)
	if err != nil {
		return nil, fmt.Errorf("failed to execute json genesis txs; %v", err)
	}

	if json.GenesisCommittee == nil {
		return nil, fmt.Errorf("genesis committee must be set")
	}
	if len(json.GenesisCommittee.Members()) == 0 {
		return nil, fmt.Errorf("genesis committee must have at least one member")
	}
	if err := json.GenesisCommittee.Validate(); err != nil {
		return nil, fmt.Errorf("genesis committee is invalid")
	}
	builder.SetGenesisCommitteeCertificate(cert.NewCertificate(
		cert.NewCommitteeStatement(
			json.Rules.NetworkID,
			scc.Period(0),
			*json.GenesisCommittee,
		),
	))

	return builder.Build(genesis.Header{
		GenesisID:   builder.CurrentHash(),
		NetworkID:   json.Rules.NetworkID,
		NetworkName: json.Rules.Name,
	}), nil
}

type VariableLenCode []byte

func (c *VariableLenCode) MarshalJSON() ([]byte, error) {
	out := make([]byte, hex.EncodedLen(len(*c))+4)
	out[0], out[1], out[2] = '"', '0', 'x'
	hex.Encode(out[3:], *c)
	out[len(out)-1] = '"'
	return out, nil
}

func (c *VariableLenCode) UnmarshalJSON(data []byte) error {
	if !bytes.HasPrefix(data, []byte(`"`)) || !bytes.HasSuffix(data, []byte(`"`)) {
		return fmt.Errorf("code must be in a string")
	}
	data = bytes.Trim(data, "\"")
	data = bytes.TrimPrefix(data, []byte("0x"))
	decoded := make([]byte, hex.DecodedLen(len(data)))
	_, err := hex.Decode(decoded, data)
	if err != nil {
		return err
	}
	*c = decoded
	return nil
}
