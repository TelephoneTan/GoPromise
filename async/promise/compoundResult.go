package promise

type CompoundResult[REQUIRED any, OPTIONAL any] struct {
	RequiredValue         []*REQUIRED
	OptionalValue         []*OPTIONAL
	OptionalReason        []error
	OptionalCancelledFlag []bool
	OptionalSucceededFlag []bool
}
