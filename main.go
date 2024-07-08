package main

import (
	"bufio"
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"math/rand"
	"strings"
	"sync"
	"unicode"

	iofs "io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/pandorasNox/lettr/pkg/github"
	"github.com/pandorasNox/lettr/pkg/middleware"
)

var Revision = "0000000"
var FaviconPath = "/static/assets/favicon"

const SESSION_COOKIE_NAME = "session"
const SESSION_MAX_AGE_IN_SECONDS = 24 * 60 * 60

//go:embed configs/*.txt
//go:embed templates/*.html.tmpl
//go:embed web/static/assets/*
//go:embed web/static/generated/*.js
//go:embed web/static/generated/*.css
var fs embed.FS

var ErrNotInWordList = errors.New("not in wordlist")

type env struct {
	port        string
	githubToken string
}

func (e env) String() string {
	s := fmt.Sprintf("port: %s\ngithub token (length): %d\n", e.port, len(e.githubToken))
	// s = s + fmt.Sprintf("foo: %s\n", e.port)
	return s
}

type counterState struct {
	mu    sync.Mutex
	count int
}

type session struct {
	// todo: think about using mutex or channel for rw session
	id                   string
	expiresAt            time.Time
	maxAgeSeconds        int
	language             language
	activeSolutionWord   word
	lastEvaluatedAttempt puzzle
	pastWords            []word
}

func (s *session) AddPastWord(w word) {
	s.pastWords = append(s.pastWords, w)
}

func (s *session) PastWords() []word {
	return slices.Clone(s.pastWords)
}

type sessions []session

func (ss sessions) String() string {
	out := ""
	for _, s := range ss {
		out = out + s.id + " " + s.expiresAt.String() + "\n"
	}

	return out
}

func (ss *sessions) updateOrSet(sess session) {
	index := slices.IndexFunc((*ss), func(s session) bool {
		return s.id == sess.id
	})
	if index == -1 {
		*ss = append(*ss, sess)
		return
	}

	(*ss)[index] = sess
}

type language string

func NewLang(maybeLang string) (language, error) {
	switch language(maybeLang) {
	case LANG_EN, LANG_DE:
		return language(maybeLang), nil
	default:
		return LANG_EN, fmt.Errorf("couldn't create new language from given value: '%s'", maybeLang)
	}
}

const (
	LANG_EN language = "en"
	LANG_DE language = "de"
)

type word [5]rune

func (w word) String() string {
	out := ""
	for _, v := range w {
		out += string(v)
	}

	return out
}

func (w word) contains(letter rune) bool {
	found := false
	for _, v := range w {
		if v == letter {
			found = true
			break
		}
	}

	return found
}

func (w word) count(letter rune) int {
	count := 0
	for _, v := range w {
		if v == letter {
			count++
		}
	}

	return count
}

func (w word) isEqual(compare word) bool {
	for i, v := range w {
		if v != compare[i] {
			return false
		}
	}

	return true
}

func (w word) hasDublicateLetters() bool {
	for _, l := range w {
		if w.count(l) >= 2 {
			return true
		}
	}

	return false
}

func (w word) ToLower() word {
	for i, v := range w {
		w[i] = unicode.ToLower(v)
	}

	return w
}

func toWord(wo string) (word, error) {
	out := word{}

	length := 0
	for i, l := range wo {
		length++
		if length > len(out) {
			return word{}, fmt.Errorf("string does not match allowed word length: length=%d, expectedLength=%d", length, len(out))
		}

		out[i] = l
	}

	if length < len(out) {
		return word{}, fmt.Errorf("string is to short: length=%d, expectedLength=%d", length, len(out))
	}

	return out, nil
}

type puzzle struct {
	Debug   string
	Guesses [6]wordGuess
}

func (p puzzle) activeRow() uint8 {
	for i, wg := range p.Guesses {
		if !wg.isFilled() {
			return uint8(i)
		}
	}

	return uint8(len(p.Guesses))
}

func (p puzzle) isSolved() bool {
	if p.activeRow() > 0 {
		return p.Guesses[p.activeRow()-1].isSolved()
	}

	return false
}

func (p puzzle) isLoose() bool {
	for _, wg := range p.Guesses {
		if !wg.isFilled() || wg.isSolved() {
			return false
		}
	}

	return true
}

func (p puzzle) letterGuesses() []letterGuess {
	lgCollector := []letterGuess{}

	for _, wg := range p.Guesses {
		if wg.isFilled() {
			lgCollector = append(lgCollector, wg.letterGuesses()...)
		}
	}

	return lgCollector
}

type wordGuess [5]letterGuess

func (wg wordGuess) isFilled() bool {
	for _, l := range wg {
		if l.Letter == 0 || l.Letter == 65533 {
			return false
		}
	}

	return true
}

func (wg wordGuess) isSolved() bool {
	for _, lg := range wg {
		if lg.Match != MatchExact {
			return false
		}
	}

	return true
}

func (wg wordGuess) letterGuesses() []letterGuess {
	s := []letterGuess{}

	if !wg.isFilled() {
		return s
	}

	for _, lg := range wg {
		s = append(s, lg)
	}

	return s
}

type letterGuess struct {
	Letter rune
	Match  match
}

type match uint

// inspiration see: https://forum.golangbridge.org/t/can-i-use-enum-in-template/25296
func (m match) is(mIn match) bool { return m == mIn }

// inspiration see: https://forum.golangbridge.org/t/can-i-use-enum-in-template/25296
var funcMap = template.FuncMap{
	"IsMatchVague": MatchVague.is,
	"IsMatchNone":  MatchNone.is,
	"IsMatchExact": MatchExact.is,
}

const (
	MatchNone match = iota + 1
	MatchVague
	MatchExact
)

type TemplateDataForm struct {
	Data                        puzzle
	Errors                      map[string]string
	IsSolved                    bool
	IsLoose                     bool
	JSCachePurgeTimestamp       int64
	Language                    language
	Revision                    string
	FaviconPath                 string
	Keyboard                    keyboard
	PastWords                   []word
	SolutionHasDublicateLetters bool
}

func (fd TemplateDataForm) New(l language, p puzzle, pastWords []word, SolutionHasDublicateLetters bool) TemplateDataForm {
	kb := keyboard{}
	kb.Init(l, p.letterGuesses())

	return TemplateDataForm{
		Data:                        p,
		Errors:                      make(map[string]string),
		JSCachePurgeTimestamp:       time.Now().Unix(),
		Language:                    l,
		Revision:                    Revision,
		FaviconPath:                 FaviconPath,
		Keyboard:                    kb,
		PastWords:                   pastWords,
		SolutionHasDublicateLetters: SolutionHasDublicateLetters,
	}
}

type wordCollection string

const (
	WC_ALL    wordCollection = "wc_all"
	WC_COMMON wordCollection = "wc_common"
)

type wordDatabase struct {
	db map[language]map[wordCollection]map[word]bool
}

func (wdb *wordDatabase) Init(fs iofs.FS, filePathsByLanguage map[language]map[wordCollection][]string) error {
	wdb.db = make(map[language]map[wordCollection]map[word]bool)

	for l, collection := range filePathsByLanguage {
		wdb.db[l] = make(map[wordCollection]map[word]bool)
		for c, paths := range collection {
			wdb.db[l][c] = make(map[word]bool)

			for _, path := range paths {
				f, err := fs.Open(path)
				if err != nil {
					return fmt.Errorf("wordDatabase init failed when opening file: %s", err)
				}
				defer f.Close()

				fInfo, err := f.Stat()
				if err != nil {
					return fmt.Errorf("wordDatabase init failed when obtaining stat: %s", err)
				}

				var allowedSize int64 = 2 * 1024 * 1024 // 2 MB
				if fInfo.Size() > allowedSize {
					return fmt.Errorf("wordDatabase init failed with forbidden file size: path='%s', size='%d'", path, fInfo.Size())
				}

				scanner := bufio.NewScanner(f)
				var line int = 0
				for scanner.Scan() {
					if line == 0 { // skip first metadata line
						line++
						continue
					}

					candidate := scanner.Text()
					word, err := toWord(candidate)
					if err != nil {
						return fmt.Errorf("wordDatabase init, couldn't parse line to word: line='%s', err=%s", candidate, err)
					}

					wdb.db[l][c][word.ToLower()] = true

					line++
				}
				if err := scanner.Err(); err != nil {
					return fmt.Errorf("wordDatabase init failed scanning file with: path='%s', err=%s", path, err)
				}
			}
		}
	}

	return nil
}

func (wdb wordDatabase) Exists(l language, w word) bool {
	db, ok := wdb.db[l]
	if !ok {
		return false
	}

	db_c, ok := db[WC_ALL]
	if !ok {
		return false
	}

	_, ok = db_c[w.ToLower()]
	return ok
}

func (wdb wordDatabase) RandomPick(l language, avoidList []word, retryAkkumulator uint8) (word, error) {
	const MAX_RETRY uint8 = 10

	if retryAkkumulator > MAX_RETRY {
		return word{}, fmt.Errorf("RandomPick exceeded retries: retryAkkumulator='%d' | MAX_RETRY='%d'", retryAkkumulator, MAX_RETRY)
	}

	db, ok := wdb.db[l]
	if !ok {
		return word{}, fmt.Errorf("RandomPick failed with unknown language: '%s'", l)
	}

	collection := WC_COMMON
	db_c, ok := db[collection]
	if !ok {
		collection = WC_ALL

		db_c, ok = db[collection]
		if !ok {
			return word{}, fmt.Errorf("RandomPick with lang '%s' failed with unknown collection: '%s'", l, collection)
		}
	}

	randsource := rand.NewSource(time.Now().UnixNano())
	randgenerator := rand.New(randsource)
	rolledLine := randgenerator.Intn(len(db_c))

	currentLine := 0
	for w := range db_c {
		if currentLine == rolledLine {

			wordContained := slices.ContainsFunc(avoidList, func(wo word) bool {
				return w.isEqual(wo)
			})
			if wordContained {
				return wdb.RandomPick(l, avoidList, retryAkkumulator+1)
			}

			return w, nil
		}

		currentLine++
	}

	return word{}, fmt.Errorf("RandomPick could not find random line aka this should never happen ^^")
}

func (wdb wordDatabase) RandomPickWithFallback(l language, avoidList []word, retryAkkumulator uint8) word {
	w, err := wdb.RandomPick(l, avoidList, retryAkkumulator)
	if err != nil {
		return word{'R', 'O', 'A', 'T', 'E'}.ToLower()
	}

	return w.ToLower()
}

type keyboard struct {
	KeyGrid [][]keyboardKey
}

func (k *keyboard) Init(l language, lgs []letterGuess) {
	k.KeyGrid = [][]keyboardKey{
		{{"Q", false, MatchNone}, {"W", false, MatchNone}, {"E", false, MatchNone}, {"R", false, MatchNone}, {"T", false, MatchNone}, {"Y", false, MatchNone}, {"U", false, MatchNone}, {"I", false, MatchNone}, {"O", false, MatchNone}, {"P", false, MatchNone}, {"Delete", false, MatchNone}},
		{{"A", false, MatchNone}, {"S", false, MatchNone}, {"D", false, MatchNone}, {"F", false, MatchNone}, {"G", false, MatchNone}, {"H", false, MatchNone}, {"J", false, MatchNone}, {"K", false, MatchNone}, {"L", false, MatchNone}, {"Enter", false, MatchNone}},
		{{"Z", false, MatchNone}, {"X", false, MatchNone}, {"C", false, MatchNone}, {"V", false, MatchNone}, {"B", false, MatchNone}, {"N", false, MatchNone}, {"M", false, MatchNone}},
	}

	for ri, r := range k.KeyGrid {
	KeyLoop:
		for ki, kk := range r {
			for _, lg := range lgs {
				if kk.Key == "Enter" || kk.Key == "Delete" {
					continue KeyLoop
				}

				KeyR := firstRune(kk.Key)
				betterMatch := (k.KeyGrid[ri][ki].Match == MatchNone) ||
					(k.KeyGrid[ri][ki].Match == MatchVague && lg.Match == MatchExact)

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
	Match  match
}

func Map[T, U any](ts []T, f func(T) U) []U {
	us := make([]U, len(ts))
	for i := range ts {
		us[i] = f(ts[i])
	}
	return us
}

func getAllFilenames(efs iofs.FS) (files []string, err error) {
	if err := iofs.WalkDir(efs, ".", func(path string, d iofs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		files = append(files, path)

		return nil
	}); err != nil {
		return nil, err
	}

	return files, nil
}

func filePathsByLang() map[language]map[wordCollection][]string {
	return map[language]map[wordCollection][]string{
		LANG_EN: {
			WC_ALL: {
				"configs/corpora-eng_news_2023_10K-export.txt",
				"configs/en-en.words.v2.txt",
			},
			WC_COMMON: {
				"configs/corpora-eng_news_2023_10K-export.txt",
			},
		},
		LANG_DE: {
			WC_ALL: {
				"configs/corpora-deu_news_2023_10K-export.txt",
				"configs/de-de.words.v2.txt",
			},
			WC_COMMON: {
				"configs/corpora-deu_news_2023_10K-export.txt",
			},
		},
	}
}

func main() {
	log.Println("staring server...")

	envCfg := envConfig()
	sessions := sessions{}

	wordDb := wordDatabase{}
	err := wordDb.Init(fs, filePathsByLang())
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
	))

	mux := http.NewServeMux()

	staticFS, err := iofs.Sub(fs, "web/static")
	if err != nil {
		log.Fatalf("subtree for 'static' dir of embed fs failed: %s", err)
	}

	mux.Handle(
		"GET /static/",
		http.StripPrefix("/static", http.FileServer(http.FS(staticFS))),
	)

	mux.HandleFunc("GET /", func(w http.ResponseWriter, req *http.Request) {
		sess := handleSession(w, req, &sessions, wordDb)

		p := sess.lastEvaluatedAttempt
		// log.Printf("debug '/' route - sess.lastEvaluatedAttempt:\n %v\n", wo)
		p.Debug = sess.activeSolutionWord.String()
		sessions.updateOrSet(sess)

		fData := TemplateDataForm{}.New(sess.language, p, sess.PastWords(), sess.activeSolutionWord.hasDublicateLetters())
		fData.IsSolved = p.isSolved()
		fData.IsLoose = p.isLoose()

		err := t.ExecuteTemplate(w, "index.html.tmpl", fData)
		if err != nil {
			log.Printf("error t.Execute '/' route: %s", err)
		}
	})

	mux.HandleFunc("GET /lettr", func(w http.ResponseWriter, r *http.Request) {
		s := handleSession(w, r, &sessions, wordDb)

		p := s.lastEvaluatedAttempt

		sessions.updateOrSet(s)

		p.Debug = s.activeSolutionWord.String()

		fData := TemplateDataForm{}.New(s.language, p, s.PastWords(), s.activeSolutionWord.hasDublicateLetters())
		fData.IsSolved = p.isSolved()
		fData.IsLoose = p.isLoose()

		err = t.ExecuteTemplate(w, "lettr-form", fData)
		if err != nil {
			log.Printf("error t.ExecuteTemplate '/lettr' route: %s", err)
		}
	})

	mux.HandleFunc("POST /lettr", func(w http.ResponseWriter, r *http.Request) {
		s := handleSession(w, r, &sessions, wordDb)

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
			w.Write([]byte("cannot parse form data"))
			return
		}

		p := s.lastEvaluatedAttempt
		p.Debug = s.activeSolutionWord.String()

		if p.isSolved() || p.isLoose() {
			w.WriteHeader(204)
			return
		}

		// log.Printf("debug '/lettr' route - row compare: activeRow='%d' / formRowCount='%d' \n", s.lastEvaluatedAttempt.activeRow(), countFilledFormRows(r.PostForm))
		if s.lastEvaluatedAttempt.activeRow() != countFilledFormRows(r.PostForm)-1 {
			w.WriteHeader(422)
			w.Write([]byte("faked rows"))
			return
		}

		p, err = parseForm(p, r.PostForm, s.activeSolutionWord, s.language, wordDb)
		if err == ErrNotInWordList {
			w.WriteHeader(422)
			w.Write([]byte("word not in word list"))
			return
		}

		s.lastEvaluatedAttempt = p
		sessions.updateOrSet(s)

		fData := TemplateDataForm{}.New(s.language, p, s.PastWords(), s.activeSolutionWord.hasDublicateLetters())
		fData.IsSolved = p.isSolved()
		fData.IsLoose = p.isLoose()

		err = t.ExecuteTemplate(w, "lettr-form", fData)
		if err != nil {
			log.Printf("error t.ExecuteTemplate '/lettr' route: %s", err)
		}
	})

	mux.HandleFunc("POST /new", func(w http.ResponseWriter, r *http.Request) {
		s := handleSession(w, r, &sessions, wordDb)

		// handle lang switch
		l := s.language
		maybeLang := r.FormValue("lang")
		if maybeLang != "" {
			l, _ = NewLang(maybeLang)
			s.language = l

			type TemplateDataLanguge struct {
				Language language
			}
			tData := TemplateDataLanguge{Language: l}

			err := t.ExecuteTemplate(w, "oob-lang-switch", tData)
			if err != nil {
				log.Printf("error t.ExecuteTemplate '/new' route: %s", err)
			}
		}

		p := puzzle{}

		s.lastEvaluatedAttempt = p
		s.AddPastWord(s.activeSolutionWord)
		s.activeSolutionWord = wordDb.RandomPickWithFallback(l, s.pastWords, 0)
		sessions.updateOrSet(s)

		p.Debug = s.activeSolutionWord.String()

		fData := TemplateDataForm{}.New(s.language, p, s.PastWords(), s.activeSolutionWord.hasDublicateLetters())
		fData.IsSolved = p.isSolved()
		fData.IsLoose = p.isLoose()

		// w.Header().Add("HX-Refresh", "true")
		err := t.ExecuteTemplate(w, "lettr-form", fData)
		if err != nil {
			log.Printf("error t.ExecuteTemplate '/new' route: %s", err)
		}
	})

	mux.HandleFunc("POST /help", func(w http.ResponseWriter, r *http.Request) {
		s := handleSession(w, r, &sessions, wordDb)

		p := s.lastEvaluatedAttempt

		sessions.updateOrSet(s)

		p.Debug = s.activeSolutionWord.String()

		fData := TemplateDataForm{}.New(s.language, p, s.PastWords(), s.activeSolutionWord.hasDublicateLetters())
		fData.IsSolved = p.isSolved()
		fData.IsLoose = p.isLoose()

		err := t.ExecuteTemplate(w, "help", fData)
		if err != nil {
			log.Printf("error t.ExecuteTemplate '/help' route: %s", err)
		}
	})

	mux.HandleFunc("GET /suggest", func(w http.ResponseWriter, r *http.Request) {
		err := t.ExecuteTemplate(w, "suggest", struct{}{})
		if err != nil {
			log.Printf("error t.ExecuteTemplate '/suggest' route: %s", err)
		}
	})

	mux.HandleFunc("POST /suggest", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			log.Printf("error: %s", err)

			w.WriteHeader(422)
			w.Write([]byte("can not parse form data"))
			return
		}

		form := r.PostForm
		log.Printf("url values: %v\n", form)

		title := fmt.Sprintf("new word suggestion: '%s'", form["word"][0])
		ir := github.IssueRequest{Title: &title}
		github.CreateIssue(context.Background(), envCfg.githubToken, ir)

		type Message string

		type TemplateDataMessages struct {
			ErrMsgs     []Message
			InfoMsgs    []Message
			SuccessMsgs []Message
		}

		err = t.ExecuteTemplate(w, "oob-messages", TemplateDataMessages{
			SuccessMsgs: []Message{"Suggestion was send, thank you!", "Happy riddling!"},
			InfoMsgs:    []Message{"Words are mad up by letters", "Happy riddling!"},
			ErrMsgs:     []Message{"Error", "Happy riddling!"},
		})
		if err != nil {
			log.Printf("error t.ExecuteTemplate 'oob-messages' route: %s", err)
		}

		err = t.ExecuteTemplate(w, "suggest", struct{}{})
		if err != nil {
			log.Printf("error t.ExecuteTemplate '/suggest' route: %s", err)
		}
	})

	counter := counterState{count: 0}
	mux.HandleFunc("POST /counter", func(w http.ResponseWriter, req *http.Request) {
		// handleSession(w, req, &sessions)
		counter.mu.Lock()
		counter.count++
		defer counter.mu.Unlock()

		b, err := io.ReadAll(req.Body)
		if err != nil {
			log.Fatalln(err)
		}

		log.Printf("Method: %s\nbody:\n%s", req.Method, b)

		io.WriteString(w, fmt.Sprintf("<span>%d</span>", counter.count))

	})

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

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", envCfg.port), muxWithMiddlewares))
}

func envConfig() env {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		panic("PORT not provided")
	}

	gt, ok := os.LookupEnv("GITHUB_TOKEN")
	if !ok {
		panic("GITHUB_TOKEN not provided")
	}

	return env{port: port, githubToken: gt}
}

func handleSession(w http.ResponseWriter, req *http.Request, sessions *sessions, wdb wordDatabase) session {
	var err error
	var sess session

	cookie, err := req.Cookie(SESSION_COOKIE_NAME)
	if err != nil {
		return newSession(w, sessions, wdb)
	}

	if cookie == nil {
		return newSession(w, sessions, wdb)
	}

	sid := cookie.Value
	i := slices.IndexFunc(*sessions, func(s session) bool {
		return s.id == sid
	})
	if i == -1 {
		return newSession(w, sessions, wdb)
	}

	sess = (*sessions)[i]

	c := constructCookie(sess)
	http.SetCookie(w, &c)

	sess.expiresAt = generateSessionLifetime()
	(*sessions)[i] = sess

	return sess
}

func newSession(w http.ResponseWriter, sessions *sessions, wdb wordDatabase) session {
	sess := generateSession(LANG_EN, wdb)
	*sessions = append(*sessions, sess)
	c := constructCookie(sess)
	http.SetCookie(w, &c)

	return sess
}

func constructCookie(s session) http.Cookie {
	return http.Cookie{
		Name:     SESSION_COOKIE_NAME,
		Value:    s.id,
		Path:     "/",
		MaxAge:   s.maxAgeSeconds,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}

func generateSession(lang language, wdb wordDatabase) session { //todo: pass it by ref not by copy?
	id := uuid.NewString()
	expiresAt := generateSessionLifetime()
	activeWord, err := wdb.RandomPick(lang, []word{}, 0)
	if err != nil {
		log.Printf("pick random word failed: %s", err)

		activeWord = word{'R', 'O', 'A', 'T', 'E'}.ToLower()
	}

	return session{id, expiresAt, SESSION_MAX_AGE_IN_SECONDS, lang, activeWord, puzzle{}, []word{}}
}

func generateSessionLifetime() time.Time {
	return time.Now().Add(SESSION_MAX_AGE_IN_SECONDS * time.Second) // todo: 24 hour, sec now only for testing
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

func parseForm(p puzzle, form url.Values, solutionWord word, l language, wdb wordDatabase) (puzzle, error) {
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

func sliceToWord(maybeGuessedWord []string) (word, error) {
	w := word{}

	if len(maybeGuessedWord) != len(w) {
		return word{}, fmt.Errorf("sliceToWord: provided slice does not match word length")
	}

	for i, l := range maybeGuessedWord {
		w[i], _ = utf8.DecodeRuneInString(strings.ToLower(l))
		if w[i] == 65533 {
			w[i] = 0
		}
	}

	return w, nil
}

func evaluateGuessedWord(guessedWord word, solutionWord word) wordGuess {
	solutionWord = solutionWord.ToLower()
	guessedLetterCountMap := make(map[rune]int)

	resultWordGuess := wordGuess{}

	// initilize
	for i, gr := range guessedWord {
		resultWordGuess[i].Letter = gr
		resultWordGuess[i].Match = MatchNone
	}

	// mark exact matches
	for i, gr := range guessedWord {
		exact := solutionWord[i] == gr

		if exact {
			guessedLetterCountMap[gr]++
			resultWordGuess[i].Match = MatchExact
		}
	}

	// mark some/vague matches
	for i, gr := range guessedWord {
		if resultWordGuess[i].Match == MatchExact {
			continue
		}

		some := solutionWord.contains(gr)

		if !(resultWordGuess[i].Match == MatchVague) || some {
			guessedLetterCountMap[gr]++
		}

		s := some && (guessedLetterCountMap[gr] <= solutionWord.count(gr))
		if s {
			resultWordGuess[i].Match = MatchVague
		}
	}

	return resultWordGuess
}
