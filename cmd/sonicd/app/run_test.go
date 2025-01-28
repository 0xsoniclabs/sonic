package app

import (
	"fmt"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"os"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/cmd/sonictool/genesis"
	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
	futils "github.com/0xsoniclabs/sonic/utils"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/docker/docker/pkg/reexec"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xsoniclabs/sonic/cmd/cmdtest"
)

func tmpdir(t *testing.T) string {
	return t.TempDir()
}

func initFakenetDatadir(dataDir string, validatorsNum idx.Validator) {
	genesisStore := makefakegenesis.FakeGenesisStore(validatorsNum, futils.ToFtm(1000000000), futils.ToFtm(5000000))
	defer genesisStore.Close()

	if err := genesis.ImportGenesisStore(genesis.ImportParams{
		GenesisStore: genesisStore,
		DataDir:      dataDir,
		CacheRatio:   cachescale.Identity,
		LiveDbCache:  1, // Set lowest cache
		ArchiveCache: 1, // Set lowest cache
	}); err != nil {
		panic(err)
	}
}

type testcli struct {
	*cmdtest.TestCmd

	// template variables for expect
	Datadir  string
	Coinbase string
}

func (tt *testcli) readConfig() {
	cfg := config.DefaultNodeConfig()
	cfg.DataDir = tt.Datadir
	addr := common.Address{} // TODO: addr = emitter coinbase
	tt.Coinbase = strings.ToLower(addr.String())
}

func init() {
	// Run the app if we've been exec'd as "opera-test" in exec().
	reexec.Register("opera-test", func() {
		initApp()
		initAppHelp()
		if err := app.Run(os.Args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	})
}

func TestMain(m *testing.M) {
	// check if we have been reexec'd
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}

// exec cli with the given command line args. If the args don't set --datadir, the
// child g gets a temporary data directory.
func exec(t *testing.T, args ...string) *testcli {
	tt := &testcli{}
	tt.TestCmd = cmdtest.NewTestCmd(t, tt)

	if len(args) < 1 || args[0] != "attach" {
		// make datadir
		for i, arg := range args {
			switch {
			case arg == "-datadir" || arg == "--datadir":
				if i < len(args)-1 {
					tt.Datadir = args[i+1]
				}
			}
		}
		if tt.Datadir == "" {
			tt.Datadir = tmpdir(t)
			args = append([]string{"-datadir", tt.Datadir}, args...)
		}

		// Remove the temporary datadir.
		tt.Cleanup = func() { os.RemoveAll(tt.Datadir) }
		defer func() {
			if t.Failed() {
				tt.Cleanup()
			}
		}()
	}

	// Boot "opera". This actually runs the test binary but the TestMain
	// function will prevent any tests from running.
	tt.Run("opera-test", args...)

	// Read the generated key
	tt.readConfig()

	return tt
}
