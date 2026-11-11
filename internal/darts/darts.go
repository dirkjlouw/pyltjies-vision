// Package darts holds one dart's score in the same shape the Pyltjies app uses:
// a number n (0 for a miss, 1 to 20, or 25 for the bull) and a multiplier m.
package darts

import (
	"fmt"
	"strconv"
	"strings"
)

// Dart is where one dart landed. The inner bull is {25, 2} and scores 50.
type Dart struct {
	N int
	M int
}

// Miss is a dart in the board outside the double ring, or one that isn't there.
var Miss = Dart{}

// Value is the points the dart scores.
func (d Dart) Value() int { return d.N * d.M }

// String gives the short form used everywhere here: S20, D16, T19, 25, 50 or M.
func (d Dart) String() string {
	switch {
	case d.N == 0:
		return "M"
	case d.N == 25 && d.M == 2:
		return "50"
	case d.N == 25:
		return "25"
	}
	return string("SDT"[d.M-1]) + strconv.Itoa(d.N)
}

// Parse reads a dart in the short form, plus the common ways people and models
// write it: "20" for a single, "SB", "OB" or "outer" for 25, "DB", "bull" for 50.
func Parse(text string) (Dart, error) {
	s := strings.ToUpper(strings.Join(strings.Fields(text), ""))
	switch s {
	case "M", "MISS", "0", "S0":
		return Miss, nil
	case "25", "SB", "OB", "OUTER", "OUTERBULL", "S25", "SBULL":
		return Dart{25, 1}, nil
	case "50", "DB", "IB", "BULL", "BULLSEYE", "INNERBULL", "D25", "DBULL":
		return Dart{25, 2}, nil
	}
	m := 1
	if s != "" {
		if i := strings.IndexByte("SDT", s[0]); i >= 0 {
			m, s = i+1, s[1:]
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 20 {
		return Miss, fmt.Errorf("not a dart: %q", text)
	}
	return Dart{n, m}, nil
}

// ParseVisit reads darts separated by spaces or commas, such as "T20 S5 D16".
func ParseVisit(text string) ([]Dart, error) {
	fields := strings.FieldsFunc(text, func(r rune) bool { return r == ' ' || r == ',' || r == ';' })
	visit := make([]Dart, 0, len(fields))
	for _, field := range fields {
		d, err := Parse(field)
		if err != nil {
			return nil, err
		}
		visit = append(visit, d)
	}
	return visit, nil
}

// MarshalText writes the short form, so JSON carries "T20".
func (d Dart) MarshalText() ([]byte, error) { return []byte(d.String()), nil }

// UnmarshalText accepts anything Parse does.
func (d *Dart) UnmarshalText(b []byte) error {
	parsed, err := Parse(string(b))
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// Matched counts how many of the read darts agree with the real ones, ignoring
// order, as a photo shows darts all at once.
func Matched(real, read []Dart) int {
	left := map[Dart]int{}
	for _, d := range real {
		left[d]++
	}
	matched := 0
	for _, d := range read {
		if left[d] > 0 {
			left[d]--
			matched++
		}
	}
	return matched
}

// Seen is one dart a reader found on a photo. X and Y are the tip's position as
// a fraction of the photo's width and height from the top left. That position,
// once a player has corrected it, is what trains a model.
type Seen struct {
	Dart Dart    `json:"score"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Sure float64 `json:"sure,omitempty"`
}

// Reading is everything a reader returns for one photo.
type Reading struct {
	Darts []Seen `json:"darts"`
	Notes string `json:"notes,omitempty"`
}

// Scores lists the darts without positions.
func (r Reading) Scores() []Dart {
	scores := make([]Dart, len(r.Darts))
	for i, seen := range r.Darts {
		scores[i] = seen.Dart
	}
	return scores
}
