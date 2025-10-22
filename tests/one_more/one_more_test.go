package onemore

import (
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/tests"
)

func TestOneMore(t *testing.T) {

	tests.StartIntegrationTestNetWithJsonGenesis(t)

	time.Sleep(30 * time.Second)
}
