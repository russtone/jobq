package jobq

type Task struct {
	ID    int
	Retry bool
	Err   bool
}

type Result struct {
	*Task
}
