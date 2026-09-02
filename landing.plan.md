# Landing Page Migration Plan

## Goal

Replace the current `/` route (which serves the game directly) with a new landing page.
The game moves to a **mode-scoped URL**: `/game/lettr`.
Henceforth, the goal is not to implement the new Pooplr game, the goal is to prepare the playing field so that we are able to implement more game modes in the future.

## Out of scope

- Putting game modes into their own sub module + sub directory. For this work everything stays in
  `pkg/router/routes/` (handlers, models, templates) as described by the naming scheme below;
  splitting the modes into separate packages is part of an upcoming ticket.

## Naming scheme

The URL carries the game mode (`/game/<mode>`), and all Go/template naming mirrors it.
A second mode (`Pooplr`, a Poople variant) is planned, so everything must be nameable per mode:

| Concern           | `lettr` mode                              | future `pooplr` mode                       |
| ----------------- | ----------------------------------------- | ------------------------------------------ |
| Page route        | `GET /game/lettr`                         | `GET /game/pooplr`                          |
| New-game route    | `POST /game/lettr/new`                    | `POST /game/pooplr/new`                     |
| Handler file      | `pkg/router/routes/game_lettr.go`         | `pkg/router/routes/game_pooplr.go`          |
| Handler funcs     | `GetGameLettr`, `PostGameLettrNew`        | `GetGamePooplr`, `PostGamePooplrNew`        |
| Template file     | `templates/game-lettr.html.tmpl`          | `templates/game-pooplr.html.tmpl`           |
| Form partial      | `templates/lettr-form.html.tmpl`          | `templates/pooplr-form.html.tmpl`           |
| Model file        | `pkg/router/routes/models/game_lettr.go`  | `pkg/router/routes/models/game_pooplr.go`   |
| Model type        | `models.TemplateDataGameLettr`            | `models.TemplateDataGamePooplr`             |

Conventions followed: Go files snake_case (as in `letter_hint.go`), template files kebab-case
(as in `lettr-form.html.tmpl`), Go identifiers PascalCase.

Mode-agnostic concerns keep flat, unscoped names: `index.go` → `routes.Index` → `landing.html.tmpl`
→ `models.TemplateDataLanding`, and `language.go` → `routes.PostLanguage`.

## Current State

- `GET /` → `index.html.tmpl` (nav + game form + footer), handler `routes.Index`, model `models.TemplateDataIndex`
- No `/game/...` route exists
- `POST /new` is **overloaded and has two callers**, both targeting `#lettr-container` and both
  rendering the `lettr-form` partial:
  1. the language dropdown in the page shell (`index.html.tmpl`), sending `lang` — sets the session
     language, renders `oob-lang-switch` (OOB innerHTML swap of `#language-dropdown-menu`) **and**
     a fresh `lettr-form`
  2. the "New Game" button inside `lettr-form.html.tmpl`, sending no `lang`
- `puzzle.GameState` stores `activeSolutionWord`, `letterHints`, `lastEvaluatedAttempt` — **no
  language**. Nothing can currently detect that a running game belongs to a different language than
  `session.Language()`.
- `GET /suggest` returns a **fragment** (`{{ define "suggest" }}`), meant to be swapped into
  `#lettr-container`; its "< Back" button does `hx-get="/lettr"` → `#lettr-container`
- `templates.go` parses an **explicit file list** in `ParseFS` — new templates are *not* picked up
  automatically just because the `//go:embed` glob matches

## Target State

- `GET /` → new landing page (hero text + game mode cards)
- `GET /game/lettr` → current game page (nav + game form + footer)
- lettr card on landing links to `/game/lettr`
- Language dropdown on landing **switches the session language in place** (HTMX, no navigation) so
  the *upcoming* game starts in that language
- Language dropdown and "New Game" on a game page keep their current in-place behaviour, via a
  mode-scoped route
- Footer "Suggest a word" always links to the suggest page

## The `/new` problem

`POST /new` cannot stay as a single shared route. It renders `lettr-form` unconditionally, which is
wrong in two directions:

- **On the landing page** there is no `#lettr-container` and no game form to render at all
- **On `/game/pooplr`** (once it exists) both the language switch and the "New Game" button must
  render `pooplr-form`, not `lettr-form`

So the handler needs to know which mode it is acting for. Three ways to get that:

| Approach | How it works | Verdict |
| --- | --- | --- |
| **Mode in the URL** | `POST /game/lettr/new`, one handler per mode, each renders its own form partial | **Chosen.** The mode is already in the path of the page doing the posting; no inference, no shared switch statement, and it matches the naming scheme above |
| Track the source | Shared `/new` sniffs `Referer` / `HX-Current-URL` and switches on it | Rejected. Header-sniffing is fragile (missing/spoofed headers, proxies), and the handler still needs an internal mode→partial switch — all the cost of per-mode handlers with none of the clarity |
| Redirect / reload back | Respond `HX-Redirect: /game/<mode>` or `HX-Refresh: true`, let the browser re-render the page in the new language | Rejected. Full page reload on *every* language switch and *every* New Game — a regression from today's in-place swap — and it still has to determine where "back" is |

Choosing mode-in-URL also means **nothing needs to track a source**: `/game/lettr/new` is only ever
reachable from the lettr page, and the landing page uses a separate mode-agnostic route that renders
no game at all.

The second half of the problem is the *state*: if the landing page changes the language, the game
started later must be in that language. Handled by giving `GameState` a language and reconciling
lazily on page load — see step 6.

## Steps

### 1. Create the game-mode handlers

New file: `pkg/router/routes/game_lettr.go`, containing both handlers for the mode:

- `GetGameLettr(sessions, wordDb, imprintUrl, revision, faviconPath) http.HandlerFunc` — the page.
  Copy the logic from `index.go` (same session handling, same template data), render
  `game-lettr.html.tmpl`, plus the language reconciliation from step 6.
- `PostGameLettrNew(sessions, wordDb, imprintUrl, revision, faviconPath) http.HandlerFunc` — the
  new-game/language-switch endpoint. Body is today's `PostNew` verbatim: optional `lang` →
  `SetLanguage` + `oob-lang-switch`, then `AddPastWord` + `NewGame`, then render `lettr-form`.

Keeping both in one file per mode makes the "add pooplr" step a single new file.

### 2. Create the game-mode model

New file: `pkg/router/routes/models/game_lettr.go`, type `TemplateDataGameLettr` — a rename of the
current `TemplateDataIndex` (embeds `shared.TemplateDataLettr` + `JSCachePurgeTimestamp`).

### 3. Create `game-lettr.html.tmpl`

New file: `pkg/router/routes/templates/game-lettr.html.tmpl`.

Move the current `index.html.tmpl` contents here, with one change:

- Nav bar language dropdown: `hx-post="/new"` → `hx-post="/game/lettr/new"` (both `<a>` entries),
  `hx-target="#lettr-container"` unchanged
- `{{ template "lettr-form" . }}`
- Footer (Suggest a word button — keeps its current `hx-get="/suggest"` → `#lettr-container`)
- Message partials (`oob-messages`, `message-success`, `message-error`, `message-info`)

Note: the message partials are `define`d in `index.html.tmpl` today and are used by other routes
(`/suggest`, `/lettr`). They must survive the move — either keep them in `game-lettr.html.tmpl` or,
cleaner, extract them into `templates/messages.html.tmpl` so neither page owns them.

Also update `lettr-form.html.tmpl`: the "New Game" button hardcodes `hx-post="/new"` → change to
`hx-post="/game/lettr/new"`. Hardcoding is fine because the partial is itself per-mode — `pooplr-form`
will hardcode `/game/pooplr/new`. (Alternative, if the form partial is ever shared across modes:
carry a `NewGameUrl` field on `shared.TemplateDataLettr`. Not needed yet.)

### 4. Turn `index.go` into the landing handler

Modify `pkg/router/routes/index.go`:

- Render `landing.html.tmpl`
- New model `models.TemplateDataLanding` (`pkg/router/routes/models/landing.go`): no puzzle state —
  `Language`, `ImprintUrl`, `Revision`, `FaviconPath`, `JSCachePurgeTimestamp`
- `Language` is still needed: the landing nav renders `{{ template "lang-btn-inner" . }}`, so the
  dropdown shows the currently selected language
- Keep `session.HandleSession(...)` so a session (and its language) exists before the user picks a mode

`models.TemplateDataIndex` is replaced by `TemplateDataLanding` + `TemplateDataGameLettr`; delete it.

### 5. Create `landing.html.tmpl`

New file: `pkg/router/routes/templates/landing.html.tmpl`, based on `option6_combined.html`, but:

- App's embedded assets: `/static/generated/output.css` and `/static/generated/main.js`, both with
  `?cachePurge={{ printf "%d" .JSCachePurgeTimestamp }}`
- Favicon block from template vars (`.FaviconPath`)
- `hx-ext="response-targets"` on `<body>` (HTMX is used by the language switcher)
- lettr card → `href="/game/lettr"`
- Pooplr card → placeholder / disabled until `/game/pooplr` exists
- Language dropdown → **no navigation**, see step 6
- Footer "Suggest a word" → plain `<a href="/suggest">` (see step 9)
- Footer revision + imprint block, same as the game page

### 6. Language switching on the landing page

The dropdown must change the language of the *upcoming* game without navigating — same visible
effect as today (button label updates in place), just with no game form to re-render.

**Route.** New file `pkg/router/routes/language.go`, func `PostLanguage(sessions, wordDb)`, wired to
`POST /language` (mode-agnostic — it deliberately renders no game):

1. `s := session.HandleSession(w, r, sessions, wdb)`
2. Parse `lang` via `language.NewLang(...)`; no-op on unknown/empty values
3. `s.SetLanguage(l)` — **and nothing else**
4. `sessions.UpdateOrSet(s)`
5. Render `oob-lang-switch` only (reuses the existing `{{ define "oob-lang-switch" }}` in
   `lettr-form.html.tmpl`, which OOB-swaps `#language-dropdown-menu`)

Landing dropdown entries:

```html
<a hx-post="/language" hx-vals='{"lang": "en"}' hx-swap="none" href="#" ...>
```

`hx-swap="none"` because the only update is the OOB swap.

**State reconciliation.** `PostLanguage` deliberately does *not* start a new game. Eagerly calling
`NewGame` on every toggle would burn a word into `pastWords` per click (en→de→en consumes three),
and it would still not protect against any other path that changes the language. Instead, make the
mismatch detectable and fix it lazily at the point of use:

- Add a `language` field to `puzzle.GameState`, set it in `puzzle.NewGame`, expose `Language()`
- In `GetGameLettr` (and every future `GetGame<Mode>`): if
  `s.GameState().Language() != s.Language()`, call `s.NewGame(s.Language(), wdb)` before rendering

This closes a latent bug that exists today independently of the landing page: a session whose
language was changed without a game reset would keep playing a solution word from the other corpus.

A zero-value `GameState` (fresh session) reports the zero language, which will not equal the session
language, so the reconciliation naturally produces the first game — verify this against however
`session.HandleSession` currently seeds new sessions, and keep whichever path is already responsible
for the initial `NewGame` from double-generating.

### 7. Retire `POST /new`

Delete `pkg/router/routes/new.go` and the `POST /new` registration; its body moves into
`PostGameLettrNew` (step 1). Update all three call sites found in the templates:

- `index.html.tmpl:40` and `:55` (language dropdown) → moved to `game-lettr.html.tmpl` with
  `hx-post="/game/lettr/new"`
- `lettr-form.html.tmpl:24` ("New Game" button) → `hx-post="/game/lettr/new"`

Nothing else references `/new` (no Go tests, no TypeScript, no Playwright specs).

Optional: factor the shared `lang` → `SetLanguage` + `oob-lang-switch` snippet out of
`PostGameLettrNew` and `PostLanguage` into one unexported helper in `language.go`, so the two cannot
drift.

### 8. Register routes

In `pkg/router/router.go`:

```go
mux.HandleFunc("GET /", routes.Index(sessions, wordDb, imprintUrl, revision, faviconPath))          // landing
mux.HandleFunc("GET /game/lettr", routes.GetGameLettr(sessions, wordDb, imprintUrl, revision, faviconPath))
mux.HandleFunc("POST /game/lettr/new", routes.PostGameLettrNew(sessions, wordDb, imprintUrl, revision, faviconPath))
mux.HandleFunc("POST /language", routes.PostLanguage(sessions, wordDb))
```

Note `GET /` in Go's `ServeMux` is a catch-all prefix pattern — it already matches `/game/...`, but the
more specific `GET /game/lettr` wins. Unknown modes (`/game/pooplr` before it exists) fall through to
the landing page rather than 404-ing; add an explicit `GET /game/{mode}` 404 handler if that matters.

### 9. Make `/suggest` reachable as a page

The landing footer links to `/suggest` directly, but `GET /suggest` currently returns a fragment
intended for `#lettr-container`. Fix `routes.GetSuggest` to render a full page when the request is
not an HTMX request (no `HX-Request` header):

- HTMX request → current behaviour, render the `suggest` fragment (game page keeps working unchanged)
- Plain request → render a new `suggest-page.html.tmpl` that wraps `{{ template "suggest" . }}` in the
  same document shell (head/favicon/assets/nav/footer) as the other pages

Also fix the fragment's "< Back" button: `hx-get="/lettr"` → `#lettr-container` only works inside the
game page. On the standalone page it must be a plain link back to `/` (or to `/game/lettr`).
Simplest: render Back as a real `<a>` in the page wrapper and keep the HTMX button in the fragment,
or pass a flag through `TemplateDataSuggest`.

`models.TemplateDataSuggest` needs the page fields (`ImprintUrl`, `Revision`, `FaviconPath`,
`JSCachePurgeTimestamp`) for the standalone render.

### 10. Register the new templates

`templates/templates.go` lists files explicitly in `ParseFS` — **the `//go:embed` glob is not enough**.
Update the list:

```go
var Routes = template.Must(template.New("landing.html.tmpl").Funcs(funcMap).ParseFS(
	templatesFs,
	"landing.html.tmpl",
	"game-lettr.html.tmpl",
	"lettr-form.html.tmpl",
	"messages.html.tmpl",   // if extracted in step 3
	"help.html.tmpl",
	"suggest.html.tmpl",
	"suggest-page.html.tmpl",
	"pages/test.html.tmpl",
))
```

`template.New(...)` names the root template — drop `index.html.tmpl` there too.

### 11. Delete `index.html.tmpl`

Replaced by `landing.html.tmpl` + `game-lettr.html.tmpl` (+ `messages.html.tmpl`).

### 12. Update tests

- `tests/playwright/tests/page.application.ts` `goto()` → `/game/lettr` (currently `/`)
- `tests/playwright/tests/0001.can-be-solved.spec.ts` → `/game/lettr`
- Add a landing-page spec: hero renders, lettr card navigates to `/game/lettr`, language dropdown
  updates the button label without navigating, footer "Suggest a word" opens `/suggest`
- Add a cross-page spec for the reconciliation in step 6: set language on `/`, then open
  `/game/lettr`, assert the game is in that language
- Check `pkg/router/routes/*_test.go` and `scripts/checks/bats/` for hardcoded `/` expectations

## Summary of behaviour changes

| Element                        | Before                            | After                                                        |
| ------------------------------ | --------------------------------- | ------------------------------------------------------------ |
| `/`                            | game page                         | landing page                                                  |
| game URL                       | `/`                               | `/game/lettr` (mode in the path)                              |
| lettr card                     | —                                 | link to `/game/lettr`                                         |
| new-game endpoint              | `POST /new` (shared)              | `POST /game/lettr/new` (per mode)                             |
| language dropdown (landing)    | —                                 | `POST /language`, OOB label swap, **no navigation**, no game render; the game started later picks the language up |
| language dropdown (game page)  | `POST /new` → new game in place   | `POST /game/lettr/new` → same behaviour, mode-scoped URL      |
| "New Game" button              | `POST /new`                       | `POST /game/lettr/new`                                        |
| stale-language game            | undetectable (`GameState` has no language) | `GameState.Language()` + reconcile on game page load   |
| "Suggest a word"               | HTMX swap into `#lettr-container` | landing: link to `/suggest` (full page); game page: unchanged |
