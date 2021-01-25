package rdf

import (
	"archive/tar"
	"encoding/xml"
	"io"
	"log"
	"strings"

	"github.com/kentquirk/little-free-library/pkg/books"
)

func extractCharDataToEndToken(d *xml.Decoder, start xml.StartElement) ([]string, error) {
	result := make([]string, 0)
outer:
	for {
		token, err := d.Token()
		if err != nil {
			return nil, err
		}
		switch tok := token.(type) {
		case xml.EndElement:
			if tok == start.End() {
				break outer
			}
		case xml.CharData:
			s := strings.Trim(string(tok), " \t\r\n")
			if s != "" {
				result = append(result, s)
			}
		}
	}
	return result, nil
}

// xmlEbook represents a single "ebook" object as read from the gutenberg catalog.
// This structure was derived by pasting XML into an XML-to-Go converter and then
// editing it down to the bare minimum.
type xmlEbook struct {
	ID              string     `xml:"about,attr"`
	Publisher       string     `xml:"publisher"`
	Title           string     `xml:"title"`
	Creators        []xmlAgent `xml:"creator>agent"`
	Illustrators    []xmlAgent `xml:"ill>agent"`
	TableOfContents string     `xml:"tableOfContents"`
	Language        string     `xml:"language>Description>value"`
	Subjects        []struct { // we need both value and memberOf because some of the subjects are useless to us
		Description struct {
			Subject  string `xml:"value"`
			MemberOf struct {
				Resource string `xml:"resource,attr"`
			} `xml:"memberOf"`
		} `xml:"Description"`
	} `xml:"subject"`
	Issued    string `xml:"issued"`
	Downloads int    `xml:"downloads"`
	Rights    string `xml:"rights"`
	License   struct {
		Text     string `xml:",chardata"`
		Resource string `xml:"resource,attr"`
	} `xml:"license"`
	Copyright string    `xml:"marc260"`
	Edition   string    `xml:"marc250"`
	Type      string    `xml:"type>Description>value"`
	Formats   []xmlFile `xml:"hasFormat>file"`
}

type xmlAgent struct {
	ID        string   `xml:"about,attr"`
	Name      string   `xml:"name"`
	Alias     []string `xml:"alias"`
	Birthdate struct {
		Text     string `xml:",chardata"`
		Datatype string `xml:"datatype,attr"`
	} `xml:"birthdate"`
	Deathdate struct {
		Text     string `xml:",chardata"`
		Datatype string `xml:"datatype,attr"`
	} `xml:"deathdate"`
	Webpage []struct {
		Resource string `xml:"resource,attr"`
	} `xml:"webpage"`
}

type xmlFile struct {
	About      string   `xml:"about,attr"`
	Formats    []string `xml:"format>Description>value"`
	Extent     int      `xml:"extent"`
	Modified   string   `xml:"modified"`
	IsFormatOf struct {
		Resource string `xml:"resource,attr"`
	} `xml:"isFormatOf"`
}

const xmlDateFormat = "2006-01-02"

func mustDate(s string) books.Date {
	if date, ix := books.ParseDate(s); ix != 0 {
		return date
	}
	return books.Date{}
}

// asAgent generates an Agent from an xmlAgent
func (x *xmlAgent) asAgent() books.Agent {
	agent := books.Agent{
		ID:        x.ID,
		Name:      x.Name,
		Aliases:   x.Alias,
		BirthDate: mustDate(x.Birthdate.Text),
		DeathDate: mustDate(x.Deathdate.Text),
		Webpages:  make([]string, 0),
	}
	for _, wp := range x.Webpage {
		agent.Webpages = append(agent.Webpages, wp.Resource)
	}
	return agent
}

// asEBook generates an EBook from an xmlEBook
func (x *xmlEbook) asEBook() books.EBook {
	et := books.EBook{
		ID:              x.ID,
		Publisher:       x.Publisher,
		Title:           x.Title,
		Creators:        make([]string, 0, 1),
		Illustrators:    make([]string, 0, 0),
		TableOfContents: x.TableOfContents,
		Language:        x.Language,
		DownloadCount:   x.Downloads,
		Rights:          x.Rights,
		Copyright:       x.Copyright,
		CopyrightDates:  books.ParseAllDates(x.Copyright),
		Edition:         x.Edition,
		Type:            x.Type,
		Files:           make([]books.PGFile, 0, 4),
		Agents:          make(map[string]books.Agent),
	}
	for i := range x.Creators {
		et.Creators = append(et.Creators, x.Creators[i].ID)
		et.Agents[x.Creators[i].ID] = x.Creators[i].asAgent()
	}
	for i := range x.Illustrators {
		et.Illustrators = append(et.Illustrators, x.Illustrators[i].ID)
		et.Agents[x.Illustrators[i].ID] = x.Illustrators[i].asAgent()
	}
	for i := range x.Subjects {
		if strings.HasSuffix(x.Subjects[i].Description.MemberOf.Resource, "LCSH") {
			et.Subjects = append(et.Subjects, x.Subjects[i].Description.Subject)
		}
	}
	et.Issued, _ = books.ParseDate(x.Issued)
	return et
}

func (x *xmlFile) asFile() books.PGFile {
	f := books.PGFile{
		Location:   x.About,
		Formats:    x.Formats,
		FileSize:   x.Extent,
		IsFormatOf: x.IsFormatOf.Resource,
	}
	f.Modified, _ = books.ParseDate(x.Modified)

	return f
}

// xmlRdf is the structure of the overall file.
type xmlRdf struct {
	XMLName    xml.Name   `xml:"RDF"`
	Namespaces []string   `xml:",any,attr"`
	EBooks     []xmlEbook `xml:"ebook"`
}

// Loader loads an RDF file given a reader to it
type Loader struct {
	reader        io.Reader
	ebookFilters  []books.EBookFilter
	pgFileFilters []books.PGFileFilter
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
		loader.ebookFilters = []books.EBookFilter{func(*books.EBook) bool { return true }}
	}

	return loader
}

// EBookFilter returns a LoaderOption that adds an EBookFilter
func EBookFilter(f books.EBookFilter) LoaderOption {
	return func(ldr *Loader) {
		ldr.ebookFilters = append(ldr.ebookFilters, f)
	}
}

// PGFileFilter returns a LoaderOption that adds a PGFileFilter
func PGFileFilter(f books.PGFileFilter) LoaderOption {
	return func(ldr *Loader) {
		ldr.pgFileFilters = append(ldr.pgFileFilters, f)
	}
}

// LoadOnly returns a LoaderOptions that limits the number of items loaded
func LoadOnly(n int) LoaderOption {
	return func(ldr *Loader) {
		ldr.loadOnly = n
	}
}

// load is a helper function used by the Load functions
func (r *Loader) load(rdr io.Reader) []books.EBook {
	var data xmlRdf
	decoder := xml.NewDecoder(rdr)
	if err := decoder.Decode(&data); err != nil {
		log.Fatal(err)
	}

	// Go through the ebooks and keep the ones that pass the filter
	ebooks := make([]books.EBook, 0)
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
				// agents =
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
func (r *Loader) LoadOne(bookdata *books.BookData) int {
	// Go through the ebooks and keep the ones that pass the filter
	ebooks := r.load(r.reader)
	bookdata.Update(ebooks)
	return 1
}

// LoadTar loads from a reader, expecting the reader to be a tar file that contains lots of files of books
// It returns the number of files that were processed within the tar, and replaces the bookdata's contents.
// If loadOnly is set, it limits the number of items loaded. This is mainly useful for testing.
func (r *Loader) LoadTar(bookdata *books.BookData) int {
	count := 0
	tr := tar.NewReader(r.reader)
	ebooks := make([]books.EBook, 0)
	for {
		_, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			log.Fatalf("Count=%d, err=%v", count, err)
		}
		newtexts := r.load(tr)
		ebooks = append(ebooks, newtexts...)
		count++
		if r.loadOnly > 0 && len(ebooks) >= r.loadOnly {
			break // end early because loadOnly
		}
	}

	bookdata.Update(ebooks)
	return count
}
