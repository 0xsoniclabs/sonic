package genesis

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/0xsoniclabs/consensus/common/bigendian"
	"github.com/0xsoniclabs/consensus/hash"
	"github.com/0xsoniclabs/consensus/inter/idx"
	"github.com/0xsoniclabs/sonic/gossip"
	"github.com/0xsoniclabs/sonic/inter/ibr"
	"github.com/0xsoniclabs/sonic/inter/ier"
	"github.com/0xsoniclabs/sonic/opera/genesis"
	"github.com/0xsoniclabs/sonic/opera/genesisstore"
	"github.com/0xsoniclabs/sonic/opera/genesisstore/fileshash"
	"github.com/0xsoniclabs/sonic/utils/devnullfile"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	gzip "github.com/klauspost/pgzip"
)

func ExportGenesis(ctx context.Context, gdb *gossip.Store, includeArchive bool, out *os.File, tmpPath string) error {
	if gdb.GetHighestLamport() != 0 {
		log.Warn("Attempting genesis export not in a beginning of an epoch. Genesis file output may contain excessive data.")
	}

	header := genesis.Header{
		GenesisID:   *gdb.GetGenesisID(),
		NetworkID:   gdb.GetEpochState().Rules.NetworkID,
		NetworkName: gdb.GetEpochState().Rules.Name,
	}
	to := gdb.GetEpoch()

	// epochs
	writer := newUnitWriter(out)
	if err := writer.Start(header, "ers", tmpPath); err != nil {
		return err
	}
	if err := exportEpochsSection(ctx, gdb, writer, 1, to); err != nil {
		return err
	}

	// blocks
	var maxBlocks int64
	if !includeArchive {
		maxBlocks = 300
	}
	writer = newUnitWriter(out)
	if err := writer.Start(header, "brs", tmpPath); err != nil {
		return err
	}
	lastBlock := gdb.GetLatestBlockIndex()
	if epochEnd := getEpochBlock(to, gdb); lastBlock != epochEnd {
		log.Warn("Last block is not the last block of the epoch", "last", lastBlock, "epochEnd", epochEnd)
	}
	if err := exportBlocksSection(ctx, gdb, writer, lastBlock, maxBlocks); err != nil {
		return err
	}

	// live state
	writer = newUnitWriter(out)
	if err := writer.Start(header, "fws", tmpPath); err != nil {
		return err
	}
	if err := exportFwsSection(ctx, gdb, writer); err != nil {
		return err
	}

	// archive
	if includeArchive {
		writer = newUnitWriter(out)
		if err := writer.Start(header, "fwa", tmpPath); err != nil {
			return err
		}
		if err := exportFwaSection(ctx, gdb, writer); err != nil {
			return err
		}
	}
	return nil
}

func exportEpochsSection(ctx context.Context, gdb *gossip.Store, writer *unitWriter, from, to idx.Epoch) error {
	log.Info("Exporting epochs", "from", from, "to", to)
	for i := to; i >= from; i-- {
		er := gdb.GetFullEpochRecord(i)
		if er == nil {
			log.Warn("No epoch record", "epoch", i)
			break
		}
		b, _ := rlp.EncodeToBytes(ier.LlrIdxFullEpochRecord{
			LlrFullEpochRecord: *er,
			Idx:                i,
		})
		_, err := writer.Write(b)
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	epochsHash, err := writer.Flush()
	if err != nil {
		return err
	}
	log.Info("Exported epochs")
	fmt.Printf("- Epochs hash: %v \n", epochsHash.String())
	return nil
}

func exportBlocksSection(ctx context.Context, gdb *gossip.Store, writer *unitWriter, to idx.Block, maxBlocks int64) error {
	toBlock := int64(to)
	fromBlock := int64(0)
	if maxBlocks != 0 && toBlock > 1+maxBlocks {
		fromBlock = toBlock - maxBlocks
	}
	log.Info("Exporting blocks", "from", fromBlock, "to", toBlock)
	for i := toBlock; i >= fromBlock; i-- {
		i := idx.Block(i)
		br := gdb.GetFullBlockRecord(i)
		if br == nil {
			return fmt.Errorf("the block record for block %d is missing in gdb", i)
		}
		if i%200000 == 0 {
			log.Info("Exporting blocks", "last", i)
		}
		b, _ := rlp.EncodeToBytes(ibr.LlrIdxFullBlockRecord{
			LlrFullBlockRecord: *br,
			Idx:                i,
		})
		_, err := writer.Write(b)
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	blocksHash, err := writer.Flush()
	if err != nil {
		return err
	}
	log.Info("Exported blocks")
	fmt.Printf("- Blocks hash: %v \n", blocksHash.String())
	return nil
}

func exportFwsSection(ctx context.Context, gdb *gossip.Store, writer *unitWriter) error {
	log.Info("Exporting Fantom World State Live data")
	if err := gdb.EvmStore().ExportLiveWorldState(ctx, writer); err != nil {
		return err
	}
	fwsHash, err := writer.Flush()
	if err != nil {
		return err
	}
	log.Info("Exported Fantom World State Live data")
	fmt.Printf("- FWS hash: %v \n", fwsHash.String())
	return nil
}

func exportFwaSection(ctx context.Context, gdb *gossip.Store, writer *unitWriter) error {
	log.Info("Exporting Fantom World State Archive data")
	if err := gdb.EvmStore().ExportArchiveWorldState(ctx, writer); err != nil {
		return err
	}

	fwaHash, err := writer.Flush()
	if err != nil {
		return err
	}
	log.Info("Exported Fantom World State Archive data")
	fmt.Printf("- FWA hash: %v \n", fwaHash.String())
	return nil
}

func getEpochBlock(epoch idx.Epoch, store *gossip.Store) idx.Block {
	bs, _ := store.GetHistoryBlockEpochState(epoch)
	if bs == nil {
		return 0
	}
	return bs.LastBlock.Idx
}

type unitWriter struct {
	plain            io.WriteSeeker
	gziper           *gzip.Writer
	fileshasher      *fileshash.Writer
	dataStartPos     int64
	uncompressedSize uint64
}

func newUnitWriter(plain io.WriteSeeker) *unitWriter {
	return &unitWriter{
		plain: plain,
	}
}

func (w *unitWriter) Start(header genesis.Header, name, tmpDirPath string) error {
	if w.plain == nil {
		// dry run
		w.fileshasher = fileshash.WrapWriter(nil, genesisstore.FilesHashPieceSize, func(int) fileshash.TmpWriter {
			return devnullfile.DevNull{}
		})
		return nil
	}
	// Write unit marker and version
	_, err := w.plain.Write(append(genesisstore.FileHeader, genesisstore.FileVersion...))
	if err != nil {
		return err
	}

	// write genesis header
	err = rlp.Encode(w.plain, genesisstore.Unit{
		UnitName: name,
		Header:   header,
	})
	if err != nil {
		return err
	}

	w.dataStartPos, err = w.plain.Seek(8+8+32, io.SeekCurrent)
	if err != nil {
		return err
	}

	w.gziper, _ = gzip.NewWriterLevel(w.plain, gzip.BestCompression)

	w.fileshasher = fileshash.WrapWriter(w.gziper, genesisstore.FilesHashPieceSize, func(tmpI int) fileshash.TmpWriter {
		tmpI++
		tmpPath := path.Join(tmpDirPath, fmt.Sprintf("genesis-%s-tmp-%d", name, tmpI))
		_ = os.MkdirAll(tmpDirPath, os.ModePerm)
		tmpFh, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_RDWR, os.ModePerm)
		if err != nil {
			panic(fmt.Errorf("file opening error; path: %s; %w", tmpPath, err))
		}
		return dropableFile{
			ReadWriteSeeker: tmpFh,
			Closer:          tmpFh,
			path:            tmpPath,
		}
	})
	return nil
}

func (w *unitWriter) Flush() (hash.Hash, error) {
	if w.plain == nil {
		return w.fileshasher.Root(), nil
	}
	h, err := w.fileshasher.Flush()
	if err != nil {
		return hash.Hash{}, err
	}

	err = w.gziper.Close()
	if err != nil {
		return hash.Hash{}, err
	}

	endPos, err := w.plain.Seek(0, io.SeekCurrent)
	if err != nil {
		return hash.Hash{}, err
	}

	_, err = w.plain.Seek(w.dataStartPos-(8+8+32), io.SeekStart)
	if err != nil {
		return hash.Hash{}, err
	}

	_, err = w.plain.Write(h.Bytes())
	if err != nil {
		return hash.Hash{}, err
	}
	_, err = w.plain.Write(bigendian.Uint64ToBytes(uint64(endPos - w.dataStartPos)))
	if err != nil {
		return hash.Hash{}, err
	}
	_, err = w.plain.Write(bigendian.Uint64ToBytes(w.uncompressedSize))
	if err != nil {
		return hash.Hash{}, err
	}

	_, err = w.plain.Seek(0, io.SeekEnd)
	if err != nil {
		return hash.Hash{}, err
	}
	return h, nil
}

func (w *unitWriter) Write(b []byte) (n int, err error) {
	n, err = w.fileshasher.Write(b)
	w.uncompressedSize += uint64(n)
	return
}

type dropableFile struct {
	io.ReadWriteSeeker
	io.Closer
	path string
}

func (f dropableFile) Drop() error {
	return os.Remove(f.path)
}
