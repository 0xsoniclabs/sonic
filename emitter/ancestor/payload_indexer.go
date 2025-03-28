package ancestor

import (
	"github.com/0xsoniclabs/cacheutils/wlru"
	"github.com/0xsoniclabs/consensus/consensus"
)

type PayloadIndexer struct {
	payloadLamports *wlru.Cache
}

func NewPayloadIndexer(cacheSize int) *PayloadIndexer {
	cache, _ := wlru.New(uint(cacheSize), cacheSize)
	return &PayloadIndexer{cache}
}

func (h *PayloadIndexer) ProcessEvent(event consensus.Event, payloadMetric Metric) {
	maxParentsPayloadMetric := h.GetMetricOf(event.Parents())
	if maxParentsPayloadMetric != 0 || payloadMetric != 0 {
		h.payloadLamports.Add(event.ID(), maxParentsPayloadMetric+payloadMetric, 1)
	}
}

func (h *PayloadIndexer) getMetricOf(id consensus.EventHash) Metric {
	parentMetric, ok := h.payloadLamports.Get(id)
	if !ok {
		return 0
	}
	return parentMetric.(Metric)
}

func (h *PayloadIndexer) GetMetricOf(ids consensus.EventHashes) Metric {
	maxMetric := Metric(0)
	for _, id := range ids {
		metric := h.getMetricOf(id)
		if maxMetric < metric {
			maxMetric = metric
		}
	}
	return maxMetric
}

func (h *PayloadIndexer) SearchStrategy() SearchStrategy {
	return NewMetricStrategy(h.GetMetricOf)
}
