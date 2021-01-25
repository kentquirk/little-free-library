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

var fullhtml = `
<!DOCTYPE html>
{{define "FULLITEM"}}
			<div class="item">
				<i>{{.title}}</i> by {{range $cr := ".creators"}}{{$.agents.cr.name}}{{end}}
			</div>
{{end}}
{{define "DOCHEAD"}}
<html>
	<head>
		<title>Little Free Library Query Results</title>
		<link rel="stylesheet"> href="/static/style.css">
	</head>
	<body>
		<div class="container">
{{end}}
{{define "DOCTAIL"}}
		</div>
	</body>
</html>
{{end}}
{{template "DOCHEAD"}}{{range .}}{{template "FULLITEM"}}{{end}}{{template "DOCTAIL"}}
`

func (svc *service) loadTemplates() {
	t := htmltmpl.Must(htmltmpl.New("full").Parse(fullhtml))
	svc.HTMLTemplates = map[string]*htmltmpl.Template{
		"full": t,
	}
}

// StaticRender implements the echo.Renderer interface so that we can render templates appropriately
// It can be used if the set of templates is predetermined.
func (svc *service) StaticRender(w io.Writer, name string, data interface{}, c echo.Context) error {
	tmpl, ok := svc.HTMLTemplates[name]
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "unknown format '%s'", name)
	}
	return tmpl.Execute(w, data)
}

// Render implements the echo.Renderer interface so that we can render templates appropriately.
// We can drop new templates into the data directory and refer to them in the request.
func (svc *service) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	name = strings.ToLower(name)
	pat := regexp.MustCompile("^[a-z0-9]{1,16}$")
	if !pat.MatchString(name) {
		return echo.NewHTTPError(http.StatusBadRequest, "bad name: "+name)
	}
	f, err := os.Open(fmt.Sprintf("./data/%s.tmpl", name))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "couldn't find template "+name)
	}
	tbody, err := ioutil.ReadAll(f)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "found, but couldn't read template "+name)
	}
	f.Close()
	tmpl, err := htmltmpl.New(name).Parse(string(tbody))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "parse failure parsing "+name+" ("+err.Error()+")")
	}
	return tmpl.Execute(w, data)
}
