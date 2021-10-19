package jobq_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/russtone/jobq"
)

var errTest = errors.New("test")

func containsInt(nn []int, n int) bool {
	for _, i := range nn {
		if i == n {
			return true
		}
	}
	return false
}

type DoMock struct {
	mock.Mock
	progress int
	wait     sync.WaitGroup
}

func (do *DoMock) Do(task jobq.Task) (jobq.Result, error) {

	do.Called(task)

	t, ok := task.(*Task)
	if !ok {
		return nil, errors.New("cast failed")
	}

	// Tasks with ID >= progress will be waiting for do.wait WaitGroup.
	if t.ID >= do.progress {
		do.wait.Wait()
	}

	// Error.
	if t.Err {
		t.Err = false
		return nil, errTest
	}

	if t.Retry {
		t.Retry = false
		return nil, jobq.Retry(nil)
	}

	return &Result{task: t}, nil
}

//
// Task
//

type Task struct {
	ID    int
	Retry bool
	Err   bool
}

func (t *Task) TaskType() {}

var _ jobq.Task = new(Task)

//
// Result
//

type Result struct {
	task *Task
}

func (r *Result) ResultType() {}

var _ jobq.Result = new(Result)

//
// Test
//

func TestAll(t *testing.T) {

	tests := []struct {
		jobs     int   // count of jobs
		workers  int   // count of workers
		errs     []int // ids of jobs which must return error
		retries  []int // ids of jobs which must be retried
		progress int   // count of jobs which must be finished (the rest will wait for do.wait)
	}{
		{
			jobs:     10,
			workers:  3,
			errs:     []int{3, 4, 5},
			retries:  []int{6, 8},
			progress: 5,
		},
		{
			jobs:     100,
			workers:  5,
			errs:     []int{5, 10, 99},
			retries:  []int{50},
			progress: 10,
		},
		{
			jobs:     100,
			workers:  50,
			errs:     []int{50},
			retries:  []int{80},
			progress: 80,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {

			if tt.progress > tt.jobs {
				panic("invald test case: progress must be less then jobs count")
			}

			for _, id := range tt.errs {
				if id > tt.jobs {
					panic("invalid test case: id must be less then jobs count")
				}

				if containsInt(tt.retries, id) {
					panic("invalid test case: id must not be same as in retries")
				}
			}

			for _, id := range tt.retries {
				if id > tt.jobs {
					panic("invalid test case: id must be less then jobs count")
				}

				if containsInt(tt.errs, id) {
					panic("invalid test case: id must not be same as in errs")
				}
			}

			do := &DoMock{
				progress: tt.progress,
			}

			// Tasks with ID >= progress will be waiting for do.wait.
			for i := 0; i < tt.jobs; i++ {
				if containsInt(tt.retries, i) {
					do.On("Do", &Task{ID: i, Retry: true}).Return().Once()
				} else if containsInt(tt.errs, i) {
					do.On("Do", &Task{ID: i, Err: true}).Return().Once()
					continue
				}
				do.On("Do", &Task{ID: i}).Return().Once()
			}

			queue := jobq.New(do.Do, tt.workers, tt.jobs)
			createdAt := time.Now()

			queue.Start()

			waitResults := sync.WaitGroup{}
			waitResults.Add(1)

			// Process results
			go func() {
				defer waitResults.Done()

				var (
					res jobq.Result
					err error
				)

				done := 0
				errs := 0

				for queue.Next(&res, &err) {
					if err != nil {
						// The only error that could happen is testErr.
						assert.Error(t, errTest, err)
						errs++
					}

					done++
				}

				// All jobs must be proccessed.
				assert.Equal(t, tt.jobs, done)

				// There must be only 1 error.
				assert.Equal(t, len(tt.errs), errs)
			}()

			group := queue.Group()
			groupCreatedAt := time.Now()

			// Collect results from group
			waitResults.Add(1)

			// Process results from group.
			go func() {
				defer waitResults.Done()

				var (
					res jobq.Result
					err error
				)

				done := 0

				for group.Next(&res, &err) {
					done++
				}

				// All jobs must be proccessed.
				assert.Equal(t, tt.progress, done)
			}()

			do.wait.Add(1)

			// Add jobs to queue or group.
			for i := 0; i < tt.jobs; i++ {
				task := &Task{
					ID:    i,
					Retry: containsInt(tt.retries, i),
					Err:   containsInt(tt.errs, i),
				}

				// Add jobs with ID < tt.progress to group.
				// Add all other jobs to queue.
				if i < tt.progress {
					group.Add(task)
				} else {
					queue.Add(task)
				}
			}

			// Wait for tasks with ID < progress
			group.Wait()
			group.Close()

			// Check progress on group
			assert.Equal(t, 1.0, group.Progress())

			// Check speed on group
			assert.InDelta(t, float64(tt.progress)/float64(time.Since(groupCreatedAt).Seconds()), group.Speed(), 500)


			// Check speed on queue.
			assert.InDelta(t, float64(tt.progress)/float64(time.Since(createdAt).Seconds()), queue.Speed(), 500)

			// Check progress on queue.
			assert.Equal(t, float64(tt.progress)/float64(tt.jobs), queue.Progress())

			// Unlock rest of jobs.
			do.wait.Done()

			// Wait for all jobs to finish.
			queue.Wait()
			queue.Close()
			waitResults.Wait()

			do.AssertExpectations(t)
		})
	}

}
