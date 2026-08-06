# Landing Page Migration Plan

## Goal

Replace the current `/` route (which serves the game directly) with a new landing page. The game moves to `/play`.

## Current State

- `GET /` → `index.html.tmpl` (nav + game form + footer)
- No `/play` route exists

## Target State

- `GET /` → new landing page (hero text + game mode cards)
- `GET /play` → current game page (nav + game form + footer)
- Classic card on landing links to `/play`
- Language dropdown on landing navigates to `/play` with selected language

## Steps

### 1. Create `play.go` handler

New file: `pkg/router/routes/play.go`

Copy logic from `index.go` — same session handling, same template data — but render `play.html.tmpl` instead of `index.html.tmpl`.

### 2. Create `play.html.tmpl` template

New file: `pkg/router/routes/templates/play.html.tmpl`

Move the current `index.html.tmpl` contents here:
- Nav bar (with language dropdown using `hx-post="/new"`)
- `{{ template "lettr-form" . }}`
- Footer (with Suggest a word button)
- Message partials (`oob-messages`, `message-success`, `message-error`, `message-info`)

### 3. Update `index.go` → landing handler

Modify `pkg/router/routes/index.go`:
- Change template from `index.html.tmpl` to `landing.html.tmpl`
- Simplify: no game state needed, just pass imprintUrl, revision, faviconPath

### 4. Create `landing.html.tmpl` template

New file: `pkg/router/routes/templates/landing.html.tmpl`

Based on `option6_combined.html`, but:
- Use app's embedded CSS/JS (`/static/generated/output.css`, `/static/generated/main.js`)
- Use favicon template vars
- Classic card `href="/play"`
- Language dropdown: clicking a language navigates to `/play` (full page nav, no HTMX needed)
- Footer: "Suggest a word" links to `/suggest` or navigates to `/play` first

### 5. Register `GET /play` route

In `pkg/router/router.go`, add:
```go
mux.HandleFunc("GET /play", routes.Play(sessions, wordDb, imprintUrl, revision, faviconPath))
```

### 6. Update `templates.go` embed

Verify `//go:embed *.html.tmpl` picks up the new files automatically.

### 7. Update existing HTMX targets

The game page (`play.html.tmpl`) uses `hx-target="#lettr-container"` for language switching and new game. This is fine since the `lettr-form` partial is rendered there. No HTMX calls originate from the landing page.

### 8. Clean up `index.html.tmpl`

Can be removed — replaced by `play.html.tmpl` and `landing.html.tmpl`.

## Language Dropdown on Landing

Clicking a language in the dropdown navigates directly to `/play` (full page load), which will start a new game in that language via the session handler.
