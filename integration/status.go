package integration

import (
	"path"

	"github.com/0xsoniclabs/sonic/utils"
)

func isInterrupted(chaindataDir string) bool {
	return utils.FileExists(path.Join(chaindataDir, "unfinished"))
}
