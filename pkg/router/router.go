package router

import (
	iofs "io/fs"
	"net/http"

	"github.com/pandorasNox/lettr/pkg/middleware"
	"github.com/pandorasNox/lettr/pkg/puzzle"
	"github.com/pandorasNox/lettr/pkg/router/routes"
	"github.com/pandorasNox/lettr/pkg/server"
	"github.com/pandorasNox/lettr/pkg/session"
)

func New(staticFS iofs.FS, server *server.Server, sessions *session.Sessions, wordDb puzzle.WordDatabase, imprintUrl string, githubToken string, revision string, faviconPath string) http.Handler {
	mux := http.NewServeMux()

	mux = addRoutes(mux, staticFS, server, sessions, wordDb, imprintUrl, githubToken, revision, faviconPath)

	handlerWithRoutesWithMiddlewares := addMiddlewares(mux)

	return handlerWithRoutesWithMiddlewares
}

func addRoutes(mux *http.ServeMux, staticFS iofs.FS, server *server.Server, sessions *session.Sessions, wordDb puzzle.WordDatabase, imprintUrl string, githubToken string, revision string, faviconPath string) *http.ServeMux {
	mux.HandleFunc("GET /static/", routes.Static(staticFS))
	mux.HandleFunc("GET /", routes.Index(sessions, wordDb, imprintUrl, revision, faviconPath))
	mux.HandleFunc("GET /letter-hint", routes.LetterHint(sessions, wordDb))
	mux.HandleFunc("GET /lettr", routes.GetLettr(sessions, wordDb, imprintUrl, revision, faviconPath))
	mux.HandleFunc("POST /lettr", routes.PostLettr(sessions, wordDb, imprintUrl, revision, faviconPath))
	mux.HandleFunc("POST /new", routes.PostNew(sessions, wordDb, imprintUrl, revision, faviconPath))
	mux.HandleFunc("POST /help", routes.Help(sessions, wordDb))
	mux.HandleFunc("GET /suggest", routes.GetSuggest(sessions, wordDb))
	mux.HandleFunc("POST /suggest", routes.PostSuggest(githubToken, sessions, wordDb, server))
	mux.HandleFunc("GET /metrics", routes.GetMetrics(server))

	// add tesing routes
	// mux.HandleFunc("GET /test", routes.GetTestPage())
	// mux.HandleFunc("POST /test/honey/increment", routes.PostIncrementHoneyTrapped(server))

	return mux
}

func addMiddlewares(mux *http.ServeMux) http.Handler {
	middlewares := []func(http.Handler) http.Handler{
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

	return muxWithMiddlewares
}
