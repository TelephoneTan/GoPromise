package test

import (
	"github.com/TelephoneTan/GoPromise/async/promise"
	"github.com/TelephoneTan/GoPromise/async/task"
	"testing"
	"time"
)

func TestOnceTask(t *testing.T) {
	task0 := (&task.Once[string]{
		Job: promise.Job[string]{
			Do: func(rs promise.Resolver[string], re promise.Rejector) {
				rs.ResolveValue(time.Now().String())
			},
		},
	}).Init()
	var allWork []*promise.Promise[any]
	for i := 0; i < 100000; i++ {
		allWork = append(allWork, promise.Then(task0.Do(), promise.FulfilledListener[string, any]{
			OnFulfilled: func(v string) any {
				println(v)
				return nil
			},
		}))
	}
	promise.AwaitAll(allWork)
}
