package txtime

import (
	"time"

	"github.com/0xsoniclabs/consensus/utils/wlru"
	"github.com/ethereum/go-ethereum/common"
)

var (
	globalFinalized, _    = wlru.New(30000, 300000)
	globalNonFinalized, _ = wlru.New(5000, 50000)
	Enabled               = false
)

func Saw(txid common.Hash, t time.Time) {
	if !Enabled {
		return
	}
	globalNonFinalized.ContainsOrAdd(txid, t, 1)
}

func Validated(txid common.Hash, t time.Time) {
	if !Enabled {
		return
	}
	v, has := globalNonFinalized.Peek(txid)
	if has {
		t = v.(time.Time)
	}
	globalFinalized.ContainsOrAdd(txid, t, 1)
}

func Of(txid common.Hash) time.Time {
	if !Enabled {
		return time.Time{}
	}
	v, has := globalFinalized.Get(txid)
	if has {
		return v.(time.Time)
	}
	v, has = globalNonFinalized.Get(txid)
	if has {
		return v.(time.Time)
	}
	now := time.Now()
	Saw(txid, now)
	return now
}

func Get(txid common.Hash) time.Time {
	if !Enabled {
		return time.Time{}
	}
	v, has := globalFinalized.Get(txid)
	if has {
		return v.(time.Time)
	}
	v, has = globalNonFinalized.Get(txid)
	if has {
		return v.(time.Time)
	}
	return time.Time{}
}
