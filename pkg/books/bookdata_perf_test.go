package books

import (
	"compress/bzip2"
	"compress/gzip"
	"flag"
	"io"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

func loadTestData(books *BookData) {
	resourcename := "/Users/kent/code/little-free-library/data/rdf-files.tar.bz2"
	var rdr io.Reader

	log.Printf("beginning book loading\n")
	// if our URL is an http resource, fetch it with exponential fallback on retry
	// it's a local file; if it fails, don't retry, just die
	// (local files are intended just for testing)
	f, err := os.Open(resourcename)
	if err != nil {
		log.Fatalf("couldn't load file %s: %s", resourcename, err)
	}
	rdr = f
	defer f.Close()

	// OK, now we have fetched something.
	// If it's a .bz2 file, unzip it
	if strings.HasSuffix(resourcename, ".bz2") {
		rdr = bzip2.NewReader(rdr)
		resourcename = resourcename[:len(resourcename)-4]
	}

	// or if it's a .gz file, unzip it
	if strings.HasSuffix(resourcename, ".gz") {
		var err error
		rdr, err = gzip.NewReader(rdr)
		if err != nil {
			log.Printf("couldn't unpack gzip: %v", err)
		}
		resourcename = resourcename[:len(resourcename)-3]
	}

	// now we have an uncompressed reader, we can start loading data from it
	count := 0
	starttime := time.Now()
	r := NewLoader(rdr,
		// We don't want to be delivering data that our users can't use, so we pre-filter the data that goes
		// into the dataset. The target language(s) and target formats can be specified in the config, and
		// only the data that meets these specifications will be saved.
		EBookFilterOpt(LanguageFilter("en")),
		PGFileFilterOpt(ContentFilter("plain_ascii")),
	)

	if strings.HasSuffix(resourcename, ".tar") {
		count = r.LoadTar(books)
	} else {
		// this is mainly useful for testing and debugging without waiting for big files
		count = r.LoadOne(books)
	}
	endtime := time.Now()
	log.Printf("book loading complete -- %d files read, %d books in dataset, took %s.\n", count, len(books.books), endtime.Sub(starttime).String())
}

var books *BookData
var constraints *ConstraintSpec

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		os.Exit(0)
	}
	books = NewBookData()
	loadTestData(books)
	os.Exit(m.Run())
}

func BenchmarkCreatorQuery(b *testing.B) {
	constraints = NewConstraintSpec()
	constraints.Limit = 1
	constraint, _, _ := ConstraintFromText("creator", "Poe")
	constraints.Includes = append(constraints.Includes, constraint)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		books.Query(constraints)
	}
}

func BenchmarkAuthorQuery(b *testing.B) {
	constraints = NewConstraintSpec()
	constraints.Limit = 1
	constraint, _, _ := ConstraintFromText("author", "Poe")
	constraints.Includes = append(constraints.Includes, constraint)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		books.Query(constraints)
	}
}

func BenchmarkIllustratorQuery(b *testing.B) {
	constraints = NewConstraintSpec()
	constraints.Limit = 1
	constraint, _, _ := ConstraintFromText("illustrator", "Parrish")
	constraints.Includes = append(constraints.Includes, constraint)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		books.Query(constraints)
	}
}

func BenchmarkTitleQuery(b *testing.B) {
	constraints = NewConstraintSpec()
	constraints.Limit = 1
	constraint, _, _ := ConstraintFromText("title", "dogs")
	constraints.Includes = append(constraints.Includes, constraint)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		books.Query(constraints)
	}
}

func BenchmarkSubjectQuery(b *testing.B) {
	constraints = NewConstraintSpec()
	constraints.Limit = 1
	constraint, _, _ := ConstraintFromText("illustrator", "music")
	constraints.Includes = append(constraints.Includes, constraint)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		books.Query(constraints)
	}
}
