package jobq

// retryable represents error which indicates
// that task must retried.
type retryable struct {
	err error
}

// Error allows retryable to implement error interface.
func (r *retryable) Error() string {
	if r.err != nil {
		return r.err.Error()
	}

	return "retry"
}

// ErrRetry is error to return from DoFunc to indicate that retry is required.
var ErrRetry = &retryable{}

// Retry wraps given error and returns retryable error.
func Retry(err error) error {
	return &retryable{err}
}
