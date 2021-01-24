package books

import (
	"testing"
	"time"
)

// We don't need all the fields for our testing
func testEText() []EText {
	return []EText{
		{
			ID:       "a",
			Title:    "Evelyn's Story",
			Creators: []string{"Evelyn Excellent"},
			Language: "en",
			Subjects: []string{"Biography"},
			Issued:   time.Date(2005, 7, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:       "h",
			Title:    "Hamilton",
			Creators: []string{"Lin-Manuel Miranda"},
			Language: "rap",
			Subjects: []string{"History - Fiction", "History - Play", "Musical"},
			Issued:   time.Date(2016, 12, 25, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:           "w",
			Title:        "Wonder Women Play Through the Ages",
			Illustrators: []string{"Lynda Carter", "Gal Gadot"},
			Language:     "en",
			Subjects:     []string{"Comics -- Fiction"},
			Issued:       time.Date(2018, 10, 10, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:       "e",
			Title:    "The Woman's Music Bible",
			Creators: []string{"Eve"},
			Language: "en",
			Subjects: []string{"Music", "Religion"},
			Issued:   time.Date(1998, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
}

func TestConstraint_testCreator(t *testing.T) {
	data := testEText()
	tests := []struct {
		name string
		p    string
		want string
	}{
		{"1", "Eve", "e"},
		{"2", "Lin-Manuel", "h"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := testCreator(tt.p)
			result := ""
			for _, book := range data {
				if f(book) {
					result += book.ID
				}
			}
			if result != tt.want {
				t.Errorf("testCreator() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestConstraint_testIllustrator(t *testing.T) {
	data := testEText()
	tests := []struct {
		name string
		p    string
		want string
	}{
		{"1", "gal", "w"},
		{"2", "Miranda", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := testIllustrator(tt.p)
			result := ""
			for _, book := range data {
				if f(book) {
					result += book.ID
				}
			}
			if result != tt.want {
				t.Errorf("testIllustrator() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestConstraint_testSubject(t *testing.T) {
	data := testEText()
	tests := []struct {
		name string
		p    string
		want string
	}{
		{"1", "fiction", "hw"},
		{"2", "music", "e"},
		{"3", "Music", "e"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := testSubject(tt.p)
			result := ""
			for _, book := range data {
				if f(book) {
					result += book.ID
				}
			}
			if result != tt.want {
				t.Errorf("testSubject() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestConstraint_testTitle(t *testing.T) {
	data := testEText()
	tests := []struct {
		name string
		p    string
		want string
	}{
		{"1", "bible", "e"},
		{"2", "the", "we"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := testTitle(tt.p)
			result := ""
			for _, book := range data {
				if f(book) {
					result += book.ID
				}
			}
			if result != tt.want {
				t.Errorf("testTitle() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestConstraint_testLanguage(t *testing.T) {
	data := testEText()
	tests := []struct {
		name string
		p    string
		want string
	}{
		{"1", "en", "awe"},
		{"2", "rap", "h"},
		{"3", "fr", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := testLanguage(tt.p)
			result := ""
			for _, book := range data {
				if f(book) {
					result += book.ID
				}
			}
			if result != tt.want {
				t.Errorf("testLanguage() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestConstraint_testYear(t *testing.T) {
	data := testEText()
	tests := []struct {
		name string
		year string
		comp yearComparison
		want string
	}{
		{"1", "2005", yearEQ, "a"},
		{"2", "2005", yearLE, "ae"},
		{"3", "2016", yearGE, "hw"},
		{"4", "1980", yearLE, ""},
		{"5", "1980", yearGE, "ahwe"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := testYear(tt.year, tt.comp)
			result := ""
			for _, book := range data {
				if f(book) {
					result += book.ID
				}
			}
			if result != tt.want {
				t.Errorf("testYear() = %v, want %v", result, tt.want)
			}
		})
	}
}

// match tests

func TestConstraint_matchCreator(t *testing.T) {
	data := testEText()
	tests := []struct {
		name string
		p    string
		want string
	}{
		{"1", "Eve", "e"},
		{"2", "Eve_", "ae"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pat, err := createRegex(tt.p)
			if err != nil {
				t.Errorf("createRegex returned error %e", err)
			}
			f := matchCreator(pat)
			result := ""
			for _, book := range data {
				if f(book) {
					result += book.ID
				}
			}
			if result != tt.want {
				t.Errorf("matchCreator() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestConstraint_matchIllustrator(t *testing.T) {
	data := testEText()
	tests := []struct {
		name string
		p    string
		want string
	}{
		{"1", "_car_", "w"},
		{"2", "car", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pat, err := createRegex(tt.p)
			if err != nil {
				t.Errorf("createRegex returned error %e", err)
			}
			f := matchIllustrator(pat)
			result := ""
			for _, book := range data {
				if f(book) {
					result += book.ID
				}
			}
			if result != tt.want {
				t.Errorf("matchIllustrator() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestConstraint_matchSubject(t *testing.T) {
	data := testEText()
	tests := []struct {
		name string
		p    string
		want string
	}{
		{"1", "_music_", "he"},
		{"2", "_o_", "ahwe"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pat, err := createRegex(tt.p)
			if err != nil {
				t.Errorf("createRegex returned error %e", err)
			}
			f := matchSubject(pat)
			result := ""
			for _, book := range data {
				if f(book) {
					result += book.ID
				}
			}
			if result != tt.want {
				t.Errorf("matchSubject() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestConstraint_matchTitle(t *testing.T) {
	data := testEText()
	tests := []struct {
		name string
		p    string
		want string
	}{
		{"1", "Bible", ""},
		{"2", "_Bible_", "e"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pat, err := createRegex(tt.p)
			if err != nil {
				t.Errorf("createRegex returned error %e", err)
			}
			f := matchTitle(pat)
			result := ""
			for _, book := range data {
				if f(book) {
					result += book.ID
				}
			}
			if result != tt.want {
				t.Errorf("matchTitle() = %v, want %v", result, tt.want)
			}
		})
	}
}

// Combiners
func TestConstraint_Or(t *testing.T) {
	data := testEText()
	tests := []struct {
		name string
		f    ConstraintFunctor
		want string
	}{
		{"1", Or(testTitle("the"), testLanguage("rap")), "hwe"},
		{"2", Or(), ""},
		{"3", Or(testTitle("bible"), testTitle("music")), "e"},
		{"3", Or(testTitle("bible"), testTitle("Story")), "ae"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ""
			for _, book := range data {
				if tt.f(book) {
					result += book.ID
				}
			}
			if result != tt.want {
				t.Errorf("testTitle() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestConstraint_And(t *testing.T) {
	data := testEText()
	tests := []struct {
		name string
		f    ConstraintFunctor
		want string
	}{
		{"1", And(testTitle("the"), testLanguage("rap")), ""},
		{"2", And(), ""},
		{"3", And(testTitle("bible"), testTitle("music")), "e"},
		{"3", And(testTitle("bible"), testTitle("Story")), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ""
			for _, book := range data {
				if tt.f(book) {
					result += book.ID
				}
			}
			if result != tt.want {
				t.Errorf("testTitle() = %v, want %v", result, tt.want)
			}
		})
	}
}