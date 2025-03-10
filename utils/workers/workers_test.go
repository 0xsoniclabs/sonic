package workers

import (
	"sync"
	"testing"
	"time"
)

func TestWorker(t *testing.T) {
	tasksC := make(chan func(), 5)
	quit := make(chan struct{})

	// Start worker in separate goroutine
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		worker(tasksC, quit)
	}()

	// Test worker processing tasks
	taskDone := make(chan bool)
	tasksC <- func() {
		taskDone <- true
	}

	select {
	case <-taskDone:
		// Task completed successfully
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for worker to process task")
	}

	close(quit)
	wg.Wait()

	// Verify worker has exited by adding a task that won't be processed
	select {
	case tasksC <- func() {
		t.Fatal("Worker should not process tasks after termination")
	}:
	default:
		// This is expected, worker has exited
	}
}

func TestWorkerQuitBeforeTask(t *testing.T) {
	tasksC := make(chan func(), 5)
	quit := make(chan struct{})

	// Start worker in separate goroutine
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		worker(tasksC, quit)
	}()

	// Terminate worker immediately
	close(quit)
	wg.Wait()

	// Verify worker has exited
	taskRan := false
	tasksC <- func() {
		taskRan = true
	}

	// Give some time for the task to potentially be processed
	time.Sleep(100 * time.Millisecond)

	if taskRan {
		t.Error("Task should not be processed after worker termination")
	}
}

func TestNew(t *testing.T) {
	wg := &sync.WaitGroup{}
	quit := make(chan struct{})
	maxTasks := 10

	w := New(wg, quit, maxTasks)

	if w.wg != wg {
		t.Errorf("Expected waitgroup %v, got %v", wg, w.wg)
	}

	if w.quit != quit {
		t.Errorf("Expected quit channel %v, got %v", quit, w.quit)
	}

	if cap(w.tasks) != maxTasks {
		t.Errorf("Expected tasks channel capacity %d, got %d", maxTasks, cap(w.tasks))
	}
}

func TestStart(t *testing.T) {
	wg := &sync.WaitGroup{}
	quit := make(chan struct{})
	maxTasks := 10
	workersN := 5

	w := New(wg, quit, maxTasks)
	w.Start(workersN)

	// buffered channel of size workersN
	doneChan := make(chan bool, workersN)
	// Ensure workers are running by enqueuing tasks and verifying they complete
	for i := 0; i < workersN; i++ {
		err := w.Enqueue(func() {
			doneChan <- true
		})
		if err != nil {
			t.Errorf("Failed to enqueue task: %v", err)
		}
	}

	// Wait for all tasks to complete
	for i := 0; i < workersN; i++ {
		select {
		case <-doneChan:
			// Task completed
		case <-time.After(time.Second):
			t.Fatal("Timeout waiting for workers to process tasks")
		}
	}

	close(quit)
	wg.Wait()
}

func TestEnqueueSuccess(t *testing.T) {
	wg := &sync.WaitGroup{}
	quit := make(chan struct{})
	maxTasks := 1

	w := New(wg, quit, maxTasks)
	w.Start(1)

	taskRan := false
	taskDone := make(chan bool)

	err := w.Enqueue(func() {
		taskRan = true
		taskDone <- true
	})

	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}

	select {
	case <-taskDone:
		if !taskRan {
			t.Error("Task did not run")
		}
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for task to run")
	}

	close(quit)
	wg.Wait()
}

// tests that Enqueue returns errTerminated when workers are terminated
func TestEnqueueTerminated(t *testing.T) {
	wg := &sync.WaitGroup{}
	quit := make(chan struct{})

	maxTasks := 1
	w := New(wg, quit, maxTasks)

	blockingTask := func() {
		// This task will never execute since we won't start any workers
		t.Fatal("This should never run")
	}
	// Fill the tasks channel to capacity (1) to block any further enqueues
	w.tasks <- blockingTask

	// Create a channel to get the result from a goroutine
	resultChan := make(chan error)

	// Try to enqueue in a separate goroutine (this will block)
	go func() {
		resultChan <- w.Enqueue(func() {
			t.Error("This task should not run")
		})
	}()

	// Give the goroutine time to reach the blocked select statement
	time.Sleep(100 * time.Millisecond)

	// Now close the quit channel
	close(quit)

	// The blocked Enqueue should now return errTerminated
	select {
	case err := <-resultChan:
		if err != errTerminated {
			t.Errorf("Expected error %v, got %v", errTerminated, err)
		}
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for Enqueue to return after termination")
	}
}

func TestDrain(t *testing.T) {
	wg := &sync.WaitGroup{}
	quit := make(chan struct{})
	maxTasks := 5

	w := New(wg, quit, maxTasks)

	for i := 0; i < 3; i++ {
		w.tasks <- func() {}
	}

	// task count should be 3, we never called Start
	if w.TasksCount() != 3 {
		t.Errorf("Expected 3 tasks, got %d", w.TasksCount())
	}

	w.Drain()

	if w.TasksCount() != 0 {
		t.Errorf("Expected 0 tasks after draining, got %d", w.TasksCount())
	}
}

func TestStartMultipleWorkers(t *testing.T) {
	wg := &sync.WaitGroup{}
	quit := make(chan struct{})
	maxTasks := 10
	workersN := 3

	w := New(wg, quit, maxTasks)
	w.Start(workersN)

	// buffered channel number of completed tasks
	taskResults := make(chan int, workersN*2)

	// Enqueue multiple tasks
	for i := 0; i < workersN*2; i++ {
		taskNum := i
		err := w.Enqueue(func() {
			// Simulate work which takes time to complete
			time.Sleep(50 * time.Millisecond)
			taskResults <- taskNum
		})
		if err != nil {
			t.Errorf("Failed to enqueue task: %v", err)
		}
	}

	// Collect results from all tasks
	completedTasks := 0
	for i := 0; i < workersN*2; i++ {
		select {
		case <-taskResults:
			completedTasks++
		// 3 * 2 * 50 = 300ms; grace period of 200ms
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Timeout waiting for workers to process tasks")
		}
	}

	if completedTasks != workersN*2 {
		t.Errorf("Expected %d completed tasks, got %d", workersN*2, completedTasks)
	}

	close(quit)
	wg.Wait()
}

func TestTasksCount(t *testing.T) {
	wg := &sync.WaitGroup{}
	quit := make(chan struct{})
	maxTasks := 5

	w := New(wg, quit, maxTasks)

	if w.TasksCount() != 0 {
		t.Errorf("Expected 0 tasks, got %d", w.TasksCount())
	}

	w.tasks <- func() {}
	w.tasks <- func() {}

	if w.TasksCount() != 2 {
		t.Errorf("Expected 2 tasks, got %d", w.TasksCount())
	}
}
