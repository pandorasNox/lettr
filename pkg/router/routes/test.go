package routes

import (
	"fmt"
	"log"
	"net/http"

	"github.com/pandorasNox/lettr/pkg/router/routes/templates"
	"github.com/pandorasNox/lettr/pkg/state"
)

type TemplateDataTestPage struct{}

func GetTestPage() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		tpDate := TemplateDataTestPage{}

		err := templates.Routes.ExecuteTemplate(w, "test.html.tmpl", tpDate)
		if err != nil {
			log.Printf("error t.Execute '/' route: %s", err)
		}
	}
}

func PostIncrementHoneyTrapped(s *state.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		before := s.Metrics().HoneyTrapped()
		s.Metrics().IncreaseHoneyTrapped()
		after := s.Metrics().HoneyTrapped()

		fmt.Fprintf(w, "before='%d'\nafter='%d'\ndone\n", before, after)
	}
}
