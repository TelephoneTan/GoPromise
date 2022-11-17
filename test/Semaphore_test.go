package test

import (
	"fmt"
	"github.com/TelephoneTan/GoPromise/async/promise"
	"sync/atomic"
	"testing"
)

func TestSemaphore(t *testing.T) {
	testN(1)
	testN(5)
}

func testN(n uint64) {
	var allWork []*promise.Type[any]
	semaphore := new(promise.Semaphore).Init(n)
	var counter atomic.Uint64
	for i := 0; i < 100; i++ {
		ii := i
		allWork = append(allWork, (&promise.Type[any]{
			Semaphore: semaphore,
			Job: promise.Job[any]{
				Do: func(rs promise.Resolver[any], re promise.Rejector) {
					if ii == 55 || ii == 66 || ii == 77 {
						panic(fmt.Sprintf("#%d 不干了", ii))
					}
					fmt.Printf("#%d 开始值 %d\n", ii, counter.Load())
					for i := 0; i < 1000000; i++ {
						counter.Add(1)
					}
					fmt.Printf("#%d 结束值 %d\n", ii, counter.Load())
					rs.ResolveValue(nil)
				},
			},
		}).Init())
	}
	p := promise.ThenRequired(promise.CompoundFulfilledListener[any, any, any]{
		OnFulfilled: func(cv *promise.CompoundResult[any, any]) any {
			println("全部成功")
			return nil
		},
	}, allWork)
	p = promise.Catch(p, promise.RejectedListener[any]{
		OnRejected: func(reason error) any {
			println("出错了:", reason.Error())
			return nil
		},
	})
	p.Await()
	fmt.Printf("总体结束值 %d\n", counter.Load())
}
