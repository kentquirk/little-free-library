package rdf

import (
	"bufio"
	"encoding/xml"
	"io"
	"log"
	"regexp"
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
	ID            string    `xml:"ID,attr"`
	Publisher     string    `xml:"publisher"`
	Title         string    `xml:"title"`
	Creator       multiname `xml:"creator"`
	Contributor   multiname `xml:"contributor"`
	FriendlyTitle string    `xml:"friendlytitle"`
	Language      string    `xml:"language>ISO639-2>value"`
	Subject       string    `xml:"subject>LCSH>value"`
	Subjects      []string  `xml:"subject>Bag>li>LCSH>value"`
	Created       string    `xml:"created>W3CDTF>value"`
	Downloads     int       `xml:"downloads>nonNegativeInteger>value"`
	Rights        struct {
		Resource string `xml:"resource,attr"`
	} `xml:"rights"`
}

const xmlDateFormat = "2006-01-02"

// asEText generates an EText from an xmlEtext
func (x *xmlEtext) asEText() books.EText {
	et := books.EText{
		ID:            x.ID,
		Publisher:     x.Publisher,
		Title:         x.Title,
		Creator:       x.Creator,
		Contributor:   x.Contributor,
		FriendlyTitle: x.FriendlyTitle,
		Language:      x.Language,
		Subjects:      x.Subjects,
		DownloadCount: x.Downloads,
		Rights:        x.Rights.Resource,
		Files:         make([]books.PGFile, 0),
	}
	et.Subjects = append(et.Subjects, x.Subject)
	// TODO: log if this gets an error
	et.Created, _ = time.Parse(xmlDateFormat, x.Created)
	return et
}

type xmlFile struct {
	About      string   `xml:"about,attr"`
	Formats    []string `xml:"format>IMT>value"`
	Extent     string   `xml:"extent"`
	Modified   string   `xml:"modified>W3CDTF>value"`
	IsFormatOf struct {
		Resource string `xml:"resource,attr"`
	} `xml:"isFormatOf"`
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
	Etexts     []xmlEtext `xml:"etext"`
	Files      []xmlFile  `xml:"file"`
}

// Loader loads an RDF file given a reader to it
type Loader struct {
	reader        *bufio.Reader
	entityMap     map[string]string
	etextFilters  []books.ETextFilter
	pgFileFilters []books.PGFileFilter
}

const bufsize = 4096

// NewLoader constructs an RDF Loader from a reader. It preloads the entities from the reader.
// We have some invented XML entities that we need to fix on the way by.
// These entities are defined in the beginning of the XML file, but it is difficult
// to get that information through the XML library. So instead, we can
// peek into the file at the first 4K bytes (without advancing the read pointer)
// to find lines that look like this:
// <!ENTITY pg  "Project Gutenberg">
// we use this to populate an entity map that we pass to the decoder when we load.
func NewLoader(r io.Reader) *Loader {
	loader := &Loader{
		reader: bufio.NewReaderSize(r, bufsize),
		entityMap: map[string]string{
			"pg":  "Project Gutenberg",
			"lic": "LICENSE",
			"f":   "WEBROOT/",
		},
	}
	if prefix, err := loader.reader.Peek(bufsize); err == nil {
		pat := regexp.MustCompile(`ENTITY +([a-z]+) +"([^"]+)"`)
		matches := pat.FindAllStringSubmatch(string(prefix), -1)
		for _, m := range matches {
			loader.entityMap[m[1]] = m[2]
		}
	}
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

// Load parses and loads the XML data within its contents.
// It only returns the entities that pass the filters that have been set up
// before calling load. If no filters are set up, this will return empty results!
func (r *Loader) Load(bookdata *books.BookData) {
	decoder := xml.NewDecoder(r.reader)
	decoder.Entity = r.entityMap

	var data xmlRdf
	if err := decoder.Decode(&data); err != nil {
		log.Fatal(err)
	}

	// first go through the files and organize them, eliminating
	// the ones that don't pass the filter
	files := make(map[string][]books.PGFile)
eachfile:
	for i := range data.Files {
		file := data.Files[i].asFile()
		for _, filt := range r.pgFileFilters {
			if !filt(&file) {
				continue eachfile
			}
		}
		key := file.IsFormatOf[1:] // this has a # at the beginning
		if f, ok := files[key]; !ok {
			files[key] = []books.PGFile{file}
		} else {
			files[key] = append(f, file)
		}
	}

	// Now go through the etexts and keep the ones that pass
	// the filter AND that have a non-empty list of files.
	etexts := make([]books.EText, 0)
	for i := range data.Etexts {
		et := data.Etexts[i].asEText()
		for _, filt := range r.etextFilters {
			if !filt(&et) {
				continue
			}
			// only store objects we have files for
			if len(files[et.ID]) != 0 {
				et.Files = files[et.ID]
				etexts = append(etexts, et)
			}
		}
	}

	bookdata.Update(etexts)
}
