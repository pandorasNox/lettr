package main

import (
	"embed"
	"fmt"
	iofs "io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/pandorasNox/lettr/pkg/puzzle"
	"github.com/pandorasNox/lettr/pkg/router"
	"github.com/pandorasNox/lettr/pkg/server"
	"github.com/pandorasNox/lettr/pkg/session"
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
	server := server.Server{}
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

	router := router.New(staticFS, &server, &sessions, wordDb, envCfg.imprintUrl, envCfg.githubToken, Revision, FaviconPath)

	// v1 := http.NewServeMux()
	// v1.Handle("/v1/", http.StripPrefix("/v1", muxWithMiddlewares))

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
