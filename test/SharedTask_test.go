package test

import (
	"github.com/TelephoneTan/GoPromise/async/promise"
	"github.com/TelephoneTan/GoPromise/async/task"
	"testing"
	"time"
)

func TestSharedTask(t *testing.T) {
	task0 := task.NewSharedTask(promise.Job[string]{
		Do: func(rs promise.Resolver[string], re promise.Rejector) {
			time.Sleep(1 * time.Second)
			rs.ResolveValue(time.Now().String())
		},
	})
	var all []promise.Promise[any]
	task.NewTimedTask(promise.Job[bool]{
		Do: func(rs promise.Resolver[bool], re promise.Rejector) {
			all = append(all, promise.Then(task0.Do(), promise.FulfilledListener[string, any]{
				OnFulfilled: func(v string) any {
					println(v)
					return nil
				},
			}))
			rs.ResolveValue(true)
		},
	}, 200*time.Millisecond, 50).Start().Await()
	promise.AwaitAll(all)
}
