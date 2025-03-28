package emitter

import (
	"io"

	"github.com/0xsoniclabs/consensus/consensus"

	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/utils"
)

var openPrevActionFile = utils.OpenFile

func (em *Emitter) writeLastEmittedEventID(id consensus.EventHash) {
	if em.emittedEventFile == nil {
		return
	}
	_, err := em.emittedEventFile.WriteAt(id.Bytes(), 0)
	if err != nil {
		log.Crit("Failed to write event file", "file", em.config.PrevEmittedEventFile.Path, "err", err)
	}
}

func (em *Emitter) readLastEmittedEventID() *consensus.EventHash {
	if em.emittedEventFile == nil {
		return nil
	}
	buf := make([]byte, 32)
	_, err := em.emittedEventFile.ReadAt(buf, 0)
	if err != nil {
		if err == io.EOF {
			return nil
		}
		log.Crit("Failed to read event file", "file", em.config.PrevEmittedEventFile.Path, "err", err)
	}
	v := consensus.BytesToEvent(buf)
	return &v
}
