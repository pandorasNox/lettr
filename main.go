package main

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"sync"

	//"io"
	"log"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/google/uuid"
)

//go:embed templates/*.html.tmpl
var fs embed.FS

type env struct {
	port string
}

func (e env) String() string {
	s := fmt.Sprintf("port: %s\n", e.port)
	// s = s + fmt.Sprintf("foo: %s\n", e.port)
	return s
}

type session struct {
	id            string
	expiresAt     time.Time
	maxAgeSeconds int
	activeWord    string
}

type sessions []session

func (ss sessions) String() string {
	out := ""
	for _, s := range ss {
		out = out + s.id + " " + s.expiresAt.String() + "\n"
	}

	return out
}

const SESSION_COOKIE_NAME = "session"
const SESSION_MAX_AGE_IN_SECONDS = 120

type counterState struct {
	mu    sync.Mutex
	count int
}

func main() {
	envCfg := envConfig()
	sessions := sessions{}

	log.Println("staring server...")
	log.Printf("env conf:\n%s", envCfg)

	t := template.Must(template.ParseFS(fs, "templates/index.html.tmpl"))

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		handleSession(w, req, &sessions)

		// io.WriteString(w, fmt.Sprintf("Hello, world!\n%s", sessions))
		//io.WriteString(w, fmt.Sprintf("Hello, world! %s\n", session.id))
		log.Printf("sessions:\n%s", sessions)

		t.Execute(w, nil)

	})

	http.HandleFunc("/wordle", func(w http.ResponseWriter, r *http.Request) {
		s := handleSession(w, r, &sessions)

		b, err := io.ReadAll(r.Body)
		if err != nil {
			log.Fatalln(err)
		}

		log.Printf("word: %s\nbody:\n%s", s.activeWord, b)

		// io.WriteString(w, fmt.Sprintf("Hello, world!\n%s", sessions))
		//io.WriteString(w, fmt.Sprintf("Hello, world! %s\n", session.id))
		log.Printf("sessions:\n%s", sessions)

		t.Execute(w, nil)

	})

	counter := counterState{count: 0}
	http.HandleFunc("/counter", func(w http.ResponseWriter, req *http.Request) {
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

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", envCfg.port), nil))
}

func envConfig() env {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		panic("PORT not provided")
	}

	return env{port}
}

func handleSession(w http.ResponseWriter, req *http.Request, sessions *sessions) session {
	var err error
	var sess session

	cookie, err := req.Cookie(SESSION_COOKIE_NAME)
	if err != nil {
		return newSession(w, sessions)
	}

	if cookie == nil {
		return newSession(w, sessions)
	}

	sid := cookie.Value
	i := slices.IndexFunc(*sessions, func(s session) bool {
		return s.id == sid
	})
	if i == -1 {
		return newSession(w, sessions)
	}

	sess = (*sessions)[i]

	c := constructCookie(sess)
	http.SetCookie(w, &c)

	sess.expiresAt = generateSession().expiresAt
	(*sessions)[i] = sess

	return sess
}

func newSession(w http.ResponseWriter, sessions *sessions) session {
	sess := generateSession()
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

func generateSession() session { //todo: pass it by ref not by copy?
	id := uuid.NewString()
	expiresAt := time.Now().Add(SESSION_MAX_AGE_IN_SECONDS * time.Second) // todo: 24 hour, sec now only for testing
	activeWord := "ROATE"
	return session{id, expiresAt, SESSION_MAX_AGE_IN_SECONDS, activeWord}
}
