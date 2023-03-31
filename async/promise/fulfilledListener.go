package promise

type FulfilledListener[NEED any, SUPPLY any] struct {
	OnFulfilled func(value NEED) any
}
