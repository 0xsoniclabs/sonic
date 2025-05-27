package app_test

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"os"
	"regexp"
	"strings"
	"testing"

	sonictool "github.com/0xsoniclabs/sonic/cmd/sonictool/app"
	"github.com/0xsoniclabs/sonic/cmd/sonictool/genesis"
	"github.com/0xsoniclabs/sonic/opera"
	ogenesis "github.com/0xsoniclabs/sonic/opera/genesis"
	"github.com/0xsoniclabs/sonic/opera/genesisstore"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/utils/caution"
	"github.com/0xsoniclabs/sonic/utils/prompt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSonicTool_check_ExecutesWithoutErrors(t *testing.T) {

	net := tests.StartIntegrationTestNet(t)
	generateNBlocks(t, net, 2)
	net.Stop()

	_, err := executeSonicTool(t,
		"--datadir", net.GetDirectory()+"/state",
		"check", "live")
	require.NoError(t, err)

	_, err = executeSonicTool(t,
		"--datadir", net.GetDirectory()+"/state",
		"check", "archive")
	require.NoError(t, err)
}

func TestSonicTool_compact_ExecutesWithoutErrors(t *testing.T) {

	net := tests.StartIntegrationTestNet(t)
	generateNBlocks(t, net, 2)
	net.Stop()

	_, err := executeSonicTool(t,
		"--datadir", net.GetDirectory()+"/state",
		"compact")
	require.NoError(t, err)
}

func TestSonicTool_account_ExecutesWithoutErrors(t *testing.T) {
	keystoreDir := t.TempDir()

	// Make a password in file (in memory process used by these tests do not allow stdinput rebinding)
	passwordFileName := t.TempDir() + "/password_file"
	require.NoError(t, generatePasswordFile(passwordFileName, "this is the passphrase"))

	// no accounts in keystore
	accounts := listAccounts(t, keystoreDir)
	require.Empty(t, accounts)

	// Create account
	accountNewOutput, err := executeSonicTool(t,
		"--datadir", keystoreDir,
		"account", "new", "--password", passwordFileName)
	require.NoError(t, err)

	accounts = listAccounts(t, keystoreDir)
	require.Len(t, accounts, 1)

	// Parse the address of the created account
	addressRe := `Public address of the key:\s+([a-zA-Z0-9]+)`
	matches := regexp.MustCompile(addressRe).FindStringSubmatch(accountNewOutput)
	require.Len(t, matches, 2)
	address := matches[1]
	require.NotEmpty(t, address)

	// Check if the account was created (check if the reported keyfile exists)
	keyFileRe := fmt.Sprintf("%s/keystore/[^.]+.[^-]+--[a-zA-Z0-9]{40}", keystoreDir)
	matches = regexp.MustCompile(keyFileRe).FindStringSubmatch(accountNewOutput)
	require.Len(t, matches, 1)
	keyfile := matches[0]
	require.FileExists(t, keyfile)

	// Update account (change password)
	mockCtrl := gomock.NewController(t)
	promptMock := prompt.NewMockUserPrompter(mockCtrl)
	revertPrompt := replaceUserPrompter(promptMock)
	promptMock.EXPECT().PromptPassword("Passphrase: ").Return("this is the passphrase", nil).Times(2)
	promptMock.EXPECT().PromptPassword("Repeat passphrase: ").Return("this is the passphrase", nil)

	_, err = executeSonicTool(t,
		"--datadir", keystoreDir,
		"account", "update", address)
	require.NoError(t, err)
	revertPrompt()

	// Generate a new key to be imported
	privateKeyFileName := t.TempDir() + "/password_file"
	_, err = generatePrivateKeyFile(privateKeyFileName)
	require.NoError(t, err)

	// Import the key
	_, err = executeSonicTool(t,
		"--datadir", keystoreDir,
		"account", "import", privateKeyFileName,
		"--password", passwordFileName) // key is not encrypted, but we need to provide a password
	require.NoError(t, err)

	// Check if the account was imported
	accounts = listAccounts(t, keystoreDir)
	require.Len(t, accounts, 2)
}

func TestSonicTool_genesis_ExecutesWithoutErrors(t *testing.T) {

	// Create a history by running some transactions
	net := tests.StartIntegrationTestNetWithFakeGenesis(t)
	generateNBlocks(t, net, 2)
	net.Stop()

	passwordFileName := fmt.Sprintf("%s/password_file", t.TempDir())
	require.NoError(t, generatePasswordFile(passwordFileName, "this is the passphrase"))

	exportFile := fmt.Sprintf("%s/genesis", t.TempDir())
	_, err := executeSonicTool(t,
		"--datadir", net.GetDirectory()+"/state",
		"genesis", "export", exportFile)
	require.NoError(t, err)
	require.FileExists(t, exportFile)

	header, sections, err := getGenesisHeaderHashes(exportFile)
	require.NoError(t, err)
	require.NotContains(t, sections, "signature", "export is signed, before signing it")

	// sign the genesis
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	hash, _, err := genesis.GetGenesisMetadata(header, sections)
	require.NoError(t, err)
	signature, err := crypto.Sign(hash[:], key)
	require.NoError(t, err)

	mockCtrl := gomock.NewController(t)
	promptMock := prompt.NewMockUserPrompter(mockCtrl)
	revertPrompt := replaceUserPrompter(promptMock)
	promptMock.EXPECT().PromptInput("Signature (hex): ").Return(hexutil.Encode(signature), nil)

	_, err = executeSonicTool(t,
		"--datadir", fmt.Sprintf("%s/state", t.TempDir()),
		"genesis", "sign", exportFile)
	// Note, this how far we can get without the actual key
	require.ErrorContains(t, err, "genesis signature does not match any trusted signer")
	revertPrompt()
}

func TestSonicTool_heal_ExecutesWithoutErrors(t *testing.T) {
	net := tests.StartIntegrationTestNet(
		t,
		tests.IntegrationTestNetOptions{
			Upgrades:             tests.AsPointer(opera.GetSonicUpgrades()),
			ClientExtraArguments: []string{"--statedb.checkpointinterval", "1"},
		},
	)
	generateNBlocks(t, net, 3)
	net.Stop()

	_, err := executeSonicTool(t, "--datadir", net.GetDirectory()+"/state", "heal")
	require.NoError(t, err)
}

func TestSonicTool_config_ExecutesWithoutErrors(t *testing.T) {

	net := tests.StartIntegrationTestNet(t)
	generateNBlocks(t, net, 2)
	net.Stop()

	configFileName := t.TempDir() + "config.toml"
	_, err := executeSonicTool(t,
		"--datadir", net.GetDirectory()+"/state",
		"dumpconfig", configFileName)
	require.NoError(t, err)

	output, err := executeSonicTool(t,
		"--datadir", net.GetDirectory()+"/state",
		"dumpconfig")
	require.NoError(t, err)

	f, err := os.Open(configFileName)
	require.NoError(t, err)
	configFromFile, err := io.ReadAll(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	require.True(t,
		strings.Contains(output, string(configFromFile)),
		"config file content is not in the output")

	_, err = executeSonicTool(t,
		"--datadir", net.GetDirectory()+"/state",
		"checkconfig", configFileName)
	require.NoError(t, err)
}

func TestSonicTool_events_ExecutesWithoutErrors(t *testing.T) {
	net := tests.StartIntegrationTestNet(t)
	generateNBlocks(t, net, 2)
	net.Stop()

	eventsExportFile := t.TempDir() + "/events.json"

	_, err := executeSonicTool(t,
		"--datadir", net.GetDirectory()+"/state",
		"events", "export", eventsExportFile)
	require.NoError(t, err)
	require.FileExists(t, eventsExportFile)

	_, err = executeSonicTool(t,
		"--datadir", net.GetDirectory()+"/state",
		"events", "import", eventsExportFile)
	require.NoError(t, err)
}

func TestSonicTool_validator_ExecutesWithoutErrors(t *testing.T) {

	dataDir := t.TempDir()

	mockCtrl := gomock.NewController(t)
	promptMock := prompt.NewMockUserPrompter(mockCtrl)
	revertPrompt := replaceUserPrompter(promptMock)
	promptMock.EXPECT().PromptPassword("Passphrase: ").Return("this is the passphrase", nil).AnyTimes()
	promptMock.EXPECT().PromptPassword("Repeat passphrase: ").Return("this is the passphrase", nil).AnyTimes()

	// Create a new validator
	log, err := executeSonicTool(t,
		"--datadir", dataDir,
		"validator", "new")
	require.NoError(t, err)
	t.Log(log)

	matches := regexp.MustCompile(fmt.Sprintf(`Path of the secret key file: (%s/keystore/validator/[a-zA-Z0-9]+)`, dataDir)).FindAllStringSubmatch(log, -1)
	t.Log(matches)
	require.Len(t, matches, 1)
	newValidatorKeyFile := matches[0][1]
	require.FileExists(t, newValidatorKeyFile)

	// Import a new account (to know pub key)
	privateKeyFileName := t.TempDir() + "/private_key"
	key, err := generatePrivateKeyFile(privateKeyFileName)
	require.NoError(t, err)
	_, err = executeSonicTool(t,
		"--datadir", dataDir,
		"account", "import", privateKeyFileName)
	require.NoError(t, err)
	accounts := listAccounts(t, dataDir)
	require.Len(t, accounts, 1)

	// Convert new account into a validator
	log, err = executeSonicTool(t,
		"--datadir", dataDir,
		"validator", "convert", hexutil.Encode(accounts[0][:]),
		hexutil.Encode(crypto.FromECDSAPub(&key.PublicKey)),
	)
	require.NoError(t, err)
	t.Log(log)

	matches = regexp.MustCompile(fmt.Sprintf(`Your key was converted and saved to (%s/keystore/validator/[a-zA-Z0-9]+)`, dataDir)).FindAllStringSubmatch(log, -1)
	t.Log(matches)
	require.Len(t, matches, 1)
	convertedValidatorKeyFile := matches[0][1]
	require.FileExists(t, convertedValidatorKeyFile)
	require.NotEqual(t, newValidatorKeyFile, convertedValidatorKeyFile, "new and converted validator keys should be different")

	revertPrompt()
}

// =============================================================================
// Helper functions
// =============================================================================

func getGenesisHeaderHashes(genesisFile string) (ogenesis.Header, ogenesis.Hashes, error) {
	genesisReader, err := os.Open(genesisFile)
	// note, genesisStore closes the reader, no need to defer close it here
	if err != nil {
		return ogenesis.Header{}, nil, fmt.Errorf("failed to open the genesis file: %w", err)
	}

	genesisStore, genesisHashes, err := genesisstore.OpenGenesisStore(genesisReader)
	if err != nil {
		return ogenesis.Header{}, nil, fmt.Errorf("failed to read genesis file: %w", err)
	}
	defer caution.CloseAndReportError(&err, genesisStore, "failed to close the genesis store")

	return genesisStore.Header(), genesisHashes, nil
}

var accountsInListRe = regexp.MustCompile(`Account\s+#\d+:\s+\{([a-zA-Z0-9]{40})\}`)

func listAccounts(t *testing.T, keystoreDir string) []common.Address {
	listAccountsOutput, err := executeSonicTool(t,
		"--datadir", keystoreDir,
		"account", "list")
	require.NoError(t, err)
	matches := accountsInListRe.FindAllStringSubmatch(listAccountsOutput, -1)

	res := make([]common.Address, 0, len(matches))
	for _, match := range matches {
		res = append(res, common.HexToAddress(match[1]))
	}
	return res
}

func generatePasswordFile(filename string, password string) (err error) {
	file, err := os.Create(filename)
	defer caution.CloseAndReportError(&err, file, "failed to close file")

	_, err = file.WriteString(password)
	if err != nil {
		return fmt.Errorf("failed to write password to file: %w", err)
	}
	return nil
}

func generatePrivateKeyFile(file string) (*ecdsa.PrivateKey, error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	err = crypto.SaveECDSA(file, key)
	if err != nil {
		return nil, fmt.Errorf("failed to save key: %w", err)
	}
	return key, err
}

// generateNBlocks generates n blocks in the blockchain.
// The transactions executed are not important, only the fact that they are
// executed synchronously and n blocks exist after the function returns.
func generateNBlocks(t *testing.T, net *tests.IntegrationTestNet, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		createAccount(t, net)
	}
}

func createAccount(t *testing.T, net *tests.IntegrationTestNet) {
	t.Helper()

	var addr common.Address
	_, err := rand.Read(addr[:])
	require.NoError(t, err)

	receipt, err := net.EndowAccount(common.Address{42}, big.NewInt(100))
	require.NoError(t, err)
	require.Equal(
		t,
		types.ReceiptStatusSuccessful,
		receipt.Status,
		"failed to deploy contract",
	)
}

// executeSonicTool executes the sonictool as if the provided arguments were
// passed on the command line.
// The standard out of the process is returned as a string.
// Only direct errors resulting from the run itself are returned, to allow
// checking specific error messages.
func executeSonicTool(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var err error

	r, w, err := os.Pipe()
	require.NoError(t, err)

	stashStdOut := os.Stdout
	defer func() {
		os.Stdout = stashStdOut
	}()
	os.Stdout = w

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()
	os.Args = append([]string{"sonictool"}, args...)

	executionErr := sonictool.Run()
	require.NoError(t, w.Close())

	output, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	return string(output), executionErr
}

func replaceUserPrompter(newPrompt prompt.UserPrompter) (cleanup func()) {
	oldPrompt := prompt.UserPrompt
	prompt.UserPrompt = newPrompt
	cleanup = func() { prompt.UserPrompt = oldPrompt }
	return
}
