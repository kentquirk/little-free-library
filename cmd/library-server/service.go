package main

import (
	"compress/bzip2"
	"compress/gzip"
	htmltmpl "html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	texttmpl "text/template"
	"time"

	"github.com/kentquirk/little-free-library/pkg/books"
	"github.com/labstack/echo/v4"
)

type service struct {
	Config        Config
	Books         *books.BookData
	HTMLTemplates map[string]*htmltmpl.Template
	TextTemplates map[string]*texttmpl.Template
}

func newService() *service {
	svc := &service{
		Books:         books.NewBookData(),
		HTMLTemplates: make(map[string]*htmltmpl.Template),
		TextTemplates: make(map[string]*texttmpl.Template),
	}
	return svc
}

func (svc *service) setupRoutes(e *echo.Echo) {
	// Routes
	e.GET("/", svc.err400)
	e.GET("/doc", svc.doc)
	e.GET("/health", svc.health)
	e.GET("/books/query", svc.bookQuery)
	e.GET("/books/count", svc.bookCount)
	e.GET("/books/query/html/:format", svc.bookQueryHTML)
	e.GET("/books/stats", svc.bookStats)
	e.GET("/book/:id", svc.bookByID)
	e.GET("/book/details/*", svc.bookDetails)

	e.GET("/qr", svc.qrcodegen)
	e.Static("/static", svc.Config.StaticRoot)
}

// load is intended to be run as a goroutine and also schedules itself to be re-run later.
func load(svc *service) {
	resourcename := svc.Config.URL
	var rdr io.Reader

	log.Printf("beginning book loading\n")
	// if our URL is an http resource, fetch it with exponential fallback on retry
	if strings.HasPrefix(resourcename, "http") {
		for retryTime, _ := time.ParseDuration("1s"); ; retryTime *= 2 {
			resp, err := http.Get(resourcename)
			log.Printf("Got %d fetching %s", resp.StatusCode, resourcename)
			if err == nil && resp.StatusCode < 300 {
				rdr = resp.Body
				defer resp.Body.Close()
				break
			}
			status := resp.Status
			if err != nil {
				status = err.Error()
			}
			log.Printf("load: couldn't fetch %s: %s -- will retry in %s", resourcename, status, retryTime)
			time.Sleep(retryTime)
		}
	} else {
		// it's a local file; if it fails, don't retry, just die
		// (local files are intended just for testing)
		f, err := os.Open(resourcename)
		if err != nil {
			log.Fatalf("couldn't load file %s: %s", resourcename, err)
		}
		rdr = f
		defer f.Close()
	}

	// We've gotten to the point where we have something we can read, so let's plan to refresh
	// whatever we get later. Note that this calls ourselves with the same payload, so
	// while it's not technically recursive it does keep starting this goroutine forever.
	time.AfterFunc(svc.Config.RefreshTime, func() {
		load(svc)
	})

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
	r := books.NewLoader(rdr,
		// We don't want to be delivering data that our users can't use, so we pre-filter the data that goes
		// into the dataset. The target language(s) and target formats can be specified in the config, and
		// only the data that meets these specifications will be saved.
		books.EBookFilterOpt(books.LanguageFilter(svc.Config.Languages...)),
		books.PGFileFilterOpt(books.ContentFilter(svc.Config.Formats...)),
		books.LoadAtMostOpt(svc.Config.LoadAtMost),
	)

	if strings.HasSuffix(resourcename, ".tar") {
		count = r.LoadTar(svc.Books)
	} else {
		// this is mainly useful for testing and debugging without waiting for big files
		count = r.LoadOne(svc.Books)
	}
	endtime := time.Now()
	log.Printf("book loading complete -- %d files read, %d books in dataset, took %s.\n", count, svc.Books.NBooks(), endtime.Sub(starttime).String())
}
