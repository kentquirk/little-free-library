package booktypes

import "github.com/kentquirk/little-free-library/pkg/date"

// PGFile is the parsed and processed structure of an object
// within the Project Gutenberg data that corresponds to a single
// downloadable entity -- a particular version of the content.
type PGFile struct {
	Location string    `json:"location,omitempty"`
	Formats  []string  `json:"formats,omitempty"`
	FileSize int       `json:"filesize,omitempty"`
	Modified date.Date `json:"modified,omitempty"`
	BookID   string    `json:"bookid,omitempty"`
}
