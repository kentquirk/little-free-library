package books

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/kentquirk/stringset/v2"
)

// ConstraintFunctor is the type of the function used to evaluate a constraint.
// We need to benchmark to see if it would make a difference to make it
// take a pointer.
type ConstraintFunctor func(EBook) bool

// ConstraintFunctorGen is a function that generates a ConstraintFunctor from a pattern.
type ConstraintFunctorGen func(pat *regexp.Regexp) ConstraintFunctor

// ConstraintCombiner is an operator that can combine a set of constraints, like AND or OR.
type ConstraintCombiner func(...ConstraintFunctor) ConstraintFunctor

func nilFunctor(EBook) bool {
	return false
}

// This pattern tests for a case-independent complete word
var wholeWord = `(?is:\b%s\b)`

// Or returns the logical OR of a set of functors; if any one of them returns true, the result is true.
// Uses short-circuit evaluation.
// If there are no arguments, returns nilFunctor.
func Or(cfs ...ConstraintFunctor) ConstraintFunctor {
	if len(cfs) == 0 {
		return nilFunctor
	}
	return func(eb EBook) bool {
		for _, cf := range cfs {
			if cf(eb) {
				return true
			}
		}
		return false
	}
}

// And returns the logical AND of a set of functors; returns true only if all of them return true.
// Uses short-circuit evaluation.
// If there are no arguments, returns nilFunctor.
func And(cfs ...ConstraintFunctor) ConstraintFunctor {
	if len(cfs) == 0 {
		return nilFunctor
	}
	return func(eb EBook) bool {
		for _, cf := range cfs {
			if !cf(eb) {
				return false
			}
		}
		return true
	}
}

// testWords evaluates a value to see if it even possibly matches any of the whole words
// in the query before passing it on to a regexp-based matcher.
func testWords(value string, matchGen ConstraintFunctorGen) ConstraintFunctor {
	words := stringset.New().Add(GetWords(value)...)
	pat, err := regexp.Compile(fmt.Sprintf(wholeWord, value))
	if err != nil {
		return nilFunctor
	}
	return func(eb EBook) bool {
		if words.Intersection(eb.Words).Length() != words.Length() {
			return false
		}
		// we know all the words in the search term were found in this
		// ebook, but now we have to test to see if they're actually in the desired field.
		f := matchGen(pat)
		return f(eb)
	}
}

func matchCreator(pat *regexp.Regexp) ConstraintFunctor {
	return func(eb EBook) bool {
		for _, s := range eb.Creators {
			if pat.MatchString(eb.Agents[s].Name) {
				return true
			}
			for _, a := range eb.Agents[s].Aliases {
				if pat.MatchString(a) {
					return true
				}
			}
		}
		return false
	}
}

// This is an optimized illustrator query because there are so few illustrators
func testIllustrator(value string) ConstraintFunctor {
	// Build this outside the functor for efficiency.
	testfunc := testWords(value, matchIllustrator)
	return func(eb EBook) bool {
		if len(eb.Illustrators) == 0 {
			return false
		}
		return testfunc(eb)
	}
}

func matchIllustrator(pat *regexp.Regexp) ConstraintFunctor {
	return func(eb EBook) bool {
		for _, s := range eb.Illustrators {
			if pat.MatchString(eb.Agents[s].Name) {
				return true
			}
			for _, a := range eb.Agents[s].Aliases {
				if pat.MatchString(a) {
					return true
				}
			}
		}
		return false
	}
}

func matchSubject(pat *regexp.Regexp) ConstraintFunctor {
	return func(eb EBook) bool {
		for _, s := range eb.Subjects {
			if pat.MatchString(s) {
				return true
			}
		}
		return false
	}
}

func matchTitle(pat *regexp.Regexp) ConstraintFunctor {
	return func(eb EBook) bool {
		return pat.MatchString(eb.Title)
	}
}

func testType(value string) ConstraintFunctor {
	pat, err := regexp.Compile(fmt.Sprintf(wholeWord, value))
	if err != nil {
		return nilFunctor
	}
	return matchType(pat)
}

func matchType(pat *regexp.Regexp) ConstraintFunctor {
	return func(eb EBook) bool {
		return pat.MatchString(eb.Type)
	}
}

// tests languages for exact equality, and allows multiple languages
// separated by period (.)
func testLanguage(value string) ConstraintFunctor {
	return func(eb EBook) bool {
		for _, l := range strings.Split(value, ".") {
			if eb.Language == l {
				return true
			}
		}
		return false
	}
}

type yearComparison int

// These comparisons are for the year of the book as compared to the target year.
// So if the book is 2005 and the target is 2010, the book is less than the targeb.
const (
	yearEQ yearComparison = iota
	yearGE yearComparison = iota
	yearLE yearComparison = iota
)

// testIssued checks the book's Issued date
func testIssued(value string, cmp yearComparison) ConstraintFunctor {
	if value == "" {
		return func(eb EBook) bool { return true }
	}
	return func(eb EBook) bool {
		d, _ := ParseDate(value)
		switch cmp {
		case yearEQ:
			return eb.Issued.CompareTo(d) == 0
		case yearGE:
			return eb.Issued.CompareTo(d) >= 0
		case yearLE:
			return eb.Issued.CompareTo(d) <= 0
		default:
			return false
		}
	}
}

// testCopyright checks the copyright dates; if any of the dates in
// CopyrightDates fits the comparison, the result is true
func testCopyright(value string, cmp yearComparison) ConstraintFunctor {
	if value == "" {
		return func(eb EBook) bool { return true }
	}
	return func(eb EBook) bool {
		if len(eb.CopyrightDates) == 0 {
			return false
		}
		d, _ := ParseDate(value)
		for _, cd := range eb.CopyrightDates {
			switch cmp {
			case yearEQ:
				if cd.CompareTo(d) == 0 {
					return true
				}
			case yearGE:
				if cd.CompareTo(d) >= 0 {
					return true
				}
			case yearLE:
				if cd.CompareTo(d) <= 0 {
					return true
				}
			}
		}
		return false
	}
}
