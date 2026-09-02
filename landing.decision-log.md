# Landing page grilling: decision log

Session date: 2026-09-02. Status: **stopped mid-decision** (see "Where we stopped").
Source plan: `landing.plan.md`. This log records what was decided, what was
discovered, and what is still open, so a follow-up session can resume without
re-asking settled questions.

## Decisions made

### D1. Scope
The ticket is the landing page and reaching the lettr game from it. Nothing
else. Consequence: the lettr gameplay endpoints stay exactly where they are:

- `POST /lettr` (word guess)
- `GET /lettr` (form refetch, used by the help/suggest Back buttons)
- `POST /help` (hints)

They are lettr-specific but unscoped. The pooplr ticket brings its own
endpoints. (Plan's naming-scheme table does not cover them and stays that way.)

### D2. Unknown mode URLs
Before pooplr exists, `GET /game/<unknown>` returns a 404 via a Go 1.22+
wildcard registration (`GET /game/{mode}`), instead of silently falling
through to the landing page (Go's `GET /` is a prefix catch-all).
`GET /game/lettr` stays more specific and wins. Go is 1.24.4. The 404 body
was sketched as plain text with status 404, not an error page template.

### D3. Landing cards
Two cards, titled with the mode names (not the mock's gameplay names):

1. **lettr** — active, links to `/game/lettr`
2. **pooplr** — disabled "Coming soon" badge (span, no link)

The mock `option6_combined.html` shows Classic / Anagram / Speed Run. The
design is reused, the speculative third card (Speed Run) is dropped, and the
titles become the mode names so the URL scheme is visible in the UI.

### D4. Template extraction, limited
Only two things are extracted into shared files:

- `messages.html.tmpl`: `oob-messages`, `message-success`, `message-error`,
  `message-info` (all currently defined in `index.html.tmpl`)
- the language partials `lang-btn-inner` and `oob-lang-switch` (currently
  owned by `lettr-form.html.tmpl`, which the landing page must not depend on)

Everything else (head/assets/favicon block, nav with the flag SVGs, footer)
is **duplicated per page**. No shared shell partials, no common base model.
Explicitly rejected: full shell extraction with a `models.Page` base.

### D5. Standalone `/suggest` page dropped (plan step 9 deleted)
"Suggest a word" stays only on the lettr game page, because it suggests lettr
words. The landing footer gets no suggest button (tagline, "Made with
HTMX", revision + imprint only). The game page footer keeps its current
in-place `hx-get="/suggest"` swap into `#lettr-container`. Therefore:

- no `suggest-page.html.tmpl`
- no `HX-Request` branching in `GetSuggest`
- no `TemplateDataSuggest` growth
- the Back-button question that motivated step 9 is moot

### D6. Landing page gets a working language switch
The landing nav has the language dropdown. It posts to a new
`POST /language`, swaps the button label in place via the existing
`oob-lang-switch` OOB partial, renders no game, and does not navigate.
(Landing-without-dropdown was considered and rejected.)

### D7. Game/language consistency: invalidation, not a stored language field
The plan proposed a `language` field on `puzzle.GameState` compared against
the session language at game-page load (option A). That was rejected in
favour of option B:

1. `puzzle.GameState` grows only `HasGame() bool` (true iff it holds a
   solution word). No language field. Language stays app state on the session.
2. Switching the session language invalidates any in-flight game
   (`ResetGame()`, a new method on session; the prototype stands in with the
   existing `SetGameState(puzzle.GameState{})`).
3. `session.HandleSession` (session.go:86) regenerates an absent game in the
   session language. Every handler already calls `HandleSession`, so this one
   check is the single reconcile point and covers `GET /game/lettr`,
   `GET /lettr`, `/help` and every future mode handler.

Known tradeoff (accepted): B trusts that every future language-changing path
resets the game. A's comparison would self-heal a missed reset. Today there
are exactly two such paths: `PostLanguage` (resets) and `PostGameLettrNew`
(regenerates immediately, as it does today).

Side notes: discarding the stale seed word without `AddPastWord` is fine
(different corpus). The game page's current semantics (switch = new game) are
preserved exactly.

### D8. Unknown-`lang` error handling, as its own first commit
`PostNew` today does `l, _ = language.NewLang(maybeLang)` (new.go:22).
`NewLang` returns a fallback of English plus an error for any value that is
not `en`/`de`, so a garbage value silently switches the player to English
and starts a new game. Latent today (the dropdowns only ever send en/de).

Decision: handle the error. First commit of the whole effort, against the
current code: log the error, keep the player's current language, continue
with the request. Same no-op-on-unknown semantics carry into the new
`PostLanguage` and `PostGameLettrNew` handlers.

### D9. Shared lang logic shape — OPEN
See "Where we stopped".

## Facts discovered during the session

These correct or extend the plan. Each is verified against the code.

1. **Plan step 6's "zero-value GameState" rationale is wrong.** Fresh
   sessions are seeded with a live English game: `newSession` →
   `generateSession(LANG_EN, wdb)` → `puzzle.NewGame` (session.go:118-125,
   139-144). A new visitor already has an EN solution word before ever
   seeing the landing page. Under D7 this detail no longer matters, but the
   plan text needs correcting.
2. **`language.NewLang` falls back to EN with an error**
   (language.go:12-18). `PostNew` discards that error (see D8).
3. **The mock's card names do not match the plan.** Classic / Anagram /
   Speed Run vs lettr / pooplr, plus a third card the plan never mentions
   (dropped in D3).
4. **Gameplay endpoints the plan never mentions:** `lettr-form.html.tmpl`
   posts guesses to `POST /lettr` and hints to `POST /help`; the help and
   suggest fragments' Back buttons do `hx-get="/lettr"`. `GET /lettr`
   (`GetLettr`) exists solely to serve that back-refetch (renders the
   `lettr-form` fragment). All stay untouched per D1.
5. **Two Back buttons exist, not one:** help.html.tmpl:6-13
   (`data-testid="back-btn"`) and suggest.html.tmpl:6-12. Both
   `hx-get="/lettr"` into `#lettr-container`. Both live game-page-only
   in-place, so both are safe under D1/D5 and need no change.
6. **The standalone suggest page (now dropped) would have broken in two
   places, not one:** the Back button and the form's own
   `hx-post="/suggest" hx-target="#lettr-container"`. Moot after D5,
   recorded in case step 9 is ever revived.
7. **No UI-visible signal of the game's language.** The keyboard grid is
   identical for EN and DE (`shared/keyboard.go`: the `l` parameter is unused
   in the grid construction). The only proof that "the game is in language X"
   is the solution word from help. A Playwright cross-page test against the
   real corpora cannot assert this reliably; a Go handler test with
   disjoint `fstest` word sets (the pattern in
   `pkg/router/routes/suggest_test.go`) asserting `wdb.Exists(LANG_DE,
   solution)` can.
8. **`session.HandleSession` returns the session struct by value**
   (session.go:86); `SetLanguage` has a pointer receiver; handlers write
   back via `sessions.UpdateOrSet(s)`. This forced the `&s` call shape in
   the helper prototype (variant 2) and is why a routes-package helper
   cannot name the session type at all (it is unexported).
9. **Test inventory for step 12:** no Go test references `/new` or `/`
   routes. `TestGetSuggest` renders the real embedded templates, so it
   indirectly exercises the `ParseFS` list. Bats checks (embeds, tailwind)
   are route-unrelated. Playwright: `page.application.ts` `goto('/')` and
   `0001.can-be-solved.spec.ts` `goto('/')` are the only callers; the spec
   toggles the theme (so `#theme-toggle` must survive on the game page,
   which it does via the copy) and solves via the help hint.
10. **No `<title>` tag exists on any current page.** The mock has one.
    Open detail, never decided.
11. **Pre-existing, not touched by this plan:** `createSuccessResponse`
    (suggest.go:115-126) renders the fresh suggest fragment with a
    zero-value model, so the honeypot input name is empty after the first
    submit.
12. **`web/app/main.ts` references no routes.** Keyboard logic keys off
    `.focusable` inputs (none on the landing), `htmx:afterSettle` resets
    state, `hx-ext response-targets` handles the 422 swaps. No changes
    needed for the landing.
13. **`templates.go`** (pkg/router/routes/templates/templates.go:23-30):
    explicit `ParseFS` list and `template.New("index.html.tmpl")` root name;
    both must change (plan step 10, unchanged by this session).
14. **`TemplateDataIndex`** is `JSCachePurgeTimestamp` + embedded
    `shared.TemplateDataLettr` (models/index.go) — the plan's rename to
    `TemplateDataGameLettr` holds. Note `PostNew` also has an inline
    struct with a typo, `TemplateDataLanguge` (new.go:25).

## Where we stopped

Mid-D9: the shape of the shared lang logic (read `lang`, validate, set on
session, render the label partial) across `PostLanguage` and
`PostGameLettrNew`. The user wanted to see concrete code before deciding and
asked for the prototypes to live in the repo.

Prototypes, written and build-verified 2026-09-02 (`go build ./...` and
`gofmt` clean via the golang:1.24-alpine container):

- `bin/prototypes/trylang/variant-1-copy/variant1.go` — block copied into
  both handlers. Compiles.
- `bin/prototypes/trylang/variant-2-helper/variant2.go` — `applyLang` plus a
  `langSwitcher` consumer-side interface (needed because the session type is
  unexported; called with `&s`). Compiles.
- `bin/prototypes/trylang/variant-3-method/variant3.go` — reading sketch
  (block comment, deliberately not compiled): `TrySetLanguage` method in
  `pkg/session` plus thin call sites. No interface, no pointer dance.

My stated recommendation was variant 3. The user had not picked.

Cleanup rule: once D9 is resolved, delete the whole `bin/prototypes/trylang`
tree.

## Open questions for the follow-up session, in suggested order

1. **D9:** pick variant 1, 2, or 3 (prototypes above).
2. **Navigate back to the landing:** the game page currently has no link to
   `/` (the `lettr` h1 in the nav is plain text). Decide whether it becomes
   `<a href="/">` on `game-lettr.html.tmpl`. Leaning yes, one line, because
   the landing becomes the site root and deserves a way back.
3. **Test plan (plan step 12):** confirm the set — Playwright: existing spec
   repointed to `/game/lettr`, plus a landing spec (hero renders, lettr card
   navigates, lang dropdown swaps the label without navigating, pooplr card
   disabled). Go: one handler test for the D7 guarantee using disjoint
   `fstest` corpora (per fact 7). The plan's "cross-page Playwright spec"
   should be re-scoped or replaced accordingly.
4. **Small items, each a one-line decision:**
   - fix the `TemplateDataLanguge` typo when the code moves
   - add a `<title>` to the landing (and optionally the game page; neither
     has one today)
   - keep the plan's flat naming (`index.go` / `routes.Index` →
     `landing.html.tmpl`) or rename to `landing.go` / `routes.Landing`
   - landing dropdown attributes: the plan's example uses
     `hx-post="/language" hx-swap="none"` with no `hx-target`; the
     label-swap-must-work guarantee belongs in the landing Playwright spec
     (OOB behaviour with `hx-swap="none"` needs browser verification)
5. **Rewrite `landing.plan.md`** to match D1-D8 (delete step 9, swap the
   step 6 mechanism to D7 and fix its factually wrong zero-value rationale,
   add the D2 404 handler to step 8, fix the step 5 card list per D3, record
   the D4 extraction limits in step 3, record D8 as the first commit and the
   no-op-on-unknown semantics in steps 1/7, drop the suggest row from the
   summary table). Not yet requested; ask first.

## Ground rules from the session

- One question at a time; facts looked up in the environment, only decisions
  put to the user.
- Scope discipline was asserted twice by the user (D1, D5): when in doubt,
  the narrower answer wins.
- The user asked that the `unslop` skill be applied to responses (plain
  words, no em dashes, no filler). This document follows it.
