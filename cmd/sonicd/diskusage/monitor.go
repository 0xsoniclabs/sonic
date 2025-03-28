package diskusage

import (
	"os"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

func MonitorFreeDiskSpace(stopNodeSig chan os.Signal, path string, freeDiskSpaceCritical uint64) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		freeSpace, err := getFreeDiskSpace(path)
		if err != nil {
			log.Warn("Failed to get free disk space", "path", path, "err", err)
			break
		}
		if freeSpace < freeDiskSpaceCritical {
			log.Error("Low disk space. Gracefully shutting down Opera to prevent database corruption.", "available", common.StorageSize(freeSpace))
			stopNodeSig <- syscall.SIGTERM
			break
		} else if freeSpace < 2*freeDiskSpaceCritical {
			log.Warn("Disk space is running low. Opera will shutdown if disk space runs below critical level.", "available", common.StorageSize(freeSpace), "critical_level", common.StorageSize(freeDiskSpaceCritical))
		}
		<-ticker.C
	}
}
