package promise

type Resolver[T any] interface {
	ResolveValue(value *T)
	ResolvePromise(promise *Type[T])
}
