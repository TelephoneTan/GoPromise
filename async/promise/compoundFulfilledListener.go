package promise

type CompoundFulfilledListener[REQUIRED any, OPTIONAL any, SUPPLY any] struct {
	OnFulfilled func(compoundValue CompoundResult[REQUIRED, OPTIONAL]) any
}
