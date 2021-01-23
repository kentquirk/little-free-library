package books

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
