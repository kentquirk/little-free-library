package rdf

import (
	"encoding/xml"
	"strings"

	"github.com/kentquirk/little-free-library/pkg/booktypes"
	"github.com/kentquirk/little-free-library/pkg/date"
)

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

// asAgent generates an Agent from an xmlAgent
func (x *xmlAgent) asAgent() booktypes.Agent {
	agent := booktypes.Agent{
		ID:        x.ID,
		Name:      x.Name,
		Aliases:   x.Alias,
		BirthDate: date.ParseOnly(x.Birthdate.Text),
		DeathDate: date.ParseOnly(x.Deathdate.Text),
		Webpages:  make([]string, 0),
	}
	for _, wp := range x.Webpage {
		agent.Webpages = append(agent.Webpages, wp.Resource)
	}
	return agent
}

// asEBook generates an EBook from an xmlEBook
func (x *xmlEbook) asEBook() booktypes.EBook {
	eb := booktypes.EBook{
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
		CopyrightDates:  date.ParseAllDates(x.Copyright),
		Edition:         x.Edition,
		Type:            x.Type,
		Files:           make([]booktypes.PGFile, 0, 4),
		Issued:          date.ParseOnly(x.Issued),
		Agents:          make(map[string]booktypes.Agent),
		Words:           nil,
	}
	for i := range x.Creators {
		eb.Creators = append(eb.Creators, x.Creators[i].ID)
		eb.Agents[x.Creators[i].ID] = x.Creators[i].asAgent()
	}
	for i := range x.Illustrators {
		eb.Illustrators = append(eb.Illustrators, x.Illustrators[i].ID)
		eb.Agents[x.Illustrators[i].ID] = x.Illustrators[i].asAgent()
	}
	for i := range x.Subjects {
		if strings.HasSuffix(x.Subjects[i].Description.MemberOf.Resource, "LCSH") {
			eb.Subjects = append(eb.Subjects, x.Subjects[i].Description.Subject)
		}
	}
	eb.ExtractWords()
	return eb
}

func (x *xmlFile) asFile() booktypes.PGFile {
	f := booktypes.BuildFile(
		x.IsFormatOf.Resource,
		x.About,
		x.Formats,
		x.Extent,
		x.Modified,
	)

	return f
}

// xmlRdf is the structure of the overall file.
type xmlRdf struct {
	XMLName    xml.Name   `xml:"RDF"`
	Namespaces []string   `xml:",any,attr"`
	EBooks     []xmlEbook `xml:"ebook"`
}
