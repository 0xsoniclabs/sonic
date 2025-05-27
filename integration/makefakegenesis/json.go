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
	"github.com/0xsoniclabs/sonic/opera/contracts/driver"
	"github.com/0xsoniclabs/sonic/opera/contracts/driver/drivercall"
	"github.com/0xsoniclabs/sonic/opera/contracts/driverauth"
	"github.com/0xsoniclabs/sonic/opera/contracts/evmwriter"
	"github.com/0xsoniclabs/sonic/opera/contracts/netinit"
	"github.com/0xsoniclabs/sonic/opera/contracts/sfc"
	"github.com/0xsoniclabs/sonic/opera/genesis"
	"github.com/0xsoniclabs/sonic/opera/genesisstore"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/bls"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/Fantom-foundation/lachesis-base/lachesis"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
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
	upgrades := opera.GetSonicUpgrades()
	decoded.Rules = opera.FakeNetRules(upgrades) // use fakenet rules as defaults
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal genesis json file; %v", err)
	}
	return &decoded, nil
}

// GenerateFakeJsonGenesis creates a JSON genesis file with fake-net rules for
// the given feature set. It includes the infrastructure contracts and a given
// number of validators with some initial tokens.
func GenerateFakeJsonGenesis(
	numValidators int,
	upgrades opera.Upgrades,
) *GenesisJson {
	jsonGenesis := &GenesisJson{
		Rules:         opera.FakeNetRules(upgrades),
		BlockZeroTime: time.Now(),
	}

	// Create infrastructure contracts.
	jsonGenesis.Accounts = []Account{
		{
			Name:    "NetworkInitializer",
			Address: netinit.ContractAddress,
			Code:    netinit.GetContractBin(),
			Nonce:   1,
		},
		{
			Name:    "NodeDriver",
			Address: driver.ContractAddress,
			Code:    driver.GetContractBin(),
			Nonce:   1,
		},
		{
			Name:    "NodeDriverAuth",
			Address: driverauth.ContractAddress,
			Code:    driverauth.GetContractBin(),
			Nonce:   1,
		},
		{
			Name:    "SFC",
			Address: sfc.ContractAddress,
			Code:    sfc.GetContractBin(),
			Nonce:   1,
		},
		{
			Name:    "ContractAddress",
			Address: evmwriter.ContractAddress,
			Code:    []byte{0},
			Nonce:   1,
		},
	}

	// Configure pre-deployed contracts, according to the hardfork of the fake-net
	if upgrades.Allegro {
		// Deploy the history storage contract
		// see: https://eips.ethereum.org/EIPS/eip-2935
		jsonGenesis.Accounts = append(jsonGenesis.Accounts, Account{
			Name:    "HistoryStorage",
			Address: params.HistoryStorageAddress,
			Code:    params.HistoryStorageCode,
			Nonce:   1,
		})
	}

	// Create the validator accounts and provide some tokens.
	tokensPerValidator := utils.ToFtm(1000000000)
	validators := GetFakeValidators(idx.Validator(numValidators))
	for _, validator := range validators {
		jsonGenesis.Accounts = append(jsonGenesis.Accounts, Account{
			Address: validator.Address,
			Balance: tokensPerValidator,
		})
	}
	totalSupply := new(big.Int).Mul(tokensPerValidator, big.NewInt(int64(numValidators)))

	var delegations []drivercall.Delegation
	for _, val := range validators {
		delegations = append(delegations, drivercall.Delegation{
			Address:            val.Address,
			ValidatorID:        val.ID,
			Stake:              utils.ToFtm(5000000),
			LockedStake:        new(big.Int),
			LockupFromEpoch:    0,
			LockupEndTime:      0,
			LockupDuration:     0,
			EarlyUnlockPenalty: new(big.Int),
			Rewards:            new(big.Int),
		})
	}

	// Create the genesis transactions.
	genesisTxs := GetGenesisTxs(0, validators, totalSupply, delegations, validators[0].Address)
	for _, tx := range genesisTxs {
		jsonGenesis.Txs = append(jsonGenesis.Txs, Transaction{
			To:   *tx.To(),
			Data: tx.Data(),
		})
	}

	// Create the genesis SCC committee.
	key := bls.NewPrivateKeyForTests(0)
	committee := scc.NewCommittee(scc.Member{
		PublicKey:         key.PublicKey(),
		ProofOfPossession: key.GetProofOfPossession(),
		VotingPower:       1,
	})

	jsonGenesis.GenesisCommittee = &committee
	return jsonGenesis
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

	if json.GenesisCommittee != nil {
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
	}

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
