package books

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
