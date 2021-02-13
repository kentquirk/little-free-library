package books

import "github.com/kentquirk/stringset/v2"

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
	ID              string               `json:"id,omitempty"`
	Publisher       string               `json:"publisher,omitempty"`
	Title           string               `json:"title,omitempty"`
	Creators        []string             `json:"creators,omitempty"`
	Illustrators    []string             `json:"illustrators,omitempty"`
	TableOfContents string               `json:"table_of_contents,omitempty"`
	Language        string               `json:"language,omitempty"`
	Subjects        []string             `json:"subjects,omitempty"`
	Issued          Date                 `json:"issued,omitempty"`
	DownloadCount   int                  `json:"download_count,omitempty"`
	Rights          string               `json:"rights,omitempty"`
	Copyright       string               `json:"copyright,omitempty"`
	Edition         string               `json:"edition,omitempty"`
	Type            string               `json:"type,omitempty"`
	Files           []PGFile             `json:"files,omitempty"`
	Agents          map[string]Agent     `json:"agents,omitempty"`
	CopyrightDates  []Date               `json:"-"`
	Words           *stringset.StringSet `json:"-"`
}

func (e *EBook) extractWords() {
	w := stringset.New().Add(GetWords(e.Title)...)
	for i := range e.Subjects {
		w.Add(GetWords(e.Subjects[i])...)
	}
	for _, v := range e.Agents {
		v.AddWords(w)
	}
	e.Words = w
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

// AddWords the list of lower-case alphanumerics in the
// Agent Name and Aliases to the given StringSet
func (a Agent) AddWords(w *stringset.StringSet) {
	w.Add(GetWords(a.Name)...)
	for i := range a.Aliases {
		w.Add(GetWords(a.Aliases[i])...)
	}
}
