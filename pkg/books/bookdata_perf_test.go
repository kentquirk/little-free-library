package books

import (
	"compress/bzip2"
	"compress/gzip"
	"io"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

func loadTestData(books *BookData) {
	if len(books.books) != 0 {
		return
	}
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

var books *BookData = NewBookData()
var constraints *ConstraintSpec

func BenchmarkCreatorQuery(b *testing.B) {
	loadTestData(books)
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
	loadTestData(books)
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
	loadTestData(books)
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
	loadTestData(books)
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
	loadTestData(books)
	constraints = NewConstraintSpec()
	constraints.Limit = 1
	constraint, _, _ := ConstraintFromText("illustrator", "music")
	constraints.Includes = append(constraints.Includes, constraint)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		books.Query(constraints)
	}
}

// Results before adding word index:
// BenchmarkCreatorQuery-12        	   34490	     34080 ns/op	     353 B/op	       2 allocs/op
// BenchmarkAuthorQuery-12         	   36412	     33400 ns/op	     353 B/op	       2 allocs/op
// BenchmarkIllustratorQuery-12    	     784	   1295505 ns/op	     495 B/op	       2 allocs/op
// BenchmarkTitleQuery-12          	    1711	    710174 ns/op	     374 B/op	       2 allocs/op
// BenchmarkSubjectQuery-12        	     100	  10073556 ns/op	     417 B/op	       1 allocs/op
//
// Results after word index:
// BenchmarkCreatorQuery-12        	  173044	      7352 ns/op	    2626 B/op	      70 allocs/op
// BenchmarkAuthorQuery-12         	  192050	      6493 ns/op	    2627 B/op	      70 allocs/op
// BenchmarkIllustratorQuery-12    	    2196	    463247 ns/op	   48323 B/op	    1472 allocs/op
// BenchmarkTitleQuery-12          	   10000	    111319 ns/op	   35308 B/op	    1084 allocs/op
// BenchmarkSubjectQuery-12        	     318	   3739078 ns/op	  303261 B/op	    9451 allocs/op
//
// Basically, 3-7x improvement.
