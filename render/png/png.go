package png

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"os"

	"github.com/srwiley/rasterx"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/math/fixed"

	"github.com/nachop51/qr-go/render"
	"github.com/nachop51/qr-go/render/style"
)

type PNG struct {
	w           io.Writer
	filename    string
	dark        color.Color
	white       color.Color
	width       int
	height      int
	quiet       int
	logo        image.Image
	logoModules int
	logoScale   int
	moduleShape style.ModuleShape
	frameShape  style.EyeShape
	ballShape   style.EyeShape
	eyeFrame    color.Color // nil follows the module color
	eyeBall     color.Color
	gradient    style.GradientKind
	gradFrom    color.Color
	gradTo      color.Color
	gradAngle   float64
	warn        render.WarningHandler
}

func New() PNG {
	return PNG{filename: "image.png", dark: color.Black, white: color.White, quiet: 4, width: 800, height: 800}
}

// Logo overlays img centred on the code. Its span defaults to the largest
// size the code's error-correction level can afford to lose (see
// Code.MaxLogoModules) and can be lowered with LogoModules. The overlay covers
// whole modules snapped to the grid, so it never slices one, and the logo is
// inset one module inside the cleared region so it never touches the
// surrounding modules.
//
// A logo hides the modules it covers; a span wider than the code's
// error-correction budget is capped to that budget so the result still scans,
// and the reduction is reported through the renderer's WarningHandler.
func (p PNG) Logo(img image.Image) PNG {
	p.logo = img
	return p
}

// LogoModules sets how many modules across the logo spans. A value <= 0 restores
// the default: the code's full error-correction budget (roughly size/3 at High,
// size/4 at Quartile, size/5 at Medium, size/6 at Low). A span wider than that
// budget is capped to it so the code stays scannable.
func (p PNG) LogoModules(n int) PNG {
	p.logoModules = n
	return p
}

// LogoScale sets how much of the cleared logo square the image fills, as a
// percentage of its edge: 100 covers the whole square, smaller values leave
// more background around the logo. A value <= 0 restores the span-dependent
// default, [render.DefaultLogoScale]. Values above 100 are capped.
func (p PNG) LogoScale(pct int) PNG {
	p.logoScale = pct
	return p
}

func (p PNG) Filename(name string) PNG {
	p.filename = name
	return p
}

func (p PNG) Writer(w io.Writer) PNG {
	p.w = w
	return p
}

// WarningHandler sets the destination for non-fatal rendering advice.
func (p PNG) WarningHandler(warn render.WarningHandler) PNG { p.warn = warn; return p }

func (p PNG) Dark(c color.Color) PNG {
	p.dark = c
	return p
}

func (p PNG) White(c color.Color) PNG {
	p.white = c
	return p
}

func (p PNG) Quiet(n int) PNG {
	p.quiet = n
	return p
}

func (p PNG) Width(w int) PNG {
	p.width = w
	return p
}

func (p PNG) Height(h int) PNG {
	p.height = h
	return p
}

// ModuleShape sets how data modules are drawn. Styled shapes assume the grid
// is a real QR code (the three finder eyes are located geometrically).
func (p PNG) ModuleShape(m style.ModuleShape) PNG { p.moduleShape = m; return p }

// EyeShape sets both the finder frame and ball shape at once.
func (p PNG) EyeShape(e style.EyeShape) PNG { p.frameShape, p.ballShape = e, e; return p }

// EyeFrameShape sets the shape of the 7x7 finder ring only.
func (p PNG) EyeFrameShape(e style.EyeShape) PNG { p.frameShape = e; return p }

// EyeBallShape sets the shape of the 3x3 finder pupil only.
func (p PNG) EyeBallShape(e style.EyeShape) PNG { p.ballShape = e; return p }

// EyeFrame colors the finder rings. Nil follows the module color (or
// gradient, when one is set).
func (p PNG) EyeFrame(c color.Color) PNG { p.eyeFrame = c; return p }

// EyeBall colors the finder pupils. Nil follows the module color (or
// gradient, when one is set).
func (p PNG) EyeBall(c color.Color) PNG { p.eyeBall = c; return p }

// GradientLinear fills the modules with a two-stop linear gradient. The angle
// is in degrees: 0 runs left to right, 90 top to bottom.
func (p PNG) GradientLinear(from, to color.Color, angleDeg float64) PNG {
	p.gradient, p.gradFrom, p.gradTo, p.gradAngle = style.GradientLinear, from, to, angleDeg
	return p
}

// GradientRadial fills the modules with a two-stop radial gradient from the
// centre of the code.
func (p PNG) GradientRadial(from, to color.Color) PNG {
	p.gradient, p.gradFrom, p.gradTo = style.GradientRadial, from, to
	return p
}

// styled reports whether any option moves rendering off the fast square path.
func (p PNG) styled() bool {
	return p.moduleShape != style.ModuleSquare ||
		p.frameShape != style.EyeSquare || p.ballShape != style.EyeSquare ||
		p.eyeFrame != nil || p.eyeBall != nil ||
		p.gradient != style.GradientNone
}

func (p PNG) drawModule(img *image.RGBA, px, py, scale int) {
	for i := range scale {
		for j := range scale {
			img.Set(px+i, py+j, p.dark)
		}
	}
}

func (p PNG) drawQuiet(img *image.RGBA) {
	draw.Draw(img, img.Bounds(), &image.Uniform{C: p.white}, image.Point{}, draw.Src)
}

// validate checks every option before allocating the output image.
func (p PNG) validate(g render.Grid) error {
	if err := render.ValidateGrid(g); err != nil {
		return err
	}
	if p.quiet < 0 || p.quiet > 256 {
		return fmt.Errorf("png: quiet zone must be between 0 and 256")
	}
	if p.width < 1 || p.height < 1 || p.width > 8192 || p.height > 8192 || int64(p.width)*int64(p.height) > 64_000_000 {
		return fmt.Errorf("png: dimensions must be positive, at most 8192x8192 and 64 megapixels")
	}
	if p.dark == nil || p.white == nil {
		return fmt.Errorf("png: dark and light colors must be non-nil")
	}
	if !p.moduleShape.Valid() || !p.frameShape.Valid() || !p.ballShape.Valid() || !p.gradient.Valid() {
		return fmt.Errorf("png: invalid style option")
	}
	if p.gradient != style.GradientNone && (p.gradFrom == nil || p.gradTo == nil || math.IsNaN(p.gradAngle) || math.IsInf(p.gradAngle, 0)) {
		return fmt.Errorf("png: gradient colors must be non-nil and angle finite")
	}
	if p.logo != nil {
		b := p.logo.Bounds()
		w, h := b.Dx(), b.Dy()
		if w < 1 || h < 1 || w > 4096 || h > 4096 || int64(w)*int64(h) > 16_000_000 {
			return fmt.Errorf("png: logo dimensions exceed 4096 pixels or 16 megapixels")
		}
	}
	return nil
}

// buildImage renders the code (and optional logo) into an RGBA image.
func (p PNG) buildImage(g render.Grid) (*image.RGBA, error) {
	if err := p.validate(g); err != nil {
		return nil, err
	}
	size := g.Size()
	modules := size + 2*p.quiet

	scale := max(min(p.width, p.height)/modules, 1)
	content := modules * scale

	// Honour the requested size, but never render smaller than the code itself.
	w := max(p.width, content)
	h := max(p.height, content)

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	p.drawQuiet(img)

	// Centre the (quiet-padded) code within the canvas.
	offX := (w - content) / 2
	offY := (h - content) / 2

	// Resolve the logo span before drawing and hide the modules it will
	// cover: shaped modules at the region's edge must round toward the
	// cleared area instead of connecting to modules the overlay erases.
	logoMods := 0
	grid := g
	if p.logo != nil {
		logoMods = render.ResolveLogoWithWarnings(g, p.logoModules, p.warn)
		grid = render.MaskLogo(g, logoMods)
	}

	if p.styled() {
		p.drawStyled(img, grid, offX, offY, scale)
	} else {
		for y := range size {
			for x := range size {
				if grid.IsDark(x, y) {
					p.drawModule(img, offX+(x+p.quiet)*scale, offY+(y+p.quiet)*scale, scale)
				}
			}
		}
	}

	if logoMods > 0 {
		p.drawLogo(img, offX, offY, size, scale, logoMods)
	}

	return img, nil
}

// drawStyled rasterizes shaped modules and whole-shape finder eyes through
// rasterx (anti-aliased, nonzero winding), in three fill passes: data
// modules, eye frames, eye balls.
func (p PNG) drawStyled(img *image.RGBA, g render.Grid, offX, offY, scale int) {
	b := img.Bounds()
	scanner := rasterx.NewScannerGV(b.Dx(), b.Dy(), img, b)
	filler := rasterx.NewFiller(b.Dx(), b.Dy(), scanner)
	rp := &rasterPath{
		f:     filler,
		scale: float64(scale),
		offX:  float64(offX + p.quiet*scale),
		offY:  float64(offY + p.quiet*scale),
	}

	moduleSrc := p.moduleSource(b)
	p.warnContrast()

	n := g.Size()
	scanner.SetColor(moduleSrc)
	for y := range n {
		for x := range n {
			if !g.IsDark(x, y) || style.InEye(x, y, n) {
				continue
			}
			var c style.Corners
			if p.moduleShape == style.ModuleRounded {
				c = style.CornerMask(g, x, y)
			}
			style.AddModule(rp, float64(x), float64(y), p.moduleShape, c)
		}
	}
	filler.Draw()
	filler.Clear()

	eyes := style.EyeRects(n)
	scanner.SetColor(orSource(p.eyeFrame, moduleSrc))
	for _, e := range eyes {
		style.AddEyeFrame(rp, e, p.frameShape)
	}
	filler.Draw()
	filler.Clear()

	scanner.SetColor(orSource(p.eyeBall, moduleSrc))
	for _, e := range eyes {
		style.AddEyeBall(rp, e, p.ballShape)
	}
	filler.Draw()
	filler.Clear()
}

// moduleSource returns what the module pass paints with: the flat dark color,
// or a per-pixel gradient spanning the whole image so all passes share one
// ramp (mirroring the SVG renderer's userSpaceOnUse gradient).
func (p PNG) moduleSource(b image.Rectangle) any {
	if p.gradient == style.GradientNone {
		return p.dark
	}
	cx, cy := float64(b.Dx())/2, float64(b.Dy())/2
	if p.gradient == style.GradientRadial {
		reach := math.Hypot(cx, cy) // corners land exactly on the end stop
		return rasterx.ColorFunc(func(x, y int) color.Color {
			t := math.Hypot(float64(x)-cx, float64(y)-cy) / reach
			return lerpColor(p.gradFrom, p.gradTo, t)
		})
	}
	rad := p.gradAngle * math.Pi / 180
	dx, dy := math.Cos(rad), math.Sin(rad)
	x1, y1 := cx*(1-dx), cy*(1-dy)
	span := (cx*(1+dx)-x1)*dx + (cy*(1+dy)-y1)*dy
	return rasterx.ColorFunc(func(x, y int) color.Color {
		t := ((float64(x)-x1)*dx + (float64(y)-y1)*dy) / span
		return lerpColor(p.gradFrom, p.gradTo, min(max(t, 0), 1))
	})
}

func orSource(c color.Color, fallback any) any {
	if c != nil {
		return c
	}
	return fallback
}

func lerpColor(a, b color.Color, t float64) color.Color {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	l := func(x, y uint32) uint16 {
		return uint16(math.Round(float64(x) + (float64(y)-float64(x))*t))
	}
	return color.RGBA64{R: l(ar, br), G: l(ag, bg), B: l(ab, bb), A: l(aa, ba)}
}

// warnContrast flags styled color choices likely to break scanning.
func (p PNG) warnContrast() {
	if p.gradient != style.GradientNone {
		style.WarnContrast(p.warn, "gradient start color", p.gradFrom, p.white)
		style.WarnContrast(p.warn, "gradient end color", p.gradTo, p.white)
	} else {
		style.WarnContrast(p.warn, "module color", p.dark, p.white)
	}
	if p.eyeFrame != nil {
		style.WarnContrast(p.warn, "eye frame color", p.eyeFrame, p.white)
	}
	if p.eyeBall != nil {
		style.WarnContrast(p.warn, "eye ball color", p.eyeBall, p.white)
	}
}

// rasterPath adapts style.Path onto a rasterx filler, mapping module units to
// device pixels: px = off + v*scale.
type rasterPath struct {
	f                 *rasterx.Filler
	scale, offX, offY float64
}

func (p *rasterPath) pt(x, y float64) fixed.Point26_6 {
	return fixed.Point26_6{
		X: fixed.Int26_6(math.Round((p.offX + x*p.scale) * 64)),
		Y: fixed.Int26_6(math.Round((p.offY + y*p.scale) * 64)),
	}
}

func (p *rasterPath) MoveTo(x, y float64) { p.f.Start(p.pt(x, y)) }
func (p *rasterPath) LineTo(x, y float64) { p.f.Line(p.pt(x, y)) }
func (p *rasterPath) CubeTo(c1x, c1y, c2x, c2y, x, y float64) {
	p.f.CubeBezier(p.pt(c1x, c1y), p.pt(c2x, c2y), p.pt(x, y))
}
func (p *rasterPath) Close() { p.f.Stop(true) }

func (p PNG) drawLogo(img *image.RGBA, offX, offY, size, scale, mods int) {
	// Cleared region, expressed in whole modules and snapped to the grid.
	start := (size - mods) / 2
	x0 := offX + (p.quiet+start)*scale
	y0 := offY + (p.quiet+start)*scale
	region := mods * scale
	draw.Draw(img, image.Rect(x0, y0, x0+region, y0+region),
		&image.Uniform{C: p.white}, image.Point{}, draw.Src)

	// Fit the logo inside its box (by default a span-dependent slice of the
	// cleared region, so the logo never touches the surrounding modules),
	// preserving the aspect ratio, centred on the region.
	box := render.LogoBoxWithWarnings(region, mods, p.logoScale, p.warn)
	sb := p.logo.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	tw, th := box, box
	switch {
	case sw > sh:
		th = box * sh / sw
	case sh > sw:
		tw = box * sw / sh
	}
	cx := x0 + region/2
	cy := y0 + region/2
	dst := image.Rect(cx-tw/2, cy-th/2, cx+tw/2, cy+th/2)
	xdraw.CatmullRom.Scale(img, dst, p.logo, sb, xdraw.Over, nil)
}

// Bytes returns the rendered QR as an encoded PNG image.
func (p PNG) Bytes(g render.Grid) ([]byte, error) {
	var buf bytes.Buffer
	img, err := p.buildImage(g)
	if err != nil {
		return nil, err
	}
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p PNG) Render(g render.Grid) error {
	img, err := p.buildImage(g)
	if err != nil {
		return err
	}

	out := p.w
	if out == nil {
		name := p.filename
		if name == "" {
			name = "image.png"
		}
		f, err := os.Create(name)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	}

	return png.Encode(out, img)
}
