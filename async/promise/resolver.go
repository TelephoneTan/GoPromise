package promise

type Resolver[T any] interface {
	Resolve(valueOrPromise any)
	ResolveValue(value *T)
	ResolvePromise(promise *Type[T])
}
