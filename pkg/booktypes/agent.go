package booktypes

import (
	"github.com/kentquirk/little-free-library/pkg/date"
	"github.com/kentquirk/stringset/v2"
)

// Agent is a record for a human (Project Gutenberg calls these agents).
// This can be an author, editor, or illustrator.
type Agent struct {
	ID        string    `json:"id,omitempty"`
	Name      string    `json:"name,omitempty"`
	Aliases   []string  `json:"aliases,omitempty"`
	BirthDate date.Date `json:"birth_date,omitempty"`
	DeathDate date.Date `json:"death_date,omitempty"`
	Webpages  []string  `json:"webpages,omitempty"`
}

// AddWords the list of lower-case alphanumerics in the
// Agent Name and Aliases to the given StringSet
func (a Agent) AddWords(w *stringset.StringSet) {
	w.Add(GetWords(a.Name)...)
	for i := range a.Aliases {
		w.Add(GetWords(a.Aliases[i])...)
	}
}
