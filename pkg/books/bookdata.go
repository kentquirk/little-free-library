package books

import (
	"math/rand"
	"time"
)

// PGFile is the parsed and processed structure of an xmlFile object
// within the Project Gutenberg data.
type PGFile struct {
	Location   string   `json:"location,omitempty"`
	Formats    []string `json:"formats,omitempty"`
	FileSize   int      `json:"filesize,omitempty"`
	Modified   Date     `json:"modified,omitempty"`
	IsFormatOf string   `json:"isformatof,omitempty"`
}

// EBook is the parsed and processed structure of an ebook object as defined in the XML.
type EBook struct {
	ID              string           `json:"id,omitempty"`
	Publisher       string           `json:"publisher,omitempty"`
	Title           string           `json:"title,omitempty"`
	Creators        []string         `json:"creators,omitempty"`
	Illustrators    []string         `json:"illustrators,omitempty"`
	TableOfContents string           `json:"table_of_contents,omitempty"`
	Language        string           `json:"language,omitempty"`
	Subjects        []string         `json:"subjects,omitempty"`
	Issued          Date             `json:"issued,omitempty"`
	DownloadCount   int              `json:"download_count,omitempty"`
	Rights          string           `json:"rights,omitempty"`
	Copyright       string           `json:"copyright,omitempty"`
	CopyrightDates  []Date           `json:"-"`
	Edition         string           `json:"edition,omitempty"`
	Type            string           `json:"type,omitempty"`
	Files           []PGFile         `json:"files,omitempty"`
	Agents          map[string]Agent `json:"agents,omitempty"`
}

// FullCreators is a helper function for templates to extract the creator name(s)
func (e *EBook) FullCreators() []Agent {
	var agents []Agent
	for _, agent := range e.Creators {
		agents = append(agents, e.Agents[agent])
	}
	return agents
}

// Agent is a record for a human (Project Gutenberg calls these agents).
// This can be an author, editor, or illustrator.
type Agent struct {
	ID        string   `json:"id,omitempty"`
	Name      string   `json:"name,omitempty"`
	Aliases   []string `json:"aliases,omitempty"`
	BirthDate Date     `json:"birth_date,omitempty"`
	DeathDate Date     `json:"death_date,omitempty"`
	Webpages  []string `json:"webpages,omitempty"`
}

// BookData is the type that we use to contain the book data and wrap all the queries.
// If we decide we want some sort of external data store, we can put it here.
// This is intended to be an opaque data structure; use accessors and query methods
// to retrieve data.
type BookData struct {
	books []EBook
}

// NewBookData constructs a BookData object
func NewBookData() *BookData {
	return &BookData{
		books: make([]EBook, 0),
	}
}

// Add inserts one or more EBook entities into the BookData
func (b *BookData) Add(bs ...EBook) {
	b.books = append(b.books, bs...)
}

// Update replaces the entire contents of the BookData
func (b *BookData) Update(bs []EBook) {
	b.books = bs
}

// NBooks returns the number of books in the dataset
func (b *BookData) NBooks() int {
	return len(b.books)
}

// Get retrieves a book by its ID, or returns false in its second argument.
// This currently searches linearly; could easily be sped up with an ID index.
func (b *BookData) Get(id string) (EBook, bool) {
	for i := range b.books {
		if b.books[i].ID == id {
			return b.books[i], true
		}
	}
	return EBook{}, false
}

// SummaryData is the data structure used to return collection-level information
// about the data on hand.
type SummaryData struct {
	TotalBooks int            `json:"total_books"`
	TotalFiles int            `json:"total_files"`
	Languages  map[string]int `json:"languages"`
	Formats    map[string]int `json:"formats"`
}

// Summary returns aggregated information about the data being stored.
func (b *BookData) Summary() SummaryData {
	sd := SummaryData{
		Languages: make(map[string]int),
		Formats:   make(map[string]int),
	}
	for i := range b.books {
		sd.TotalBooks++
		lang := b.books[i].Language
		sd.Languages[lang]++
		for _, f := range b.books[i].Files {
			for _, fmt := range f.Formats {
				sd.TotalFiles++
				sd.Formats[fmt]++
			}
		}
	}
	return sd
}

// Query does a query against the book data according to a ConstraintSpec.
// If the random flag is set, we choose a random subset of matching items.
//
// We want to select items fairly, so we use a replacement algorithm
// that adjusts the replacement probability based on the number of items
// that we have already seen.
// To choose n out of a stream of items, we generate the items one at a time,
// keeping the first n items in a set S.
// Then, when reading the m-th item I (m>n now), we keep it with probability n/m.
// When we keep it, we select item U uniformly at random from S, and replace
// U with I.
func (b *BookData) Query(constraints *ConstraintSpec) []EBook {
	result := make([]EBook, 0)

	// create the random number generator only if we need it
	var random *rand.Rand
	if constraints.Random {
		random = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	matchCount := 0
	replace := false
	include := constraints.IncludeCombiner(constraints.Includes...)
	exclude := constraints.ExcludeCombiner(constraints.Excludes...)
	for k := range b.books {
		if len(result) >= constraints.Limit {
			if !constraints.Random {
				break
			} else {
				replace = true
			}
		}
		// empty include list means include all; empty exclude list means exclude none
		if len(constraints.Includes) == 0 || include(b.books[k]) {
			if len(constraints.Excludes) == 0 || !exclude(b.books[k]) {
				matchCount++
				if !constraints.Random && matchCount < constraints.Limit*constraints.Page {
					continue
				}
				if replace {
					keep := (random.Float64() < (float64(constraints.Limit) / float64(matchCount)))
					if keep {
						randomIndex := random.Intn(constraints.Limit)
						result[randomIndex] = b.books[k]
					}
				} else {
					result = append(result, b.books[k])
				}
			}
		}
	}
	return result
}

// Count does a query against the book data according to a ConstraintSpec and returns the number
// of matching items (ignoring Limit and Random).
func (b *BookData) Count(constraints *ConstraintSpec) int {
	matchCount := 0
	include := constraints.IncludeCombiner(constraints.Includes...)
	exclude := constraints.ExcludeCombiner(constraints.Excludes...)
	for k := range b.books {
		// empty include list means include all; empty exclude list means exclude none
		if len(constraints.Includes) == 0 || include(b.books[k]) {
			if len(constraints.Excludes) == 0 || !exclude(b.books[k]) {
				matchCount++
			}
		}
	}
	return matchCount
}
