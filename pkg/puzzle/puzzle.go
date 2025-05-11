package puzzle

import "github.com/pandorasNox/lettr/pkg/assert"

type Puzzle struct {
	Guesses [6]WordGuess
}

func (p Puzzle) ActiveRow() uint8 {
	for i, wg := range p.Guesses {
		if !wg.isFilled() {
			return uint8(i)
		}
	}

	return uint8(len(p.Guesses))
}

func (p Puzzle) IsSolved() bool {
	if p.ActiveRow() > 0 {
		return p.Guesses[p.ActiveRow()-1].isSolved()
	}

	return false
}

func (p Puzzle) IsLoose() bool {
	for _, wg := range p.Guesses {
		if !wg.isFilled() || wg.isSolved() {
			return false
		}
	}

	return true
}

func (p Puzzle) LetterGuesses() []LetterGuess {
	lgCollector := []LetterGuess{}

	for _, wg := range p.Guesses {
		if wg.isFilled() {
			lgCollector = append(lgCollector, wg.LetterGuesses()...)
		}
	}

	return lgCollector
}

type WordGuess [5]LetterGuess

func (wg WordGuess) isFilled() bool {
	for _, l := range wg {
		if l.Letter == 0 || l.Letter == 65533 {
			return false
		}
	}

	return true
}

func (wg WordGuess) isSolved() bool {
	for _, lg := range wg {
		if lg.Match != MatchExact {
			return false
		}
	}

	return true
}

func (wg WordGuess) LetterGuesses() []LetterGuess {
	s := []LetterGuess{}

	if !wg.isFilled() {
		return s
	}

	for _, lg := range wg {
		s = append(s, lg)
	}

	return s
}

type LetterGuess struct {
	Letter rune
	Match  Match
}

func EvaluateGuessedWord(guessedWord Word, solutionWord Word) WordGuess {
	solutionWord = solutionWord.ToLower()
	solutionLettersCountMap := make(map[rune]int) // Tracks letter counts in solution
	guessEvaluation := WordGuess{}

	// Count letter occurrences in solution word
	for _, char := range solutionWord {
		solutionLettersCountMap[char]++
	}

	// initilize resultWordGuess
	for i, guessLetter := range guessedWord {
		guessEvaluation[i].Letter = guessLetter
		guessEvaluation[i].Match = MatchNone
	}

	// mark exact matches
	for i, guessLetter := range guessedWord {
		if solutionWord[i] == guessLetter {
			guessEvaluation[i].Match = MatchExact
			solutionLettersCountMap[guessLetter]--
		}
	}

	// mark vague matches
	for i, guessLetter := range guessedWord {
		if guessEvaluation[i].Match == MatchExact {
			continue
		}
		assert.Assert(guessEvaluation[i].Match == MatchNone, "guessedWord letter match entry should be None")

		if solutionLettersCountMap[guessLetter] > 0 {
			guessEvaluation[i].Match = MatchVague
			solutionLettersCountMap[guessLetter]--
		}
	}

	return guessEvaluation
}
