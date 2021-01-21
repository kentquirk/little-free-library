package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
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

// ETextFilter is a function that evaluates an EText object and returns
// true if the object "passes". Only if an object passes all filters is
// it included in the output.
type ETextFilter func(*EText) bool

// PGFileFilter is a function that evaluates an PGFile object and returns
// true if the object "passes". Only if an object passes all filters is
// it included in the output.
type PGFileFilter func(*PGFile) bool

// LanguageFilter is a convenience function that returns an ETextFilter which
// returns true if the etext is in any of the languages specified.
func LanguageFilter(languages ...string) ETextFilter {
	return func(e *EText) bool {
		for _, l := range languages {
			if e.Language == l {
				return true
			}
		}
		return false
	}
}

// Convenience constants for content types
const (
	TextPlain      = "text/plain"
	TextPlainUTF8  = "text/plain"
	TextPlainLatin = `text/plain; charset="iso-8859-1"`
	TextPlainASCII = `text/plain; charset="us-ascii"`
	Mobi           = "application/x-mobipocket-ebook"
	EPub           = "application/epub+zip"
	Plucker        = "application/prs.plucker"
	HTML           = "text/html"
	Zip            = "application/zip"
)

// ContentFilter is a convenience function that returns a PGFileFilter which
// returns true if the file is an exact match for any one of the specified content types.
// Some files have two content types -- the base type, and Zip (if there is a zipped version
// of the file).
func ContentFilter(contentTypes ...string) PGFileFilter {
	return func(f *PGFile) bool {
		for _, ct := range contentTypes {
			for _, format := range f.Formats {
				if format == ct {
					return true
				}
			}
		}
		return false
	}
}

// BookData is the type that we use to contain the book data and wrap all the queries.
// If we decide we want some sort of external data store, we can put it here.
type BookData struct {
	Books map[string]EText
}

// Get retrieves a book by ID
func (b *BookData) Get(id string) (EText, bool) {
	et, ok := b.Books[id]
	return et, ok
}

type summaryData struct {
	TotalBooks int            `json:"total_books"`
	TotalFiles int            `json:"total_files"`
	Languages  map[string]int `json:"languages"`
	Formats    map[string]int `json:"formats"`
}

func (b *BookData) bookSummary(c echo.Context) error {
	sd := summaryData{
		Languages: make(map[string]int),
		Formats:   make(map[string]int),
	}
	for i := range b.Books {
		sd.TotalBooks++
		lang := b.Books[i].Language
		sd.Languages[lang]++
		for _, f := range b.Books[i].Files {
			for _, fmt := range f.Formats {
				sd.TotalFiles++
				sd.Formats[fmt]++
			}
		}
	}
	return c.JSON(http.StatusOK, sd)
}

// ConstraintSpec is used to store a complete set of constraints
type ConstraintSpec struct {
	Includes        []ConstraintFunctor
	IncludeCombiner ConstraintCombiner
	Excludes        []ConstraintFunctor
	ExcludeCombiner ConstraintCombiner
	Limit           int
}

func newConstraintSpec() *ConstraintSpec {
	return &ConstraintSpec{
		Includes:        make([]ConstraintFunctor, 0),
		IncludeCombiner: And,
		Excludes:        make([]ConstraintFunctor, 0),
		ExcludeCombiner: Or,
		Limit:           25,
	}
}

func (b *BookData) doQuery(constraints *ConstraintSpec) []EText {
	result := make([]EText, 0)
	log.Println(constraints)

	for k := range b.Books {
		if len(result) >= constraints.Limit {
			break
		}
		include := constraints.IncludeCombiner(constraints.Includes...)
		// exclude := constraints.ExcludeCombiner(constraints.Excludes...)
		// empty include list means include all; empty exclude list means exclude none
		if len(constraints.Includes) == 0 || include(b.Books[k]) {
			// if len(constraints.Excludes) == 0 || !exclude(b.Books[k]) {
			result = append(result, b.Books[k])
			// }
		}
	}
	return result
}

// bookQuery does a book query based on a query specification.
func (b *BookData) bookQuery(c echo.Context) error {
	values := c.QueryParams()
	constraints := newConstraintSpec()

	for k, vals := range values {
		// once for each copy of a given key
		for _, v := range vals {
			switch k {
			case "or":
				constraints.IncludeCombiner = Or
			case "and":
				constraints.IncludeCombiner = And
			case "-or":
				constraints.ExcludeCombiner = Or
			case "-and":
				constraints.ExcludeCombiner = And
			case "limit":
				n, _ := strconv.Atoi(v)
				// TODO: read upper limit from config
				if n > 0 && n < 100 {
					constraints.Limit = n
				}
			default:
				constraint, exclude, err := ConstraintFromText(k, v)
				if err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, "constraint error: %e", err)
				}
				if exclude {
					constraints.Excludes = append(constraints.Excludes, constraint)
				} else {
					constraints.Includes = append(constraints.Includes, constraint)
				}
			}
		}
	}
	// ok, we have a constraint spec -- execute it
	result := b.doQuery(constraints)
	return c.JSON(http.StatusOK, result)
}

func bookByID(c echo.Context) error {
	return c.String(http.StatusOK, "Ok\n")
}
