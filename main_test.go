package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/google/uuid"
	"github.com/pandorasNox/lettr/pkg/puzzle"
)

func Test_constructCookie(t *testing.T) {
	fixedUuid := "9566c74d-1003-4c4d-bbbb-0407d1e2c649"
	expireDate := time.Date(2024, 02, 27, 0, 0, 0, 0, time.Now().Location())

	type args struct {
		s session
	}
	tests := []struct {
		name string
		args args
		want http.Cookie
	}{
		// add test cases here
		{
			"test_name",
			args{session{fixedUuid, expireDate, SESSION_MAX_AGE_IN_SECONDS, LANG_EN, puzzle.Word{}, puzzle.Puzzle{}, []puzzle.Word{}}},
			http.Cookie{
				Name:     SESSION_COOKIE_NAME,
				Value:    fixedUuid,
				Path:     "/",
				MaxAge:   SESSION_MAX_AGE_IN_SECONDS,
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := constructCookie(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("constructCookie() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_handleSession(t *testing.T) {
	type args struct {
		w        http.ResponseWriter
		req      *http.Request
		sessions *sessions
		wdb      wordDatabase
	}

	// monkey patch time.Now
	patchFnTime := func() time.Time {
		return time.Unix(1615256178, 0)
	}
	patchesTime := gomonkey.ApplyFunc(time.Now, patchFnTime)
	defer patchesTime.Reset()
	// monkey patch uuid.NewString
	patches := gomonkey.ApplyFuncReturn(uuid.NewString, "12345678-abcd-1234-abcd-ab1234567890")
	defer patches.Reset()

	tests := []struct {
		name string
		args args
		want session
	}{
		// add test cases here
		{
			"test handleSession is generating new session if no cookie is set",
			args{
				httptest.NewRecorder(),
				httptest.NewRequest("get", "/", strings.NewReader("Hello, Reader!")),
				&sessions{},
				wordDatabase{db: map[language]map[wordCollection]map[puzzle.Word]bool{
					LANG_EN: {
						WC_COMMON: {
							puzzle.Word{'R', 'O', 'A', 'T', 'E'}: true,
						},
					},
				}},
			},
			session{
				id:                 "12345678-abcd-1234-abcd-ab1234567890",
				expiresAt:          time.Unix(1615256178, 0).Add(SESSION_MAX_AGE_IN_SECONDS * time.Second),
				maxAgeSeconds:      86400,
				language:           LANG_EN,
				activeSolutionWord: puzzle.Word{'R', 'O', 'A', 'T', 'E'},
				pastWords:          []puzzle.Word{},
			},
		},
		// {
		// 	// todo // check out https://gist.github.com/jonnyreeves/17f91155a0d4a5d296d6 for inspiration
		// 	"test got cookie but no session corresponding session on server",
		// 	args{},
		// 	session{
		// 		id:            "12345678-abcd-1234-abcd-ab1234567890",
		// 		expiresAt:     time.Unix(1615256178, 0).Add(SESSION_MAX_AGE_IN_SECONDS * time.Second),
		// 		maxAgeSeconds: 120,
		// 		activeWord:    word{'R','O','A','T','E'},
		// 	},
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handleSession(tt.args.w, tt.args.req, tt.args.sessions, tt.args.wdb); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleSession() = %v, want %v", got, tt.want)
			}
		})
	}

	// fmt.Println("")
	// fmt.Println("foooooooooooooooo")
	// fmt.Println("")

	// t.Run("test", func(t *testing.T) {
	// 	// t.Errorf("fail %v", session{})
	// 	t.Errorf("fail %v", handleSession(httptest.NewRecorder(), httptest.NewRequest("get", "/", strings.NewReader("Hello, Reader!")), &sessions{}))
	// })
}

func Test_parseForm(t *testing.T) {
	type args struct {
		p            puzzle.Puzzle
		form         url.Values
		solutionWord puzzle.Word
		language     language
		wdb          wordDatabase
	}
	tests := []struct {
		name    string
		args    args
		want    puzzle.Puzzle
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "no hits, neither same or exact",
			// args: args{puzzle{}, url.Values{}, word{'M', 'I', 'S', 'S', 'S'}},
			args: args{
				p:            puzzle.Puzzle{},
				form:         url.Values{"r0": make([]string, 5)},
				solutionWord: puzzle.Word{'M', 'I', 'S', 'S', 'S'},
				language:     LANG_EN,
				wdb: wordDatabase{db: map[language]map[wordCollection]map[puzzle.Word]bool{
					LANG_EN: {
						WC_COMMON: {
							puzzle.Word{'m', 'i', 's', 's', 's'}: true,
							puzzle.Word{0, 0, 0, 0, 0}:           true, // equals make([]string, 5)
						},
						WC_ALL: {
							puzzle.Word{'m', 'i', 's', 's', 's'}: true,
							puzzle.Word{0, 0, 0, 0, 0}:           true, // equals make([]string, 5)
						},
					},
				}},
			},
			want: puzzle.Puzzle{
				Guesses: [6]puzzle.WordGuess{
					{
						puzzle.LetterGuess{Match: puzzle.MatchNone},
						puzzle.LetterGuess{Match: puzzle.MatchNone},
						puzzle.LetterGuess{Match: puzzle.MatchNone},
						puzzle.LetterGuess{Match: puzzle.MatchNone},
						puzzle.LetterGuess{Match: puzzle.MatchNone},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "full exact match",
			args: args{
				p:            puzzle.Puzzle{},
				form:         url.Values{"r0": []string{"M", "A", "T", "C", "H"}},
				solutionWord: puzzle.Word{'M', 'A', 'T', 'C', 'H'},
				language:     LANG_EN,
				wdb: wordDatabase{db: map[language]map[wordCollection]map[puzzle.Word]bool{
					LANG_EN: {
						WC_COMMON: {
							puzzle.Word{'m', 'a', 't', 'c', 'h'}: true,
						},
						WC_ALL: {
							puzzle.Word{'m', 'a', 't', 'c', 'h'}: true,
						},
					},
				}},
			},
			want: puzzle.Puzzle{Debug: "", Guesses: [6]puzzle.WordGuess{
				{
					{Letter: 'm', Match: puzzle.MatchExact},
					{Letter: 'a', Match: puzzle.MatchExact},
					{Letter: 't', Match: puzzle.MatchExact},
					{Letter: 'c', Match: puzzle.MatchExact},
					{Letter: 'h', Match: puzzle.MatchExact},
				},
			}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := parseForm(tt.args.p, tt.args.form, tt.args.solutionWord, tt.args.language, tt.args.wdb); !reflect.DeepEqual(got, tt.want) || (err != nil) != tt.wantErr {
				t.Errorf("parseForm() = %v, %v; want %v, %v", got, err != nil, tt.want, tt.wantErr)
			}
		})
	}
}

func Test_evaluateGuessedWord(t *testing.T) {
	type args struct {
		guessedWord  puzzle.Word
		solutionWord puzzle.Word
	}
	tests := []struct {
		name string
		args args
		want puzzle.WordGuess
	}{
		// test cases
		{
			name: "no hits, neither same or exact",
			args: args{
				guessedWord:  puzzle.Word{},
				solutionWord: puzzle.Word{'M', 'I', 'S', 'S', 'S'},
			},
			want: puzzle.WordGuess{
				{Match: puzzle.MatchNone},
				{Match: puzzle.MatchNone},
				{Match: puzzle.MatchNone},
				{Match: puzzle.MatchNone},
				{Match: puzzle.MatchNone},
			},
		},
		{
			name: "full exact match",
			args: args{
				guessedWord:  puzzle.Word{'m', 'a', 't', 'c', 'h'},
				solutionWord: puzzle.Word{'M', 'A', 'T', 'C', 'H'},
			},
			want: puzzle.WordGuess{
				{'m', puzzle.MatchExact},
				{'a', puzzle.MatchExact},
				{'t', puzzle.MatchExact},
				{'c', puzzle.MatchExact},
				{'h', puzzle.MatchExact},
			},
		},
		{
			name: "partial exact and partial some match",
			args: args{
				guessedWord:  puzzle.Word{'r', 'a', 'u', 'l', 'o'},
				solutionWord: puzzle.Word{'R', 'O', 'A', 'T', 'E'},
			},
			want: puzzle.WordGuess{
				{'r', puzzle.MatchExact},
				{'a', puzzle.MatchVague},
				{'u', puzzle.MatchNone},
				{'l', puzzle.MatchNone},
				{'o', puzzle.MatchVague},
			},
		},
		{
			name: "guessed word contains duplicats",
			args: args{
				guessedWord:  puzzle.Word{'r', 'o', 't', 'o', 'r'},
				solutionWord: puzzle.Word{'R', 'O', 'A', 'T', 'E'},
			},
			want: puzzle.WordGuess{
				{'r', puzzle.MatchExact},
				{'o', puzzle.MatchExact},
				{'t', puzzle.MatchVague},
				{'o', puzzle.MatchNone}, // both false bec we already found it or even already guesst the exact match
				{'r', puzzle.MatchNone}, // both false bec we already found it or even already guesst the exact match
			},
		},
		{
			name: "guessed word contains duplicats at end",
			args: args{
				guessedWord:  puzzle.Word{'i', 'x', 'i', 'i', 'i'},
				solutionWord: puzzle.Word{'L', 'X', 'I', 'I', 'I'},
			},
			want: puzzle.WordGuess{
				{'i', puzzle.MatchNone},
				{'x', puzzle.MatchExact},
				{'i', puzzle.MatchExact},
				{'i', puzzle.MatchExact},
				{'i', puzzle.MatchExact},
			},
		},
		{
			name: "guessed word contains duplicats at end fpp",
			args: args{
				guessedWord:  puzzle.Word{'l', 'i', 'i', 'i', 'i'},
				solutionWord: puzzle.Word{'I', 'L', 'X', 'I', 'I'},
			},
			want: puzzle.WordGuess{
				{'l', puzzle.MatchVague},
				{'i', puzzle.MatchVague},
				{'i', puzzle.MatchNone},
				{'i', puzzle.MatchExact},
				{'i', puzzle.MatchExact},
			},
		},
		// {
		// 	name: "target word contains duplicats / guessed word contains duplicats",
		// 	args: args{
		// 		puzzle.Puzzle{},
		// 		url.Values{"r0c0": []string{"M"}, "r0c1": []string{"A"}, "r0c2": []string{"T"}, "r0c3": []string{"C"}, "r0c4": []string{"H"}},
		// 		word{'M', 'A', 'T', 'C', 'H'},
		// 	},
		// 	want: puzzle.Puzzle{"", puzzle.wordGuess{
		// 		{
		// 			{'r', puzzle.LetterExact},
		// 			{'o', puzzle.LetterExact},
		// 			{'t', puzzle.LetterExact},
		// 			{'o', puzzle.LetterExact},
		// 			{'r', puzzle.LetterExact},
		// 		},
		// 	}},
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := evaluateGuessedWord(tt.args.guessedWord, tt.args.solutionWord); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("evaluateGuessedWord() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sessions_updateOrSet(t *testing.T) {
	type args struct {
		sess session
	}
	tests := []struct {
		name string
		ss   *sessions
		args args
		want sessions
	}{
		{
			"set new session",
			&sessions{},
			args{session{id: "foo"}},
			sessions{session{id: "foo"}},
		},
		{
			"update session",
			&sessions{session{id: "foo", maxAgeSeconds: 1}},
			args{session{id: "foo", maxAgeSeconds: 2}},
			sessions{session{id: "foo", maxAgeSeconds: 2}},
		},
		{
			"update session changes only correct session",
			&sessions{session{id: "foo"}, session{id: "bar"}, session{id: "baz", maxAgeSeconds: 1}, session{id: "foobar"}},
			args{session{id: "baz", maxAgeSeconds: 2}},
			sessions{session{id: "foo"}, session{id: "bar"}, session{id: "baz", maxAgeSeconds: 2}, session{id: "foobar"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ss.updateOrSet(tt.args.sess)
			if !reflect.DeepEqual((*tt.ss), tt.want) {
				t.Errorf("evaluateGuessedWord() = %v, want %v", tt.ss, tt.want)
			}
		})
	}
}

// todo: test for ???:
//   files, err := getAllFilenames(staticFS)
//   log.Printf("  debug fsys:\n    %v\n    %s\n", files, err)

func TestTemplateDataSuggest_validate(t *testing.T) {
	type fields struct {
		Word string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{name: "Suggested word match", fields: fields{Word: "gamer"}, wantErr: false},
		{name: "Suggested word match", fields: fields{Word: "GAMER"}, wantErr: false},
		{name: "Suggested word match", fields: fields{Word: "preuß"}, wantErr: false},
		{name: "Suggested word match", fields: fields{Word: "höste"}, wantErr: false},
		{name: "Suggested word match", fields: fields{Word: "hÖste"}, wantErr: false},
		{name: "Suggested word match", fields: fields{Word: "HÖSTE"}, wantErr: false},
		{name: "Suggested word match", fields: fields{Word: "fülle"}, wantErr: false},
		{name: "Suggested word match", fields: fields{Word: "FÜLLE"}, wantErr: false},
		{name: "Suggested word match", fields: fields{Word: "größe"}, wantErr: false},
		{name: "Suggested word match", fields: fields{Word: "GRÖßE"}, wantErr: false},

		{name: "Suggested word invalid (special chars: ?)", fields: fields{Word: "?????"}, wantErr: true},
		{name: "Suggested word invalid (special chars: ô)", fields: fields{Word: "grôss"}, wantErr: true},
		{name: "Suggested word invalid (special chars: emoji's (😁))", fields: fields{Word: "😁,😁,😁"}, wantErr: true},
		{name: "Suggested word invalid (word to short en)", fields: fields{Word: "tiny"}, wantErr: true},
		{name: "Suggested word invalid (word to short de)", fields: fields{Word: "kurz"}, wantErr: true},
		{name: "Suggested word invalid (word to long en)", fields: fields{Word: "toolong"}, wantErr: true},
		{name: "Suggested word invalid (word to long de)", fields: fields{Word: "zulang"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tds := TemplateDataSuggest{
				Word: tt.fields.Word,
			}
			if err := tds.validate(); (err != nil) != tt.wantErr {
				t.Errorf("TemplateDataSuggest.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
