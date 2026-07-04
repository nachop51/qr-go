package spec

import (
	"reflect"
	"testing"
)

func TestCharValue(t *testing.T) {
	cases := map[byte]int{
		'0': 0, '9': 9,
		'A': 10, 'Z': 35,
		' ': 36, '$': 37, '%': 38, '*': 39,
		'+': 40, '-': 41, '.': 42, '/': 43, ':': 44,
		'a': -1, '!': -1, // not in the alphanumeric charset
	}
	for c, want := range cases {
		if got := CharValue(c); got != want {
			t.Errorf("CharValue(%q) = %d, want %d", c, got, want)
		}
	}
}

// Modules per side is 17 + 4*version for every version 1..40.
func TestModules(t *testing.T) {
	for v := 1; v <= MaxVersion(); v++ {
		if got, want := Modules(v), 17+4*v; got != want {
			t.Errorf("Modules(%d) = %d, want %d", v, got, want)
		}
	}
}

// Data codewords + EC codewords must equal the total codewords, for every
// version and error-correction level.
func TestCodewordConservation(t *testing.T) {
	for v := 1; v <= MaxVersion(); v++ {
		for level := 0; level < 4; level++ {
			data := DataCodewords(v, level)
			ec := ECCodewords(v, level)
			total := TotalCodewords(v)
			if data+ec != total {
				t.Errorf("v%d L%d: data(%d)+ec(%d)=%d, want total %d",
					v, level, data, ec, data+ec, total)
			}
			if data <= 0 {
				t.Errorf("v%d L%d: non-positive data codewords %d", v, level, data)
			}
		}
	}
}

func TestMaxVersion(t *testing.T) {
	if MaxVersion() != 40 {
		t.Fatalf("MaxVersion = %d, want 40", MaxVersion())
	}
}

// Alignment pattern centre coordinates, cross-checked against the QR standard
// (ISO/IEC 18004) Annex E.
func TestAlignmentCoords(t *testing.T) {
	cases := map[int][]int{
		1:  nil,
		2:  {6, 18},
		6:  {6, 34},
		7:  {6, 22, 38},
		10: {6, 28, 50},
		14: {6, 26, 46, 66},
		20: {6, 34, 62, 90},
		32: {6, 34, 60, 86, 112, 138},
		40: {6, 30, 58, 86, 114, 142, 170},
	}
	for v, want := range cases {
		if got := AlignmentCoords(v); !reflect.DeepEqual(got, want) {
			t.Errorf("AlignmentCoords(%d) = %v, want %v", v, got, want)
		}
	}
}

// The format information region is always 15 bits, placed twice.
func TestFormatModules(t *testing.T) {
	for v := 1; v <= MaxVersion(); v++ {
		if got := len(FormatModules(v)); got != 30 {
			t.Errorf("FormatModules(%d) has %d positions, want 30", v, got)
		}
	}
}

// Version information exists only for version >= 7 (18 bits, placed twice).
func TestVersionModules(t *testing.T) {
	for v := 1; v < 7; v++ {
		if got := VersionModules(v); got != nil {
			t.Errorf("VersionModules(%d) = %v, want nil", v, got)
		}
	}
	for v := 7; v <= MaxVersion(); v++ {
		if got := len(VersionModules(v)); got != 36 {
			t.Errorf("VersionModules(%d) has %d positions, want 36", v, got)
		}
	}
}

func TestHasNonASCII(t *testing.T) {
	if HasNonASCII([]byte("HELLO WORLD 123")) {
		t.Error("ASCII text reported as non-ASCII")
	}
	if !HasNonASCII([]byte("café")) {
		t.Error("non-ASCII text reported as ASCII")
	}
}

func TestDarkModule(t *testing.T) {
	// The dark module always sits at (8, 4*version+9).
	for _, v := range []int{1, 7, 40} {
		x, y := DarkModule(v)
		if x != 8 || y != 4*v+9 {
			t.Errorf("DarkModule(%d) = (%d,%d), want (8,%d)", v, x, y, 4*v+9)
		}
	}
}
