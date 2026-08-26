package gnubgparser

import (
	"math"
	"os"
	"strings"
	"testing"
)

// countLuck reports how many parsed rolls carry a luck value, and how the
// values split by sign.
func countLuck(t *testing.T, filename string) (withLuck, positive, negative int) {
	t.Helper()
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", filename)
	}
	match, err := ParseSGFFile(filename)
	if err != nil {
		t.Fatalf("ParseSGFFile(%s): %v", filename, err)
	}
	for _, game := range match.Games {
		for _, mr := range game.Moves {
			if mr.Luck == nil {
				continue
			}
			withLuck++
			switch {
			case mr.Luck.Value > 0:
				positive++
			case mr.Luck.Value < 0:
				negative++
			}
		}
	}
	return withLuck, positive, negative
}

// TestParseLuckFromAnalysedFile is the regression this file exists for: gnuBG
// writes the luck of a roll as a single float, LU[-0.00537], and requiring a
// rating word in front of it made every real match parse as carrying no luck
// whatsoever — silently, since an absent LU is a legitimate state.
func TestParseLuckFromAnalysedFile(t *testing.T) {
	const filename = "test/charlot1-charlot2_7p_2025-11-08-2305_analysed.sgf"

	raw, err := os.ReadFile(filename)
	if err != nil {
		t.Skipf("Test file not readable: %v", err)
	}
	wantLU := strings.Count(string(raw), "LU[")
	if wantLU == 0 {
		t.Fatal("fixture carries no LU property; it cannot test luck parsing")
	}

	withLuck, positive, negative := countLuck(t, filename)
	if withLuck != wantLU {
		t.Errorf("rolls carrying luck: got %d, want %d (one per LU property in the file)",
			withLuck, wantLU)
	}
	// Luck is signed. A parser that dropped the sign would leave a match in
	// which nobody was ever unlucky.
	if positive == 0 || negative == 0 {
		t.Errorf("signed luck lost: %d lucky and %d unlucky rolls", positive, negative)
	}
}

// TestParseLuckUncomputedFile covers the other real shape of the property.
// gnuBG writes LU[-inf] — its ERR_VAL — for a match whose luck it never
// computed, and Go parses that string happily as -Inf. Handing callers an
// infinity to average is worse than handing them nothing, so those rolls must
// come back with no luck at all.
func TestParseLuckUncomputedFile(t *testing.T) {
	const filename = "test/charlot1-charlot2_7p_2025-11-08-2305.sgf"

	raw, err := os.ReadFile(filename)
	if err != nil {
		t.Skipf("Test file not readable: %v", err)
	}
	if !strings.Contains(string(raw), "LU[-inf]") {
		t.Skip("fixture no longer carries uncomputed luck")
	}

	withLuck, _, _ := countLuck(t, filename)
	if withLuck != 0 {
		t.Errorf("got %d rolls carrying luck, want none: LU[-inf] means gnuBG never computed it",
			withLuck)
	}
}

// TestParseLuckValueFormats covers the property text itself, including the
// two-field form kept for tolerance and the inputs that must yield no luck at
// all rather than a fabricated number.
func TestParseLuckValueFormats(t *testing.T) {
	cases := []struct {
		name       string
		lu         string
		wantOK     bool
		wantValue  float64
		wantRating string
	}{
		{"gnuBG's own format, negative", "-0.00537", true, -0.00537, ""},
		{"gnuBG's own format, positive", "0.21400", true, 0.21400, ""},
		{"a neutral roll is a value, not an absence", "0.00000", true, 0, ""},
		{"rating in front of the value is tolerated", "VeryGood 0.64000", true, 0.64000, "VeryGood"},
		{"gnuBG's ERR_VAL is not a measurement", "-inf", false, 0, ""},
		{"positive infinity either", "inf", false, 0, ""},
		{"nor is NaN", "nan", false, 0, ""},
		{"a non-numeric value yields no luck", "VeryGood", false, 0, ""},
		{"empty property yields no luck", "", false, 0, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			node := &SGFNode{Properties: map[string][]string{}}
			if c.lu != "" {
				node.Properties["LU"] = []string{c.lu}
			}
			var mr MoveRecord
			parseLuck(node, &mr)

			if !c.wantOK {
				if mr.Luck != nil {
					t.Fatalf("got luck %+v, want none", *mr.Luck)
				}
				return
			}
			if mr.Luck == nil {
				t.Fatalf("got no luck, want value %v", c.wantValue)
			}
			if math.Abs(mr.Luck.Value-c.wantValue) > 1e-9 {
				t.Errorf("value: got %v, want %v", mr.Luck.Value, c.wantValue)
			}
			if mr.Luck.Rating != c.wantRating {
				t.Errorf("rating: got %q, want %q", mr.Luck.Rating, c.wantRating)
			}
		})
	}
}
