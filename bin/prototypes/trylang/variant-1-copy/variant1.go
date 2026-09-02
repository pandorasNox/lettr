// Prototype variant 1: no shared helper, the lang snippet is copied into each handler.
//
// Note: s.SetGameState(puzzle.GameState{}) stands in for a session.ResetGame()
// method that the real implementation would add (see the shared-understanding
// notes on game invalidation). The prototype must compile before that method
// exists, hence the existing setter.
package variant1

import (
	"log"
	"net/http"

	"github.com/pandorasNox/lettr/pkg/language"
	"github.com/pandorasNox/lettr/pkg/puzzle"
	"github.com/pandorasNox/lettr/pkg/router/routes/models/shared"
	"github.com/pandorasNox/lettr/pkg/router/routes/templates"
	"github.com/pandorasNox/lettr/pkg/session"
)

func PostLanguage(sessions *session.Sessions, wdb puzzle.WordDatabase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := session.HandleSession(w, r, sessions, wdb)

		if maybeLang := r.FormValue("lang"); maybeLang != "" {
			if l, err := language.NewLang(maybeLang); err != nil {
				log.Printf("PostLanguage: ignoring unknown lang %q: %s", maybeLang, err)
			} else {
				s.SetLanguage(l)
				s.SetGameState(puzzle.GameState{}) // final code: s.ResetGame()
			}
		}

		sessions.UpdateOrSet(s)

		tData := struct{ Language language.Language }{Language: s.Language()}
		if err := templates.Routes.ExecuteTemplate(w, "oob-lang-switch", tData); err != nil {
			log.Printf("PostLanguage: render oob-lang-switch: %s", err)
		}
	}
}

func PostGameLettrNew(sessions *session.Sessions, wdb puzzle.WordDatabase, imprintUrl, revision, faviconPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := session.HandleSession(w, r, sessions, wdb)

		if maybeLang := r.FormValue("lang"); maybeLang != "" {
			if l, err := language.NewLang(maybeLang); err != nil {
				log.Printf("PostGameLettrNew: ignoring unknown lang %q: %s", maybeLang, err)
			} else {
				s.SetLanguage(l)

				tData := struct{ Language language.Language }{Language: l}
				if err := templates.Routes.ExecuteTemplate(w, "oob-lang-switch", tData); err != nil {
					log.Printf("PostGameLettrNew: render oob-lang-switch: %s", err)
				}
			}
		}

		s.AddPastWord(s.GameState().ActiveSolutionWord())
		s.NewGame(s.Language(), wdb)
		sessions.UpdateOrSet(s)

		fData := shared.TemplateDataLettr{}.New(s.Language(), puzzle.Puzzle{}, s.GameState().LetterHints(), s.PastWords(), imprintUrl, revision, faviconPath)
		if err := templates.Routes.ExecuteTemplate(w, "lettr-form", fData); err != nil {
			log.Printf("PostGameLettrNew: render lettr-form: %s", err)
		}
	}
}
