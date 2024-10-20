package routes

import (
	"context"
	"html/template"
	"log"
	"net/http"

	"github.com/pandorasNox/lettr/pkg/github"
	"github.com/pandorasNox/lettr/pkg/notification"
	"github.com/pandorasNox/lettr/pkg/routes/models"
)

func GetSuggest(t *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := t.ExecuteTemplate(w, "suggest", models.TemplateDataSuggest{})
		if err != nil {
			log.Printf("error t.ExecuteTemplate '/suggest' route: %s", err)
		}
	}
}

func PostSuggest(t *template.Template, githubToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notifier := notification.NewNotifier()

		err := r.ParseForm()
		if err != nil {
			log.Printf("error: %s", err)

			w.WriteHeader(422)
			notifier.AddError("can not parse form data")
			err = t.ExecuteTemplate(w, "oob-messages", notifier.ToTemplate())
			if err != nil {
				log.Printf("error t.ExecuteTemplate 'oob-messages': %s", err)
			}
			return
		}

		form := r.PostForm
		tds := models.TemplateDataSuggest{
			Word:     form["word"][0],
			Message:  form["message"][0],
			Language: form["language-pick"][0],
			Action:   form["suggest-action"][0],
		}

		err = tds.Validate()
		if err != nil {
			w.WriteHeader(422)
			w.Header().Add("HX-Reswap", "none")

			notifier.AddError(err.Error())
			err = t.ExecuteTemplate(w, "oob-messages", notifier.ToTemplate())
			if err != nil {
				log.Printf("error t.ExecuteTemplate 'oob-messages' route: %s", err)
			}

			return
		}

		err = github.CreateWordSuggestionIssue(
			context.Background(), githubToken, tds.Word, tds.Language, tds.Action, tds.Message,
		)
		if err != nil {
			w.WriteHeader(422)
			w.Header().Add("HX-Reswap", "none")

			notifier.AddError("Could not send suggestion.")
			err = t.ExecuteTemplate(w, "oob-messages", notifier.ToTemplate())
			if err != nil {
				log.Printf("error t.ExecuteTemplate 'oob-messages' route: %s", err)
			}

			return
		}

		notifier.AddSuccess("Suggestion send, thank you!")
		err = t.ExecuteTemplate(w, "oob-messages", notifier.ToTemplate())
		if err != nil {
			log.Printf("error t.ExecuteTemplate 'oob-messages' route: %s", err)
		}

		err = t.ExecuteTemplate(w, "suggest", models.TemplateDataSuggest{})
		if err != nil {
			log.Printf("error t.ExecuteTemplate '/suggest' route: %s", err)
		}
	}
}
