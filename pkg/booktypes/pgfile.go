package booktypes

import (
	"log"

	"github.com/kentquirk/little-free-library/pkg/date"
)

// CompType indicates the type of standalone compression in use
// (Note that some formats are inherently compressed.)
type CompType int

// Types of compression
// Currently PG only supports zip, but if they ever add more, it would go here
const (
	CompNone CompType = iota
	CompZip
)

// PGFile is the parsed and processed structure of an object
// within the Project Gutenberg data that corresponds to a single
// downloadable entity -- a particular version of the content.
type PGFile struct {
	Location string    `json:"location,omitempty"`
	Format   string    `json:"format,omitempty"`
	Comp     CompType  `json:"comp,omitempty"`
	FileSize int       `json:"filesize,omitempty"`
	Modified date.Date `json:"modified,omitempty"`
	BookID   string    `json:"bookid,omitempty"`
}

// BuildFile makes a PGFile object from a set of parameters. In particular, it gets a slice of formats,
// which will be either one or two items, one of which might be a compression format. These get broken
// out into base format and an optional compression format.
func BuildFile(id string, loc string, formats []string, siz int, modified string) PGFile {
	f := PGFile{
		Location: loc,
		FileSize: siz,
		BookID:   id,
		Modified: date.ParseOnly(modified),
	}
	for _, fmt := range formats {
		switch fmt {
		case "application/zip":
			f.Comp = CompZip
		default:
			f.Format = fmt
		}
	}
	if len(formats) >= 2 && f.Comp == 0 {
		log.Printf("book %s has suspect formats; new compression type?", f.BookID)
	}
	return f
}
