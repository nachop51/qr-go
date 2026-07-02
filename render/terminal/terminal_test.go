package terminal

import (
	"bytes"
	"strings"
	"testing"
)

// fakeGrid satisfies render.Grid structurally — no import needed.
type fakeGrid struct{ n int }

func (f fakeGrid) Size() int            { return f.n }
func (f fakeGrid) IsDark(x, y int) bool { return (x+y)%2 == 0 }

func TestTerminalRender(t *testing.T) {
	var buf bytes.Buffer
	if err := New().Writer(&buf).Quiet(1).Render(fakeGrid{n: 3}); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if want := 3 + 2*1; len(lines) != want { // size + 2*quiet rows
		t.Fatalf("want %d rows, got %d:\n%s", want, len(lines), out)
	}
	if !strings.Contains(out, "██") {
		t.Fatalf("expected dark modules:\n%s", out)
	}
}
