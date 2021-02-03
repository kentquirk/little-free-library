package books

import (
	"errors"
	"regexp"
	"strings"
)

// createRegex constructs a regex from a glob-style expression.
// glob-style: . means any single character and _ means any number of characters.
// This is similar to file pattern matching on the command line, except that ? and * are replaced
// by . and _, in order to be URL-safe.
//
// The expression is evaluated against the entire string, and case is insignificant.
// For example, if a list of creators is Evelyn, Linda, Lynn, and Eve, these are matches for
// various globs:
// Eve_ matches Evelyn and Eve
// Eve matches only Eve
// L.n_ matches Lynn and Linda
// _l.n_ matches Linda, Evelyn and Lynn
func createRegex(value string) (*regexp.Regexp, error) {
	// we can leave . alone, since it matches any single character.
	// _ needs to be replaced by ".*"
	glob := strings.Replace(value, "_", ".*", -1)
	return regexp.Compile("(?is:^" + glob + "$)")
}

// ConstraintFromText creates a ConstraintFunctor by parsing name and value fields.
//
// Constraints supported are:
// year: value is a single numeric year, or a range with one end omitted (1855, 1855-1899, -1920, 1900-)
// creator: value matches creator field
// contributor: value matches contributor field
// author: value matches creator OR contributor fields
// title: value matches title field
// subject: value matches subject field
// topic: value matches subject or title
// any: value matches any of subject, title, creator, contributor
// language: value matches 2- or 3-char language field, multiple values separated by .
//
// All matches are case-insensitive. For non-glob queries, the specified string is tested at
// word boundaries for the specified field or fields (including multi-valued fields).
// If the subject is "History - Fiction", "fiction" is considered a match, but "story" is not.
//
// Patterns can be specified with "glob-style" queries, which are queries whose names
// are preceded by a tilde (~) character.
// For glob-style queries, the value is treated as a "glob"-style expression (see below).
//
// Names can also be preceded by a hyphen (-) character, which means that the match is
// inverted -- matched items are *excluded* from the results. If an item is included by
// one constraint but excluded by another, the exclusion wins.
//
// Both - and ~ can be used on the same name in either order.
//
// glob-style: . means any single character and _ means any number of characters.
// This is similar to file pattern matching on the command line, except that ? and * are replaced
// by . and _, in order to be URL-safe.
//
// The expression is evaluated against the entire string, and case is insignificant.
// For example, if a list of creators is Evelyn, Linda, Lynn, and Eve, these are matches for
// various globs:
// Eve_ matches Evelyn and Eve
// Eve matches only Eve
// L.n_ matches Lynn and Linda
// _l.n_ matches Linda, Evelyn and Lynn
//
// The return values are the generated constraint functor, a boolean indicating if the
// constraint is an exclude constraint, and an error.
func ConstraintFromText(name string, value string) (ConstraintFunctor, bool, error) {
	exclude := false
	useRegexp := false
	name = strings.ToLower(name)
	value = strings.ToLower(value)
outer:
	for len(name) > 0 {
		switch name[0] {
		case '-':
			exclude = true
			name = name[1:]
		case '~':
			useRegexp = true
			name = name[1:]
		default:
			break outer
		}
	}
	var pat *regexp.Regexp
	var err error
	if useRegexp {
		pat, err = createRegex(value)
		if err != nil {
			return nilFunctor, false, err
		}
	}

	retfunc := nilFunctor
	switch name {
	case "author", "auth":
		if useRegexp {
			retfunc = matchCreator(pat)
		} else {
			retfunc = testWords(value, matchCreator)
		}

	case "illustrator", "ill":
		if useRegexp {
			retfunc = matchIllustrator(pat)
		} else {
			retfunc = testIllustrator(value)
		}
	case "creator", "cre":
		if useRegexp {
			retfunc = Or(matchCreator(pat), matchIllustrator(pat))
		} else {
			retfunc = Or(testWords(value, matchCreator), testIllustrator(value))
		}
	case "title":
		if useRegexp {
			retfunc = matchTitle(pat)
		} else {
			retfunc = testWords(value, matchTitle)
		}
	case "subject", "subj":
		if useRegexp {
			retfunc = matchSubject(pat)
		} else {
			retfunc = testWords(value, matchSubject)
		}
	case "topic", "top":
		if useRegexp {
			retfunc = Or(matchTitle(pat), matchSubject(pat))
		} else {
			retfunc = Or(testWords(value, matchTitle), testWords(value, matchSubject))
		}
	case "type", "typ":
		if useRegexp {
			retfunc = matchType(pat)
		} else {
			retfunc = testType(value)
		}
	case "any":
		if useRegexp {
			retfunc = Or(matchCreator(pat), matchIllustrator(pat), matchTitle(pat), matchSubject(pat))
		} else {
			retfunc = Or(testWords(value, matchCreator), testIllustrator(value), testWords(value, matchTitle), testWords(value, matchSubject))
		}
	case "language", "lang":
		retfunc = testLanguage(value)
	case "issued", "iss":
		splits := strings.Split(value, "-")
		if len(splits) == 1 {
			retfunc = testIssued(splits[0], yearEQ)
		} else if len(splits) == 2 {
			retfunc = And(testIssued(splits[0], yearGE), testIssued(splits[1], yearLE))
		}
	case "copyright", "cop", "copr":
		splits := strings.Split(value, "-")
		if len(splits) == 1 {
			retfunc = testCopyright(splits[0], yearEQ)
		} else if len(splits) == 2 {
			retfunc = And(testCopyright(splits[0], yearGE), testCopyright(splits[1], yearLE))
		}
	default:
		return retfunc, false, errors.New("bad constraint definition")
	}
	return retfunc, exclude, nil
}
