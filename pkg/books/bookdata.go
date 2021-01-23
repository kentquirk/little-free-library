package books

import (
	"log"
	"time"
)

// PGFile is the parsed and processed structure of an XML file object
// within the Project Gutenberg data.
type PGFile struct {
	Location   string    `json:"location,omitempty"`
	Formats    []string  `json:"formats,omitempty"`
	FileSize   string    `json:"filesize,omitempty"`
	Modified   time.Time `json:"-"`
	IsFormatOf string    `json:"isformatof,omitempty"`
}

// EText is the parsed and processed structure of an etext object as defined in the XML.
type EText struct {
	ID            string    `json:"id,omitempty"`
	Publisher     string    `json:"publisher,omitempty"`
	Title         string    `json:"title,omitempty"`
	Creator       []string  `json:"creator,omitempty"`
	Contributor   []string  `json:"contributor,omitempty"`
	FriendlyTitle string    `json:"friendly_title,omitempty"`
	Language      string    `json:"language,omitempty"`
	Subjects      []string  `json:"subjects,omitempty"`
	Created       time.Time `json:"created,omitempty"`
	DownloadCount int       `json:"download_count,omitempty"`
	Rights        string    `json:"-"`
	Files         []PGFile  `json:"files,omitempty"`
}

// BookData is the type that we use to contain the book data and wrap all the queries.
// If we decide we want some sort of external data store, we can put it here.
// This is intended to be an opaque data structure; use accessors and query methods
// to retrieve data.
type BookData struct {
	books []EText
}

// NewBookData constructs a BookData object
func NewBookData() *BookData {
	return &BookData{
		books: make([]EText, 0),
	}
}

// Add inserts one or more EText entities into the BookData
func (b *BookData) Add(bs ...EText) {
	b.books = append(b.books, bs...)
}

// Update replaces the entire contents of the BookData
func (b *BookData) Update(bs []EText) {
	b.books = bs
}

// Get retrieves a book by its ID, or returns false in its second argument.
// This currently searches linearly; could easily be sped up with an ID index.
func (b *BookData) Get(id string) (EText, bool) {
	for i := range b.books {
		if b.books[i].ID == id {
			return b.books[i], true
		}
	}
	return EText{}, false
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
// TODO: Fix and test excludes
func (b *BookData) Query(constraints *ConstraintSpec) []EText {
	result := make([]EText, 0)
	log.Println(constraints)

	matchCount := 0
	for k := range b.books {
		if len(result) >= constraints.Limit {
			break
		}
		include := constraints.IncludeCombiner(constraints.Includes...)
		// exclude := constraints.ExcludeCombiner(constraints.Excludes...)
		// empty include list means include all; empty exclude list means exclude none
		if len(constraints.Includes) == 0 || include(b.books[k]) {
			// if len(constraints.Excludes) == 0 || !exclude(b.Books[k]) {
			matchCount++
			if matchCount < constraints.Limit*constraints.Page {
				continue
			}
			result = append(result, b.books[k])
			// }
		}
	}
	return result
}
