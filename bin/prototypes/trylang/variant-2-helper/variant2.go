// Prototype variant 2: one private helper both handlers call.
//
// The session type in pkg/session is unexported, so a helper living in the
// routes package cannot name it as a parameter type. The minimal interface
// below is the consumer-side workaround; it names only the methods needed.
//
// Note: s.SetGameState(puzzle.GameState{}) stands in for a session.ResetGame()
// method the real implementation would add; the prototype must compile before
// that method exists.
package variant2

import (
	"log"
	"net/http"

	"github.com/pandorasNox/lettr/pkg/language"
	"github.com/pandorasNox/lettr/pkg/puzzle"
	"github.com/pandorasNox/lettr/pkg/router/routes/models/shared"
	"github.com/pandorasNox/lettr/pkg/router/routes/templates"
	"github.com/pandorasNox/lettr/pkg/session"
)

type langSwitcher interface {
	SetLanguage(l language.Language)
}

// applyLang parses the optional `lang` form value. Known language: switches
// the session and reports it. Empty or unknown: ignored. Renders nothing,
// the caller renders oob-lang-switch.
//
// HandleSession returns the session struct by value and SetLanguage has a
// pointer receiver, so callers pass &s.
func applyLang(s langSwitcher, maybeLang string) (language.Language, bool) {
	if maybeLang == "" {
		return "", false
	}

	l, err := language.NewLang(maybeLang)
	if err != nil {
		log.Printf("applyLang: ignoring unknown lang %q: %s", maybeLang, err)
		return "", false
	}

	s.SetLanguage(l)
	return l, true
}

func PostLanguage(sessions *session.Sessions, wdb puzzle.WordDatabase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := session.HandleSession(w, r, sessions, wdb)

		if _, ok := applyLang(&s, r.FormValue("lang")); ok {
			s.SetGameState(puzzle.GameState{}) // final code: s.ResetGame()
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

		lang, switched := applyLang(&s, r.FormValue("lang"))
		if switched {
			tData := struct{ Language language.Language }{Language: lang}
			if err := templates.Routes.ExecuteTemplate(w, "oob-lang-switch", tData); err != nil {
				log.Printf("PostGameLettrNew: render oob-lang-switch: %s", err)
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
