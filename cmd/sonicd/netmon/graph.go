package netmon

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/goccy/go-graphviz"
)

func createNetworkSvg(ctxt context.Context, summary *SummaryPage) (result string, err error) {
	g, err := graphviz.New(ctxt)
	if err != nil {
		return "", nil
	}

	graph, err := g.Graph()
	if err != nil {
		return "", err
	}
	defer func() {
		err = errors.Join(err, graph.Close(), g.Close())
	}()

	// add nodes
	var R = max(300, float64(150*len(summary.Nodes))/(2*math.Pi))
	for i, node := range summary.Nodes {
		n, err := graph.CreateNodeByName(node.ID)
		if err != nil {
			return "", err
		}
		n.SetLabel(node.IP)
		sin, cos := math.Sincos(math.Pi * 2 * float64(i) / float64(len(summary.Nodes)))
		n.SetPos(R+sin*R, R+R*cos)
		if node.ID == summary.Local.ID {
			n.SetColor("red")
		}
	}

	// add edges from local node to direct peers
	for _, peer := range summary.Peers {
		n, err := graph.NodeByName(summary.Local.ID)
		if err != nil {
			continue
		}
		m, err := graph.NodeByName(peer.ID)
		if err != nil {
			continue
		}
		e, err := graph.CreateEdgeByName(fmt.Sprintf("%s-%s", summary.Local.ID, peer.ID), n, m)
		if err != nil {
			continue
		}
		if e != nil {
			e.SetColor("red")
		}
	}

	// add all other edges
	for _, node := range summary.Peers {
		for _, peer := range node.Peers {
			n, err := graph.NodeByName(node.ID)
			if err != nil {
				continue
			}
			m, err := graph.NodeByName(peer.ID)
			if err != nil {
				continue
			}
			_, err = graph.CreateEdgeByName(fmt.Sprintf("%s-%s", node.ID, peer.ID), n, m)
			if err != nil {
				continue
			}
		}
	}

	var buf bytes.Buffer
	g.SetLayout(graphviz.NOP)
	if err := g.Render(ctxt, graph, graphviz.SVG, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
