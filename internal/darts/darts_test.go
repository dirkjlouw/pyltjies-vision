package darts

import (
	"encoding/json"
	"testing"
)

func TestParseAndString(t *testing.T) {
	cases := map[string]string{
		"T20": "T20", "t20": "T20", "20": "S20", "S5": "S5", "D 16": "D16",
		"25": "25", "SB": "25", "outer bull": "25", "50": "50", "Bull": "50", "DB": "50",
		"M": "M", "miss": "M", "0": "M",
	}
	for in, want := range cases {
		d, err := Parse(in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", in, err)
		}
		if d.String() != want {
			t.Errorf("Parse(%q) = %s, want %s", in, d, want)
		}
	}
	for _, bad := range []string{"", "T25", "21", "Q5", "D", "treble"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) should fail", bad)
		}
	}
}

func TestValue(t *testing.T) {
	for in, want := range map[string]int{"T20": 60, "D16": 32, "S1": 1, "25": 25, "50": 50, "M": 0} {
		d, _ := Parse(in)
		if d.Value() != want {
			t.Errorf("%s scores %d, want %d", in, d.Value(), want)
		}
	}
}

func TestParseVisit(t *testing.T) {
	visit, err := ParseVisit("T20, S5 D16")
	if err != nil || len(visit) != 3 || visit[2] != (Dart{16, 2}) {
		t.Fatalf("got %v %v", visit, err)
	}
	if _, err := ParseVisit("T20 X9"); err == nil {
		t.Fatal("a bad dart should fail the visit")
	}
}

func TestMatchedIgnoresOrderAndCountsRepeats(t *testing.T) {
	real, _ := ParseVisit("T20 T20 S1")
	read, _ := ParseVisit("S1 T20 S20")
	if got := Matched(real, read); got != 2 {
		t.Fatalf("matched %d, want 2", got)
	}
	read, _ = ParseVisit("T20 T20 T20 S1")
	if got := Matched(real, read); got != 3 {
		t.Fatalf("an extra dart can't match twice: %d", got)
	}
}

func TestJSONUsesTheShortForm(t *testing.T) {
	var r Reading
	if err := json.Unmarshal([]byte(`{"darts":[{"score":"bull","x":0.5,"y":0.5}]}`), &r); err != nil {
		t.Fatal(err)
	}
	out, _ := json.Marshal(r)
	if string(out) != `{"darts":[{"score":"50","x":0.5,"y":0.5}]}` {
		t.Fatalf("got %s", out)
	}
}
