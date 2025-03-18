package gossip

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/hash"
)

func TestComputePrevRandao_ComputationIsDeterministic(t *testing.T) {
	events := hash.FakeEvents(5)
	randao1 := computePrevRandao(events)
	rand.Shuffle(len(events), func(i, j int) {
		events[i], events[j] = events[j], events[i]
	})
	randao2 := computePrevRandao(events)
	if randao1 != randao2 {
		t.Error("computation is not deterministic")
	}
}

func TestComputePrevRandao_ComputationProducesCorrectValue(t *testing.T) {
	tests := []struct {
		name   string
		events hash.Events
		want   string
	}{
		{
			name:   "empty_events",
			events: hash.Events{},
			want:   "0x9d908ecfb6b256def8b49a7c504e6c889c4b0e41fe6ce3e01863dd7b61a20aa0",
		},
		{
			name: "one_event",
			events: hash.Events{
				hash.HexToEventHash("0x1234"),
			},
			want: "0x445c47179cf0e0e25fc47fcd611f2fff71742cfa2da9f42ff1a2aba577562bde",
		},
		{
			name: "multiple_events",
			events: hash.Events{
				hash.HexToEventHash("0x5678"),
				hash.HexToEventHash("0x9012"),
				hash.HexToEventHash("0x3456"),
			},
			want: "0xd260b051cbc12b222995f09e75d1596850a94bb257015bee25b84c7e8015de06",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := computePrevRandao(test.events)
			if !strings.EqualFold(got.String(), test.want) {
				t.Errorf("unexpected hash; got: %s, want: %s", got, test.want)
			}
		})
	}
}
