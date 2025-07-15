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

func TestIntegration(t *testing.T) {

	nets := make(map[common.Hash]*IntegrationTestNet)

	for _, test := range testRegistry.test {
		testType := reflect.TypeOf(test)

		up := test.GetUpgradesForTest()

		for _, upgrade := range up {

			// TODO: name the net by the upgrade:
			// - generate name?
			// - create enums with presets?

			var net *IntegrationTestNet
			if catched, ok := nets[hashUpgrades(upgrade)]; ok {
				net = catched
			} else {
				net = StartIntegrationTestNet(t, IntegrationTestNetOptions{Upgrades: &upgrade})
				nets[hashUpgrades(upgrade)] = net
			}

			// TODO: parallelize net creation?  this may require pre-pass over all tests
			// TODO: inject session sponsor into genesis

			t.Run(testType.Name(), func(t *testing.T) {
				t.Parallel()
				for i := range testType.NumMethod() {
					testCase := testType.Method(i)

					if strings.HasPrefix(testCase.Name, "Test") {
						t.Run(testCase.Name, func(t *testing.T) {
							session := net.SpawnSession(t)
							testCase.Func.Call([]reflect.Value{
								reflect.ValueOf(test),
								reflect.ValueOf(t),
								reflect.ValueOf(session),
							})
						})
					}
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
