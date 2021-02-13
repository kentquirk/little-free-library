// Package main provides the backend to support the Digital Little Free Library project
package main

import (
	"context"
	"crypto/sha512"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/codingconcepts/env"
	"github.com/kentquirk/stringset/v2"
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
	ValidUsers       []string      `env:"VALID_USERS"`
	AuthSecret       string        `env:"AUTH_SECRET"`
	AuthSalt         string        `env:"AUTH_SALT" default:"This is sample salt."`
	CacheDir         string        `env:"CACHE_DIR" default:"/var/www/.cache"`
	StaticRoot       string        `env:"STATIC_ROOT" required:"true"`
	Port             int           `env:"PORT" required:"true"`
	MaxLimit         int           `env:"MAXLIMIT" default:"100"`
	ShutdownTimeout  time.Duration `env:"SHUTDOWN_TIMEOUT" default:"5s"`
	Languages        []string      `env:"LANGUAGES" delimiter:"," default:"en"`
	Formats          []string      `env:"FORMATS" delimiter:"," default:"plain_8859.1,plain_ascii,plain_utf8,mobi,epub,html_text"`
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

func authValidator(cfg Config) func(key string, c echo.Context) (bool, error) {
	keys := stringset.New()
	for _, u := range cfg.ValidUsers {
		h := sha512.New()
		h.Write([]byte(cfg.AuthSecret))
		h.Write([]byte(cfg.AuthSalt))
		h.Write([]byte(u))
		key := fmt.Sprintf("%x", h.Sum(nil))[:16]
		keys.Add(key)
	}
	// log.Println(keys.WrappedJoin("Authorized keys:", ", ", ""))

	return func(key string, c echo.Context) (bool, error) {
		return keys.Contains(key), nil
	}
}

func setupMiddleware(e *echo.Echo, cfg Config) {
	e.Use(
		// don't allow big bodies to choke us
		middleware.BodyLimit("64K"),
		// add a request ID
		middleware.RequestID(),
		// logging
		middleware.Logger(),
		// crash handling
		middleware.Recover(),
		// key auth
		middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
			Validator: authValidator(cfg),
			// skip auth for local queries
			Skipper: func(c echo.Context) bool {
				return strings.HasPrefix(c.Request().Host, "localhost")
			},
		}),
		// TODO: add rate limiter
	)
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

	setupMiddleware(e, svc.Config)
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
