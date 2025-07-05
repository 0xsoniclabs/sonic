package datasemaphore

import (
	"sync"
	"testing"
	"time"

	"github.com/0xsoniclabs/consensus/consensus"
)

func TestNew(t *testing.T) {
	maxProcessing := consensus.Metric{Num: 10, Size: 1000}
	warningFunc := func(received, processing, releasing consensus.Metric) {}

	semaphore := New(maxProcessing, warningFunc)

	if semaphore == nil {
		t.Fatal("New should return a non-nil DataSemaphore")
	}

	if semaphore.maxProcessing != maxProcessing {
		t.Errorf("Expected maxProcessing to be %v, got %v", maxProcessing, semaphore.maxProcessing)
	}

	if semaphore.cond == nil {
		t.Error("Condition variable should be initialized")
	}
}

func TestTryAcquire(t *testing.T) {
	maxProcessing := consensus.Metric{Num: 10, Size: 1000}
	semaphore := New(maxProcessing, nil)

	// Test successful acquisition
	weight := consensus.Metric{Num: 5, Size: 500}
	if !semaphore.TryAcquire(weight) {
		t.Errorf("Expected to acquire semaphore with weight %v", weight)
	}

	processing := semaphore.Processing()
	if processing.Num != 5 || processing.Size != 500 {
		t.Errorf("Expected processing to be {Num=5,Size=500}, got %v", processing)
	}

	// Test successful second acquisition
	weight2 := consensus.Metric{Num: 3, Size: 300}
	if !semaphore.TryAcquire(weight2) {
		t.Errorf("Expected to acquire semaphore with weight %v", weight2)
	}

	processing = semaphore.Processing()
	if processing.Num != 8 || processing.Size != 800 {
		t.Errorf("Expected processing to be {Num=8,Size=800}, got %v", processing)
	}

	// Test failed acquisition due to exceeding Num
	weightExceedNum := consensus.Metric{Num: 3, Size: 100}
	if semaphore.TryAcquire(weightExceedNum) {
		t.Errorf("Expected acquisition to fail with weight %v", weightExceedNum)
	}

	// Test failed acquisition due to exceeding Size
	weightExceedSize := consensus.Metric{Num: 1, Size: 201}
	if semaphore.TryAcquire(weightExceedSize) {
		t.Errorf("Expected acquisition to fail with weight %v", weightExceedSize)
	}
}

func TestAcquire(t *testing.T) {
	// Test 1: Immediate acquisition (no waiting)
	maxProcessing := consensus.Metric{Num: 10, Size: 1000}
	semaphore := New(maxProcessing, nil)

	weight := consensus.Metric{Num: 5, Size: 500}
	if !semaphore.Acquire(weight, 100*time.Millisecond) {
		t.Errorf("Expected immediate acquisition to succeed")
	}

	// Test 2: Acquisition that should fail due to timeout
	weightExceed := consensus.Metric{Num: 6, Size: 600}
	if semaphore.Acquire(weightExceed, 10*time.Millisecond) {
		t.Errorf("Expected acquisition to fail due to timeout")
	}

	// Test 3: Immediate failure due to weight > maxProcessing
	weightTooLarge := consensus.Metric{Num: 11, Size: 500}
	if semaphore.Acquire(weightTooLarge, 10*time.Millisecond) {
		t.Errorf("Expected immediate failure for weight > maxProcessing")
	}

	// Test 4: Immediate failure due to weight.Size > maxProcessing.Size
	weightTooLargeSize := consensus.Metric{Num: 5, Size: 1001}
	if semaphore.Acquire(weightTooLargeSize, 10*time.Millisecond) {
		t.Errorf("Expected immediate failure for weight.Size > maxProcessing.Size")
	}
}

// Acquire with a separate goroutine for release
func TestAcquireWithRelease(t *testing.T) {
	maxProcessing := consensus.Metric{Num: 10, Size: 1000}
	semaphore := New(maxProcessing, nil)

	// First, acquire all resources
	semaphore.TryAcquire(maxProcessing)
	// Set up a channel to signal completion
	done := make(chan bool, 1)

	// Start a goroutine that will release resources after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		semaphore.Release(maxProcessing) // Release all resources
		done <- true
	}()

	smallWeight := consensus.Metric{Num: 1, Size: 100}
	result := semaphore.Acquire(smallWeight, 200*time.Millisecond)

	// Wait for the release goroutine to complete
	select {
	case <-done:
		// Release (should have) completed
	case <-time.After(300 * time.Millisecond):
		t.Fatal("Release goroutine did not complete")
	}

	if !result {
		t.Error("Expected acquisition to succeed after resources were released")
	}
}

func TestRelease(t *testing.T) {
	var warningCalled bool
	warningFunc := func(received, processing, releasing consensus.Metric) {
		warningCalled = true
	}

	maxProcessing := consensus.Metric{Num: 10, Size: 1000}
	semaphore := New(maxProcessing, warningFunc)
	weight := consensus.Metric{Num: 5, Size: 500}
	semaphore.TryAcquire(weight)

	// Test normal release
	semaphore.Release(weight)
	processing := semaphore.Processing()
	if processing.Num != 0 || processing.Size != 0 {
		t.Errorf("Expected processing to be {Num=0,Size=0}, got %v", processing)
	}

	// Test over-release (should trigger warning)
	semaphore.TryAcquire(weight)
	overWeight := consensus.Metric{Num: 6, Size: 500}
	semaphore.Release(overWeight)

	if !warningCalled {
		t.Error("Warning function should have been called for over-release")
	}

	processing = semaphore.Processing()
	if processing.Num != 0 || processing.Size != 0 {
		t.Errorf("Processing should be reset to zero after over-release, got %v", processing)
	}

	// Test warning is nil
	semaphore = New(maxProcessing, nil)
	semaphore.TryAcquire(weight)
	// This should NOT panic even though warning is nil
	semaphore.Release(overWeight)
}

func TestTerminate(t *testing.T) {
	maxProcessing := consensus.Metric{Num: 10, Size: 1000}
	semaphore := New(maxProcessing, nil)

	semaphore.Terminate()

	weight := consensus.Metric{Num: 1, Size: 1}
	if semaphore.TryAcquire(weight) {
		t.Error("Acquisition should fail after termination")
	}
}

func TestTerminateWithWaiting(t *testing.T) {
	maxProcessing := consensus.Metric{Num: 10, Size: 1000}
	semaphore := New(maxProcessing, nil)

	semaphore.TryAcquire(consensus.Metric{Num: 8, Size: 800})

	// Set up a channel to signal completion
	done := make(chan bool, 1)

	// Start a goroutine that will try to acquire more resources than available
	go func() {
		defer func() {
			done <- true
		}()

		// This should block until terminated
		largeWeight := consensus.Metric{Num: 5, Size: 500}
		result := semaphore.Acquire(largeWeight, 500*time.Millisecond)
		if result {
			t.Error("Expected acquisition to fail after terminate")
		}
	}()

	// Give goroutine time to start waiting
	time.Sleep(50 * time.Millisecond)

	// Now terminate the semaphore
	semaphore.Terminate()

	// Wait for the acquisition goroutine to complete
	select {
	case <-done:
		// Test passed - goroutine completed
	case <-time.After(600 * time.Millisecond):
		t.Fatal("Acquire did not complete after Terminate was called")
	}
}

func TestAvailable(t *testing.T) {
	maxProcessing := consensus.Metric{Num: 10, Size: 1000}
	semaphore := New(maxProcessing, nil)

	// Test initial available resources
	available := semaphore.Available()
	if available.Num != 10 || available.Size != 1000 {
		t.Errorf("Expected initial available to be {Num=10,Size=1000}, got %v", available)
	}

	// Test available after acquisition
	weight := consensus.Metric{Num: 4, Size: 400}
	semaphore.TryAcquire(weight)

	available = semaphore.Available()
	if available.Num != 6 || available.Size != 600 {
		t.Errorf("Expected available after acquisition to be {Num=6,Size=600}, got %v", available)
	}
}

func TestConcurrentAcquireRelease(t *testing.T) {
	maxProcessing := consensus.Metric{Num: 10, Size: 1000}
	semaphore := New(maxProcessing, nil)

	const numGoroutines = 5
	const iterations = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				weight := consensus.Metric{Num: 1, Size: 100}
				if semaphore.TryAcquire(weight) {
					// Brief work simulation
					time.Sleep(1 * time.Millisecond)
					semaphore.Release(weight)
				} else {
					// Brief pause before retry
					time.Sleep(1 * time.Millisecond)
				}
			}
		}(i)
	}

	// Use a timeout to prevent test hanging
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Test completed normally
	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out")
	}

	processing := semaphore.Processing()
	if processing.Num != 0 || processing.Size != 0 {
		t.Errorf("Expected processing to be {Num=0,Size=0} after all releases, got %v", processing)
	}
}

func TestAcquireWithWaiting(t *testing.T) {
	maxProcessing := consensus.Metric{Num: 10, Size: 1000}
	semaphore := New(maxProcessing, nil)

	// Acquire most resources
	semaphore.TryAcquire(consensus.Metric{Num: 8, Size: 800})
	// Set up a channel to signal completion
	done := make(chan bool, 1)
	acquireSucceeded := false

	go func() {
		// This should initially block, then succeed
		weight := consensus.Metric{Num: 3, Size: 300}
		acquireSucceeded = semaphore.Acquire(weight, 500*time.Millisecond)
		if acquireSucceeded {
			semaphore.Release(weight)
		}
		done <- true
	}()

	// Give goroutine time to start waiting
	time.Sleep(50 * time.Millisecond)

	// Release resources to allow waiting goroutine to proceed
	semaphore.Release(consensus.Metric{Num: 2, Size: 200})

	// Wait for goroutine completion with timeout
	select {
	case <-done:
		// Test completed normally
	case <-time.After(600 * time.Millisecond):
		t.Fatal("Test timed out waiting for acquisition to complete")
	}

	if !acquireSucceeded {
		t.Error("Expected acquisition to succeed after resources were released")
	}

	// Release remaining resources
	semaphore.Release(consensus.Metric{Num: 6, Size: 600})

	// Check final state
	processing := semaphore.Processing()
	if processing.Num != 0 || processing.Size != 0 {
		t.Errorf("Expected processing to be {Num=0,Size=0} after all releases, got %v", processing)
	}
}
