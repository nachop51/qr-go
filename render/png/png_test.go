package png

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"testing"

	"github.com/nachop51/qr-go/render"
)

// budgetGrid is a dark grid that also advertises a logo budget.
type budgetGrid struct {
	n      int
	budget int
}

func (b budgetGrid) Size() int            { return b.n }
func (b budgetGrid) IsDark(x, y int) bool { return true }
func (b budgetGrid) MaxLogoModules() int  { return b.budget }

// centerClearedModules measures the central cleared region on the middle row,
// in modules. The grid must be fully dark so only that region is non-black.
func centerClearedModules(img image.Image, scale int) int {
	b := img.Bounds()
	y := b.Dy() / 2
	isBlack := func(x int) bool {
		r, g, bl, _ := img.At(x, y).RGBA()
		return r == 0 && g == 0 && bl == 0
	}
	l, r := b.Dx()/2, b.Dx()/2
	for l > 0 && !isBlack(l-1) {
		l--
	}
	for r < b.Dx()-1 && !isBlack(r+1) {
		r++
	}
	return ((r + 1) - l) / scale
}

// A span within budget is drawn as requested; a span beyond it is capped to the
// budget, and the cap is reported once through Warnf.
func TestLogoCappedToBudget(t *testing.T) {
	logo := image.NewRGBA(image.Rect(0, 0, 32, 32))
	draw.Draw(logo, logo.Bounds(), &image.Uniform{C: color.RGBA{200, 30, 30, 255}}, image.Point{}, draw.Src)

	var warns int
	orig := render.Warnf
	render.Warnf = func(string, ...any) { warns++ }
	defer func() { render.Warnf = orig }()

	scale, _, _ := geom(800, 800, 4, 25)
	render := func(span int) image.Image {
		var buf bytes.Buffer
		if err := New().Writer(&buf).Logo(logo).LogoModules(span).Render(budgetGrid{n: 25, budget: 5}); err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(&buf)
		if err != nil {
			t.Fatal(err)
		}
		return img
	}

	// Within budget: 3 <= 5 -> drawn as requested, no cap.
	warns = 0
	if got := centerClearedModules(render(3), scale); got != 3 {
		t.Errorf("within budget: drew %d modules, want 3", got)
	}
	if warns != 0 {
		t.Errorf("within budget: got %d warnings, want 0", warns)
	}

	// Over budget: 9 -> capped to 5, reported once.
	warns = 0
	if got := centerClearedModules(render(9), scale); got != 5 {
		t.Errorf("over budget: drew %d modules, want 5 (capped)", got)
	}
	if warns != 1 {
		t.Errorf("over budget: got %d warnings, want 1", warns)
	}
}

// darkGrid is entirely dark, so after a logo overlay the only non-black pixels
// in the code area are the cleared region, making its edges easy to measure.
type darkGrid struct{ n int }

func (d darkGrid) Size() int            { return d.n }
func (d darkGrid) IsDark(x, y int) bool { return true }

// geom recomputes the scale and centring offset used by buildImage.
func geom(width, height, quiet, size int) (scale, offX, offY int) {
	modules := size + 2*quiet
	scale = max(min(width, height)/modules, 1)
	content := modules * scale
	offX = (max(width, content) - content) / 2
	offY = (max(height, content) - content) / 2
	return
}

// The cleared logo region must cover whole modules, snapped to the grid, with
// equal margins on both sides (centred).
func TestLogoRegionModuleAligned(t *testing.T) {
	const n = 25
	logo := image.NewRGBA(image.Rect(0, 0, 40, 40))
	draw.Draw(logo, logo.Bounds(), &image.Uniform{C: color.RGBA{200, 30, 30, 255}}, image.Point{}, draw.Src)

	var buf bytes.Buffer
	if err := New().Writer(&buf).Logo(logo).Render(darkGrid{n: n}); err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	scale, offX, _ := geom(800, 800, 4, n)

	isBlack := func(x, y int) bool {
		r, g, bl, _ := img.At(x, y).RGBA()
		return r == 0 && g == 0 && bl == 0
	}

	// Walk the centre row outward from the middle to find the cleared region.
	y := b.Dy() / 2
	l, r := b.Dx()/2, b.Dx()/2
	for l > 0 && !isBlack(l-1, y) {
		l--
	}
	for r < b.Dx()-1 && !isBlack(r+1, y) {
		r++
	}
	width := (r + 1) - l

	if width%scale != 0 {
		t.Errorf("cleared width %d is not a whole number of modules (scale %d)", width, scale)
	}
	if (l-offX)%scale != 0 {
		t.Errorf("cleared left edge %d is not on a module boundary (offX %d, scale %d)", l, offX, scale)
	}
	// Symmetric margins => centred. Left margin from code start; right margin to code end.
	codeStart := offX + 4*scale // quiet = 4
	codeEnd := codeStart + n*scale
	leftMods := (l - codeStart) / scale
	rightMods := (codeEnd - (r + 1)) / scale
	if leftMods != rightMods {
		t.Errorf("not centred: %d modules left of logo, %d modules right", leftMods, rightMods)
	}
}

func TestRenderWithLogo(t *testing.T) {
	logo := image.NewRGBA(image.Rect(0, 0, 64, 64))
	draw.Draw(logo, logo.Bounds(), &image.Uniform{C: color.RGBA{200, 30, 30, 255}}, image.Point{}, draw.Src)

	var buf bytes.Buffer
	if err := New().Writer(&buf).Logo(logo).Render(fakeGrid{n: 25}); err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	// The centre of the canvas must be the red logo, not a black/white module.
	b := img.Bounds()
	r, g, bl, _ := img.At(b.Dx()/2, b.Dy()/2).RGBA()
	if !(r>>8 > 150 && g>>8 < 80 && bl>>8 < 80) {
		t.Errorf("centre pixel not logo-red: r=%d g=%d b=%d", r>>8, g>>8, bl>>8)
	}
}

type fakeGrid struct{ n int }

func (f fakeGrid) Size() int            { return f.n }
func (f fakeGrid) IsDark(x, y int) bool { return (x+y)%2 == 0 }

func renderBounds(t *testing.T, p PNG, g fakeGrid) image.Rectangle {
	t.Helper()
	var buf bytes.Buffer
	if err := p.Writer(&buf).Render(g); err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return img.Bounds()
}

// A non-square canvas must not truncate the code: it fits to the smaller side.
func TestRenderNonSquareDoesNotTruncate(t *testing.T) {
	b := renderBounds(t, New().Width(400).Height(200), fakeGrid{n: 21})
	if b.Dx() < 400 || b.Dy() < 200 {
		t.Fatalf("image %dx%d smaller than requested 400x200", b.Dx(), b.Dy())
	}
	// modules = 21 + 2*4 quiet = 29; scale = min(400,200)/29 = 6; content = 174.
	if b.Dy() < 174 {
		t.Fatalf("height %d cannot hold the 174px code", b.Dy())
	}
}

// A canvas smaller than the code grows to fit rather than clipping.
func TestRenderGrowsToFitTinyCanvas(t *testing.T) {
	b := renderBounds(t, New().Width(10).Height(10), fakeGrid{n: 21})
	modules := 21 + 2*4
	if b.Dx() < modules || b.Dy() < modules {
		t.Fatalf("image %dx%d cannot hold %d modules", b.Dx(), b.Dy(), modules)
	}
}

// darkMargins returns the whitespace on each side of the dark-pixel bounding
// box: left, right, top, bottom.
func darkMargins(img image.Image) (l, r, tp, b int) {
	bb := img.Bounds()
	minX, minY, maxX, maxY := bb.Dx(), bb.Dy(), -1, -1
	for y := 0; y < bb.Dy(); y++ {
		for x := 0; x < bb.Dx(); x++ {
			if rr, _, _, _ := img.At(x, y).RGBA(); rr == 0 {
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	return minX, bb.Dx() - 1 - maxX, minY, bb.Dy() - 1 - maxY
}

// The rendered code must be centered to within one pixel on both axes, for
// square, non-square, and odd-leftover canvases. fakeGrid is a checkerboard, so
// all four corners are dark and its bounding box equals the full code area.
func TestRenderCentersWithinOnePixel(t *testing.T) {
	for _, c := range []struct{ w, h int }{
		{800, 800}, {500, 500}, {800, 400}, {1000, 640}, {813, 641},
	} {
		var buf bytes.Buffer
		if err := New().Writer(&buf).Width(c.w).Height(c.h).Render(fakeGrid{n: 21}); err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(&buf)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		l, r, tp, b := darkMargins(img)
		if d := l - r; d < -1 || d > 1 {
			t.Errorf("%dx%d: horizontal margins L=%d R=%d differ by %d (>1px)", c.w, c.h, l, r, d)
		}
		if d := tp - b; d < -1 || d > 1 {
			t.Errorf("%dx%d: vertical margins T=%d B=%d differ by %d (>1px)", c.w, c.h, tp, b, d)
		}
	}
}

func TestRenderDefaultsToImagePNG(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := New().Render(fakeGrid{n: 21}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat("image.png"); err != nil {
		t.Fatalf("expected default image.png to be created: %v", err)
	}
}

func TestRenderUsesConfiguredFilename(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := New().Filename("custom.png").Render(fakeGrid{n: 21}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat("custom.png"); err != nil {
		t.Fatalf("expected custom.png to be created: %v", err)
	}
	if _, err := os.Stat("image.png"); err == nil {
		t.Fatal("did not expect default image.png when Filename is set")
	}
}

func TestDrawQuietFillsBackgroundAndPreservesModules(t *testing.T) {
	p := New()
	img := image.NewRGBA(image.Rect(0, 0, 6, 6))

	p.drawQuiet(img)
	p.drawModule(img, 2, 2, 2)

	wantWhite := color.RGBAModel.Convert(p.white).(color.RGBA)
	wantDark := color.RGBAModel.Convert(p.dark).(color.RGBA)

	for y := range 6 {
		for x := range 6 {
			want := wantWhite
			if x >= 2 && x < 4 && y >= 2 && y < 4 {
				want = wantDark
			}

			if got := img.RGBAAt(x, y); got != want {
				t.Fatalf("pixel (%d, %d): got %s, want %s", x, y, rgbaString(got), rgbaString(want))
			}
		}
	}
}

func rgbaString(c color.RGBA) string {
	return fmt.Sprintf("rgba(%d,%d,%d,%d)", c.R, c.G, c.B, c.A)
}

// Compile-time check: PNG satisfies the full renderer contract.
var _ render.Renderer = New()

// Bytes returns exactly what Render writes, and the result decodes as a PNG.
func TestPNGBytesMatchesRender(t *testing.T) {
	cfg := New().Quiet(1).Width(64).Height(64)
	g := budgetGrid{n: 21, budget: 7}

	var buf bytes.Buffer
	if err := cfg.Writer(&buf).Render(g); err != nil {
		t.Fatalf("Render: %v", err)
	}
	got, err := cfg.Bytes(g)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if !bytes.Equal(got, buf.Bytes()) {
		t.Error("Bytes != Render output")
	}

	img, err := png.Decode(bytes.NewReader(got))
	if err != nil {
		t.Fatalf("decode Bytes output: %v", err)
	}
	if img.Bounds().Dx() < 64 || img.Bounds().Dy() < 64 {
		t.Errorf("decoded image %v smaller than requested 64x64", img.Bounds())
	}
}
