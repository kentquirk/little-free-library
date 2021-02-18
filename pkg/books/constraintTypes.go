package books

import (
	"regexp"

	"github.com/kentquirk/little-free-library/pkg/booktypes"
)

// ConstraintFunctor is the type of the function used to evaluate a constraint.
// We need to benchmark to see if it would make a difference to make it
// take a pointer.
type ConstraintFunctor func(booktypes.EBook) bool

// ConstraintFunctorGen is a function that generates a ConstraintFunctor from a pattern.
type ConstraintFunctorGen func(pat *regexp.Regexp) ConstraintFunctor

// ConstraintCombiner is an operator that can combine a set of constraints, like AND or OR.
type ConstraintCombiner func(...ConstraintFunctor) ConstraintFunctor

// ConstraintSpec is used to store a complete set of constraints.
// Page is in units of a multiple of Limit.
// If Random is true, Page is ignored.
type ConstraintSpec struct {
	Includes        []ConstraintFunctor
	IncludeCombiner ConstraintCombiner
	Excludes        []ConstraintFunctor
	ExcludeCombiner ConstraintCombiner
	Limit           int
	Page            int
	Random          bool
}

// NewConstraintSpec creates an empty constraint spec that will return all results 25 at a time.
func NewConstraintSpec() *ConstraintSpec {
	return &ConstraintSpec{
		Includes:        make([]ConstraintFunctor, 0),
		IncludeCombiner: And,
		Excludes:        make([]ConstraintFunctor, 0),
		ExcludeCombiner: Or,
		Limit:           25,
		Page:            0,
	}
}
