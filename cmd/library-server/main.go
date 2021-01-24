// Package main provides the backend to support the Digital Little Free Library project
package main

import (
	"compress/bzip2"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
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
// CACHEDIR (default:"/var/www/.cache"). Specifies where on disk the cache information for TLS/Let's Encrypt is stored.
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
type Config struct {
	CacheDir        string        `env:"CACHEDIR" default:"/var/www/.cache"`
	Port            int           `env:"PORT" required:"true"`
	MaxLimit        int           `env:"MAXLIMIT" default:"100"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" default:"5s"`
	Languages       []string      `env:"LANGUAGES" delimiter:"," default:"en"`
	Formats         []string      `env:"FORMATS" delimiter:"," default:"plain_8859.1,plain_ascii,plain_utf8,mobi,epub"`
	RefreshTime     time.Duration `env:"REFRESH_TIME" default:"23h17m"`
	URL             string        `env:"URL" default:"/Users/kent/code/little-free-library/data/rdf-files.tar.bz2"`
	// URL             string        `env:"URL" default:"http://www.gutenberg.org/cache/epub/feeds/rdf-files.tar.bz2"`
}

func setupMiddleware(e *echo.Echo) {
	// logging
	e.Use(middleware.Logger())
	// crash handling
	e.Use(middleware.Recover())
}

var bookData *books.BookData = new(books.BookData)

func load(svc *service) {
	resourcename := svc.Config.URL
	var rdr io.Reader
	if strings.HasPrefix(resourcename, "http") {
		resp, err := http.Get(resourcename)
		if err != nil {
			log.Fatalf("couldn't load from %s: %s", resourcename, err)
		}
		rdr = resp.Body
		defer resp.Body.Close()
	} else {
		// it's a local file
		f, err := os.Open(resourcename)
		if err != nil {
			log.Fatalf("couldn't load file %s: %s", resourcename, err)
		}
		rdr = f
		defer f.Close()
	}

	if strings.HasSuffix(resourcename, ".bz2") {
		rdr = bzip2.NewReader(rdr)
		resourcename = resourcename[:len(resourcename)-4]
	}

	log.Printf("beginning book loading\n")
	count := 0
	starttime := time.Now()
	if strings.HasSuffix(resourcename, ".tar") {
		r := rdf.NewLoader(rdr)
		r.AddETextFilter(books.LanguageFilter(svc.Config.Languages...))
		r.AddPGFileFilter(books.ContentFilter(svc.Config.Formats...))
		count = r.LoadTar(svc.Books)
	} else {
		r := rdf.NewLoader(rdr)
		r.AddETextFilter(books.LanguageFilter(svc.Config.Languages...))
		r.AddPGFileFilter(books.ContentFilter(svc.Config.Formats...))

		r.LoadBulk(svc.Books)
		count++
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
