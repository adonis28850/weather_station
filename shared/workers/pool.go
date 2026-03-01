package workers

import (
	"sync"

	"weather-station/shared/logger"
)

// JobFunc is a function that processes a job and returns an error
type JobFunc[T any] func(job T) error

// Pool manages a pool of worker goroutines
// T is the type of job to process
type Pool[T any] struct {
	wg          *sync.WaitGroup
	restartChan chan int
	jobChan     chan T
	resultChan  chan error
}

// StartWorkerPool initializes and manages the worker pool
// Creates workers, handles worker restarts on panic, and returns a Pool struct
func StartWorkerPool[T any](
	jobFunc JobFunc[T],
	numWorkers int,
	jobQueueSize int,
	resultQueueSize int,
) *Pool[T] {
	jobChan := make(chan T, jobQueueSize)
	resultChan := make(chan error, resultQueueSize)
	
	var wg sync.WaitGroup
	restartChan := make(chan int, numWorkers)

	// Worker restart handler
	go func() {
		for workerID := range restartChan {
			logger.Info("Restarting worker %d after panic", workerID)
			wg.Add(1)
			go worker(workerID, jobFunc, jobChan, resultChan, &wg, restartChan)
		}
	}()

	// Create initial workers
	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go worker(i, jobFunc, jobChan, resultChan, &wg, restartChan)
		logger.Info("Started worker %d", i)
	}

	return &Pool[T]{
		wg:          &wg,
		restartChan: restartChan,
		jobChan:     jobChan,
		resultChan:  resultChan,
	}
}

// GetJobChan returns the channel for submitting jobs to the worker pool
func (p *Pool[T]) GetJobChan() chan<- T {
	return p.jobChan
}

// GetResultChan returns the channel for receiving results from the worker pool
func (p *Pool[T]) GetResultChan() <-chan error {
	return p.resultChan
}

// Wait waits for all workers to finish
func (p *Pool[T]) Wait() {
	p.wg.Wait()
}

// Shutdown gracefully shuts down the worker pool
func (p *Pool[T]) Shutdown() {
	close(p.jobChan)
	close(p.restartChan)
	p.Wait()
}

// worker processes jobs from the job queue
// Each worker goroutine continuously pulls jobs from the job channel and processes them
// Uses sync.WaitGroup to ensure proper shutdown without race conditions
// Automatically restarts on panic to maintain worker pool size
func worker[T any](
	id int,
	jobFunc JobFunc[T],
	jobChan <-chan T,
	resultChan chan<- error,
	wg *sync.WaitGroup,
	restartChan chan<- int,
) {
	// Panic recovery to prevent worker crashes from taking down the entire application
	// When a panic occurs, the worker logs it and signals for restart
	// Use non-blocking send to prevent deadlock if restartChan is full
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Worker %d recovered from panic: %v. Requesting restart...", id, r)
			select {
			case restartChan <- id:
				// Successfully sent restart request
			default:
				// Channel is full, log warning but don't block
				logger.Error("Warning: restart channel full, worker %d will not be restarted", id)
			}
		}
	}()

	// Signal completion when the worker exits (after jobChan is closed)
	// This prevents race conditions during shutdown
	defer wg.Done()

	// Process jobs until the job channel is closed
	for job := range jobChan {
		logger.Info("Worker %d processing job", id)
		resultChan <- jobFunc(job)
	}
}