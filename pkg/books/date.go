package books

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// Date represents a date without time information; it can represent only a year.
type Date struct {
	Year  int
	Month int
	Day   int
}

// AsTime converts the date object into the best representation of a time.Time
func (d Date) AsTime() time.Time {
	switch {
	case d.Year == 0:
		return time.Time{}
	case d.Month == 0, d.Day == 0:
		return time.Date(d.Year, 1, 1, 0, 0, 0, 0, time.UTC)
	default:
		return time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC)
	}
}

// ToString returns a string representation of the date to the appropriate level of precision.
func (d Date) ToString() string {
	switch {
	case d.Year == 0:
		return "N/A"
	case d.Month == 0, d.Day == 0:
		return strconv.Itoa(d.Year)
	default:
		return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
	}
}

// CompareTo compares two date objects to an appropriate level of precision,
// and returns <0 if the receiver is less than other, >0 if it's greater than,
// and 0 if they're equal (at the level to which they can be compared).
// If one of the dates has different precision than the other, they are only compared
// to the lesser precision. An empty Date is considered to be less than any non-empty
// date and equal to itself.
func (d Date) CompareTo(other Date) int {
	switch {
	case d.Year == 0:
		if other.Year == 0 {
			return 0
		}
		return -1
	case d.Month == 0, d.Day == 0:
		return d.Year - other.Year
	default:
		switch {
		case other.Year == 0:
			return 1
		case other.Month == 0, other.Day == 0:
			return d.Year - other.Year
		default:
			if other.Year != d.Year {
				return d.Year - other.Year
			}
			if other.Month != d.Month {
				return d.Month - other.Month
			}
			return d.Day - other.Day
		}
	}
}

// IsZero returns true if the Date object is the zero value
func (d Date) IsZero() bool {
	return d.Year == 0 && d.Month == 0 && d.Day == 0
}

// AsDate converts a time.Time into a Date object.
func AsDate(t time.Time) Date {
	return Date{
		Year:  t.Year(),
		Month: int(t.Month()),
		Day:   t.Day(),
	}
}

// ParseDate parses a string and looks for the first thing in it that could be a date.
// If none are found, it returns a zero date.
// It also returns an index into the string pointing past the date that was found.
// If no date was found, the index is 0.
// The regex and logic are fairly finicky, which avoids lots of cases perhaps at
// the expense of clarity.
func ParseDate(s string) (Date, int) {
	// look for a 4-digit year that is not part of a longer string
	patYMD := regexp.MustCompile(`\b([0-9]{4})([./-]([0-9]{1,2})[./-]([0-9]{1,2}))?\b`)
	ixs := patYMD.FindStringSubmatchIndex(s)
	if len(ixs) != 0 {
		y, _ := strconv.Atoi(s[ixs[2]:ixs[3]])
		if ixs[4] != -1 {
			m, _ := strconv.Atoi(s[ixs[6]:ixs[7]])
			d, _ := strconv.Atoi(s[ixs[8]:ixs[9]])
			return Date{y, m, d}, ixs[1]
		}
		return Date{Year: y}, ixs[1]
	}
	return Date{}, 0
}

// ParseAllDates returns a slice of Date objects found in the given string.
func ParseAllDates(s string) []Date {
	dates := make([]Date, 0)
	for {
		d, ix := ParseDate(s)
		if ix == 0 {
			break
		}
		dates = append(dates, d)
		s = s[ix:]
	}
	return dates
}

// MarshalJSON implements json.Marshaler
func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.ToString())
}
