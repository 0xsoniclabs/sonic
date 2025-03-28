package vecmt

import (
	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/kvdb"
)

func (vi *Index) getBytes(table kvdb.Store, id consensus.EventHash) []byte {
	key := id.Bytes()
	b, err := table.Get(key)
	if err != nil {
		vi.crit(err)
	}
	return b
}

func (vi *Index) setBytes(table kvdb.Store, id consensus.EventHash, b []byte) {
	key := id.Bytes()
	err := table.Put(key, b)
	if err != nil {
		vi.crit(err)
	}
}

// GetHighestBeforeTime reads the vector from DB
func (vi *Index) GetHighestBeforeTime(id consensus.EventHash) *HighestBeforeTime {
	if bVal, okGet := vi.cache.HighestBeforeTime.Get(id); okGet {
		return bVal.(*HighestBeforeTime)
	}

	b := HighestBeforeTime(vi.getBytes(vi.table.HighestBeforeTime, id))
	if b == nil {
		return nil
	}
	vi.cache.HighestBeforeTime.Add(id, &b, uint(len(b)))
	return &b
}

// GetHighestBefore reads the vector from DB
func (vi *Index) GetHighestBefore(id consensus.EventHash) *HighestBefore {
	return &HighestBefore{
		VSeq:  vi.Engine.GetHighestBefore(id),
		VTime: vi.GetHighestBeforeTime(id),
	}
}

// SetHighestBeforeTime stores the vector into DB
func (vi *Index) SetHighestBeforeTime(id consensus.EventHash, vec *HighestBeforeTime) {
	vi.setBytes(vi.table.HighestBeforeTime, id, *vec)

	vi.cache.HighestBeforeTime.Add(id, vec, uint(len(*vec)))
}

// SetHighestBefore stores the vectors into DB
func (vi *Index) SetHighestBefore(id consensus.EventHash, vec *HighestBefore) {
	vi.Engine.SetHighestBefore(id, vec.VSeq)
	vi.SetHighestBeforeTime(id, vec.VTime)
}
