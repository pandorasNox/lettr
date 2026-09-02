// Prototype variant 3: the shared logic becomes a method on the session type.
//
// Pure reading sketch, intentionally not compiled: the method must live in
// pkg/session where the type is nameable, and the handlers below only compile
// once that method exists. Keep this file around until one of the three
// variants is chosen, then delete the whole bin/prototypes/trylang tree.
package variant3

/*
PART 1: goes into pkg/session/session.go

func (s *session) TrySetLanguage(maybe string) (language.Language, bool) {
	if maybe == "" {
		return s.Language(), false
	}

	l, err := language.NewLang(maybe)
	if err != nil {
		log.Printf("TrySetLanguage: ignoring unknown lang %q: %s", maybe, err)
		return s.Language(), false
	}

	s.SetLanguage(l)
	return l, true
}

PART 2: pkg/router/routes/language.go and game_lettr.go

func PostLanguage(sessions *session.Sessions, wdb puzzle.WordDatabase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := session.HandleSession(w, r, sessions, wdb)

		if _, ok := s.TrySetLanguage(r.FormValue("lang")); ok {
			s.ResetGame()
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

		lang, ok := s.TrySetLanguage(r.FormValue("lang"))
		if ok {
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
*/
