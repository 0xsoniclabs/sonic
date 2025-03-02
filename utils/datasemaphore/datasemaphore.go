package datasemaphore

import (
	"sync"
	"time"

	"github.com/0xsoniclabs/consensus/inter/dag"
)

type DataSemaphore struct {
	processing    dag.Metric
	maxProcessing dag.Metric

	mu   sync.Mutex
	cond *sync.Cond

	warning func(received dag.Metric, processing dag.Metric, releasing dag.Metric)
}

func New(maxProcessing dag.Metric, warning func(received dag.Metric, processing dag.Metric, releasing dag.Metric)) *DataSemaphore {
	s := &DataSemaphore{
		maxProcessing: maxProcessing,
		warning:       warning,
	}
	s.cond = sync.NewCond(&s.mu)
	return s
}

func (s *DataSemaphore) Acquire(weight dag.Metric, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	s.mu.Lock()
	defer s.mu.Unlock()
	for !s.tryAcquire(weight) {
		if weight.Size > s.maxProcessing.Size || weight.Num > s.maxProcessing.Num || time.Now().After(deadline) {
			return false
		}
		s.cond.Wait()
	}
	return true
}

func (s *DataSemaphore) TryAcquire(weight dag.Metric) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tryAcquire(weight)
}

func (s *DataSemaphore) tryAcquire(metric dag.Metric) bool {
	tmp := s.processing
	tmp.Num += metric.Num
	tmp.Size += metric.Size
	if tmp.Num > s.maxProcessing.Num || tmp.Size > s.maxProcessing.Size {
		return false
	}
	s.processing = tmp
	return true
}

func (s *DataSemaphore) Release(weight dag.Metric) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.processing.Num < weight.Num || s.processing.Size < weight.Size {
		if s.warning != nil {
			s.warning(s.processing, s.processing, weight)
		}
		s.processing = dag.Metric{}
	} else {
		s.processing.Num -= weight.Num
		s.processing.Size -= weight.Size
	}
	s.cond.Broadcast()
}

func (s *DataSemaphore) Terminate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maxProcessing = dag.Metric{}
	s.cond.Broadcast()
}

func (s *DataSemaphore) Processing() dag.Metric {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.processing
}

func (s *DataSemaphore) Available() dag.Metric {
	s.mu.Lock()
	defer s.mu.Unlock()
	return dag.Metric{
		Num:  s.maxProcessing.Num - s.processing.Num,
		Size: s.maxProcessing.Size - s.processing.Size,
	}
}
