package test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var errTest = errors.New("test")

type DoMock struct {
	mock.Mock
	progress   int
	progressWG sync.WaitGroup
	wait       sync.WaitGroup
}

func (do *DoMock) Do(task *Task) (*Result, error) {

	do.Called(task)

	defer func() {
		if task.ID < do.progress {
			do.progressWG.Done()
		}
	}()

	if task.ID >= do.progress {
		do.wait.Wait()
	}

	// Error.
	if task.Err {
		task.Err = false
		return nil, errTest
	}

	if task.Retry {
		task.Retry = false
		return nil, Retry(nil)
	}

	return &Result{Task: task}, nil
}

func TestJobqueue(t *testing.T) {

	tests := []struct {
		jobs     int
		workers  int
		errID    int
		retryID  int
		destID   int
		progress int
	}{
		{jobs: 10, workers: 3, errID: 5, retryID: 6, destID: 7, progress: 5},
		{jobs: 100, workers: 5, errID: 5, retryID: 50, destID: 69, progress: 10},
		{jobs: 100, workers: 50, errID: 50, retryID: 80, destID: 33, progress: 80},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {

			if tt.progress > tt.jobs {
				panic("invald test case: progress must be less then jobs count")
			}

			if tt.errID > tt.jobs || tt.retryID > tt.jobs || tt.destID > tt.jobs {
				panic("invalid test case: id must be less then jobs count")
			}

			do := &DoMock{
				progress: tt.progress,
			}

			do.progressWG.Add(tt.progress)

			for i := 0; i < tt.jobs; i++ {
				if i == tt.retryID {
					do.On("Do", &Task{ID: i, Retry: true}).Return().Once()
				} else if i == tt.errID {
					do.On("Do", &Task{ID: i, Err: true}).Return().Once()
					continue
				}
				do.On("Do", &Task{ID: i}).Return().Once()
			}

			queue := newQueue(do.Do, tt.workers, tt.jobs)

			queue.Start()

			wg := sync.WaitGroup{}
			wg.Add(2)

			// Process errors
			go func() {
				defer wg.Done()

				var err error

				count := 0

				for queue.Err(&err) {
					if err != nil {
						// The only error that could happen is testErr.
						assert.Error(t, errTest, err)
						count++
					}
				}

				// There must be only 1 error.
				assert.Equal(t, 1, count)
			}()

			// Process results
			go func() {
				defer wg.Done()

				var res Result

				count := 0

				for queue.Next(&res) {
					count += 1
				}

				// All jobs must be proccessed.
				assert.Equal(t, count, tt.jobs-1)
			}()

			var dest Result

			do.wait.Add(1)

			for i := 0; i < tt.jobs; i++ {
				task := &Task{
					ID:    i,
					Retry: i == tt.retryID,
					Err:   i == tt.errID,
				}

				if i != tt.destID {
					queue.Add(task)
				} else {
					queue.Add(task, Dest(&dest))
				}
			}

			// Wait for jobs to finish to compare progress.
			do.progressWG.Wait()
			assert.Equal(t, float64(tt.progress)/float64(tt.jobs), queue.Progress())

			// Unlock rest of jobs.
			do.wait.Done()

			queue.WaitJobs()

			// Check job with dest.
			assert.Equal(t, Result{Task: &Task{ID: tt.destID}}, dest)

			queue.Stop()

			queue.WaitWorkers()

			// Wait erors and output processors.
			wg.Wait()

			do.AssertExpectations(t)
		})
	}

}
