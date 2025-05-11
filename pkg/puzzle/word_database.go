package puzzle

import (
	"bufio"
	"fmt"
	iofs "io/fs"
	"math/rand"
	"slices"
	"time"

	"github.com/pandorasNox/lettr/pkg/language"
)

type WordCollection string

const (
	WC_ALL    WordCollection = "wc_all"
	WC_COMMON WordCollection = "wc_common"
)

type WordDatabase struct {
	Db map[language.Language]map[WordCollection]map[Word]bool
}

func (wdb *WordDatabase) Init(fs iofs.FS, filePathsByLanguage map[language.Language]map[WordCollection][]string) error {
	wdb.Db = make(map[language.Language]map[WordCollection]map[Word]bool)

	for l, collectionFilePaths := range filePathsByLanguage {
		wdb.Db[l] = make(map[WordCollection]map[Word]bool)

		for c, paths := range collectionFilePaths {
			wdb.Db[l][c] = make(map[Word]bool)

			for _, path := range paths {
				f, err := fs.Open(path)
				if err != nil {
					return fmt.Errorf("wordDatabase init failed when opening file: %s", err)
				}
				defer f.Close()

				fInfo, err := f.Stat()
				if err != nil {
					return fmt.Errorf("wordDatabase init failed when obtaining stat: %s", err)
				}

				var allowedSize int64 = 2 * 1024 * 1024 // 2 MB
				if fInfo.Size() > allowedSize {
					return fmt.Errorf("wordDatabase init failed with forbidden file size: path='%s', size='%d'", path, fInfo.Size())
				}

				scanner := bufio.NewScanner(f)
				line := 0
				for scanner.Scan() {
					if line == 0 { // skip first metadata line
						line++
						continue
					}

					candidate := scanner.Text()
					word, err := toWord(candidate)
					if err != nil {
						return fmt.Errorf("wordDatabase init, couldn't parse line to word: line='%s', err=%s", candidate, err)
					}

					wdb.Db[l][c][word.ToLower()] = true

					line++
				}
				if err := scanner.Err(); err != nil {
					return fmt.Errorf("wordDatabase init failed scanning file with: path='%s', err=%s", path, err)
				}
			}
		}

		for w := range wdb.Db[l][WC_COMMON] {
			wdb.Db[l][WC_ALL][w] = true
		}
	}

	return nil
}

func (wdb WordDatabase) Exists(l language.Language, w Word) bool {
	db, ok := wdb.Db[l]
	if !ok {
		return false
	}

	db_c, ok := db[WC_ALL]
	if !ok {
		return false
	}

	_, ok = db_c[w.ToLower()]
	return ok
}

func (wdb WordDatabase) RandomPick(l language.Language, avoidList []Word, retryAkkumulator uint8) (Word, error) {
	const MAX_RETRY uint8 = 10

	if retryAkkumulator > MAX_RETRY {
		return Word{}, fmt.Errorf("RandomPick exceeded retries: retryAkkumulator='%d' | MAX_RETRY='%d'", retryAkkumulator, MAX_RETRY)
	}

	db, ok := wdb.Db[l]
	if !ok {
		return Word{}, fmt.Errorf("RandomPick failed with unknown language: '%s'", l)
	}

	collection := WC_COMMON
	db_c, ok := db[collection]
	if !ok {
		collection = WC_ALL

		db_c, ok = db[collection]
		if !ok {
			return Word{}, fmt.Errorf("RandomPick with lang '%s' failed with unknown collection: '%s'", l, collection)
		}
	}

	randsource := rand.NewSource(time.Now().UnixNano())
	randgenerator := rand.New(randsource)
	rolledLine := randgenerator.Intn(len(db_c))

	currentLine := 0
	for w := range db_c {
		if currentLine == rolledLine {

			wordContained := slices.ContainsFunc(avoidList, func(wo Word) bool {
				return w.IsEqual(wo)
			})
			if wordContained {
				return wdb.RandomPick(l, avoidList, retryAkkumulator+1)
			}

			return w, nil
		}

		currentLine++
	}

	return Word{}, fmt.Errorf("RandomPick could not find random line aka this should never happen ^^")
}

func (wdb WordDatabase) RandomPickWithFallback(l language.Language, avoidList []Word, retryAkkumulator uint8) Word {
	w, err := wdb.RandomPick(l, avoidList, retryAkkumulator)
	if err != nil {
		return Word{'R', 'O', 'A', 'T', 'E'}.ToLower()
	}

	return w.ToLower()
}

func FilePathsByLang() map[language.Language]map[WordCollection][]string {
	return map[language.Language]map[WordCollection][]string{
		language.LANG_EN: {
			WC_ALL: {
				"configs/corpora-eng_news_2023_10K-export.txt",
				"configs/en-en.words.v2.txt",
			},
			WC_COMMON: {
				"configs/corpora-eng_news_2023_10K-export.txt",
				"configs/github.com.ajeetdsouza.clidle.words.go.txt",
			},
		},
		language.LANG_DE: {
			WC_ALL: {
				"configs/corpora-deu_news_2023_10K-export.txt",
				"configs/de-de.words.v2.txt",
			},
			WC_COMMON: {
				"configs/corpora-deu_news_2023_10K-export.txt",
			},
		},
	}
}
