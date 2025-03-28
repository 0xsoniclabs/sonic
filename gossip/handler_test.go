package gossip

import (
	"testing"

	"github.com/ethereum/go-ethereum/p2p/discover/discfilter"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

func TestIsUseless(t *testing.T) {
	validEnode := enode.MustParse("enode://3f4306c065eaa5d8079e17feb56c03a97577e67af3c9c17496bb8916f102f1ff603e87d2a4ebfa0a2f70b780b85db212618857ea4e9627b24a9b0dd2faeb826e@127.0.0.1:5050")
	sonicName := "Sonic/v1.0.0-a-61af51c2-1715085138/linux-amd64/go1.21.7"
	operaName := "go-opera/v1.1.2-rc.6-8e84c9dc-1688013329/linux-amd64/go1.19.11"
	invalidName := "bot"

	discfilter.Enable()
	if isUseless(validEnode, sonicName) {
		t.Errorf("sonic peer reported as useless")
	}
	if isUseless(validEnode, operaName) {
		t.Errorf("opera peer reported as useless")
	}
	if !isUseless(validEnode, invalidName) {
		t.Errorf("invalid peer not reported as useless")
	}
	if !isUseless(validEnode, operaName) {
		t.Errorf("peer not banned after marking as useless")
	}
}
