package task

import (
	"github.com/TelephoneTan/GoPromise/async/promise"
	"sync"
)

type _Once[T any] struct {
	Job       promise.Job[T]
	Semaphore promise.Semaphore
	once      sync.Once
	promise   promise.Promise[T]
}

type Once[T any] struct {
	*_Once[T]
}

func NewOnceTask[T any](job promise.Job[T]) Once[T] {
	return Once[T]{&_Once[T]{Job: job}}.init()
}

func NewOnceTaskWithSemaphore[T any](job promise.Job[T], semaphore promise.Semaphore) Once[T] {
	return Once[T]{&_Once[T]{Job: job, Semaphore: semaphore}}.init()
}

func NewOnceTaskEmpty[T any]() Once[T] {
	return Once[T]{&_Once[T]{}}.init()
}

func (o Once[T]) init() Once[T] {
	return o
}

func (o Once[T]) Cancel() {
	o.once.Do(func() {
		o.promise = promise.Cancelled[T]()
	})
	o.promise.Cancel()
}

func (o Once[T]) DoJob(job promise.Job[T]) promise.Promise[T] {
	o.once.Do(func() {
		o.promise = promise.NewPromiseWithSemaphore(job, o.Semaphore)
	})
	return o.promise
}

func (o Once[T]) Do() promise.Promise[T] {
	return o.DoJob(o.Job)
}
