![Project build, test and lint status badge](https://github.com/pandorasNox/lettr/actions/workflows/go.yml/badge.svg)

# lettr

## known issues
* If server restarts while playing, the next guess will end in a "fake rows" error. Resolved by reloading the page.

## plan of action
* [x] generate session (cookie) when none is present 
* [x] keep user data in server memory
* [ ] optional: session memory management based on cookie lifetime

## quiz
### what happens on server side
* [x] word generation – requires: allowed word list
* [x] input validation (check matches) – requires: allowed word list, generated/current word, input of last word, number of tries
* [x] trigger like quiz win/fail – requires: number of tries
* [ ] refactoring
    * [x] rename wordle to `lettr` bec of trademark
    * [x] use packages instead of everythiung in one file
    * [x] move leftover routing from main.go to routes packages
    * [ ] session handling
* [ ] check out https://github.com/torenware/vite-go
    * https://vitejs.dev/guide/backend-integration VS https://www.npmjs.com/package/webpack-assets-manifest

## open todo
- must
    * [x] word not InWordDB
    * [x] website app version via ??? (assets? or on webpage?)
    * [x] (mobile) keyboard (use + indication for used letters)
    * [x] protect against request size + correct http 413 error code throw
    * [x] fix scripts/tools.sh func_exec_cli passing parameter issue
    * [x] bugfix: full page get form submit request on random occations when it should just be a htmx post
    * [x] avoid same word twice (words to exclude (previous taken quizes))
    * [ ] editorial work: e.g. words like games or gamer are missing + maybe we introduce a common vs uncommen word list
        * [x] word suggestion (button to save (unknown) word eg. in LiteFS/email/github-issue/something)
            * [ ] form spam protection mechanism
                * [ ] easy honeypot (html form fields)
                * [ ] captcha: https://github.com/altcha-org/altcha
                * [ ] other (research task)?
        * [x] corpora dataset export https://corpora.uni-leipzig.de/en/res?corpusId=eng_news_2023&word=would
            * https://github.com/Leipzig-Corpora-Collection
        * https://api.wortschatz-leipzig.de/ws/swagger-ui/index.html#/Words/getWordInformation
        * https://wortschatz.uni-leipzig.de/en/download/English
    * [x] fix past words display
    * [x] add github action ci pipeline build + tests
        * [x] build
        * [x] tests
        * [x] linting
        * [x] test image build
        * [x] formatting
    * [x] add imprint/link to imprint
    * [x] fix letter hints is not reset with new game bug
    * [x] add metrics endpoint
    * [x] add scheduled RemoveExpiredSessions func (go routine in main.go)
    * [ ] tailwind check build succes (with files)
    * [ ] os.SIGNAL handling (gracefull server Shutdown)
- nice-to-have
    * [x] option for double letter hint
    * [ ] Circuit Breaker Support
    * [ ] pick word dataset picker
    * [ ] get definition (e.g. wikitionary)
        * options:
            * https://raw.githubusercontent.com/wordset/wordset-dictionary/master/data/%s.json
            * wikitionary API
                * en
                    * https://en.wiktionary.org/api/rest_v1/page/definition/hund
                    * https://en.wiktionary.org/wiki/hund
                * de
                    * no api
                    * https://de.wiktionary.org/wiki/hund
            * openthesaurus (de only)
                * https://www.openthesaurus.de/synonyme/search?q=test&format=application/json
    * [x] hint feature / give me one letter
    * [ ] ui languge should also change
    * [ ] ESLint
    * [ ] http error codes: <!-- was this ment for additional middleware??? -->
        * [ ] 414 URI Too Long
        * [ ] 431 Request Header Fields Too Large (RFC 6585)
    * [ ] improve VS Code dev container
    * [ ] htmx routing ? (keep nav + footer but change body + change url)
    * [x] mark suggested word on keyboard (pink letters)
