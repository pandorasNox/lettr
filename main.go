package main

import (
	"context"
	"embed"
	"fmt"
	iofs "io/fs"
	"log"
	"net"
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

	ticker := time.NewTicker(1 * time.Hour)
	go cleanupSessions(ticker, &sessions)

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

	// v1 := http.NewServeMux()
	// v1.Handle("/v1/", http.StripPrefix("/v1", muxWithMiddlewares))

	// log.Fatal(testserver.ListenAndServe())
	// ctx, cancel := context.WithCancel(context.Background())
	httpServer := &http.Server{
		Addr:        ":8080",
		Handler:     mux,
		BaseContext: func(_ net.Listener) context.Context { return ctx },
	}
	// httpServer.RegisterOnShutdown(cancel)

	go func() {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", envCfg.port), router))
		log.Println("Stopped serving new connections.")
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}
	log.Println("Graceful shutdown complete.")

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", envCfg.port), router))
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

func cleanupSessions(ticker *time.Ticker, sesssions *session.Sessions) {
	for range ticker.C {
		sesssions.RemoveExpiredSessions()
	}
}
