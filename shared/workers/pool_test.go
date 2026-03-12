package workers

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestStartWorkerPool(t *testing.T) {
	// Test 1: Pool creation with valid config
	t.Run("ValidConfig", func(t *testing.T) {
		jobFunc := func(job string) error {
			return nil
		}

		pool := StartWorkerPool(jobFunc, 2, 10, 10)
		if pool == nil {
			t.Error("Expected pool to be created, got nil")
		}
		if pool.GetJobChan() == nil {
			t.Error("Expected job channel to be created, got nil")
		}
		if pool.GetResultChan() == nil {
			t.Error("Expected result channel to be created, got nil")
		}

		pool.Shutdown()
	})

	// Test 2: Pool with zero workers
	t.Run("ZeroWorkers", func(t *testing.T) {
		jobFunc := func(job string) error {
			return nil
		}

		pool := StartWorkerPool(jobFunc, 0, 10, 10)
		if pool == nil {
			t.Error("Expected pool to be created even with zero workers, got nil")
		}

		pool.Shutdown()
	})

	// Test 3: Pool with single worker
	t.Run("SingleWorker", func(t *testing.T) {
		jobFunc := func(job string) error {
			return nil
		}

		pool := StartWorkerPool(jobFunc, 1, 10, 10)
		if pool == nil {
			t.Error("Expected pool to be created with single worker, got nil")
		}

		pool.Shutdown()
	})

	// Test 4: Pool with multiple workers
	t.Run("MultipleWorkers", func(t *testing.T) {
		jobFunc := func(job string) error {
			return nil
		}

		pool := StartWorkerPool(jobFunc, 5, 10, 10)
		if pool == nil {
			t.Error("Expected pool to be created with multiple workers, got nil")
		}

		pool.Shutdown()
	})

	// Test 5: Pool with zero queue size
	t.Run("ZeroQueueSize", func(t *testing.T) {
		jobFunc := func(job string) error {
			return nil
		}

		pool := StartWorkerPool(jobFunc, 2, 0, 10)
		if pool == nil {
			t.Error("Expected pool to be created with zero queue size, got nil")
		}

		pool.Shutdown()
	})
}

func TestPoolGetJobChan(t *testing.T) {
	// Test 1: Job channel retrieval
	t.Run("JobChannelRetrieval", func(t *testing.T) {
		jobFunc := func(job string) error {
			return nil
		}

		pool := StartWorkerPool(jobFunc, 2, 10, 10)
		defer pool.Shutdown()

		jobChan := pool.GetJobChan()
		if jobChan == nil {
			t.Error("Expected job channel to be returned, got nil")
		}

		// Verify we can send jobs to the channel
		select {
		case jobChan <- "test":
			// Successfully sent job
		default:
			t.Error("Failed to send job to job channel")
		}
	})
}

func TestPoolGetResultChan(t *testing.T) {
	// Test 1: Result channel retrieval
	t.Run("ResultChannelRetrieval", func(t *testing.T) {
		jobFunc := func(job string) error {
			return nil
		}

		pool := StartWorkerPool(jobFunc, 2, 10, 10)
		defer pool.Shutdown()

		resultChan := pool.GetResultChan()
		if resultChan == nil {
			t.Error("Expected result channel to be returned, got nil")
		}
	})

	// Test 2: Result channel buffer size
	t.Run("ResultChannelBufferSize", func(t *testing.T) {
		jobFunc := func(job string) error {
			return nil
		}

		bufferSize := 3
		pool := StartWorkerPool(jobFunc, 1, 10, bufferSize)
		defer pool.Shutdown()

		resultChan := pool.GetResultChan()
		jobChan := pool.GetJobChan()

		// Send more jobs than result buffer size
		for i := 0; i < bufferSize+2; i++ {
			jobChan <- "test"
		}

		// Wait for some results
		time.Sleep(100 * time.Millisecond)

		// Verify results are being sent
		receivedResults := 0
		timeout := false
		for i := 0; i < bufferSize+2 && !timeout; i++ {
			select {
			case <-resultChan:
				receivedResults++
			case <-time.After(50 * time.Millisecond):
				timeout = true
			}
		}

		if receivedResults == 0 {
			t.Error("Expected to receive results from result channel")
		}
	})
}

func TestPoolWait(t *testing.T) {
	// Test 1: Waiting for workers to finish
	t.Run("WaitForWorkers", func(t *testing.T) {
		jobFunc := func(job string) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		}

		pool := StartWorkerPool(jobFunc, 2, 10, 10)

		// Send jobs
		jobChan := pool.GetJobChan()
		for i := 0; i < 5; i++ {
			jobChan <- "test"
		}

		// Close job channel to signal workers to stop waiting for jobs
		close(jobChan)

		// Wait for all jobs to complete
		pool.Wait()
	})

	// Test 2: Wait with no jobs
	t.Run("WaitWithNoJobs", func(t *testing.T) {
		jobFunc := func(job string) error {
			return nil
		}

		pool := StartWorkerPool(jobFunc, 2, 10, 10)

		// Close job channel immediately
		close(pool.GetJobChan())

		// Wait for workers to finish
		pool.Wait()
	})
}

func TestPoolShutdown(t *testing.T) {
	// Test 1: Graceful shutdown
	t.Run("GracefulShutdown", func(t *testing.T) {
		jobFunc := func(job string) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		}

		pool := StartWorkerPool(jobFunc, 2, 10, 10)

		// Send jobs
		jobChan := pool.GetJobChan()
		for i := 0; i < 3; i++ {
			jobChan <- "test"
		}

		// Shutdown should wait for jobs to complete
		pool.Shutdown()

		// After shutdown, job channel should be closed
		// Note: We can't test this directly because the channel is already closed
		// but we can verify shutdown completed without error
	})

	// Test 2: Shutdown with active jobs
	t.Run("ShutdownWithActiveJobs", func(t *testing.T) {
		jobFunc := func(job string) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		}

		pool := StartWorkerPool(jobFunc, 2, 10, 10)

		// Send jobs
		jobChan := pool.GetJobChan()
		for i := 0; i < 5; i++ {
			jobChan <- "test"
		}

		// Shutdown should wait for all jobs to complete
		done := make(chan bool)
		go func() {
			pool.Shutdown()
			done <- true
		}()

		// Verify shutdown completes
		select {
		case <-done:
			// Shutdown completed successfully
		case <-time.After(1 * time.Second):
			t.Error("Expected shutdown to complete within 1 second")
		}
	})

	// Test 3: Multiple shutdowns should panic
	t.Run("MultipleShutdowns", func(t *testing.T) {
		jobFunc := func(job string) error {
			return nil
		}

		pool := StartWorkerPool(jobFunc, 2, 10, 10)

		// First shutdown
		pool.Shutdown()

		// Second shutdown should panic (closing already-closed channels)
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic on second shutdown, but got none")
			}
		}()

		pool.Shutdown()
	})
}

func TestPoolJobProcessing(t *testing.T) {
	// Test 1: Successful job processing
	t.Run("SuccessfulJobProcessing", func(t *testing.T) {
		var mu sync.Mutex
		processedJobs := []string{}

		jobFunc := func(job string) error {
			mu.Lock()
			processedJobs = append(processedJobs, job)
			mu.Unlock()
			return nil
		}

		pool := StartWorkerPool(jobFunc, 2, 10, 10)
		defer pool.Shutdown()

		jobChan := pool.GetJobChan()
		resultChan := pool.GetResultChan()

		// Send jobs
		jobs := []string{"job1", "job2", "job3"}
		for _, job := range jobs {
			jobChan <- job
		}

		// Wait for results
		for i := 0; i < len(jobs); i++ {
			select {
			case err := <-resultChan:
				if err != nil {
					t.Errorf("Expected no error for job, got: %v", err)
				}
			case <-time.After(1 * time.Second):
				t.Error("Expected to receive result within 1 second")
			}
		}

		// Verify all jobs were processed
		mu.Lock()
		if len(processedJobs) != len(jobs) {
			t.Errorf("Expected %d jobs to be processed, got %d", len(jobs), len(processedJobs))
		}
		mu.Unlock()
	})

	// Test 2: Job processing with errors
	t.Run("JobProcessingWithErrors", func(t *testing.T) {
		jobFunc := func(job string) error {
			if job == "error" {
				return errors.New("test error")
			}
			return nil
		}

		pool := StartWorkerPool(jobFunc, 2, 10, 10)
		defer pool.Shutdown()

		jobChan := pool.GetJobChan()
		resultChan := pool.GetResultChan()

		// Send jobs
		jobChan <- "success"
		jobChan <- "error"

		// Wait for results
		errorCount := 0
		for i := 0; i < 2; i++ {
			select {
			case err := <-resultChan:
				if err != nil {
					errorCount++
				}
			case <-time.After(1 * time.Second):
				t.Error("Expected to receive result within 1 second")
			}
		}

		if errorCount != 1 {
			t.Errorf("Expected 1 error, got %d", errorCount)
		}
	})
}

func TestPoolPanicRecovery(t *testing.T) {
	// Test 1: Worker panic recovery
	t.Run("WorkerPanicRecovery", func(t *testing.T) {
		panicCount := 0
		var mu sync.Mutex

		jobFunc := func(job string) error {
			if job == "panic" {
				panic("test panic")
			}
			return nil
		}

		pool := StartWorkerPool(jobFunc, 2, 10, 10)
		defer pool.Shutdown()

		jobChan := pool.GetJobChan()
		resultChan := pool.GetResultChan()

		// Send a job that will cause panic
		jobChan <- "panic"

		// Wait a bit for panic to be handled
		time.Sleep(200 * time.Millisecond)

		// Send another job to verify worker is still working
		jobChan <- "normal"

		// Wait for result
		select {
		case err := <-resultChan:
			if err != nil {
				mu.Lock()
				panicCount++
				mu.Unlock()
			}
		case <-time.After(1 * time.Second):
			t.Error("Expected to receive result after panic recovery")
		}
	})
}

func TestPoolConcurrency(t *testing.T) {
	// Test 1: Concurrent job submission
	t.Run("ConcurrentJobSubmission", func(t *testing.T) {
		var mu sync.Mutex
		processedJobs := 0

		jobFunc := func(job string) error {
			time.Sleep(10 * time.Millisecond)
			mu.Lock()
			processedJobs++
			mu.Unlock()
			return nil
		}

		pool := StartWorkerPool(jobFunc, 3, 100, 100)
		defer pool.Shutdown()

		jobChan := pool.GetJobChan()

		// Submit jobs concurrently
		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(jobNum int) {
				defer wg.Done()
				jobChan <- "test"
			}(i)
		}

		wg.Wait()

		// Wait for all jobs to be processed
		time.Sleep(500 * time.Millisecond)

		mu.Lock()
		if processedJobs != 20 {
			t.Errorf("Expected 20 jobs to be processed, got %d", processedJobs)
		}
		mu.Unlock()
	})
}
