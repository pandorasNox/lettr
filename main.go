package main

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"strings"
	"unicode"

	iofs "io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/pandorasNox/lettr/pkg/language"
	"github.com/pandorasNox/lettr/pkg/middleware"
	"github.com/pandorasNox/lettr/pkg/notification"
	"github.com/pandorasNox/lettr/pkg/puzzle"
	"github.com/pandorasNox/lettr/pkg/routes"
	"github.com/pandorasNox/lettr/pkg/session"
)

var Revision = "0000000"
var FaviconPath = "/static/assets/favicon"

//go:embed configs/*.txt
//go:embed templates/*.html.tmpl
//go:embed templates/**/*.html.tmpl
//go:embed web/static/assets/*
//go:embed web/static/generated/*.js
//go:embed web/static/generated/*.css
var fs embed.FS

var ErrNotInWordList = errors.New("not in wordlist")

type env struct {
	port        string
	githubToken string
	imprintUrl  string
}

func (e env) String() string {
	s := fmt.Sprintf("port: %s", e.port)

	if e.githubToken != "" {
		s = fmt.Sprintf("%s\ngithub token (length): %d", s, len(e.githubToken))
	}

	if e.imprintUrl != "" {
		s = fmt.Sprintf("%s\nimprint: %s", s, e.imprintUrl)
	}
	// s = s + fmt.Sprintf("foo: %s\n", e.port)
	return s
}

func NewLang(maybeLang string) (language.Language, error) {
	switch language.Language(maybeLang) {
	case language.LANG_EN, language.LANG_DE:
		return language.Language(maybeLang), nil
	default:
		return language.LANG_EN, fmt.Errorf("couldn't create new language from given value: '%s'", maybeLang)
	}
}

// inspiration see: https://forum.golangbridge.org/t/can-i-use-enum-in-template/25296
var funcMap = template.FuncMap{
	"IsMatchVague": puzzle.MatchVague.Is,
	"IsMatchNone":  puzzle.MatchNone.Is,
	"IsMatchExact": puzzle.MatchExact.Is,
}

type TemplateDataForm struct {
	Data                        puzzle.Puzzle
	Errors                      map[string]string
	IsSolved                    bool
	IsLoose                     bool
	JSCachePurgeTimestamp       int64
	Language                    language.Language
	Revision                    string
	FaviconPath                 string
	Keyboard                    keyboard
	PastWords                   []puzzle.Word
	SolutionHasDublicateLetters bool
	ImprintUrl                  string
}

func (fd TemplateDataForm) New(l language.Language, p puzzle.Puzzle, pastWords []puzzle.Word, solutionHasDublicateLetters bool, imprintUrl string) TemplateDataForm {
	kb := keyboard{}
	kb.Init(l, p.LetterGuesses())

	return TemplateDataForm{
		Data:                        p,
		Errors:                      make(map[string]string),
		JSCachePurgeTimestamp:       time.Now().Unix(),
		Language:                    l,
		Revision:                    Revision,
		FaviconPath:                 FaviconPath,
		Keyboard:                    kb,
		PastWords:                   pastWords,
		SolutionHasDublicateLetters: solutionHasDublicateLetters,
		ImprintUrl:                  imprintUrl,
	}
}

type keyboard struct {
	KeyGrid [][]keyboardKey
}

func (k *keyboard) Init(l language.Language, lgs []puzzle.LetterGuess) {
	k.KeyGrid = [][]keyboardKey{
		{{"Q", false, puzzle.MatchNone}, {"W", false, puzzle.MatchNone}, {"E", false, puzzle.MatchNone}, {"R", false, puzzle.MatchNone}, {"T", false, puzzle.MatchNone}, {"Y", false, puzzle.MatchNone}, {"U", false, puzzle.MatchNone}, {"I", false, puzzle.MatchNone}, {"O", false, puzzle.MatchNone}, {"P", false, puzzle.MatchNone}, {"Delete", false, puzzle.MatchNone}},
		{{"A", false, puzzle.MatchNone}, {"S", false, puzzle.MatchNone}, {"D", false, puzzle.MatchNone}, {"F", false, puzzle.MatchNone}, {"G", false, puzzle.MatchNone}, {"H", false, puzzle.MatchNone}, {"J", false, puzzle.MatchNone}, {"K", false, puzzle.MatchNone}, {"L", false, puzzle.MatchNone}, {"Enter", false, puzzle.MatchNone}},
		{{"Z", false, puzzle.MatchNone}, {"X", false, puzzle.MatchNone}, {"C", false, puzzle.MatchNone}, {"V", false, puzzle.MatchNone}, {"B", false, puzzle.MatchNone}, {"N", false, puzzle.MatchNone}, {"M", false, puzzle.MatchNone}},
	}

	for ri, r := range k.KeyGrid {
	KeyLoop:
		for ki, kk := range r {
			for _, lg := range lgs {
				if kk.Key == "Enter" || kk.Key == "Delete" {
					continue KeyLoop
				}

				KeyR := firstRune(kk.Key)
				betterMatch := (k.KeyGrid[ri][ki].Match == puzzle.MatchNone) ||
					(k.KeyGrid[ri][ki].Match == puzzle.MatchVague && lg.Match == puzzle.MatchExact)

				if lg.Letter == unicode.ToLower(KeyR) && betterMatch {
					k.KeyGrid[ri][ki].IsUsed = true
					k.KeyGrid[ri][ki].Match = lg.Match
				}
			}
		}
	}
}

func firstRune(s string) rune {
	for _, r := range s {
		return r
	}

	return 0
}

type keyboardKey struct {
	Key    string
	IsUsed bool
	Match  puzzle.Match
}

func Map[T, U any](ts []T, f func(T) U) []U {
	us := make([]U, len(ts))
	for i := range ts {
		us[i] = f(ts[i])
	}
	return us
}

func main() {
	log.Println("staring server...")

	envCfg := envConfig()
	sessions := session.NewSessions()

	wordDb := puzzle.WordDatabase{}
	err := wordDb.Init(fs, puzzle.FilePathsByLang())
	if err != nil {
		log.Fatalf("init wordDatabase failed: %s", err)
	}

	log.Printf("env conf:\n%s", envCfg)

	// t := template.Must(template.ParseFS(fs, "templates/index.html.tmpl", "templates/lettr-form.html.tmpl"))
	// log.Printf("template name: %s", t.Name())
	t := template.Must(template.New("index.html.tmpl").Funcs(funcMap).ParseFS(
		fs,
		"templates/index.html.tmpl",
		"templates/lettr-form.html.tmpl",
		"templates/help.html.tmpl",
		"templates/suggest.html.tmpl",
		"templates/pages/test.html.tmpl",
	))

	staticFS, err := iofs.Sub(fs, "web/static")
	if err != nil {
		log.Fatalf("subtree for 'static' dir of embed fs failed: %s", err) //TODO
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /static/", routes.Static(staticFS))

	mux.HandleFunc("GET /", func(w http.ResponseWriter, req *http.Request) {
		sess := session.HandleSession(w, req, &sessions, wordDb)

		p := sess.GameState().LastEvaluatedAttempt()
		sessions.UpdateOrSet(sess)

		fData := TemplateDataForm{}.New(sess.Language(), p, sess.PastWords(), sess.GameState().ActiveSolutionWord().HasDublicateLetters(), envCfg.imprintUrl)
		fData.IsSolved = p.IsSolved()
		fData.IsLoose = p.IsLoose()

		err := t.ExecuteTemplate(w, "index.html.tmpl", fData)
		if err != nil {
			log.Printf("error t.Execute '/' route: %s", err)
		}
	})

	mux.HandleFunc("GET /test", routes.TestPage(t))

	mux.HandleFunc("GET /letter-hint", routes.LetterHint(t, &sessions, wordDb))

	mux.HandleFunc("GET /lettr", func(w http.ResponseWriter, r *http.Request) {
		s := session.HandleSession(w, r, &sessions, wordDb)
		sessions.UpdateOrSet(s)

		p := s.GameState().LastEvaluatedAttempt()

		fData := TemplateDataForm{}.New(
			s.Language(),
			p,
			s.PastWords(),
			s.GameState().ActiveSolutionWord().HasDublicateLetters(),
			envCfg.imprintUrl,
		)
		fData.IsSolved = p.IsSolved()
		fData.IsLoose = p.IsLoose()

		err = t.ExecuteTemplate(w, "lettr-form", fData)
		if err != nil {
			log.Printf("error t.ExecuteTemplate '/lettr' route: %s", err)
		}
	})

	mux.HandleFunc("POST /lettr", func(w http.ResponseWriter, r *http.Request) {
		s := session.HandleSession(w, r, &sessions, wordDb)
		notifier := notification.NewNotifier()

		// b, err := io.ReadAll(r.Body)
		// if err != nil {
		// 	// log.Fatalln(err)
		// 	log.Printf("error: %s", err)
		// }
		// log.Printf("word: %s\nbody:\n%s", s.activeWord, b)

		err := r.ParseForm()
		if err != nil {
			log.Printf("error: %s", err)

			w.WriteHeader(422)
			notifier.AddError("cannot parse form data")
			err = t.ExecuteTemplate(w, "oob-messages", notifier.ToTemplate())
			if err != nil {
				log.Printf("error t.ExecuteTemplate 'oob-messages': %s", err)
			}
			return
		}

		g := s.GameState()
		p := g.LastEvaluatedAttempt()

		if p.IsSolved() || p.IsLoose() {
			w.WriteHeader(204)
			return
		}

		if p.ActiveRow() != countFilledFormRows(r.PostForm)-1 {
			w.WriteHeader(422)
			notifier.AddError("faked rows")
			err = t.ExecuteTemplate(w, "oob-messages", notifier.ToTemplate())
			if err != nil {
				log.Printf("error t.ExecuteTemplate 'oob-messages': %s", err)
			}
			return
		}

		p, err = parseForm(p, r.PostForm, g.ActiveSolutionWord(), s.Language(), wordDb)
		if err == ErrNotInWordList {
			w.WriteHeader(422)
			notifier.AddError("word not in word list")
			err = t.ExecuteTemplate(w, "oob-messages", notifier.ToTemplate())
			if err != nil {
				log.Printf("error t.ExecuteTemplate 'oob-messages': %s", err)
			}
			return
		}

		g.SetLastEvaluatedAttempt(p)
		s.SetGameState(*g) //todo move gamestate from pointer to copy
		sessions.UpdateOrSet(s)

		fData := TemplateDataForm{}.New(s.Language(), p, s.PastWords(), g.ActiveSolutionWord().HasDublicateLetters(), envCfg.imprintUrl)
		fData.IsSolved = p.IsSolved()
		fData.IsLoose = p.IsLoose()

		err = t.ExecuteTemplate(w, "lettr-form", fData)
		if err != nil {
			log.Printf("error t.ExecuteTemplate '/lettr' route: %s", err)
		}
	})

	mux.HandleFunc("POST /new", func(w http.ResponseWriter, r *http.Request) {
		s := session.HandleSession(w, r, &sessions, wordDb)

		// handle lang switch
		l := s.Language()
		maybeLang := r.FormValue("lang")
		if maybeLang != "" {
			l, _ = NewLang(maybeLang)
			s.SetLanguage(l)

			type TemplateDataLanguge struct {
				Language language.Language
			}
			tData := TemplateDataLanguge{Language: l}

			err := t.ExecuteTemplate(w, "oob-lang-switch", tData)
			if err != nil {
				log.Printf("error t.ExecuteTemplate '/new' route: %s", err)
			}
		}

		p := puzzle.Puzzle{}

		s.SetLastEvaluatedAttempt(p)
		s.AddPastWord(s.ActiveSolutionWord())
		s.SetActiveSolutionWord(wordDb.RandomPickWithFallback(l, s.PastWords(), 0))
		s.NewGame(l, wordDb)
		sessions.UpdateOrSet(s)

		fData := TemplateDataForm{}.New(s.Language(), p, s.PastWords(), s.ActiveSolutionWord().HasDublicateLetters(), envCfg.imprintUrl)
		fData.IsSolved = p.IsSolved()
		fData.IsLoose = p.IsLoose()

		// w.Header().Add("HX-Refresh", "true")
		err := t.ExecuteTemplate(w, "lettr-form", fData)
		if err != nil {
			log.Printf("error t.ExecuteTemplate '/new' route: %s", err)
		}
	})

	mux.HandleFunc("POST /help", routes.Help(t, &sessions, wordDb))

	mux.HandleFunc("GET /suggest", routes.GetSuggest(t))

	mux.HandleFunc("POST /suggest", routes.PostSuggest(t, envCfg.githubToken))

	middlewares := []func(h http.Handler) http.Handler{
		func(h http.Handler) http.Handler {
			return middleware.NewRequestSize(h, 32*1024 /* 32kiB */)
		},
		func(h http.Handler) http.Handler {
			return middleware.NewBodySize(h, 32*1024 /* 32kiB */)
		},
	}

	var muxWithMiddlewares http.Handler = mux
	for _, fm := range middlewares {
		muxWithMiddlewares = fm(muxWithMiddlewares)
	}

	// v1 := http.NewServeMux()
	// v1.Handle("/v1/", http.StripPrefix("/v1", muxWithMiddlewares))

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", envCfg.port), muxWithMiddlewares))
}

func envConfig() env {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		panic("PORT not provided")
	}

	gt, ok := os.LookupEnv("GITHUB_TOKEN")
	if !ok {
		log.Printf("(optional) environment variable GITHUB_TOKEN not set")
	}

	imprintUrl, ok := os.LookupEnv("IMPRINT_URL")
	if !ok {
		log.Printf("(optional) environment variable IMPRINT_URL not set")
	}

	return env{port: port, githubToken: gt, imprintUrl: imprintUrl}
}

func countFilledFormRows(postPuzzleForm url.Values) uint8 {
	isfilled := func(row []string) bool {
		emptyButWithLen := make([]string, len(row)) // we need empty slice but with right elem length
		return !(slices.Compare(row, emptyButWithLen) == 0)
	}

	var count uint8 = 0
	l := len(postPuzzleForm)
	for i := 0; i < l; i++ {
		guessedWord, ok := postPuzzleForm[fmt.Sprintf("r%d", i)]
		if ok && isfilled(guessedWord) {
			count++
		}
	}

	return count
}

func parseForm(p puzzle.Puzzle, form url.Values, solutionWord puzzle.Word, l language.Language, wdb puzzle.WordDatabase) (puzzle.Puzzle, error) {
	for ri := range p.Guesses {
		maybeGuessedWord, ok := form[fmt.Sprintf("r%d", ri)]
		if !ok {
			continue
		}

		guessedWord, err := sliceToWord(maybeGuessedWord)
		if err != nil {
			return p, fmt.Errorf("parseForm could not create guessedWord from form input: %s", err.Error())
		}

		if !wdb.Exists(l, guessedWord) {
			return p, ErrNotInWordList
		}

		wg := evaluateGuessedWord(guessedWord, solutionWord)

		p.Guesses[ri] = wg
	}

	return p, nil
}

func sliceToWord(maybeGuessedWord []string) (puzzle.Word, error) {
	w := puzzle.Word{}

	if len(maybeGuessedWord) != len(w) {
		return puzzle.Word{}, fmt.Errorf("sliceToWord: provided slice does not match word length")
	}

	for i, l := range maybeGuessedWord {
		w[i], _ = utf8.DecodeRuneInString(strings.ToLower(l))
		if w[i] == 65533 {
			w[i] = 0
		}
	}

	return w, nil
}

func evaluateGuessedWord(guessedWord puzzle.Word, solutionWord puzzle.Word) puzzle.WordGuess {
	solutionWord = solutionWord.ToLower()
	guessedLetterCountMap := make(map[rune]int)

	resultWordGuess := puzzle.WordGuess{}

	// initilize
	for i, gr := range guessedWord {
		resultWordGuess[i].Letter = gr
		resultWordGuess[i].Match = puzzle.MatchNone
	}

	// mark exact matches
	for i, gr := range guessedWord {
		exact := solutionWord[i] == gr

		if exact {
			guessedLetterCountMap[gr]++
			resultWordGuess[i].Match = puzzle.MatchExact
		}
	}

	// mark some/vague matches
	for i, gr := range guessedWord {
		if resultWordGuess[i].Match == puzzle.MatchExact {
			continue
		}

		some := solutionWord.Contains(gr)

		if !(resultWordGuess[i].Match == puzzle.MatchVague) || some {
			guessedLetterCountMap[gr]++
		}

		s := some && (guessedLetterCountMap[gr] <= solutionWord.Count(gr))
		if s {
			resultWordGuess[i].Match = puzzle.MatchVague
		}
	}

	return resultWordGuess
}
