package processor

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"
)

// Define a simple job and result type for testing.
type TestJob struct {
	Data string
}

type TestResult struct {
	Processed bool
}

func batchProcessor(jobs []TestJob) ([]TestResult, error) {
	results := make([]TestResult, len(jobs))
	for i := range jobs {
		results[i] = TestResult{Processed: true}
	}
	return results, nil
}

func TestProcessor(t *testing.T) {

	testCases := []struct {
		name          string
		jobCount      int
		maxBatchSize  int
		maxBatchDelay time.Duration
	}{
		{
			name:          "should process lots of small batches",
			jobCount:      10,
			maxBatchSize:  10,
			maxBatchDelay: 1_000 * time.Millisecond,
		},
		{
			name:          "should process after delay time",
			jobCount:      1_000,
			maxBatchSize:  10_000,
			maxBatchDelay: 1_000 * time.Millisecond,
		},
		{
			name:          "should process after reaching max batch size",
			jobCount:      10_000,
			maxBatchSize:  1_000,
			maxBatchDelay: 1 * time.Minute,
		},
	}

	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			jobsChannel := make(chan TestJob)
			wg := &sync.WaitGroup{}
			processor := New(NewProcessorParams[TestJob, TestResult]{
				Ctx:            ctx,
				MaxBatchDelay:  tc.maxBatchDelay,
				MaxBatchSize:   tc.maxBatchSize,
				Jobs:           jobsChannel,
				BatchProcessor: batchProcessor,
				Wg:             wg,
			})

			go processor.Start()

			for i := 0; i < tc.jobCount; i++ {
				jobsChannel <- TestJob{Data: "Job " + strconv.Itoa(i+1)}
			}

			doneCh := make(chan bool)
			go func() {
				wg.Wait()
				close(doneCh)
			}()

			select {
			case <-doneCh:
				t.Log("All jobs processed")
			case <-time.After(10 * time.Second):
				t.Errorf("Timeout: Not all jobs were processed in time")
			}
		})

	}

}
