package promise

type SettledListener[T any] struct {
	OnSettled func() *Type[T]
}
