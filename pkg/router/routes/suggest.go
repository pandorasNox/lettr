package routes

import (
	"context"
	"log"
	"net/http"

	"github.com/pandorasNox/lettr/pkg/github"
	"github.com/pandorasNox/lettr/pkg/notification"
	"github.com/pandorasNox/lettr/pkg/router/routes/models"
	"github.com/pandorasNox/lettr/pkg/router/routes/templates"
)

func GetSuggest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := templates.Routes.ExecuteTemplate(w, "suggest", models.TemplateDataSuggest{})
		if err != nil {
			log.Printf("error t.ExecuteTemplate '/suggest' route: %s", err)
		}
	}
}

func PostSuggest(githubToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notifier := notification.NewNotifier()

		err := r.ParseForm()
		if err != nil {
			log.Printf("error: %s", err)

			w.WriteHeader(422)
			notifier.AddError("can not parse form data")
			err = templates.Routes.ExecuteTemplate(w, "oob-messages", notifier.ToTemplate())
			if err != nil {
				log.Printf("error t.ExecuteTemplate 'oob-messages': %s", err)
			}
			return
		}

		form := r.PostForm
		tds := models.TemplateDataSuggest{
			Word:     form["word"][0],
			Message:  form["123456789"][0],
			Language: form["language-pick"][0],
			Action:   form["suggest-action"][0],
		}

		isHoneypotFilled := len(form["message"][0]) != 0
		if isHoneypotFilled {
			createSuccessResponse(w, &notifier)
			return
		}

		err = tds.Validate()
		if err != nil {
			w.WriteHeader(422)
			w.Header().Add("HX-Reswap", "none")

			notifier.AddError(err.Error())
			err = templates.Routes.ExecuteTemplate(w, "oob-messages", notifier.ToTemplate())
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
			err = templates.Routes.ExecuteTemplate(w, "oob-messages", notifier.ToTemplate())
			if err != nil {
				log.Printf("error t.ExecuteTemplate 'oob-messages' route: %s", err)
			}

			return
		}

		createSuccessResponse(w, &notifier)
	}
}

func createSuccessResponse(w http.ResponseWriter, n *notification.Notifier) {
	n.AddSuccess("Suggestion send, thank you!")
	err := templates.Routes.ExecuteTemplate(w, "oob-messages", n.ToTemplate())
	if err != nil {
		log.Printf("error t.ExecuteTemplate 'oob-messages' route: %s", err)
	}

	err = templates.Routes.ExecuteTemplate(w, "suggest", models.TemplateDataSuggest{})
	if err != nil {
		log.Printf("error t.ExecuteTemplate '/suggest' route: %s", err)
	}
}
