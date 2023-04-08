package test

import (
	"fmt"
	"github.com/TelephoneTan/GoPromise/async/promise"
	"sync/atomic"
	"testing"
	"time"
)

func TestPromise(t *testing.T) {
	num := 20
	var x atomic.Int32
	var allWork []promise.Promise[string]
	interval := 500 * time.Millisecond
	semaphore := promise.NewSemaphore(1)
	for i := 0; i < num; i++ {
		ii := i
		p := promise.NewPromiseWithSemaphore(promise.Job[string]{
			Do: func(resolver promise.Resolver[string], rejector promise.Rejector) {
				time.Sleep(interval)
				resolver.ResolveValue("")
			},
		}, semaphore)
		allWork = append(allWork, p)
		p.SetTimeout(interval*5, &promise.TimeOutListener{OnTimeOut: func(duration time.Duration) {
			fmt.Printf("任务 %d 已超时（%f s）\n", ii, duration.Seconds())
		}}).SetTimeout(interval*10, &promise.TimeOutListener{OnTimeOut: func(duration time.Duration) {
			fmt.Printf("任务 %d 已超时（%f s）\n", ii, duration.Seconds())
		}})
		p = promise.Then(p, promise.FulfilledListener[string, string]{
			OnFulfilled: func(value string) any {
				return fmt.Sprintf("%d -> x = %d", ii, x.Add(1))
			},
		})
		allWork = append(allWork, p)
		p = promise.Then(p, promise.FulfilledListener[string, string]{
			OnFulfilled: func(value string) any {
				println(value)
				return nil
			},
		})
		allWork = append(allWork, p)
		p = promise.Catch(p, promise.RejectedListener[string]{
			OnRejected: func(reason error) any {
				println(ii, "Error!", reason.Error())
				return nil
			},
		})
		allWork = append(allWork, p)
		for i := 0; i < 10; i++ {
			p = promise.Finally[string, any](p, promise.SettledListener[any]{})
			allWork = append(allWork, p)
		}
	}
	promise.ThenAll[any](promise.CompoundFulfilledListener[string, string, any]{}, allWork, allWork).Await()
	fmt.Printf("执行 %d 个任务，最后的结果是 %d\n", num, x.Load())
	time.Sleep(100 * time.Hour)
}
