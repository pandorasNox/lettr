package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	iofs "io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pandorasNox/lettr/pkg/puzzle"
	"github.com/pandorasNox/lettr/pkg/router"
	"github.com/pandorasNox/lettr/pkg/session"
	"github.com/pandorasNox/lettr/pkg/state"
)

var Revision = "0000000"
var FaviconPath = "/static/assets/favicon"

//go:embed configs/*.txt
//go:embed web/static/assets/*
//go:embed web/static/generated/*.js
//go:embed web/static/generated/*.js.map
//go:embed web/static/generated/*.css
var embedFs embed.FS

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

func main() {
	log.Println("staring server...")

	envCfg := envConfig()
	serverState := state.Server{}
	sessions := session.NewSessions()

	wordDb := puzzle.WordDatabase{}
	err := wordDb.Init(embedFs, puzzle.FilePathsByLang())
	if err != nil {
		log.Fatalf("init wordDatabase failed: %s", err)
	}

	log.Printf("env conf:\n%s", envCfg)

	staticFS, err := iofs.Sub(embedFs, "web/static")
	if err != nil {
		log.Fatalf("subtree for 'static' dir of embed fs failed: %s", err) //TODO
	}

	router := router.New(staticFS, &serverState, &sessions, wordDb, envCfg.imprintUrl, envCfg.githubToken, Revision, FaviconPath)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", envCfg.port),
		Handler: router,
		// BaseContext: func(_ net.Listener) context.Context { return shutdownCtx },
	}

	sigChan := make(chan os.Signal, 1)
	shutdownDoneChan := make(chan bool, 2)

	run(sigChan, shutdownDoneChan, server, sessions)
}

func run(sigChan chan os.Signal, shutdownDoneChan chan bool, server *http.Server, sessions session.Sessions) {
	ticker := time.NewTicker(1 * time.Hour)
	quitScheduleChan := make(chan bool)
	go cleanupSessions(ticker, &sessions, quitScheduleChan, shutdownDoneChan)

	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}

		log.Println("Stopped serving new connections.")
		shutdownDoneChan <- true
	}()

	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	go func() { quitScheduleChan <- true }()

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}

	<-shutdownDoneChan
	<-shutdownDoneChan

	log.Println("Graceful shutdown complete.")
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

func cleanupSessions(ticker *time.Ticker, sesssions *session.Sessions, quitChan chan bool, shutdownDoneChan chan bool) {
	for {
		select {
		case <-ticker.C:
			sesssions.RemoveExpiredSessions()
		case <-quitChan:
			sesssions.RemoveExpiredSessions()
			shutdownDoneChan <- true
			return
		default:
			// do non-blocking select, as we can't know how long the ticker would block
			// so we can wait for quitChan signal
			// but also sleep to lighten cpu load from endless loop
			time.Sleep(500 * time.Millisecond)
		}
	}
}
