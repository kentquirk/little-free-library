package date

import (
	"reflect"
	"testing"
)

func TestDate_CompareTo(t *testing.T) {
	tests := []struct {
		name  string
		date1 string
		date2 string
		want  int
	}{
		{"a", "2020", "2019", 1},
		{"b", "2020", "2020", 0},
		{"c", "2020", "2021", -1},
		{"d", "2020", "0000", 1},
		{"e", "2019", "2020", -1},
		{"f", "0000", "2019", -1},
		{"g", "0000", "0000", 0},
		{"h", "2020-2-3", "2019", 1},
		{"i", "2020-2-3", "2020", 0},
		{"j", "2020-2-3", "2021", -1},
		{"k", "2020", "2019-2-3", 1},
		{"l", "2020", "2020-2-3", 0},
		{"m", "2020", "2021-2-3", -1},
		{"n", "0000", "2019-2-3", -1},
		{"o", "2020-2-3", "2019-2-3", 1},
		{"p", "2020-12-3", "2020-2-3", 1},
		{"q", "2020-2-3", "2020-2-3", 0},
		{"r", "2020-2-3", "2020-12-3", -1},
		{"s", "2020-2-3", "2021-2-3", -1},
	}
	samesign := func(a, b int) bool {
		return a < 0 && b < 0 || a == 0 && b == 0 || a > 0 && b > 0
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d1, _ := ParseDate(tt.date1)
			d2, _ := ParseDate(tt.date2)
			if got := d1.CompareTo(d2); !samesign(got, tt.want) {
				t.Errorf("Date.CompareTo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseAllDates(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want []Date
	}{
		{"a", "2010", []Date{{Year: 2010}}},
		{"b", "  2010 2020", []Date{{Year: 2010}, {Year: 2020}}},
		{"c", "xyz 2010, 2011, 2012", []Date{{Year: 2010}, {Year: 2011}, {Year: 2012}}},
		{"d", "2010-12-13, 2011, 2012", []Date{{Year: 2010, Month: 12, Day: 13}, {Year: 2011}, {Year: 2012}}},
		{"e", "2010, 2011.7.18, 2012", []Date{{Year: 2010}, {Year: 2011, Month: 7, Day: 18}, {Year: 2012}}},
		{"f", "2010, xyz2011, 1977/10/30", []Date{{Year: 2010}, {Year: 1977, Month: 10, Day: 30}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseAllDates(tt.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseAllDates() = %v, want %v", got, tt.want)
			}
		})
	}
}
