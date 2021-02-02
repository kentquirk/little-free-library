// Package main provides the backend to support the Digital Little Free Library project
package main

import (
	"compress/bzip2"
	"compress/gzip"
	"context"
	"fmt"
	htmltmpl "html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	texttmpl "text/template"
	"time"

	"github.com/codingconcepts/env"
	"github.com/kentquirk/little-free-library/pkg/books"
	"github.com/kentquirk/little-free-library/pkg/rdf"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/acme/autocert"
)

// Config stores configuration variables that can be specified in the environment.
// They are:
// PORT (required). Specifies the port number. If the port number is 443, we automatically do a TLS setup and
//   get a certificate from Let's Encrypt. Otherwise, we just do a normal HTTP setup.
// CACHE_DIR (default:"/var/www/.cache"). Specifies where on disk the cache information for TLS/Let's Encrypt is stored.
// STATIC_ROOT (no default). Specifies the path that should be statically served. There is no safe default.
// MAXLIMIT (default 100). The maximum number of items that can be returned at once, even if the query
//   specifies a limit value.
// SHUTDOWN_TIMEOUT (default 5s): maximum time the server will wait to try to shutdown nicely when interrupted.
// LANGUAGES (comma-separated, default 'en'). When loading the data, only books listing one of the specified
//   languages will be stored in the database.
// FORMATS (comma-separated, by default the most popular formats). Friendly format names are specified in
//   formats.go.
// REFRESH_TIME (default 23h17m to avoid hitting the servers at the same time every day. This is the frequency
//   at which the data is refreshed by downloading it from Project Gutenberg.
// URL. The URL used to fetch catalog.rdf.zip from Project Gutenberg.
// LOAD_AT_MOST. If this is a nonzero number, the system will load no more than this many books. Useful for debugging.
// NO_CACHE_TEMPLATES. If this is true, templates will be reloaded on every fetch (useful for editing templates).
type Config struct {
	CacheDir         string        `env:"CACHE_DIR" default:"/var/www/.cache"`
	StaticRoot       string        `env:"STATIC_ROOT" required:"true"`
	Port             int           `env:"PORT" required:"true"`
	MaxLimit         int           `env:"MAXLIMIT" default:"100"`
	ShutdownTimeout  time.Duration `env:"SHUTDOWN_TIMEOUT" default:"5s"`
	Languages        []string      `env:"LANGUAGES" delimiter:"," default:"en"`
	Formats          []string      `env:"FORMATS" delimiter:"," default:"plain_8859.1,plain_ascii,plain_utf8,mobi,epub"`
	RefreshTime      time.Duration `env:"REFRESH_TIME" default:"23h17m"`
	URL              string        `env:"URL" default:"/Users/kent/code/little-free-library/data/rdf-files.tar.bz2"`
	LoadAtMost       int           `env:"LOAD_AT_MOST"`
	NoCacheTemplates bool          `env:"NO_CACHE_TEMPLATES"`
	// This is the URL that is current for the latest catalog at gutenberg.org as of January 2021. Please do not
	// use it for testing; download a local copy. Only use this URL once you are confident that your code is running
	// properly and will not spam the server with requests. Best to leave the default value as a local file and override
	// it in your production server configuration.
	// URL             string        `env:"URL" default:"http://www.gutenberg.org/cache/epub/feeds/rdf-files.tar.bz2"`
}

func setupMiddleware(e *echo.Echo) {
	e.Use(
		// don't allow big bodies to choke us
		middleware.BodyLimit("64K"),
		// add a request ID
		middleware.RequestID(),
		// logging
		middleware.Logger(),
		// crash handling
		middleware.Recover(),
		// TODO: add rate limiter
	)
}

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
	r := rdf.NewLoader(rdr,
		// We don't want to be delivering data that our users can't use, so we pre-filter the data that goes
		// into the dataset. The target language(s) and target formats can be specified in the config, and
		// only the data that meets these specifications will be saved.
		rdf.EBookFilter(books.LanguageFilter(svc.Config.Languages...)),
		rdf.PGFileFilter(books.ContentFilter(svc.Config.Formats...)),
		rdf.LoadAtMost(svc.Config.LoadAtMost),
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

func main() {
	svc := newService()

	// Load config
	if err := env.Set(&(svc.Config)); err != nil {
		log.Fatal(err)
	}

	// Echo instance
	e := echo.New()
	// TODO: put dircache in config
	e.AutoTLSManager.Cache = autocert.DirCache(svc.Config.CacheDir)

	setupMiddleware(e)
	svc.setupRoutes(e)
	e.Renderer = svc

	// If the port is the SSL port, then do a TLS setup, otherwise just do normal HTTP
	startfunc := e.Start
	if svc.Config.Port == 443 {
		startfunc = e.StartAutoTLS
	}

	// background-load the data
	go load(svc)

	// Start server
	go func() {
		if err := startfunc(fmt.Sprintf(":%d", svc.Config.Port)); err != nil {
			e.Logger.Info("shutting down the server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), svc.Config.ShutdownTimeout)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}
