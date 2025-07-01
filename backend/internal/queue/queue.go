package queue

import "log"

// Job represents a task to be executed by a worker.
type Job struct {
	// A function that does the work.
	Action func() error
}

// JobQueue is a channel that holds all the pending jobs.
var JobQueue = make(chan Job, 100)

// Worker is a struct that pulls jobs from the queue and executes them.
type Worker struct {
	ID         int
	JobQueue   chan Job
	Quit       chan bool
}

// NewWorker creates a new worker.
func NewWorker(id int, jobQueue chan Job) Worker {
	return Worker{
		ID:         id,
		JobQueue:   jobQueue,
		Quit:       make(chan bool),
	}
}

// Start makes the worker listen for jobs.
func (w Worker) Start() {
	go func() {
		for {
			select {
			case job := <-w.JobQueue:
				if err := job.Action(); err != nil {
					log.Printf("Worker %d: Error processing job: %s", w.ID, err)
				}
			case <-w.Quit:
				return
			}
		}
	}()
}

// StartDispatcher creates and starts a number of workers.
func StartDispatcher(nworkers int) {
	for i := 1; i <= nworkers; i++ {
		worker := NewWorker(i, JobQueue)
		worker.Start()
	}
}