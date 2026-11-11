package bench

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dirkjlouw/pyltjies-vision/internal/darts"
)

type stub map[string]string

func (s stub) Read(_ context.Context, photo []byte, _ string) (darts.Reading, error) {
	text, ok := s[string(photo)]
	if !ok {
		return darts.Reading{}, errors.New("rate limited")
	}
	visit, _ := darts.ParseVisit(text)
	var reading darts.Reading
	for _, d := range visit {
		reading.Darts = append(reading.Darts, darts.Seen{Dart: d})
	}
	return reading, nil
}

func write(t *testing.T, dir, name, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLabelsAndAFullRun(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "labels.csv", "file,darts\n# a comment\na.jpg,T20 S5 D16\nb.JPG,\"25, 50\"\nc.png,S1\nd.gif,S2\n")
	write(t, dir, "a.jpg", "A")
	write(t, dir, "b.JPG", "B")
	write(t, dir, "c.png", "C")
	write(t, dir, "d.gif", "D")
	photos, err := LoadLabels(filepath.Join(dir, "labels.csv"))
	if err != nil || len(photos) != 4 {
		t.Fatalf("photos %v %v", photos, err)
	}
	reader := stub{"A": "D16 T20 S5", "B": "50 S25 S3"}
	mimes := map[string]string{".jpg": "image/jpeg", ".png": "image/png"}
	var results []Result
	for _, photo := range photos {
		results = append(results, Run(context.Background(), reader, dir, photo, mimes))
	}
	if !results[0].Exact() || results[1].Exact() || results[2].Error != "rate limited" || !strings.Contains(results[3].Error, "unknown photo type") {
		t.Fatalf("results %+v", results)
	}
	s := Summarize(results)
	want := Summary{Photos: 4, Exact: 1, Failed: 2, RealDarts: 7, Matched: 5, Extras: 1}
	if s != want {
		t.Fatalf("summary %+v, want %+v", s, want)
	}
	if !strings.Contains(s.String(), "5 of 7 (71%)") {
		t.Errorf("report %q", s)
	}
	if !strings.HasPrefix(Line(results[1]), "miss") || !strings.Contains(Line(results[1]), "read 50 25 S3") {
		t.Errorf("line %q", Line(results[1]))
	}
}

func TestLabelsRefuseNonsense(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "bad.csv", "a.jpg,T20 Q7\n")
	if _, err := LoadLabels(filepath.Join(dir, "bad.csv")); err == nil || !strings.Contains(err.Error(), "line 1") {
		t.Fatalf("err %v", err)
	}
	write(t, dir, "empty.csv", "file,darts\n")
	if _, err := LoadLabels(filepath.Join(dir, "empty.csv")); err == nil {
		t.Fatal("an empty label file should fail")
	}
}
