package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/kentquirk/little-free-library/pkg/books"
	"github.com/labstack/echo/v4"
	"github.com/skip2/go-qrcode"
)

func parseIntWithDefault(input string, def int) (int, error) {
	if input == "" {
		return def, nil
	}

	n, err := strconv.Atoi(input)
	if err != nil {
		return def, echo.NewHTTPError(http.StatusBadRequest, "parameter must be an integer")
	}
	return n, nil
}

// err400 returns 400 and is used to discourage random queries
func (svc *service) err400(c echo.Context) error {
	return c.String(http.StatusBadRequest, "Go away.")
}

// doc returns a documentation page
// TODO: elaborate!
func (svc *service) doc(c echo.Context) error {
	doctext := `
	<h1>Little Free Library</h1>
	<p>This service generates data for the digital little free library project.
	The point of the project is to deliver a small collection of freely shareable
	book content to a digital device, in much the same way that physical little
	free library boxes can hold a small collection of books.
	</p>
	`
	return c.String(http.StatusOK, doctext)
}

// health returns 200 Ok and can be used by a load balancer to indicate
// that the service is stable
func (svc *service) health(c echo.Context) error {
	return c.String(http.StatusOK, "Ok\n")
}

// qrcodegen is a handler that returns a png image of a QR code
// It's intended to be used within a templated img tag, so it doesn't do anything
// other than render a QR code of the URL parameter passed in. It supports a couple of
// parameters to control the output
//
// Required query parameter is url, which is used as the body of the QR code
//
// Optional query parameters are:
// * size is a number of the pixel size of the png; default is 512.
// * level is the recovery level - options are "l" (low), "m" (medium -- default), "h" (high), "x" (max)
func (svc *service) qrcodegen(c echo.Context) error {
	url := c.QueryParam("url")
	if url == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "URL query parameter required")
	}

	level := qrcode.Medium
	s := c.QueryParam("level")
	switch s {
	case "l":
		level = qrcode.Low
	case "m":
		level = qrcode.Medium
	case "h":
		level = qrcode.High
	case "x":
		level = qrcode.Highest
	case "":
	// do nothing
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "level parameter must be one of l,m,h,x")
	}

	size, err := parseIntWithDefault(c.QueryParam("size"), 256)
	if err != nil {
		return err
	}
	if size < 128 || size > 1024 {
		return echo.NewHTTPError(http.StatusBadRequest, "parameter must be between 128 and 1024")
	}

	png, err := qrcode.Encode(url, level, size)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "could not encode that URL")
	}
	return c.Blob(http.StatusOK, "image/png", png)
}

func (svc *service) buildConstraints(values url.Values) (*books.ConstraintSpec, error) {
	constraints := books.NewConstraintSpec()
	for k, vals := range values {
		// once for each copy of a given key
		for _, v := range vals {
			switch k {
			case "or":
				constraints.IncludeCombiner = books.Or
			case "and":
				constraints.IncludeCombiner = books.And
			case "-or":
				constraints.ExcludeCombiner = books.Or
			case "-and":
				constraints.ExcludeCombiner = books.And
			case "limit", "lim":
				n, _ := strconv.Atoi(v)
				if n > 0 && n <= svc.Config.MaxLimit {
					constraints.Limit = n
				} else {
					return nil, echo.NewHTTPError(http.StatusBadRequest,
						fmt.Sprintf("limit must be >0 and <=%d", svc.Config.MaxLimit))
				}
			case "page", "pg":
				n, err := strconv.Atoi(v)
				if err != nil || n < 0 {
					return nil, echo.NewHTTPError(http.StatusBadRequest, "page must be numeric and >0")
				}
				constraints.Page = n
			case "random", "rand":
				constraints.Random = true
			default:
				var constraint books.ConstraintFunctor
				exclude := false

				// if there are multiple words in the query, use them all with an AND
				words := books.GetWords(v)
				switch len(words) {
				case 0:
					// no words at all, bad query
					return nil, echo.NewHTTPError(http.StatusBadRequest, "invalid search string: "+v)
				case 1:
					// just one word, make a simple constraint
					c, ex, err := books.ConstraintFromText(k, words[0])
					if err != nil {
						return nil, echo.NewHTTPError(http.StatusBadRequest, "constraint error: "+err.Error())
					}
					exclude = ex
					constraint = c
				default:
					// multiple words, build an AND constraint
					cs := make([]books.ConstraintFunctor, 0)
					for _, word := range words {
						c, ex, err := books.ConstraintFromText(k, word)
						if err != nil {
							return nil, echo.NewHTTPError(http.StatusBadRequest, "constraint error: "+err.Error())
						}
						cs = append(cs, c)
						exclude = ex
					}
					constraint = books.And(cs...)
				}
				if exclude {
					constraints.Excludes = append(constraints.Excludes, constraint)
				} else {
					constraints.Includes = append(constraints.Includes, constraint)
				}
			}
		}
	}
	return constraints, nil
}

// bookQuery does a book query based on a query specification.
// TODO: if an accept header is specified, format the result appropriately. For now we just do JSON.
func (svc *service) bookQuery(c echo.Context) error {
	constraints, err := svc.buildConstraints(c.QueryParams())
	if err != nil {
		return err
	}
	result := svc.Books.Query(constraints)
	return c.JSON(http.StatusOK, result)
}

// bookCount does a book query based on a query specification and returns the
// number of items that would result from that query.
func (svc *service) bookCount(c echo.Context) error {
	constraints, err := svc.buildConstraints(c.QueryParams())
	if err != nil {
		return err
	}
	result := svc.Books.Count(constraints)
	return c.JSON(http.StatusOK, result)
}

// bookQueryHTML does a book query based on a query specification and then
// runs the result through an HTML template.
func (svc *service) bookQueryHTML(c echo.Context) error {
	constraints, err := svc.buildConstraints(c.QueryParams())
	if err != nil {
		return err
	}
	result := svc.Books.Query(constraints)
	return c.Render(http.StatusOK, c.Param("format"), result)
}

func (svc *service) bookStats(c echo.Context) error {
	return c.JSON(http.StatusOK, svc.Books.Stats())
}

func (svc *service) bookDetails(c echo.Context) error {
	// strip off the fixed path and just take the part that matches the *
	id := c.Request().URL.Path
	if strings.HasSuffix(c.Path(), "*") {
		id = id[len(c.Path())-1:]
	}
	book, ok := svc.Books.Get(id)
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "no book found with id "+id)
	}
	return c.JSON(http.StatusOK, book)
}

// choices returns a json collection of the possibilities for several fields
// in a query:
// formats -- all the values allowed for format
// types -- all the types
// languages -- all the languages
// Note that all of these are dependent on the data actually loaded; allowed values of
// these fields may well have been restricted during loading.
func (svc *service) choices(c echo.Context) error {
	switch c.Param("field") {
	case "types", "type", "typ":
		stats := svc.Books.Stats()
		types := make([]string, 0)
		for k := range stats.Types {
			types = append(types, k)
		}
		return c.JSON(http.StatusOK, types)
	case "formats", "format", "fmt":
		stats := svc.Books.Stats()
		ctypes := make(map[string]string)
		for k, v := range books.ContentTypes {
			if _, ok := stats.Formats[v]; ok {
				ctypes[k] = v
			}
		}
		return c.JSON(http.StatusOK, ctypes)
	case "languages", "language", "lang":
		stats := svc.Books.Stats()
		langs := make([]string, 0)
		for k := range stats.Languages {
			langs = append(langs, k)
		}
		return c.JSON(http.StatusOK, langs)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "unrecognized field name")
	}
}
