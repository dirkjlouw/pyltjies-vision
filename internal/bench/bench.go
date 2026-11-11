// Package bench scores a reader against photos whose darts are known, so any
// reader, Gemini today or our own model later, is judged on the same photos.
package bench

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dirkjlouw/pyltjies-vision/internal/darts"
)

// Reader is anything that reads darts off a photo.
type Reader interface {
	Read(ctx context.Context, photo []byte, mime string) (darts.Reading, error)
}

// Photo is one labelled photo: the file and the darts really in the board.
type Photo struct {
	File string
	Real []darts.Dart
}

// LoadLabels reads labels.csv: one line per photo, the file name and the darts
// in it, such as `board-01.jpg,T20 S5 D16`. Lines starting with # are skipped,
// and so is a header line starting with "file".
func LoadLabels(path string) ([]Photo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comment = '#'
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true
	var photos []Photo
	for line := 1; ; line++ {
		record, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(record) == 0 || strings.TrimSpace(record[0]) == "" {
			continue
		}
		if line == 1 && strings.EqualFold(strings.TrimSpace(record[0]), "file") {
			continue
		}
		if len(record) < 2 {
			return nil, fmt.Errorf("%s line %d: needs a file and its darts", path, line)
		}
		real, err := darts.ParseVisit(record[1])
		if err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, line, err)
		}
		photos = append(photos, Photo{File: strings.TrimSpace(record[0]), Real: real})
	}
	if len(photos) == 0 {
		return nil, fmt.Errorf("%s has no photos", path)
	}
	return photos, nil
}

// Result is how a reader did on one photo.
type Result struct {
	File    string        `json:"file"`
	Real    []darts.Dart  `json:"real"`
	Reading darts.Reading `json:"reading"`
	Matched int           `json:"matched"`
	Error   string        `json:"error,omitempty"`
}

// Exact is true when every dart was read right and nothing extra was found.
func (r Result) Exact() bool {
	return r.Error == "" && r.Matched == len(r.Real) && len(r.Reading.Darts) == len(r.Real)
}

// Score works out a result from what the reader returned.
func Score(photo Photo, reading darts.Reading, err error) Result {
	result := Result{File: photo.File, Real: photo.Real, Reading: reading}
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Matched = darts.Matched(photo.Real, reading.Scores())
	return result
}

// Run reads one photo from dir and scores it.
func Run(ctx context.Context, reader Reader, dir string, photo Photo, mimeTypes map[string]string) Result {
	mime, ok := mimeTypes[strings.ToLower(filepath.Ext(photo.File))]
	if !ok {
		return Score(photo, darts.Reading{}, fmt.Errorf("unknown photo type %q", filepath.Ext(photo.File)))
	}
	data, err := os.ReadFile(filepath.Join(dir, photo.File))
	if err != nil {
		return Score(photo, darts.Reading{}, err)
	}
	reading, err := reader.Read(ctx, data, mime)
	return Score(photo, reading, err)
}

// Summary adds up a run.
type Summary struct {
	Photos, Exact, Failed      int
	RealDarts, Matched, Extras int
}

// Summarize adds up results. Extras are darts read that don't match a real one:
// a wrong score or a dart that isn't there.
func Summarize(results []Result) Summary {
	var s Summary
	for _, r := range results {
		s.Photos++
		s.RealDarts += len(r.Real)
		if r.Error != "" {
			s.Failed++
			continue
		}
		s.Matched += r.Matched
		s.Extras += len(r.Reading.Darts) - r.Matched
		if r.Exact() {
			s.Exact++
		}
	}
	return s
}

func percent(part, whole int) string {
	if whole == 0 {
		return "-"
	}
	return fmt.Sprintf("%.0f%%", 100*float64(part)/float64(whole))
}

// String is the report line: the darts figure is the one to watch against the
// 60% target.
func (s Summary) String() string {
	return fmt.Sprintf("Darts read right: %d of %d (%s). Photos fully right: %d of %d (%s). Wrong or extra darts: %d. Failed calls: %d.",
		s.Matched, s.RealDarts, percent(s.Matched, s.RealDarts), s.Exact, s.Photos, percent(s.Exact, s.Photos), s.Extras, s.Failed)
}

// Line describes one result for the report.
func Line(r Result) string {
	real := visit(r.Real)
	if r.Error != "" {
		return fmt.Sprintf("FAIL  %-24s real %-14s %s", r.File, real, r.Error)
	}
	mark := "ok  "
	if !r.Exact() {
		mark = "miss"
	}
	return fmt.Sprintf("%s  %-24s real %-14s read %s", mark, r.File, real, visit(r.Reading.Scores()))
}

func visit(ds []darts.Dart) string {
	parts := make([]string, len(ds))
	for i, d := range ds {
		parts[i] = d.String()
	}
	if len(parts) == 0 {
		return "(none)"
	}
	return strings.Join(parts, " ")
}
