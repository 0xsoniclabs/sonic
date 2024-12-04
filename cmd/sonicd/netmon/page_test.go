package netmon

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/stretchr/testify/require"
)

func TestSummaryPage_CanRenderEmptyPage(t *testing.T) {
	require := require.New(t)
	summary, err := CreateSummaryPage(
		context.Background(),
		enode.ID{},
		map[enode.ID][]*enode.Node{},
	)
	require.NoError(err)
	page, err := renderToString(&summary)
	require.NoError(err)
	require.Contains(page, "<head>")
	require.Contains(page, "</head>")
	require.Contains(page, "function reloadPage()")
}

func TestSummaryPage_RenderPageWithSingleNode(t *testing.T) {
	require := require.New(t)

	localId := enode.ID{1, 2, 3, 4}
	summary, err := CreateSummaryPage(
		context.Background(),
		localId,
		map[enode.ID][]*enode.Node{
			localId: {},
		},
	)
	require.NoError(err)

	page, err := renderToString(&summary)
	require.NoError(err)
	require.Contains(page, "<head>")
	require.Contains(page, "</head>")
	// check that there is a svg element
	require.Contains(page, "<svg")
	require.Contains(page, "</svg>")
}

func TestSummaryPage_RenderPageWithMultipleNodes(t *testing.T) {
	require := require.New(t)

	localId := enode.ID{1}
	remote1Id := enode.ID{2}
	remote2Id := enode.ID{3}
	localEnode := createNode(1)
	remote1Enode := createNode(2)
	remote2Enode := createNode(3)
	summary, err := CreateSummaryPage(
		context.Background(),
		localId,
		map[enode.ID][]*enode.Node{
			remote1Id: {localEnode, remote2Enode},
			remote2Id: {localEnode, remote1Enode},
		},
	)
	require.NoError(err)

	page, err := renderToString(&summary)
	require.NoError(err)
	require.Contains(page, "<head>")
	require.Contains(page, "</head>")
	// check that there is a svg element
	require.Contains(page, "<svg>")
	require.Contains(page, "</svg>")
	require.Contains(page, "127.0.0.1")
	require.Contains(page, "127.0.0.2")
	require.Contains(page, "127.0.0.3")
}

func renderToString(p *SummaryPage) (string, error) {
	builder := &strings.Builder{}
	err := p.Render(builder)
	return builder.String(), err
}

func createNode(id int) *enode.Node {
	ip := net.ParseIP(fmt.Sprintf("127.0.0.%d", id))
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	return enode.NewV4(&key.PublicKey, ip, 5050, 5050)
}
