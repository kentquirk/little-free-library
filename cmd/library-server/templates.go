package main

import (
	"fmt"
	htmltmpl "html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"
)

// Render implements the echo.Renderer interface so that we can render templates appropriately.
// We can drop new templates into the data directory and refer to them in the request.
// If you wish to have only a static list of templates, use the "embed" package now included with Go 1.16.
func (svc *service) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	name = strings.ToLower(name)
	pat := regexp.MustCompile("^[a-z0-9]{1,16}$")
	if !pat.MatchString(name) {
		return echo.NewHTTPError(http.StatusBadRequest, "bad name: "+name)
	}

	tmpl, ok := svc.HTMLTemplates[name]
	if !ok || svc.Config.NoCacheTemplates {
		f, err := os.Open(fmt.Sprintf("./data/%s.tmpl", name))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "couldn't find template "+name)
		}
		tbody, err := ioutil.ReadAll(f)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "found, but couldn't read template "+name)
		}
		f.Close()
		tmpl, err = htmltmpl.New(name).Parse(string(tbody))
		svc.HTMLTemplates[name] = tmpl
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "parse failure parsing "+name+" ("+err.Error()+")")
		}
	}
	return tmpl.Execute(w, data)
}
