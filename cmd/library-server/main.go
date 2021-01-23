// Package main provides the backend to support the Digital Little Free Library project
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/codingconcepts/env"
	"github.com/kentquirk/little-free-library/pkg/books"
	"github.com/kentquirk/little-free-library/pkg/rdf"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/acme/autocert"
)

type config struct {
	CacheDir        string        `env:"CACHEDIR" default:"/var/www/.cache"`
	Port            int           `env:"PORT" required:"true"`
	MaxLimit        int           `env:"MAXLIMIT" default:"100"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" default:"5s"`
	Languages       []string      `env:"LANGUAGES" delimiter:"," default:"en"`
	Formats         []string      `env:"FORMATS" delimiter:"," default:"plain_8859.1,plain_ascii,plain_utf8,mobi,epub"`
}

func setupMiddleware(e *echo.Echo) {
	// logging
	e.Use(middleware.Logger())
	// crash handling
	e.Use(middleware.Recover())
}

var bookData *books.BookData = new(books.BookData)

func load(svc *service) {
	f, err := os.Open("./data/catalog.rdf")
	if err != nil {
		log.Fatal("couldn't load!")
	}
	r := rdf.NewLoader(f)
	r.AddETextFilter(books.LanguageFilter(svc.Config.Languages...))
	r.AddPGFileFilter(books.ContentFilter(svc.Config.Formats...))
	log.Printf("beginning book loading")
	r.Load(svc.Books)
	log.Printf("book loading complete")
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
