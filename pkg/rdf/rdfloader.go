package rdf

import (
	"archive/tar"
	"encoding/xml"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

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

// multiname is an array of names; we need the special type because
// the XML uses a single "creator" or "contributor" object if there's only one, but uses a "bag" if there
// are more than one. So we need a special unmarshaler to handle it.
type multiname []string

// UnmarshalXML implements the Unmarshaler interface for the multiname type.
func (c *multiname) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	result, err := extractCharDataToEndToken(d, start)
	if err != nil {
		return err
	}
	(*c) = result
	return nil
}

// xmlEtext represents a single "etext" object as read from the gutenberg catalog.
// This structure was derived by pasting XML into an XML-to-Go converter and then
// editing it down to the bare minimum.
// Subject/Subjects is like Creators, but since the internals of the two representations
// differ, we can handle them both in the XML definition and fix it up in postprocessing.
type xmlEtext struct {
	ID              string     `xml:"about,attr"`
	Publisher       string     `xml:"publisher"`
	Title           string     `xml:"title"`
	Creators        []xmlAgent `xml:"creator>agent"`
	Illustrators    []xmlAgent `xml:"ill>agent"`
	Contributors    multiname  `xml:"contributor"`
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

// asEText generates an EText from an xmlEtext
func (x *xmlEtext) asEText() books.EText {
	et := books.EText{
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
		CopyrightYears:  make([]int, 0, 0),
		Edition:         x.Edition,
		Type:            x.Type,
		Files:           make([]books.PGFile, 0, 4),
	}
	for i := range x.Creators {
		et.Creators = append(et.Creators, x.Creators[i].ID)
	}
	for i := range x.Illustrators {
		et.Illustrators = append(et.Illustrators, x.Illustrators[i].ID)
	}
	for i := range x.Subjects {
		if strings.HasSuffix(x.Subjects[i].Description.MemberOf.Resource, "LCSH") {
			et.Subjects = append(et.Subjects, x.Subjects[i].Description.Subject)
		}
	}
	if len(x.Copyright) > 4 {
		p := regexp.MustCompile("[12][0-9]{3}")
		years := p.FindAllString(x.Copyright, -1)
		for _, y := range years {
			year, _ := strconv.Atoi(y)
			et.CopyrightYears = append(et.CopyrightYears, year)
		}
	}
	// TODO: log if this gets an error
	et.Issued, _ = time.Parse(xmlDateFormat, x.Issued)
	return et
}

func (x *xmlFile) asFile() books.PGFile {
	f := books.PGFile{
		Location:   x.About,
		Formats:    x.Formats,
		FileSize:   x.Extent,
		IsFormatOf: x.IsFormatOf.Resource,
	}
	f.Modified, _ = time.Parse(xmlDateFormat, x.Modified)

	return f
}

// xmlRdf is the structure of the overall file.
type xmlRdf struct {
	XMLName    xml.Name   `xml:"RDF"`
	Namespaces []string   `xml:",any,attr"`
	Etexts     []xmlEtext `xml:"ebook"`
}

// Loader loads an RDF file given a reader to it
type Loader struct {
	reader        io.Reader
	etextFilters  []books.ETextFilter
	pgFileFilters []books.PGFileFilter
}

const bufsize = 4096

// NewLoader constructs an RDF Loader from a reader.
func NewLoader(r io.Reader) *Loader {
	loader := &Loader{reader: r}
	return loader
}

// AddETextFilter adds an ETextFilter
func (r *Loader) AddETextFilter(f books.ETextFilter) {
	r.etextFilters = append(r.etextFilters, f)
}

// AddPGFileFilter adds a PGFileFilter
func (r *Loader) AddPGFileFilter(f books.PGFileFilter) {
	r.pgFileFilters = append(r.pgFileFilters, f)
}

// load is a helper function used by the Load functions
func (r *Loader) load(rdr io.Reader) []books.EText {
	var data xmlRdf
	decoder := xml.NewDecoder(rdr)
	if err := decoder.Decode(&data); err != nil {
		log.Fatal(err)
	}

	// Go through the etexts and keep the ones that pass the filter
	etexts := make([]books.EText, 0)
	for i := range data.Etexts {
		et := data.Etexts[i].asEText()
		for _, filt := range r.etextFilters {
			if !filt(&et) {
				continue
			}
		eachfile:
			for _, xf := range data.Etexts[i].Formats {
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
				etexts = append(etexts, et)
			}
		}
	}
	return etexts
}

// LoadOne parses and loads the XML data within its contents, expecting the contents to
// be a single file containing one or more EText entities.
// It only returns the entities that pass the filters that have been set up
// before calling load.
// Returns 1 (the number of files processed).
func (r *Loader) LoadOne(bookdata *books.BookData) int {
	// if there are no etextFilters, add a dummy one that passes everything
	if len(r.etextFilters) == 0 {
		r.AddETextFilter(func(*books.EText) bool { return true })
	}

	// Go through the etexts and keep the ones that pass the filter
	etexts := r.load(r.reader)
	bookdata.Update(etexts)
	return 1
}

// LoadTar loads from a reader, expecting the reader to be a tar file that contains lots of files of books
// It returns the number of files that were processed within the tar, and replaces the bookdata's contents.
func (r *Loader) LoadTar(bookdata *books.BookData) int {
	// if there are no etextFilters, add a dummy one that passes everything
	if len(r.etextFilters) == 0 {
		r.AddETextFilter(func(*books.EText) bool { return true })
	}

	count := 0
	tr := tar.NewReader(r.reader)
	etexts := make([]books.EText, 0)
	for {
		_, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			log.Fatalf("Count=%d, err=%v", count, err)
		}
		newtexts := r.load(tr)
		etexts = append(etexts, newtexts...)
		count++
	}

	bookdata.Update(etexts)
	return count
}
