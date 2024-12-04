package netmon

import (
	"context"
	_ "embed"
	"html/template"
	"io"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
)

// SummaryPage represents the data to be shown on the network
// monitoring summary page.
type SummaryPage struct {
	Local PeerInfo      // < the local node's information
	Peers []PeerInfo    // < information about peers
	Nodes []PeerInfo    // < information about all known nodes
	Graph template.HTML // < a SVG network graph to show
}

type PeerInfo struct {
	ID    string
	IP    string
	Port  int
	Peers []PeerInfo
}

// CreateSummaryPage creates a new summary page from the given neighborhood.
func CreateSummaryPage(
	context context.Context,
	localId enode.ID,
	neighborhood map[enode.ID][]*enode.Node,
) (SummaryPage, error) {
	// Index all known nodes by their ID.
	peers := map[enode.ID]*enode.Node{}
	for _, nodes := range neighborhood {
		for _, node := range nodes {
			peers[node.ID()] = node
		}
	}

	getPeerInfo := func(peer enode.ID) PeerInfo {
		info := PeerInfo{}
		if node, found := peers[peer]; found {
			info.ID = node.ID().String()
			info.IP = node.IP().String()
			info.Port = node.TCP()
		} else {
			info.ID = peer.String()
		}
		return info
	}

	page := SummaryPage{}
	page.Local = getPeerInfo(localId)

	// Collect peer information.
	for peer, peers := range neighborhood {
		info := getPeerInfo(peer)
		for _, peer := range peers {
			info.Peers = append(info.Peers, getPeerInfo(peer.ID()))
		}
		page.Peers = append(page.Peers, info)
	}

	sort.Slice(page.Peers, func(i, j int) bool {
		return page.Peers[i].IP < page.Peers[j].IP
	})

	for i := range page.Peers {
		sort.Slice(page.Peers[i].Peers, func(j, k int) bool {
			return page.Peers[i].Peers[j].IP < page.Peers[i].Peers[k].IP
		})
	}

	// Collect all known nodes.
	for id := range peers {
		page.Nodes = append(page.Nodes, getPeerInfo(id))
	}
	sort.Slice(page.Nodes, func(i, j int) bool {
		return page.Nodes[i].IP < page.Nodes[j].IP
	})

	// Render the graph using graphviz.
	graph, err := createNetworkSvg(context, &page)
	if err != nil {
		return SummaryPage{}, err
	}
	page.Graph = template.HTML(graph)
	return page, nil
}

// Render renders a summary HTML page to the given writer.
func (p *SummaryPage) Render(out io.Writer) error {
	return pageTemplate.Execute(out, *p)
}

var funcMap = template.FuncMap{
	"inc": func(i int) int {
		return i + 1
	},
	"now": func() string {
		return time.Now().Format("2006-01-02 15:04:05")
	},
}

//go:embed page.html
var rawPageTemplate string
var pageTemplate = template.Must(template.New("page").Funcs(funcMap).Parse(rawPageTemplate))
