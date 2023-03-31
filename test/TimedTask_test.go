package test

import (
	"fmt"
	"github.com/TelephoneTan/GoPromise/async/promise"
	"github.com/TelephoneTan/GoPromise/async/task"
	"github.com/TelephoneTan/GoPromise/util"
	"testing"
	"time"
)

func TestTimedTask(t *testing.T) {
	task0 := (&task.Timed{
		Job: promise.Job[bool]{
			Do: func(rs promise.Resolver[bool], re promise.Rejector) {
				println(time.Now().String())
				rs.ResolveValue(util.Ptr(true))
			},
		},
	}).
		Init(50*time.Millisecond, 152)
	//Init(50*time.Millisecond, 2)
	//Init(50*time.Millisecond, 3)
	//Init(50*time.Millisecond, 4)
	addTimes := task0.AddTimesBy(1)
	aS := promise.Then(addTimes, promise.FulfilledListener[int64, any]{
		OnFulfilled: func(v *int64) any {
			fmt.Printf("成功增加了 %d 次运行\n", *v)
			return nil
		},
	})
	aF := promise.Catch(aS, promise.RejectedListener[any]{
		OnRejected: func(reason error) any {
			fmt.Printf("增加运行失败：%v\n", reason)
			return nil
		},
	})
	aF.Await()
	reduceTimes := task0.ReduceTimeBy(4)
	rS := promise.Then(reduceTimes, promise.FulfilledListener[int64, any]{
		OnFulfilled: func(v *int64) any {
			fmt.Printf("成功减少了 %d 次运行\n", *v)
			return nil
		},
	})
	rF := promise.Catch(rS, promise.RejectedListener[any]{
		OnRejected: func(reason error) any {
			fmt.Printf("减少运行失败：%v\n", reason)
			return nil
		},
	})
	rF.Await()
	task0Promise := task0.Start()
	for i := 0; i < 1000; i++ {
		task0Promise = task0.Start()
	}
	(&promise.Promise[any]{
		Job: promise.Job[any]{
			Do: func(rs promise.Resolver[any], re promise.Rejector) {
				time.Sleep(9 * time.Second)
				task0.Pause()
				println("已暂停")
				time.Sleep(6 * time.Second)
				task0.SetInterval(1 * time.Second)
				println("已更改间隔")
				d := 5 * time.Second
				task0.Resume(d)
				println("已恢复，下一次任务将延迟", d.Seconds(), "秒执行")
				time.Sleep(10 * time.Second)
				task0.Cancel()
				println("已取消")
				rs.ResolveValue(nil)
			},
		},
	}).Init()
	tS := promise.Then(task0Promise, promise.FulfilledListener[int64, any]{
		OnFulfilled: func(v *int64) any {
			println("一共运行了", *v, "次")
			return nil
		},
	})
	tC := promise.ForCancel[any](tS, promise.CancelledListener{
		OnCancelled: func() {
			println("不妙，被取消掉了")
		},
	})
	tC.Await()
}
