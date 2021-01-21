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
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/acme/autocert"
)

type config struct {
	CacheDir        string        `env:"CACHEDIR" default:"/var/www/.cache"`
	Port            int           `env:"PORT" required:"true"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" default:"5s"`
	// Secret          []byte        `env:"SECRET" required:"true"`
	// Peers    []string `env:"PEERS"` // you can use `delimiter` tag to specify separator, for example `delimiter:" "`
	// ConnectionTimeout time.Duration `env:"TIMEOUT" default:"10s"`
}

func setupMiddleware(e *echo.Echo) {
	// logging
	e.Use(middleware.Logger())
	// crash handling
	e.Use(middleware.Recover())
}

var bookData *BookData = new(BookData)

func load() {
	f, err := os.Open("./data/catalog.rdf")
	if err != nil {
		log.Fatal("couldn't load!")
	}
	rdf := NewRDFLoader(f)
	rdf.AddETextFilter(LanguageFilter("en"))
	rdf.AddPGFileFilter(ContentFilter(TextPlain, TextPlainASCII, TextPlainLatin, Mobi))
	rdf.Load(bookData)
}

func setupRoutes(e *echo.Echo) {
	// Routes
	e.GET("/", err400)
	e.GET("/doc", doc)
	e.GET("/health", health)
	e.GET("/books/query", bookData.bookQuery)
	e.GET("/books/summary", bookData.bookSummary)
	e.GET("/details/:id", bookDetails)
	e.GET("/qr/:id", qrcodegen)
	e.GET("/book/:id", bookByID)
}

func main() {
	// Load config
	cfg := config{}
	if err := env.Set(&cfg); err != nil {
		log.Fatal(err)
	}

	// Echo instance
	e := echo.New()
	// TODO: put dircache in config
	e.AutoTLSManager.Cache = autocert.DirCache(cfg.CacheDir)

	setupMiddleware(e)
	setupRoutes(e)

	// If the port is the SSL port, then do a TLS setup, otherwise just do normal HTTP
	startfunc := e.Start
	if cfg.Port == 443 {
		startfunc = e.StartAutoTLS
	}

	// background-load the data
	go load()

	// Start server
	go func() {
		if err := startfunc(fmt.Sprintf(":%d", cfg.Port)); err != nil {
			e.Logger.Info("shutting down the server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}
