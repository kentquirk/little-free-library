package rdf

import (
	"strings"

	"github.com/kentquirk/little-free-library/pkg/booktypes"
)

// EBookFilter is a function that evaluates an EBook object and returns
// true if the object "passes". Only if an object passes all filters is
// it included in the output.
type EBookFilter func(*booktypes.EBook) bool

// PGFileFilter is a function that evaluates an PGFile object and returns
// true if the object "passes". Only if an object passes all filters is
// it included in the output.
type PGFileFilter func(*booktypes.PGFile) bool

// LanguageFilter is a convenience function that returns an EBookFilter which
// returns true if the ebook is in any of the languages specified.
func LanguageFilter(languages ...string) EBookFilter {
	return func(e *booktypes.EBook) bool {
		for _, l := range languages {
			if e.Language == l {
				return true
			}
		}
		return false
	}
}

// ContentFilter is a convenience function that returns a PGFileFilter which
// returns true if the file has a matching prefix of for any one of the specified content types.
// Some files have two content types -- the base type, and Zip (if there is a zipped version
// of the file).
func ContentFilter(contentTypes ...string) PGFileFilter {
	return func(f *booktypes.PGFile) bool {
		for _, ctname := range contentTypes {
			if ct, ok := ContentTypes[ctname]; ok {
				for _, format := range f.Formats {
					if strings.HasPrefix(format, ct) {
						return true
					}
				}
			}
		}
		return false
	}
}
