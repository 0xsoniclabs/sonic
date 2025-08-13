package rpctest

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestRpc(t *testing.T) {

	net := tests.StartIntegrationTestNetWithJsonGenesis(t)

	scenarios := map[string]scenario{
		"cancun": {
			contracts: map[string]contract{
				"transient": {
					abi:      `[{"inputs":[],"stateMutability":"nonpayable","type":"constructor"},{"inputs":[],"name":"testVal","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"v","type":"uint256"}],"name":"transientStore","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"nonpayable","type":"function"}]`,
					bytecode: "0x6080604052348015600e575f5ffd5b506040516019906074565b604051809103905ff0801580156031573d5f5f3e3d5ffd5b505f5f6101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506081565b610168806103e783390190565b6103598061008e5f395ff3fe608060405234801561000f575f5ffd5b5060043610610034575f3560e01c806377001dd614610038578063cbe929a114610056575b5f5ffd5b610040610086565b60405161004d919061024c565b60405180910390f35b610070600480360381019061006b9190610293565b610119565b60405161007d919061024c565b60405180910390f35b5f5f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16633c6bb4366040518163ffffffff1660e01b8152600401602060405180830381865afa1580156100f0573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061011491906102d2565b905090565b5f5f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16636057361d836040518263ffffffff1660e01b8152600401610173919061024c565b5f604051808303815f87803b15801561018a575f5ffd5b505af115801561019c573d5f5f3e3d5ffd5b505050505f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16633c6bb4366040518163ffffffff1660e01b8152600401602060405180830381865afa158015610209573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061022d91906102d2565b9050919050565b5f819050919050565b61024681610234565b82525050565b5f60208201905061025f5f83018461023d565b92915050565b5f5ffd5b61027281610234565b811461027c575f5ffd5b50565b5f8135905061028d81610269565b92915050565b5f602082840312156102a8576102a7610265565b5b5f6102b58482850161027f565b91505092915050565b5f815190506102cc81610269565b92915050565b5f602082840312156102e7576102e6610265565b5b5f6102f4848285016102be565b9150509291505056fea264697066735822122001e6db08fe6c450db4ae197d9cbd8dcdcc33630a693647fe94a9d4cf8e3e6fe164736f6c637828302e382e32392d646576656c6f702e323032342e31312e312b636f6d6d69742e666361306264333100596080604052348015600e575f5ffd5b5061014c8061001c5f395ff3fe6080604052348015600e575f5ffd5b50600436106030575f3560e01c80633c6bb4361460345780636057361d14604e575b5f5ffd5b603a6066565b60405160459190608a565b60405180910390f35b606460048036038101906060919060ca565b606e565b005b5f5f5c905090565b805f5d50565b5f819050919050565b6084816074565b82525050565b5f602082019050609b5f830184607d565b92915050565b5f5ffd5b60ac816074565b811460b5575f5ffd5b50565b5f8135905060c48160a5565b92915050565b5f6020828403121560dc5760db60a1565b5b5f60e78482850160b8565b9150509291505056fea2646970667358221220f9c9312ee992efb7beb50187fb3d4e5d9648f7b3ad87c4971bae457baf878a8e64736f6c637828302e382e32392d646576656c6f702e323032342e31312e312b636f6d6d69742e66636130626433310059",
				},
				"withness": {
					abi:      `[{"inputs":[],"name":"getData","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"},{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"bytes32","name":"_data","type":"bytes32"},{"internalType":"uint256","name":"_data2","type":"uint256"}],"name":"storeData","outputs":[],"stateMutability":"nonpayable","type":"function"}]`,
					bytecode: "0x6080604052348015600e575f5ffd5b506101dc8061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610034575f3560e01c8063379d7010146100385780633bc5de3014610054575b5f5ffd5b610052600480360381019061004d91906100fd565b610073565b005b61005c610084565b60405161006a929190610159565b60405180910390f35b815f81905550806001819055505050565b5f5f5f54600154915091509091565b5f5ffd5b5f819050919050565b6100a981610097565b81146100b3575f5ffd5b50565b5f813590506100c4816100a0565b92915050565b5f819050919050565b6100dc816100ca565b81146100e6575f5ffd5b50565b5f813590506100f7816100d3565b92915050565b5f5f6040838503121561011357610112610093565b5b5f610120858286016100b6565b9250506020610131858286016100e9565b9150509250929050565b61014481610097565b82525050565b610153816100ca565b82525050565b5f60408201905061016c5f83018561013b565b610179602083018461014a565b939250505056fea26469706673582212201806040da0ead935dd8ef956ca230a7b70191b1c58e84e9c416fffbc8f2ab26f64736f6c637828302e382e32392d646576656c6f702e323032342e31312e312b636f6d6d69742e66636130626433310059",
				},
			},
			transactions: []transaction{
				{
					contract: "transient",
					method:   "transientStore",
					args:     []any{987654},
				},
				{
					contract: "withness",
					method:   "storeData",
					args:     []any{"0x61686f6a00000000000000000000000000000000000000000000000000000000", 987654},
				},
			},

			testCases: map[string]testCase{
				"Get transient storage test call result": {
					contract: "transient",
					method:   "debug_traceTransaction",

					// arguments:  []any{"0x239fa7623354ec26520de878b52f13fe84b06971", {"timeout": "2m", "tracer": "callTracer"}},
				},

				// 	- test_name: "Get transient storage test call result"
				//     call_id: 1
				//     method_name: debug_traceTransaction
				//     arguments: [tx.hash,{"timeout":"2m","tracer":"callTracer"}]
				//     result: {"from":"0x239fa7623354ec26520de878b52f13fe84b06971","gas":"0x6f3a","gasUsed":"0x6f3a","to":"0x86989606a88841e2fb56787806a5f7a3a89641e2","input":"0xcbe929a100000000000000000000000000000000000000000000000000000000000f1206","output":"0x00000000000000000000000000000000000000000000000000000000000f1206","calls":[{"from":"0x86989606a88841e2fb56787806a5f7a3a89641e2","gas":"0x6c6","gasUsed":"0x1fa","to":"0x831cd2c34e3c50d10c9fd84bd893942c97b6d5a0","input":"0x6057361d00000000000000000000000000000000000000000000000000000000000f1206","value":"0x0","type":"CALL"},{"from":"0x86989606a88841e2fb56787806a5f7a3a89641e2","gas":"0x38b","gasUsed":"0x19a","to":"0x831cd2c34e3c50d10c9fd84bd893942c97b6d5a0","input":"0x3c6bb436","output":"0x00000000000000000000000000000000000000000000000000000000000f1206","type":"STATICCALL"}],"value":"0x0","type":"CALL"}
				//     ignore_fields: ["to", "from","gas","gasUsed"]

				//   - test_name: "Get proof of contract storage position 0x0"
				//     call_id: 2
				//     contract_id: 2
				//     method_name: eth_getProof
				//     arguments: [contract.address, ["0x0"], "latest"]
				//     result: { "accountProof": ["0xf90211a02413593ac746477651bbb9d4fa8e8e1afa17f2b8b8c0662e04ed3dd1cb84d977a0f866eeef7beb0aa8d8531b6ccab024f20f4948627c732641d332cf115784500da07c9f814ab75a5efbc975b716d9237be9bbff04dfa7a037c5a1a9f77efb4ddd6ea02e2840ceb4da5687c81e1ec4f1c1ca1c4c366638a05f2e261b145d44e027f2fba0edc40eefbbb5f6abb9afc9d28282dd4144a0a2ca22bd020061ad47e04382867ba03e2ec41dff86c9f11907420c228328147ed1ecb122b790b8f2c3ea3a93e37578a0e61681efbb63a7f4773b7e529ae32d6145e6cbb1116af82230dd5a0f319ac510a044f11f602ff95e130fc480d4376ce40b9064ed5ff0afcd389242d118a173ea2aa0c98dab4d7b244e658f8967a1556602a0e2ce51348b7f67efee0912f250ec9e48a0b38dcbafe52889df1bd116574cb91706f88c45c63f0a7bd69963fccdd658960aa09ae23f940a32bb29c5e5cc1e12eaa56fbdcb2e5dc38f5ab1e6d694373ee65d69a0083ff38d81733a35e0ffe679cf033922b52af8e0a73024ce39bf5899dfb9fe81a05b068f3904e9d96fbe08277c75f1b26b34bd4dabcb29a83c6b12819cecbb3b93a09e2cbbdf659ed462a3efa872aff602794634555dc5055f201df0f672c033c4e0a0683921fef1c7e92c5edc2e3630b8256e0c40aa4002e49dd33a467911a7aa1647a0c474add54993eb6be38ed04ac850223b7cc579bd297fb2e98334d15dbee4f58580", "0xf89180808080a01884fdbcd1baf2a3071971dd122ab2bf54fffdc666deb145a871a6c3054b9c3880a04ec9440a03758730e2bb6d7a866362014e669ac788cbb9a0de18b668e710f29880808080a0de1e091578b17dd369b08bbb31c139dcdc83c99fa3391ae74d87daac3102c8e18080a0c55a33a98a3a84613e0ce23227b0ad7fafa2a64b4c9f22bdea942ef295f6e30f8080", "0xf869a0207f659e7d55bf7dab219ff4b89b17ea40b2b3e994858295da0e0adf2d279252b846f8440180a0bafb58a40f8f3c01cfd71f9012a5def778829f09a85401593361bfea4152ace6a028ed02873b0baeb83b356d2f2560df83f56084ffdb097e32d653df9b16773a1a"], "address": "0x04dabb381b152755d0930310269da3ddef8938bb", "balance": "0x0", "codeHash": "0x83311ca9d05d239d0ff13effb5a4b4f89ca1788e8060293bb794049ed7e41085", "nonce": "0x1", "storageHash": "0xbafb58a40f8f3c01cfd71f9012a5def778829f09a85401593361bfea4152ace6", "storageProof": [{"key": "0x0000000000000000000000000000000000000000000000000000000000000000", "proof": ["0xf8518080a0699553c51e84f8424a08e2e5331d1f979fab75f3d02680b374ddb2f8240f619b8080808080808080a09a04a68a869e458b1ef15d154cce994a65f31e677aafcc18b1f02385f92bad688080808080", "0xf843a0390decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563a1a061686f6a00000000000000000000000000000000000000000000000000000000"], "value": "0x61686f6a00000000000000000000000000000000000000000000000000000000"}]}
				//     ignore_fields: ["accountProof","proof", "address"] # TODO: order of proof values can be different and test fails
			},
		},
	}

	for name, scenario := range scenarios {
		t.Run(name, func(t *testing.T) {
			scenario.setup(t, net)
			for name, testCase := range scenario.testCases {
				t.Run(name, func(t *testing.T) {
					_ = testCase
				})
			}
		})
	}
}

type scenario struct {
	contracts    map[string]contract
	transactions []transaction
	testCases    map[string]testCase
}

func (s *scenario) setup(t *testing.T, session tests.IntegrationTestNetSession) {
	t.Helper()

	contracts := make(map[string]boundContract)

	// deploy contracts
	for name, contract := range s.contracts {
		contracts[name] = contract.deploy(t, session)
	}

	// execute scenario history of transactions
	for _, tx := range s.transactions {
		tx.execute(t, session, contracts)
	}
}

type contract struct {
	abi      string
	bytecode string
}

func (c *contract) deploy(
	t testing.TB,
	session tests.IntegrationTestNetSession,
) boundContract {
	t.Helper()

	opts, err := session.GetTransactOptions(session.GetSessionSponsor())
	require.NoError(t, err)

	client, err := session.GetClient()
	require.NoError(t, err)

	var abi abi.ABI
	err = json.Unmarshal([]byte(c.abi), &abi)
	require.NoError(t, err)

	bytecode := hexutil.MustDecode(c.bytecode)

	address, tx, _, err := bind.DeployContract(opts, abi, bytecode, client)
	require.NoError(t, err, "failed to deploy contract; %v", err)

	receipt, err := session.GetReceipt(tx.Hash())
	require.NoError(t, err, "failed to get receipt for contract deployment")
	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful, "contract deployment failed")

	return boundContract{abi, address}
}

type transaction struct {
	contract string
	method   string
	args     []any
}

func (tx *transaction) execute(
	t *testing.T,
	session tests.IntegrationTestNetSession,
	contracts map[string]boundContract,
) {
	t.Helper()

	contract, ok := contracts[tx.contract]
	require.True(t, ok, "contract not found")

	args, err := ConvertArgumentsWithAbi(&contract.abi, tx.method, tx.args)
	require.NoError(t, err, "failed to convert arguments for method %s", tx.method)

	client, err := session.GetClient()
	require.NoError(t, err, "failed to get client")

	boundContract := bind.NewBoundContract(contract.address, contract.abi, client, client, client)

	opts, err := session.GetTransactOptions(session.GetSessionSponsor())
	require.NoError(t, err, "failed to get transaction options")

	sentTx, err := boundContract.Transact(opts, tx.method, args...)
	require.NoError(t, err, "failed to execute transaction for method %s", tx.method)

	receipt, err := session.GetReceipt(sentTx.Hash())
	require.NoError(t, err, "failed to execute transaction")
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "transaction failed")
}

type testCase struct {
	contract string
	method   string
}

type boundContract struct {
	abi     abi.ABI
	address common.Address
}

func ConvertArgumentsWithAbi(contractABI *abi.ABI, methodName string, arguments []any) ([]interface{}, error) {

	method := contractABI.Methods[methodName]
	args := make([]interface{}, len(method.Inputs))

	for i, input := range method.Inputs {
		arg, err := convertArgument(input.Type.String(), arguments[i])
		if err != nil {
			return nil, fmt.Errorf("failed to convert argument: %s, %s", arguments[i], err)
		}
		args[i] = arg
	}
	return args, nil
}

func convertArgument(input string, argument interface{}) (arg interface{}, err error) {

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	switch input {
	case "uint256":
		arg = big.NewInt(int64(argument.(int)))
	case "string":
		arg = argument.(string)
	case "bool":
		arg = argument.(bool)
	case "address":
		arg = common.HexToAddress(argument.(string))
	case "bytes32":
		arg = common.HexToHash(argument.(string))

	default:
		return nil, fmt.Errorf("unsupported type: %s", input)
	}
	return arg, nil
}
