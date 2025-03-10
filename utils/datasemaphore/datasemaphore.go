package datasemaphore

import (
	"sync"
	"time"

	"github.com/0xsoniclabs/consensus/inter/dag"
)

// DataSemaphore implements a resource control mechanism for limiting concurrent processing
// of DAG elements based on two resource dimensions: number of events and total size.
// It allows callers to acquire resources up to configured limits and blocks if limits would be exceeded.
type DataSemaphore struct {
	// github.com/0xsoniclabs/consensus/inter/dag
	//
	// type Metric struct {
	//     Num  idx.Event // Event is a uint64
	//     Size uint64
	// }
	processing    dag.Metric                                                             // Tracks currently used resources (event count and size)
	maxProcessing dag.Metric                                                             // Maximum allowed resources to be used concurrently
	mu            sync.Mutex                                                             // Mutex for thread-safe access to semaphore state
	cond          *sync.Cond                                                             // Condition variable for signaling when resources become available
	warning       func(received dag.Metric, processing dag.Metric, releasing dag.Metric) // Callback for resource accounting anomalies
}

func New(maxProcessing dag.Metric, warning func(received dag.Metric, processing dag.Metric, releasing dag.Metric)) *DataSemaphore {
	s := &DataSemaphore{
		maxProcessing: maxProcessing,
		warning:       warning,
	}
	s.cond = sync.NewCond(&s.mu)
	return s
}

// Acquire attempts to acquire resources specified by weight, blocking until resources
// are available or timeout occurs.
// weight: resources to acquire (event count and size)
// timeout: maximum time to wait for resources to become available
// Returns true if resources were successfully acquired, false otherwise
func (s *DataSemaphore) Acquire(weight dag.Metric, timeout time.Duration) bool {
	// Calculate deadline for timeout
	deadline := time.Now().Add(timeout)

	for {
		s.mu.Lock()
		// Check if we've exceeded the deadline
		if time.Now().After(deadline) {
			s.mu.Unlock()
			return false
		}

		// Try to acquire resources, blocking and waiting if not immediately available
		if s.tryAcquire(weight) {
			s.mu.Unlock()
			return true
		}

		// Set up a timer to wake goroutines up when the deadline is reached
		timer := time.AfterFunc(time.Until(deadline), func() {
			s.cond.Broadcast() // Wake up anyone waiting
		})

		// Wait for a signal that resources have been released or timeout
		s.cond.Wait()

		// Stop the timer if it hasn't fired yet
		timer.Stop()

		s.mu.Unlock()
	}
}

// TryAcquire attempts to acquire resources without blocking.
// weight: resources to acquire (event count and size)
// Returns true if resources were acquired, false if not enough resources were available
func (s *DataSemaphore) TryAcquire(weight dag.Metric) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tryAcquire(weight)
}

// Must be called with mutex already locked.
func (s *DataSemaphore) tryAcquire(metric dag.Metric) bool {
	tmp := s.processing // Create temporary copy to check if acquisition is possible
	tmp.Num += metric.Num
	tmp.Size += metric.Size

	// Check if acquisition would exceed either resource limit
	if tmp.Num > s.maxProcessing.Num || tmp.Size > s.maxProcessing.Size {
		return false
	}

	// If we reach here, acquisition is possible - update the processing state
	s.processing = tmp
	return true
}

// Release returns previously acquired resources to the semaphore.
// If more resources are released than were acquired, triggers the warning callback
// and resets the semaphore state to prevent further anomalies
func (s *DataSemaphore) Release(weight dag.Metric) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for resource accounting anomaly (releasing more than what's being processed)
	if s.processing.Num < weight.Num || s.processing.Size < weight.Size {
		if s.warning != nil {
			// Call warning callback with the anomalous state
			s.warning(s.processing, s.processing, weight)
		}
		// Reset processing to prevent further issues
		s.processing = dag.Metric{}
	} else {
		// Normal case - subtract the released resources
		s.processing.Num -= weight.Num
		s.processing.Size -= weight.Size
	}

	// Notify all waiting goroutines that resources have been released
	s.cond.Broadcast()
}

// Terminate stops the semaphore by setting max capacity to zero.
// This effectively prevents any new acquisitions and wakes up all
// waiting goroutines so they can detect termination
func (s *DataSemaphore) Terminate() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Set max processing to zero to prevent new acquisitions
	s.maxProcessing = dag.Metric{}

	// Wake up all waiting goroutines so they can detect termination
	s.cond.Broadcast()
}

// Processing returns the current resource usage
func (s *DataSemaphore) Processing() dag.Metric {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.processing
}

// Available returns the currently available resource capacity.
func (s *DataSemaphore) Available() dag.Metric {
	s.mu.Lock()
	defer s.mu.Unlock()
	return dag.Metric{
		Num:  s.maxProcessing.Num - s.processing.Num,
		Size: s.maxProcessing.Size - s.processing.Size,
	}
}
