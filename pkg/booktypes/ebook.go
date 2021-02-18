package booktypes

import (
	"regexp"
	"strings"

	"github.com/kentquirk/little-free-library/pkg/date"
	"github.com/kentquirk/stringset/v2"
)

// wordPat is a pattern we use when we need to extract all the alphanumeric elements in a string
var wordPat = regexp.MustCompile("[^a-z0-9_]+")

// I looked at eliminating "noise words" to reduce the size of the word indices, but it only
// reduced it by 20%, and that didn't seem worth the extra logic and the reduced
// precision of the search.
// var noiseWords = stringset.New().Add(wordPat.Split(`
// 				an and but is it of or the to
// 				a b c d e f g h i j k l m n o p q r s t u v w x y z
// 				0 1 2 3 4 5 6 7 8 9
// 				`, -1)...)

// GetWords retrieves a lowercased list of alphanumeric strings from an input string
func GetWords(s string) []string {
	return wordPat.Split(strings.ToLower(s), -1)
}

// EBook is the parsed and processed structure of an ebook object.
type EBook struct {
	ID              string               `json:"id,omitempty"`
	Publisher       string               `json:"publisher,omitempty"`
	Title           string               `json:"title,omitempty"`
	Creators        []string             `json:"creators,omitempty"`
	Illustrators    []string             `json:"illustrators,omitempty"`
	TableOfContents string               `json:"table_of_contents,omitempty"`
	Language        string               `json:"language,omitempty"`
	Subjects        []string             `json:"subjects,omitempty"`
	Issued          date.Date            `json:"issued,omitempty"`
	DownloadCount   int                  `json:"download_count,omitempty"`
	Rights          string               `json:"rights,omitempty"`
	Copyright       string               `json:"copyright,omitempty"`
	Edition         string               `json:"edition,omitempty"`
	Type            string               `json:"type,omitempty"`
	Files           []PGFile             `json:"files,omitempty"`
	Agents          map[string]Agent     `json:"agents,omitempty"`
	CopyrightDates  []date.Date          `json:"-"`
	Words           *stringset.StringSet `json:"-"`
}

// ExtractWords retrieves a stringSet of individual words
func (e *EBook) ExtractWords() {
	w := stringset.New().Add(GetWords(e.Title)...)
	for i := range e.Subjects {
		w.Add(GetWords(e.Subjects[i])...)
	}
	for _, v := range e.Agents {
		v.AddWords(w)
	}
	e.Words = w
}

// FullCreators is a helper function for templates to extract the creator name(s)
func (e *EBook) FullCreators() []Agent {
	var agents []Agent
	for _, agent := range e.Creators {
		agents = append(agents, e.Agents[agent])
	}
	return agents
}
