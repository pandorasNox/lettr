package routes

import (
	"html/template"
	"log"
	"net/http"
)

type TemplateDataTestPage struct{}

func TestPage(t *template.Template) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		tpDate := TemplateDataTestPage{}

		err := t.ExecuteTemplate(w, "test.html.tmpl", tpDate)
		if err != nil {
			log.Printf("error t.Execute '/' route: %s", err)
		}
	}
}
