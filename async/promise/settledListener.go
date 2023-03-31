package promise

type SettledListener[T any] struct {
	OnSettled func() *Promise[T]
}
