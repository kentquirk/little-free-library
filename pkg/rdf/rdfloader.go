package rdf

import (
	"archive/tar"
	"encoding/xml"
	"io"
	"log"

	"github.com/kentquirk/little-free-library/pkg/booktypes"
)

// Loader loads an RDF file given a reader to it
type Loader struct {
	reader        io.Reader
	ebookFilters  []EBookFilter
	pgFileFilters []PGFileFilter
	loadOnly      int
}

// LoaderOption is the type of a function used to set loader options;
// It modifies the Loader passed into it.
type LoaderOption func(*Loader)

// NewLoader constructs an RDF Loader from a reader.
func NewLoader(r io.Reader, options ...LoaderOption) *Loader {
	loader := &Loader{reader: r}
	for _, opt := range options {
		opt(loader)
	}
	// if after this there are no ebookFilters, add a dummy one that passes everything
	if len(loader.ebookFilters) == 0 {
		loader.ebookFilters = []EBookFilter{func(*booktypes.EBook) bool { return true }}
	}

	return loader
}

// EBookFilterOpt returns a LoaderOption that adds an EBookFilter
func EBookFilterOpt(f EBookFilter) LoaderOption {
	return func(ldr *Loader) {
		ldr.ebookFilters = append(ldr.ebookFilters, f)
	}
}

// PGFileFilterOpt returns a LoaderOption that adds a PGFileFilter
func PGFileFilterOpt(f PGFileFilter) LoaderOption {
	return func(ldr *Loader) {
		ldr.pgFileFilters = append(ldr.pgFileFilters, f)
	}
}

// LoadAtMostOpt returns a LoaderOptions that limits the number of items loaded
func LoadAtMostOpt(n int) LoaderOption {
	return func(ldr *Loader) {
		ldr.loadOnly = n
	}
}

// UntarOpt returns a LoaderOptions that wraps the reader in a tar reader
func UntarOpt(n int) LoaderOption {
	return func(ldr *Loader) {
		ldr.reader = tar.NewReader(ldr.reader)
	}
}

// Load is a helper function used by the Load functions
func (r *Loader) Load(rdr io.Reader) []booktypes.EBook {
	var data xmlRdf
	decoder := xml.NewDecoder(rdr)
	if err := decoder.Decode(&data); err != nil {
		log.Fatal(err)
	}

	// Go through the ebooks and keep the ones that pass the filter
	ebooks := make([]booktypes.EBook, 0)
	for i := range data.EBooks {
		et := data.EBooks[i].asEBook()
		for _, filt := range r.ebookFilters {
			if !filt(&et) {
				continue
			}
		eachfile:
			for _, xf := range data.EBooks[i].Formats {
				file := xf.asFile()
				for _, filt := range r.pgFileFilters {
					if !filt(&file) {
						continue eachfile
					}
				}
				et.Files = append(et.Files, file)
			}
			// only store objects we have files for
			if len(et.Files) != 0 {
				ebooks = append(ebooks, et)
			}
		}
	}
	return ebooks
}

// LoadOne parses and loads the XML data within its contents, expecting the contents to
// be a single file containing one or more EBook entities.
// It only returns the entities that pass the filters that have been set up
// before calling load.
// Returns 1 (the number of files processed).
func (r *Loader) LoadOne() ([]booktypes.EBook, int) {
	// Go through the ebooks and keep the ones that pass the filter
	ebooks := r.Load(r.reader)
	return ebooks, 1
}

// LoadTar loads from a reader, expecting the reader to be a tar file that contains lots of files of books
// It returns a slide of EBooks and the number of files that were processed within the tar.
// If loadOnly is set, it limits the number of items loaded. This is mainly useful for testing.
func (r *Loader) LoadTar() ([]booktypes.EBook, int) {
	count := 0
	tr := tar.NewReader(r.reader)
	ebooks := make([]booktypes.EBook, 0)
	for {
		_, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			log.Fatalf("Count=%d, err=%v", count, err)
		}
		newtexts := r.Load(tr)
		ebooks = append(ebooks, newtexts...)
		count++
		if r.loadOnly > 0 && len(ebooks) >= r.loadOnly {
			break // end early because loadOnly
		}
	}
	return ebooks, count
}
