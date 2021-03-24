package test

//go:generate jobq -package test -task Task -result Result

type Task struct {
	ID    int
	Retry bool
	Err   bool
}

type Result struct {
	*Task
}
