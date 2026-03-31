package workers

import (
	"errors"
	"sync"
)

// errTerminated is a custom error indicating that the worker pool has been terminated.
var errTerminated = errors.New("terminated")

// worker is the function that executes tasks from the tasks channel.
// It runs in a goroutine and continuously checks for new tasks or a quit signal.
func worker(tasksC <-chan func(), quit <-chan struct{}) {
	// The 'for' loop runs indefinitely until the quit channel is closed.
	for {
		// 'select' allows us to wait on multiple channel operations.
		select {
		// If the quit channel is closed, it means we should stop working.
		case <-quit:
			// Return from the function, which terminates the goroutine.
			return
		// If a task is available in the tasks channel, execute it.
		case job := <-tasksC:
			// Execute the function received from the tasks channel.
			job()
		}
	}
}

// Workers is a struct that manages a pool of worker goroutines.
type Workers struct {
	// quit is a channel used to signal workers to stop.
	quit chan struct{}
	// wg is a WaitGroup used to wait for all worker goroutines to finish.
	wg *sync.WaitGroup
	// tasks is a buffered channel for tasks to be executed by workers.
	tasks chan func()
}

// New creates a new Workers instance.
// It initializes the tasks channel, quit channel, and WaitGroup.
// maxTasks specifies the maximum number of tasks that can be buffered.
func New(wg *sync.WaitGroup, quit chan struct{}, maxTasks int) *Workers {
	return &Workers{
		// Create a buffered channel for tasks. The buffer size is maxTasks.
		tasks: make(chan func(), maxTasks),
		// Assign the quit channel.
		quit: quit,
		// Assign the WaitGroup.
		wg: wg,
	}
}

// Start starts a specified number of worker goroutines.
// workersN specifies the number of workers to start.
func (w *Workers) Start(workersN int) {
	// Start the desired number of workers.
	for i := 0; i < workersN; i++ {
		// Increment the WaitGroup counter.
		w.wg.Add(1)
		// Start a new goroutine for each worker.
		go func() {
			// Decrement the WaitGroup counter when the goroutine finishes.
			defer w.wg.Done()
			// Call the worker function to start processing tasks.
			worker(w.tasks, w.quit)
		}()
	}
}

// Enqueue adds a task to the tasks channel.
// It returns an error if the worker pool has been terminated.
func (w *Workers) Enqueue(fn func()) error {
	// 'select' allows us to wait on multiple channel operations.
	select {
	// If the tasks channel is not full, add the task.
	case w.tasks <- fn:
		// Return nil, indicating success.
		return nil
	// If the quit channel is closed, it means the worker pool has been terminated.
	case <-w.quit:
		// Return the errTerminated error.
		return errTerminated
	}
}

// Drain removes all remaining tasks from the tasks channel without executing them.
func (w *Workers) Drain() {
	// Keep looping until the tasks channel is empty.
	for {
		// 'select' allows us to check if there are any tasks in the channel.
		select {
		// If a task is available, remove it and continue.
		case <-w.tasks:
			continue
		// If the tasks channel is empty, exit the loop.
		default:
			return
		}
	}
}

// TasksCount returns the number of tasks currently in the tasks channel.
func (w *Workers) TasksCount() int {
	// returns the number of elements in the tasks channel.
	return len(w.tasks)
}
