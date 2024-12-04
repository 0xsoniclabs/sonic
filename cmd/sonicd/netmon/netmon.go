package netmon

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/Fantom-foundation/go-opera/gossip/topology"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/urfave/cli.v1"
)

var (
	NetworkMonitoringEnabledFlag = cli.BoolFlag{
		Name:  "netmon",
		Usage: "Enable monitoring of the network structure",
	}
	NetworkMonitoringPortFlag = cli.IntFlag{
		Name:  "netmon.port",
		Usage: "Port publishing the network monitoring data",
		Value: 30303,
	}
)

type NetworkMonitor struct {
	tracker    topology.ConnectionTracker
	serverPort int
	server     *http.Server
}

func NewNetworkMonitor(
	tracker topology.ConnectionTracker,
	serverPort int,
) *NetworkMonitor {
	return &NetworkMonitor{
		tracker:    tracker,
		serverPort: serverPort,
	}
}

func (nm *NetworkMonitor) Start() error {
	log.Info("Starting network monitor...")
	port := nm.serverPort
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: &networkMonitorHandler{nm.tracker},
	}
	serverPort, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return err
	}
	go server.Serve(serverPort)
	log.Info("Network monitor started", "address", fmt.Sprintf("http://localhost:%d", port))
	nm.server = server
	return nil
}

func (nm *NetworkMonitor) Stop() error {
	if nm.server == nil {
		return nil
	}
	err := nm.server.Shutdown(context.Background())
	nm.server = nil
	return err
}

type networkMonitorHandler struct {
	tracker topology.ConnectionTracker
}

func (h *networkMonitorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	localId := h.tracker.GetLocalId()
	neighborhood := h.tracker.GetNeighborhood()
	page, err := CreateSummaryPage(r.Context(), localId, neighborhood)
	if err != nil {
		log.Warn("Failed to create network monitor page", "err", err)
		return
	}

	if err := page.Render(w); err != nil {
		log.Warn("Failed to render network monitor page", "err", err)
	}
}
