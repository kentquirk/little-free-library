package books

import (
	"math/rand"
	"sync"
	"time"

	"github.com/kentquirk/little-free-library/pkg/booktypes"
)

// BookData is the type that we use to contain the book data and wrap all the queries.
// If we decide we want some sort of external data store, we can put it here.
// This is intended to be an opaque data structure; use accessors and query methods
// to retrieve data.
type BookData struct {
	mu      sync.RWMutex
	books   []booktypes.EBook
	bookIDs map[string]int
}

// NewBookData constructs a BookData object
func NewBookData() *BookData {
	return &BookData{
		books:   make([]booktypes.EBook, 0),
		bookIDs: make(map[string]int),
	}
}

func (b *BookData) updateIDs(start int) {
	for i := start; i < len(b.books); i++ {
		b.bookIDs[b.books[i].ID] = i
	}
}

// Add inserts one or more EBook entities into the BookData
func (b *BookData) Add(bs ...booktypes.EBook) {
	b.mu.Lock()
	defer b.mu.Unlock()
	start := len(b.books)
	b.books = append(b.books, bs...)
	b.updateIDs(start)
}

// Update replaces the entire contents of the BookData
func (b *BookData) Update(bs []booktypes.EBook) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.books = bs
	b.updateIDs(0)
}

// NBooks returns the number of books in the dataset
func (b *BookData) NBooks() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.books)
}

// Get retrieves a book by its ID, or returns false in its second argument.
// This currently searches linearly; could easily be sped up with an ID index.
func (b *BookData) Get(id string) (booktypes.EBook, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if ix, ok := b.bookIDs[id]; ok {
		return b.books[ix], true
	}
	return booktypes.EBook{}, false
}

// StatsData is the data structure used to return collection-level information
// about the data on hand.
type StatsData struct {
	TotalBooks   int            `json:"total_books"`
	TotalFiles   int            `json:"total_files"`
	AvgIndexSize float64        `json:"avg_index_size"`
	Languages    map[string]int `json:"languages"`
	Formats      map[string]int `json:"formats"`
	Types        map[string]int `json:"types"`
}

// Stats returns aggregated information about the data being stored.
func (b *BookData) Stats() StatsData {
	var totalWordsInIndex float64
	sd := StatsData{
		Languages: make(map[string]int),
		Formats:   make(map[string]int),
		Types:     make(map[string]int),
	}

	b.mu.RLock()
	defer b.mu.RUnlock()
	for i := range b.books {
		totalWordsInIndex += float64(b.books[i].Words.Length())
		sd.TotalBooks++
		lang := b.books[i].Language
		sd.Languages[lang]++
		sd.Types[b.books[i].Type]++
		for _, f := range b.books[i].Files {
			for _, fmt := range f.Formats {
				sd.TotalFiles++
				sd.Formats[fmt]++
			}
		}
	}
	sd.AvgIndexSize = totalWordsInIndex / float64(sd.TotalBooks)
	return sd
}

// Query does a query against the book data according to a ConstraintSpec.
// If the random flag is set, we choose a random subset of matching items.
//
// We want to select items fairly, so we use a replacement algorithm
// that adjusts the replacement probability based on the number of items
// that we have already seen.
// To choose n out of a stream of items, we generate the items one at a time,
// keeping the first n items in a set S.
// Then, when reading the m-th item I (m>n now), we keep it with probability n/m.
// When we keep it, we select item U uniformly at random from S, and replace
// U with I.
func (b *BookData) Query(constraints *ConstraintSpec) []booktypes.EBook {
	result := make([]booktypes.EBook, 0)

	// create the random number generator only if we need it
	var random *rand.Rand
	if constraints.Random {
		random = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	matchCount := 0
	replace := false
	include := constraints.IncludeCombiner(constraints.Includes...)
	exclude := constraints.ExcludeCombiner(constraints.Excludes...)

	b.mu.RLock()
	defer b.mu.RUnlock()
	for k := range b.books {
		if len(result) >= constraints.Limit {
			if !constraints.Random {
				break
			} else {
				replace = true
			}
		}
		// empty include list means include all; empty exclude list means exclude none
		if len(constraints.Includes) == 0 || include(b.books[k]) {
			if len(constraints.Excludes) == 0 || !exclude(b.books[k]) {
				matchCount++
				if !constraints.Random && matchCount < constraints.Limit*constraints.Page {
					continue
				}
				if replace {
					keep := (random.Float64() < (float64(constraints.Limit) / float64(matchCount)))
					if keep {
						randomIndex := random.Intn(constraints.Limit)
						result[randomIndex] = b.books[k]
					}
				} else {
					result = append(result, b.books[k])
				}
			}
		}
	}
	return result
}

// Count does a query against the book data according to a ConstraintSpec and returns the number
// of matching items (ignoring Limit and Random).
func (b *BookData) Count(constraints *ConstraintSpec) int {
	matchCount := 0
	include := constraints.IncludeCombiner(constraints.Includes...)
	exclude := constraints.ExcludeCombiner(constraints.Excludes...)

	b.mu.RLock()
	defer b.mu.RUnlock()
	for k := range b.books {
		// empty include list means include all; empty exclude list means exclude none
		if len(constraints.Includes) == 0 || include(b.books[k]) {
			if len(constraints.Excludes) == 0 || !exclude(b.books[k]) {
				matchCount++
			}
		}
	}
	return matchCount
}
