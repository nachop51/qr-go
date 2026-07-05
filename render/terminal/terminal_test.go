package terminal

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nachop51/qr-go/render"
)

// fakeGrid satisfies render.Grid structurally — no import needed.
type fakeGrid struct{ n int }

func (f fakeGrid) Size() int            { return f.n }
func (f fakeGrid) IsDark(x, y int) bool { return (x+y)%2 == 0 }

// The default (half-block) style packs two module rows per text row, so the
// output has ceil(total/2) lines and uses half-block glyphs.
func TestTerminalHalfBlockDefault(t *testing.T) {
	var buf bytes.Buffer
	if err := New().Writer(&buf).Quiet(1).Render(fakeGrid{n: 3}); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	total := 3 + 2*1        // size + 2*quiet module rows
	want := (total + 1) / 2 // two module rows per text row
	if len(lines) != want {
		t.Fatalf("want %d text rows, got %d:\n%s", want, len(lines), out)
	}

	if !strings.ContainsAny(out, string([]rune{glyphFull, glyphUpper, glyphLower})) {
		t.Fatalf("expected half-block glyphs:\n%s", out)
	}
	if strings.Contains(out, "██") {
		t.Fatalf("half-block output should not contain full-width block fills:\n%s", out)
	}
}

// Block style keeps one text row per module row and uses the fill strings.
func TestTerminalBlockMode(t *testing.T) {
	var buf bytes.Buffer
	if err := New().Writer(&buf).Quiet(1).Block().Render(fakeGrid{n: 3}); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if want := 3 + 2*1; len(lines) != want {
		t.Fatalf("want %d rows, got %d:\n%s", want, len(lines), out)
	}
	if !strings.Contains(out, "██") {
		t.Fatalf("expected full-block dark modules:\n%s", out)
	}
}

// Custom fill strings imply Block style.
func TestTerminalDarkImpliesBlock(t *testing.T) {
	var buf bytes.Buffer
	if err := New().Writer(&buf).Quiet(0).Dark("XX").Light("..").Render(fakeGrid{n: 3}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "XX") || !strings.Contains(out, "..") {
		t.Fatalf("expected custom fills XX/.. :\n%s", out)
	}
}

// Invert swaps which modules are drawn as ink; the two outputs must differ and
// a fully light quiet border must flip to ink.
func TestTerminalInvert(t *testing.T) {
	var normal, inverted bytes.Buffer
	if err := New().Writer(&normal).Quiet(1).Render(fakeGrid{n: 3}); err != nil {
		t.Fatal(err)
	}
	if err := New().Writer(&inverted).Quiet(1).Invert().Render(fakeGrid{n: 3}); err != nil {
		t.Fatal(err)
	}
	if normal.String() == inverted.String() {
		t.Fatal("Invert() produced identical output")
	}
}

func TestTerminal_renderBlock(t *testing.T) {
	tests := []struct {
		name string
		g    render.Grid
		want string
	}{
		// fakeGrid.IsDark(0,0) == true; with quiet 0 that's one dark fill + newline.
		{"single dark module", fakeGrid{n: 1}, "██\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New().Quiet(0).renderBlock(tt.g); got != tt.want {
				t.Errorf("renderBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}
