package svg

import (
	"bytes"
	"strings"
	"testing"
)

type fakeGrid struct{ n int }

func (f fakeGrid) Size() int            { return f.n }
func (f fakeGrid) IsDark(x, y int) bool { return (x+y)%2 == 0 }

func TestSVGRender(t *testing.T) {
	var buf bytes.Buffer

	err := New().Writer(&buf).Quiet(1).Module(2).Render(fakeGrid{n: 3})
	if err != nil {
		t.Fatal(err)
	}

	out := buf.String()

	if !strings.Contains(out, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10" width="10" height="10" shape-rendering="crispEdges">`) {
		t.Fatalf("expected svg root with dimensions, got: %s", out)
	}
	if !strings.Contains(out, `<rect width="10" height="10" fill="#ffffff"/>`) {
		t.Fatalf("expected background rect, got: %s", out)
	}
	if !strings.Contains(out, `<rect x="2" y="2" width="2" height="2" fill="#000000"/>`) {
		t.Fatalf("expected dark module at first quiet-zone offset, got: %s", out)
	}
	if got := strings.Count(out, `fill="#000000"`); got != 5 {
		t.Fatalf("expected 5 dark modules, got %d: %s", got, out)
	}
}
