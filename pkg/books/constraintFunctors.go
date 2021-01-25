package books

import (
	"fmt"
	"regexp"
	"strings"
)

// ConstraintFunctor is the type of the function used to evaluate a constraint.
// We need to benchmark to see if it would make a difference to make it
// take a pointer.
type ConstraintFunctor func(EBook) bool

// ConstraintCombiner is an operator that can combine a set of constraints, like AND or OR.
type ConstraintCombiner func(...ConstraintFunctor) ConstraintFunctor

func nilFunctor(EBook) bool {
	return false
}

// Or returns the logical OR of a set of functors; if any one of them returns true, the result is true.
// Uses short-circuit evaluation.
// If there are no arguments, returns nilFunctor.
func Or(cfs ...ConstraintFunctor) ConstraintFunctor {
	if len(cfs) == 0 {
		return nilFunctor
	}
	return func(et EBook) bool {
		for _, cf := range cfs {
			if cf(et) {
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
	return func(et EBook) bool {
		for _, cf := range cfs {
			if !cf(et) {
				return false
			}
		}
		return true
	}
}

func testCreator(value string) ConstraintFunctor {
	pat, err := regexp.Compile(fmt.Sprintf(`(?is:\b%s\b)`, value))
	if err != nil {
		return nilFunctor
	}
	return matchCreator(pat)
}

func matchCreator(pat *regexp.Regexp) ConstraintFunctor {
	return func(et EBook) bool {
		for _, s := range et.Creators {
			if pat.MatchString(et.Agents[s].Name) {
				return true
			}
			for _, a := range et.Agents[s].Aliases {
				if pat.MatchString(a) {
					return true
				}
			}
		}
		return false
	}
}

func testIllustrator(value string) ConstraintFunctor {
	pat, err := regexp.Compile(fmt.Sprintf(`(?is:\b%s\b)`, value))
	if err != nil {
		return nilFunctor
	}
	return matchIllustrator(pat)
}

func matchIllustrator(pat *regexp.Regexp) ConstraintFunctor {
	return func(et EBook) bool {
		for _, s := range et.Illustrators {
			if pat.MatchString(et.Agents[s].Name) {
				return true
			}
			for _, a := range et.Agents[s].Aliases {
				if pat.MatchString(a) {
					return true
				}
			}
		}
		return false
	}
}

func testSubject(value string) ConstraintFunctor {
	pat, err := regexp.Compile(fmt.Sprintf(`(?is:\b%s\b)`, value))
	if err != nil {
		return nilFunctor
	}
	return matchSubject(pat)
}

func matchSubject(pat *regexp.Regexp) ConstraintFunctor {
	return func(et EBook) bool {
		for _, s := range et.Subjects {
			if pat.MatchString(s) {
				return true
			}
		}
		return false
	}
}

func testTitle(value string) ConstraintFunctor {
	pat, err := regexp.Compile(fmt.Sprintf(`(?is:\b%s\b)`, value))
	if err != nil {
		return nilFunctor
	}
	return matchTitle(pat)
}

func matchTitle(pat *regexp.Regexp) ConstraintFunctor {
	return func(et EBook) bool {
		return pat.MatchString(et.Title)
	}
}

func testType(value string) ConstraintFunctor {
	pat, err := regexp.Compile(fmt.Sprintf(`(?is:\b%s\b)`, value))
	if err != nil {
		return nilFunctor
	}
	return matchType(pat)
}

func matchType(pat *regexp.Regexp) ConstraintFunctor {
	return func(et EBook) bool {
		return pat.MatchString(et.Type)
	}
}

// tests languages for exact equality, and allows multiple languages
// separated by period (.)
func testLanguage(value string) ConstraintFunctor {
	return func(et EBook) bool {
		for _, l := range strings.Split(value, ".") {
			if et.Language == l {
				return true
			}
		}
		return false
	}
}

type yearComparison int

// These comparisons are for the year of the book as compared to the target year.
// So if the book is 2005 and the target is 2010, the book is less than the target.
const (
	yearEQ yearComparison = iota
	yearGE yearComparison = iota
	yearLE yearComparison = iota
)

// testIssued checks the book's Issued date
func testIssued(value string, cmp yearComparison) ConstraintFunctor {
	if value == "" {
		return func(et EBook) bool { return true }
	}
	return func(et EBook) bool {
		d, _ := ParseDate(value)
		switch cmp {
		case yearEQ:
			return et.Issued.CompareTo(d) == 0
		case yearGE:
			return et.Issued.CompareTo(d) >= 0
		case yearLE:
			return et.Issued.CompareTo(d) <= 0
		default:
			return false
		}
	}
}

// testCopyright checks the copyright dates; if any of the dates in
// CopyrightDates fits the comparison, the result is true
func testCopyright(value string, cmp yearComparison) ConstraintFunctor {
	if value == "" {
		return func(et EBook) bool { return true }
	}
	return func(et EBook) bool {
		if len(et.CopyrightDates) == 0 {
			return false
		}
		d, _ := ParseDate(value)
		for _, cd := range et.CopyrightDates {
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
