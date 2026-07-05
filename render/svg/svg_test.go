package svg

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"testing"

	"github.com/nachop51/qr-go/render"
)

type fakeGrid struct{ n int }

func (f fakeGrid) Size() int            { return f.n }
func (f fakeGrid) IsDark(x, y int) bool { return (x+y)%2 == 0 }

// budgetGrid is a dark grid that advertises a logo budget.
type budgetGrid struct {
	n      int
	budget int
}

func (b budgetGrid) Size() int            { return b.n }
func (b budgetGrid) IsDark(x, y int) bool { return true }
func (b budgetGrid) MaxLogoModules() int  { return b.budget }

func solidLogo(n int) image.Image {
	l := image.NewRGBA(image.Rect(0, 0, n, n))
	draw.Draw(l, l.Bounds(), &image.Uniform{C: color.RGBA{200, 30, 30, 255}}, image.Point{}, draw.Src)
	return l
}

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

// A logo produces a module-aligned cleared region and an embedded <image>.
func TestSVGLogo(t *testing.T) {
	var buf bytes.Buffer
	if err := New().Writer(&buf).Logo(solidLogo(64)).Render(fakeGrid{n: 25}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	// Default span size/5 = 5; module 10, quiet 4 -> region at (140,140), 50x50.
	if !strings.Contains(out, `<rect x="140" y="140" width="50" height="50" fill="#ffffff"/>`) {
		t.Fatalf("expected module-aligned cleared region, got: %s", out)
	}
	// Logo box leaves a one-module ring: (145,145), 40x40, embedded as a data URI.
	if !strings.Contains(out, `<image x="145" y="145" width="40" height="40" preserveAspectRatio="xMidYMid meet" href="data:image/png;base64,`) {
		t.Fatalf("expected embedded logo image, got: %s", out)
	}
}

// An oversized span is capped to the grid's budget, reported once via Warnf.
func TestSVGLogoCapped(t *testing.T) {
	var warns int
	orig := render.Warnf
	render.Warnf = func(string, ...any) { warns++ }
	defer func() { render.Warnf = orig }()

	var buf bytes.Buffer
	if err := New().Writer(&buf).Logo(solidLogo(64)).LogoModules(9).Render(budgetGrid{n: 25, budget: 3}); err != nil {
		t.Fatal(err)
	}
	// Capped to 3: start=(25-3)/2=11 -> region at (150,150), 30x30.
	if !strings.Contains(buf.String(), `<rect x="150" y="150" width="30" height="30" fill="#ffffff"/>`) {
		t.Fatalf("expected capped region (span 3), got: %s", buf.String())
	}
	if warns != 1 {
		t.Fatalf("expected 1 cap warning, got %d", warns)
	}
}
