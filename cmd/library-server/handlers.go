package main

import (
	"net/http"
	"strconv"

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

type service struct {
	Config config
	Books  *books.BookData
}

func newService() *service {
	return &service{
		Books: books.NewBookData(),
	}
}

func (svc *service) setupRoutes(e *echo.Echo) {
	// Routes
	e.GET("/", svc.err400)
	e.GET("/doc", svc.doc)
	e.GET("/health", svc.health)
	e.GET("/books/query", svc.bookQuery)
	e.GET("/books/summary", svc.bookSummary)
	e.GET("/details/:id", svc.bookDetails)
	e.GET("/qr/:id", svc.qrcodegen)
	e.GET("/book/:id", svc.bookByID)
}

// err400 returns 400 and is used to discourage random queries
func (svc *service) err400(c echo.Context) error {
	return c.String(http.StatusBadRequest, "Go away.")
}

// doc returns a documentation page
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

func (svc *service) bookDetails(c echo.Context) error {
	return c.String(http.StatusOK, "Ok\n")
}

// bookQuery does a book query based on a query specification.
func (svc *service) bookQuery(c echo.Context) error {
	values := c.QueryParams()
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
				// TODO: read max limit from config
				if n > 0 && n < 100 {
					constraints.Limit = n
				}
			case "page", "pg":
				n, _ := strconv.Atoi(v)
				constraints.Page = n
			default:
				constraint, exclude, err := books.ConstraintFromText(k, v)
				if err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, "constraint error: %e", err)
				}
				if exclude {
					constraints.Excludes = append(constraints.Excludes, constraint)
				} else {
					constraints.Includes = append(constraints.Includes, constraint)
				}
			}
		}
	}
	// ok, we have a constraint spec -- execute it
	result := svc.Books.Query(constraints)
	return c.JSON(http.StatusOK, result)
}

func (svc *service) bookSummary(c echo.Context) error {
	return c.JSON(http.StatusOK, svc.Books.Summary())
}

func (svc *service) bookByID(c echo.Context) error {
	return c.String(http.StatusOK, "Ok\n")
}
