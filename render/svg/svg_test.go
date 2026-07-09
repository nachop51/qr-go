package svg

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"testing"

	"github.com/nachop51/qr-go/render"
	"github.com/nachop51/qr-go/render/style"
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
	if !strings.Contains(out, `<path fill="#000000" d="`) {
		t.Fatalf("expected dark modules path, got: %s", out)
	}
	if !strings.Contains(out, `M2 2h2v2h-2z`) {
		t.Fatalf("expected dark module at first quiet-zone offset, got: %s", out)
	}
	// The 3x3 checkerboard has 5 dark modules, none adjacent: 5 path commands.
	if got := strings.Count(out, "z"); got != 5 {
		t.Fatalf("expected 5 dark module runs, got %d: %s", got, out)
	}
}

// Horizontally adjacent dark modules merge into a single path command per row.
func TestSVGRunLength(t *testing.T) {
	var buf bytes.Buffer
	if err := New().Writer(&buf).Quiet(0).Module(1).Render(budgetGrid{n: 3}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if !strings.Contains(out, `d="M0 0h3v1h-3zM0 1h3v1h-3zM0 2h3v1h-3z"`) {
		t.Fatalf("expected one merged run per all-dark row, got: %s", out)
	}
}

// A logo produces a module-aligned cleared region and an embedded <image>.
func TestSVGLogo(t *testing.T) {
	var buf bytes.Buffer
	if err := New().Writer(&buf).Logo(solidLogo(64)).Render(fakeGrid{n: 25}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	// fakeGrid has no budget, so the span falls back to size/5 = 5; module 10,
	// quiet 4 -> region at (140,140), 50x50.
	if !strings.Contains(out, `<rect x="140" y="140" width="50" height="50" fill="#ffffff"/>`) {
		t.Fatalf("expected module-aligned cleared region, got: %s", out)
	}
	// Logo box leaves a one-module ring on every side: (150,150), 30x30,
	// embedded as a data URI.
	if !strings.Contains(out, `<image x="150" y="150" width="30" height="30" preserveAspectRatio="xMidYMid meet" href="data:image/png;base64,`) {
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

// Styled output drops crispEdges and paints modules, frames and balls as
// three separate paths, skipping the eye areas in the module loop.
func TestSVGStyled(t *testing.T) {
	var buf bytes.Buffer
	r := New().Writer(&buf).Quiet(0).Module(1).
		ModuleShape(style.ModuleRounded).
		EyeShape(style.EyeCircle).
		EyeFrame("#1d4ed8").
		EyeBall("#b91c1c")
	if err := r.Render(fakeGrid{n: 21}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if strings.Contains(out, "crispEdges") {
		t.Fatal("styled output must not set crispEdges")
	}
	if got := strings.Count(out, `<path fill=`); got != 3 {
		t.Fatalf("expected 3 paint groups, got %d: %s", got, out)
	}
	if !strings.Contains(out, `<path fill="#1d4ed8" d="`) || !strings.Contains(out, `<path fill="#b91c1c" d="`) {
		t.Fatalf("expected eye frame and ball colors, got: %s", out)
	}
	// The module path must not touch the eye areas: with quiet 0 and module 1
	// the top-left eye spans x in [0,7); the checkerboard's (0,0) module is
	// dark but must be skipped, so no module subpath starts at the origin.
	dark := out[strings.Index(out, `<path fill="#000000" d="`):]
	dark = dark[:strings.Index(dark, `"/>`)]
	if strings.Contains(dark, "M0 0") {
		t.Fatalf("module path should skip eye modules: %s", dark)
	}
}

// Setting only an eye color still routes through the styled branch; the eyes
// stay square but become their own paths.
func TestSVGStyledEyeColorOnly(t *testing.T) {
	var buf bytes.Buffer
	if err := New().Writer(&buf).EyeFrame("red").Render(fakeGrid{n: 21}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if got := strings.Count(out, `<path fill=`); got != 3 {
		t.Fatalf("expected 3 paint groups, got %d: %s", got, out)
	}
	if !strings.Contains(out, `<path fill="red" d="`) {
		t.Fatalf("expected red eye frame path, got: %s", out)
	}
}

// A gradient adds a <defs> ramp shared by modules and (uncolored) eyes.
func TestSVGGradient(t *testing.T) {
	var buf bytes.Buffer
	err := New().Writer(&buf).Quiet(0).Module(10).
		GradientLinear("#0f172a", "#7f1d1d", 90).
		Render(fakeGrid{n: 21})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if !strings.Contains(out, `<linearGradient id="qrgo-gradient" gradientUnits="userSpaceOnUse" x1="105" y1="0" x2="105" y2="210">`) {
		t.Fatalf("expected vertical linear gradient across the 210px image, got: %s", out)
	}
	if !strings.Contains(out, `<stop offset="0" stop-color="#0f172a"/><stop offset="1" stop-color="#7f1d1d"/>`) {
		t.Fatalf("expected two gradient stops, got: %s", out)
	}
	if got := strings.Count(out, `<path fill="url(#qrgo-gradient)"`); got != 3 {
		t.Fatalf("all 3 paint groups should share the gradient, got %d: %s", got, out)
	}

	buf.Reset()
	if err := New().Writer(&buf).GradientRadial("#000", "#333").Render(fakeGrid{n: 21}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `<radialGradient id="qrgo-gradient"`) {
		t.Fatalf("expected radial gradient, got: %s", buf.String())
	}
}

// Compile-time check: SVG satisfies the full renderer contract.
var _ render.Renderer = New()

// Bytes returns exactly what Render writes, with and without a logo.
func TestSVGBytesMatchesRender(t *testing.T) {
	for name, tc := range map[string]struct {
		cfg SVG
		g   render.Grid
	}{
		"plain": {New().Quiet(1).Module(2), fakeGrid{n: 3}},
		"logo":  {New().Quiet(1).Module(4).Logo(solidLogo(8)), budgetGrid{n: 21, budget: 7}},
	} {
		var buf bytes.Buffer
		if err := tc.cfg.Writer(&buf).Render(tc.g); err != nil {
			t.Fatalf("%s: Render: %v", name, err)
		}
		got, err := tc.cfg.Bytes(tc.g)
		if err != nil {
			t.Fatalf("%s: Bytes: %v", name, err)
		}
		if !bytes.Equal(got, buf.Bytes()) {
			t.Errorf("%s: Bytes != Render output\nBytes:\n%s\nRender:\n%s", name, got, buf.Bytes())
		}
	}
}
