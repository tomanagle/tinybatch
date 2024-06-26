package batcher

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type Message struct {
	Topic  string
	Offset int64
	Key    []byte
	Value  []byte
}

type Job struct {
	ID      string
	Message Message
}

type JobResult struct {
	ID   string
	Data []byte
}

func TestBatcher(t *testing.T) {

	testCases := []struct {
		name               string
		maxBatchSize       int
		maxBatchDelay      time.Duration
		jobCount           int
		cancelContextAfter time.Duration
	}{
		{
			name:          "should complete all the jobs before returning",
			jobCount:      100_00,
			maxBatchSize:  10,
			maxBatchDelay: 1_500 * time.Millisecond,
		},
		{
			name:          "should complete after batch delay",
			jobCount:      5_000, // will never reach the maxBatchSize
			maxBatchSize:  100_000,
			maxBatchDelay: 1_000 * time.Millisecond,
		},
		{
			name:               "should handle errors gracefully if context is cancelled",
			jobCount:           5_000,
			maxBatchSize:       1_000,
			maxBatchDelay:      1_000 * time.Millisecond,
			cancelContextAfter: 3 * time.Millisecond,
		},
		{
			name:     "should set default options",
			jobCount: 10_000,
		},
	}

	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {

			assert := assert.New(t)

			count := tc.jobCount
			var successCounter atomic.Int64
			var errCounter atomic.Int64

			var processJobs = func(jobs []Job) ([]JobResult, error) {

				results := make([]JobResult, 0, len(jobs))

				for _, job := range jobs {
					result := JobResult{
						ID:   job.ID,
						Data: job.Message.Value,
					}
					successCounter.Add(1)

					results = append(results, result)
				}

				return results, nil
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var batcher *Batcher[Job, JobResult]

			if tc.maxBatchSize > 0 && tc.maxBatchDelay > 0 {
				batcher = New(
					ctx,
					processJobs,
					WithMaxBatchSize(tc.maxBatchSize),
					WithMaxBatchDelay(tc.maxBatchDelay),
				)
			} else {
				batcher = New(ctx, processJobs)
			}

			if tc.maxBatchSize != 0 {
				assert.Equal(tc.maxBatchSize, batcher.MaxBatchSize)
			} else { // default value
				assert.Equal(100, batcher.MaxBatchSize)
			}

			if tc.maxBatchDelay != 0 {
				assert.Equal(tc.maxBatchDelay, batcher.MaxBatchDelay)
			} else { // default value
				assert.Equal(1_000*time.Millisecond, batcher.MaxBatchDelay)
			}

			if tc.cancelContextAfter > 0 {
				time.AfterFunc(tc.cancelContextAfter, func() {
					cancel()
				})
			}

			batcher.Start()

			for i := 0; i < count; i++ {
				err := batcher.Add(Job{
					ID: strconv.Itoa(i),
					Message: Message{
						Topic:  "test",
						Offset: int64(i),
						Key:    []byte("key"),
						Value:  []byte("value"),
					},
				})

				if err != nil {
					errCounter.Add(1)
				}
			}

			if tc.cancelContextAfter == 0 {
				batcher.Stop()
			}

			success := successCounter.Load()
			errs := errCounter.Load()

			assert.True(true)
			assert.Equal(count, int(errs+success))

		})

	}

}
