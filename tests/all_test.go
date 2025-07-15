package tests

import (
	"reflect"
	"strings"
	"testing"

	"github.com/0xsoniclabs/carmen/go/common"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/crypto"
)

type TestFixture interface {
	GetUpgradesForTest() []opera.Upgrades
}

type Registry struct {
	test []TestFixture
}

func (r *Registry) Register(test TestFixture) {
	r.test = append(r.test, test)
}

var testRegistry Registry

func TestAll(t *testing.T) {
	t.Parallel()

	nets := make(map[common.Hash]*IntegrationTestNet)

	for _, test := range testRegistry.test {
		testType := reflect.TypeOf(test)

		up := test.GetUpgradesForTest()
		for _, upgrade := range up {
			// TODO: inject session sponsor into genesis

			// TODO: name the net by the upgrade:
			// - generate name?
			// - create enums with presets?
			t.Run(testType.Name(), func(t *testing.T) {
				_, ok := nets[hashUpgrades(upgrade)]
				if !ok {
					nets[hashUpgrades(upgrade)] = StartIntegrationTestNet(t, IntegrationTestNetOptions{Upgrades: &upgrade})
				}

				t.Parallel()

				for i := range testType.NumMethod() {
					testCase := testType.Method(i)

					// TODO: check signature of testCase.Func
					// - may make sense to support multiple signatures

					if strings.HasPrefix(testCase.Name, "Test") {
						t.Run(testCase.Name, func(t *testing.T) {
							net := nets[hashUpgrades(upgrade)]
							session := net.SpawnSession(t)
							testCase.Func.Call([]reflect.Value{
								reflect.ValueOf(test),
								reflect.ValueOf(t),
								reflect.ValueOf(session),
							})
						})
					}

					// TODO: error on ignored public method, this may be an
					// unlinked and ignored test
				}
			})
		}
	}
}

func hashUpgrades(upgrades opera.Upgrades) common.Hash {
	sha := crypto.NewKeccakState()
	_ = upgrades.EncodeRLP(sha)
	var h common.Hash
	_, _ = sha.Read(h[:])
	return h
}
