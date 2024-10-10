package puzzle

import (
	"slices"

	"github.com/pandorasNox/lettr/pkg/language"
)

type GameState struct {
	puzzle               Puzzle
	activeSolutionWord   Word
	letterHints          []rune
	lastEvaluatedAttempt Puzzle
}

func NewGame(l language.Language, wdb WordDatabase, excludeWords []Word) GameState {
	newSolutionWord := wdb.RandomPickWithFallback(l, excludeWords, 0)

	return GameState{
		puzzle:               Puzzle{},
		activeSolutionWord:   newSolutionWord,
		letterHints:          []rune{},
		lastEvaluatedAttempt: Puzzle{},
	}
}

func (g *GameState) ActiveSolutionWord() Word {
	return g.activeSolutionWord
}

func (g *GameState) SetActiveSolutionWord(w Word) {
	g.activeSolutionWord = w
}

func (g *GameState) LetterHints() []rune {
	return slices.Clone(g.letterHints)
}

func (g *GameState) AddLetterHint(l rune) {
	g.letterHints = append(g.letterHints, l)
}

func (g *GameState) LastEvaluatedAttempt() Puzzle {
	return g.lastEvaluatedAttempt
}

func (g *GameState) SetLastEvaluatedAttempt(p Puzzle) {
	g.lastEvaluatedAttempt = p
}
