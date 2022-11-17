# Go Promise

一个仿照 JavaScript 中 `Promise` 风格的 Go 语言异步任务框架。

## Promise 风格

除了下述的不同点外，其余特性与 JavaScript 中的 `Promise` 保持一致。

不同点：

1. 由于 Go 语言的 `强类型` 特性，成功值无法在 `Promise` 链条上持续传递，只能传递到下一个节点。

## 用法

具体的用法体现在测试用例中：

### [MultipleSemaphore_test.go](./test/MultipleSemaphore_test.go)

同一个任务受到多个并发度控制

### [OnceTask_test.go](./test/OnceTask_test.go)

无论结果如何，只会被执行一次的任务

### [Promise_test.go](./test/Promise_test.go)

* 通过构造任务之间的依赖关系来创建任务链
* 给任务设置超时

### [Semaphore_test.go](./test/Semaphore_test.go)

受单个并发度控制的任务

### [SharedTask_test.go](./test/SharedTask_test.go)

同一时间只会有一个任务在执行，但可以接受多次结果请求，任务结束后统一发布结果的“共享任务”

### [TimedTask_test.go](./test/TimedTask_test.go)

* 创建定时任务
* 更改定时任务的计划运行次数
* 更改定时任务的运行时间间隔

### [CatchFault_test.go](./test/CatchFault_test.go)

当异步任务在被执行的过程中引发了 Segmentation fault，会自动被 `Catch` 分支捕捉到