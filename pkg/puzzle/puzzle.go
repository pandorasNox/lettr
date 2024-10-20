package puzzle

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
	guessedLetterCountMap := make(map[rune]int)

	resultWordGuess := WordGuess{}

	// initilize
	for i, gr := range guessedWord {
		resultWordGuess[i].Letter = gr
		resultWordGuess[i].Match = MatchNone
	}

	// mark exact matches
	for i, gr := range guessedWord {
		exact := solutionWord[i] == gr

		if exact {
			guessedLetterCountMap[gr]++
			resultWordGuess[i].Match = MatchExact
		}
	}

	// mark some/vague matches
	for i, gr := range guessedWord {
		if resultWordGuess[i].Match == MatchExact {
			continue
		}

		some := solutionWord.Contains(gr)

		if !(resultWordGuess[i].Match == MatchVague) || some {
			guessedLetterCountMap[gr]++
		}

		s := some && (guessedLetterCountMap[gr] <= solutionWord.Count(gr))
		if s {
			resultWordGuess[i].Match = MatchVague
		}
	}

	return resultWordGuess
}
