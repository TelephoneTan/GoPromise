package task

import (
	"github.com/TelephoneTan/GoPromise/async/promise"
	"sync"
)

type Once[T any] struct {
	Job       promise.Job[T]
	Semaphore *promise.Semaphore
	once      sync.Once
	promise   *promise.Promise[T]
}

func NewOnceTask[T any](job promise.Job[T]) *Once[T] {
	return (&Once[T]{Job: job}).Init()
}

func NewOnceTaskWithSemaphore[T any](job promise.Job[T], semaphore *promise.Semaphore) *Once[T] {
	return (&Once[T]{Job: job, Semaphore: semaphore}).Init()
}

func NewOnceTaskEmpty[T any]() *Once[T] {
	return (&Once[T]{}).Init()
}

func (o *Once[T]) Init() *Once[T] {
	return o
}

func (o *Once[T]) Cancel() {
	o.once.Do(func() {
		o.promise = promise.Cancelled[T]()
	})
	o.promise.Cancel()
}

func (o *Once[T]) DoJob(job promise.Job[T]) *promise.Promise[T] {
	o.once.Do(func() {
		o.promise = promise.NewPromiseWithSemaphore(job, o.Semaphore)
	})
	return o.promise
}

func (o *Once[T]) Do() *promise.Promise[T] {
	return o.DoJob(o.Job)
}
