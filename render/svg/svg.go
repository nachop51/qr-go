package svg

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	xdraw "golang.org/x/image/draw"

	"github.com/nachop51/qr-go/render"
	"github.com/nachop51/qr-go/render/style"
)

type SVG struct {
	w           io.Writer
	dark        string
	light       string
	quiet       int
	module      int
	logo        image.Image
	logoModules int
	logoScale   int
	moduleShape style.ModuleShape
	frameShape  style.EyeShape
	ballShape   style.EyeShape
	eyeFrame    string
	eyeBall     string
	gradient    style.GradientKind
	gradFrom    string
	gradTo      string
	gradAngle   float64
	warn        render.WarningHandler
}

func New() SVG {
	return SVG{
		w:      os.Stdout,
		dark:   "#000000",
		light:  "#ffffff",
		quiet:  4,
		module: 10,
	}
}

func (s SVG) Writer(w io.Writer) SVG                        { s.w = w; return s }
func (s SVG) Dark(c string) SVG                             { s.dark = c; return s }
func (s SVG) Light(c string) SVG                            { s.light = c; return s }
func (s SVG) Quiet(n int) SVG                               { s.quiet = n; return s }
func (s SVG) Module(n int) SVG                              { s.module = n; return s }
func (s SVG) WarningHandler(warn render.WarningHandler) SVG { s.warn = warn; return s }

// ModuleShape sets how data modules are drawn. Styled shapes assume the grid
// is a real QR code (the three finder eyes are located geometrically).
func (s SVG) ModuleShape(m style.ModuleShape) SVG { s.moduleShape = m; return s }

// EyeShape sets both the finder frame and ball shape at once.
func (s SVG) EyeShape(e style.EyeShape) SVG { s.frameShape, s.ballShape = e, e; return s }

// EyeFrameShape sets the shape of the 7x7 finder ring only.
func (s SVG) EyeFrameShape(e style.EyeShape) SVG { s.frameShape = e; return s }

// EyeBallShape sets the shape of the 3x3 finder pupil only.
func (s SVG) EyeBallShape(e style.EyeShape) SVG { s.ballShape = e; return s }

// EyeFrame colors the finder rings. Empty follows the module color (or
// gradient, when one is set).
func (s SVG) EyeFrame(c string) SVG { s.eyeFrame = c; return s }

// EyeBall colors the finder pupils. Empty follows the module color (or
// gradient, when one is set).
func (s SVG) EyeBall(c string) SVG { s.eyeBall = c; return s }

// GradientLinear fills the modules with a two-stop linear gradient. The angle
// is in degrees: 0 runs left to right, 90 top to bottom.
func (s SVG) GradientLinear(from, to string, angleDeg float64) SVG {
	s.gradient, s.gradFrom, s.gradTo, s.gradAngle = style.GradientLinear, from, to, angleDeg
	return s
}

// GradientRadial fills the modules with a two-stop radial gradient from the
// centre of the code.
func (s SVG) GradientRadial(from, to string) SVG {
	s.gradient, s.gradFrom, s.gradTo = style.GradientRadial, from, to
	return s
}

// styled reports whether any option moves rendering off the fast square path.
func (s SVG) styled() bool {
	return s.moduleShape != style.ModuleSquare ||
		s.frameShape != style.EyeSquare || s.ballShape != style.EyeSquare ||
		s.eyeFrame != "" || s.eyeBall != "" ||
		s.gradient != style.GradientNone
}

// Logo overlays img centred on the code, embedded as a data URI. Its span
// defaults to the largest size the code's error-correction level can afford
// to lose (see Code.MaxLogoModules) and can be lowered with LogoModules. The
// overlay covers whole modules snapped to the grid, so it never slices one,
// and the logo is inset one module inside the cleared region so it never
// touches the surrounding modules.
//
// A logo hides the modules it covers; a span wider than the code's
// error-correction budget is capped to that budget so the result still scans,
// and the reduction is reported through the renderer's WarningHandler.
func (s SVG) Logo(img image.Image) SVG { s.logo = img; return s }

// LogoModules sets how many modules across the logo spans. A value <= 0 restores
// the default: the code's full error-correction budget (roughly size/3 at High,
// size/4 at Quartile, size/5 at Medium, size/6 at Low). A span wider than that
// budget is capped to it so the code stays scannable.
func (s SVG) LogoModules(n int) SVG { s.logoModules = n; return s }

// LogoScale sets how much of the cleared logo square the image fills, as a
// percentage of its edge: 100 covers the whole square, smaller values leave
// more background around the logo. A value <= 0 restores the span-dependent
// default, [render.DefaultLogoScale]. Values above 100 are capped.
func (s SVG) LogoScale(pct int) SVG { s.logoScale = pct; return s }

func (s SVG) Render(g render.Grid) error {
	w := s.w
	if w == nil {
		w = os.Stdout
	}

	markup, err := s.markup(g)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, markup)
	return err
}

// Bytes returns the rendered QR as SVG markup.
func (s SVG) Bytes(g render.Grid) ([]byte, error) {
	markup, err := s.markup(g)
	return []byte(markup), err
}

func (s SVG) validate(g render.Grid) (SVG, error) {
	if err := render.ValidateGrid(g); err != nil {
		return s, err
	}
	if s.quiet < 0 || s.quiet > 256 {
		return s, fmt.Errorf("svg: quiet zone must be between 0 and 256")
	}
	if s.module < 1 || s.module > 1024 {
		return s, fmt.Errorf("svg: module size must be between 1 and 1024")
	}
	if !s.moduleShape.Valid() || !s.frameShape.Valid() || !s.ballShape.Valid() || !s.gradient.Valid() || math.IsNaN(s.gradAngle) || math.IsInf(s.gradAngle, 0) {
		return s, fmt.Errorf("svg: invalid style option")
	}
	parse := func(name, raw string, optional bool) (string, error) {
		if optional && raw == "" {
			return "", nil
		}
		c, err := render.ParseCSSColor(raw)
		if err != nil {
			return "", fmt.Errorf("svg: %s: %w", name, err)
		}
		return c.CSS, nil
	}
	var err error
	if s.dark, err = parse("dark color", s.dark, false); err != nil {
		return s, err
	}
	if s.light, err = parse("light color", s.light, false); err != nil {
		return s, err
	}
	if s.eyeFrame, err = parse("eye frame color", s.eyeFrame, true); err != nil {
		return s, err
	}
	if s.eyeBall, err = parse("eye ball color", s.eyeBall, true); err != nil {
		return s, err
	}
	if s.gradient != style.GradientNone {
		if s.gradFrom, err = parse("gradient start color", s.gradFrom, false); err != nil {
			return s, err
		}
		if s.gradTo, err = parse("gradient end color", s.gradTo, false); err != nil {
			return s, err
		}
	}
	if s.logo != nil {
		b := s.logo.Bounds()
		w, h := b.Dx(), b.Dy()
		if w < 1 || h < 1 || w > 4096 || h > 4096 || int64(w)*int64(h) > 16_000_000 {
			return s, fmt.Errorf("svg: logo dimensions exceed 4096 pixels or 16 megapixels")
		}
	}
	return s, nil
}

func (s SVG) markup(g render.Grid) (string, error) {
	var err error
	s, err = s.validate(g)
	if err != nil {
		return "", err
	}
	module := s.module
	quiet := s.quiet

	totalModules := g.Size() + 2*quiet
	size := totalModules * module

	// Resolve the logo span before drawing and hide the modules it will
	// cover: shaped modules at the region's edge must round toward the
	// cleared area instead of connecting to modules the overlay erases.
	logoMods := 0
	if s.logo != nil {
		logoMods = render.ResolveLogoWithWarnings(g, s.logoModules, s.warn)
		g = render.MaskLogo(g, logoMods)
	}

	if s.styled() {
		return s.styledMarkup(g, logoMods, module, quiet, size)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d" shape-rendering="crispEdges">`, size, size, size, size)
	fmt.Fprintf(&sb, `<rect width="%d" height="%d" fill="%s"/>`, size, size, escapeAttr(s.light))

	// All dark modules as a single <path>: one DOM node instead of one <rect>
	// per module, with horizontal runs merged so a solid row is one command.
	sb.WriteString(`<path fill="` + escapeAttr(s.dark) + `" d="`)
	for y := 0; y < g.Size(); y++ {
		for x := 0; x < g.Size(); {
			if !g.IsDark(x, y) {
				x++
				continue
			}
			run := 1
			for x+run < g.Size() && g.IsDark(x+run, y) {
				run++
			}
			px := (x + quiet) * module
			py := (y + quiet) * module
			w := run * module
			fmt.Fprintf(&sb, "M%d %dh%dv%dh-%dz", px, py, w, module, w)
			x += run
		}
	}
	sb.WriteString(`"/>`)

	if logoMods > 0 {
		if err := s.drawLogo(&sb, g.Size(), quiet, module, logoMods); err != nil {
			return "", err
		}
	}

	sb.WriteString(`</svg>`)
	return sb.String(), nil
}

// styledMarkup renders shaped modules and whole-shape finder eyes. Unlike the
// fast path it omits shape-rendering="crispEdges", which would destroy the
// anti-aliasing that curves depend on, and draws each of the three finder
// eyes as one ring path plus one pupil path instead of per-module squares.
func (s SVG) styledMarkup(g render.Grid, logoMods, module, quiet, size int) (string, error) {
	s.warnContrast()

	var sb strings.Builder
	fmt.Fprintf(&sb, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`, size, size, size, size)
	fmt.Fprintf(&sb, `<rect width="%d" height="%d" fill="%s"/>`, size, size, escapeAttr(s.light))

	moduleFill := s.dark
	if s.gradient != style.GradientNone {
		s.writeGradient(&sb, size)
		moduleFill = "url(#qrgo-gradient)"
	}
	frameFill := s.eyeFrame
	if frameFill == "" {
		frameFill = moduleFill
	}
	ballFill := s.eyeBall
	if ballFill == "" {
		ballFill = moduleFill
	}

	p := &svgPath{sb: &sb, scale: float64(module), off: float64(quiet)}
	n := g.Size()

	sb.WriteString(`<path fill="` + escapeAttr(moduleFill) + `" d="`)
	for y := range n {
		for x := range n {
			if !g.IsDark(x, y) || style.InEye(x, y, n) {
				continue
			}
			var c style.Corners
			if s.moduleShape == style.ModuleRounded {
				c = style.CornerMask(g, x, y)
			}
			style.AddModule(p, float64(x), float64(y), s.moduleShape, c)
		}
	}
	sb.WriteString(`"/>`)

	eyes := style.EyeRects(n)
	sb.WriteString(`<path fill="` + escapeAttr(frameFill) + `" d="`)
	for _, e := range eyes {
		style.AddEyeFrame(p, e, s.frameShape)
	}
	sb.WriteString(`"/>`)
	sb.WriteString(`<path fill="` + escapeAttr(ballFill) + `" d="`)
	for _, e := range eyes {
		style.AddEyeBall(p, e, s.ballShape)
	}
	sb.WriteString(`"/>`)

	if logoMods > 0 {
		if err := s.drawLogo(&sb, n, quiet, module, logoMods); err != nil {
			return "", err
		}
	}

	sb.WriteString(`</svg>`)
	return sb.String(), nil
}

// warnContrast flags styled color choices likely to break scanning. SVG
// colors are arbitrary CSS strings, so only simple hex values are checked;
// anything else is skipped silently.
func (s SVG) warnContrast() {
	bg, ok := style.ParseHex(s.light)
	if !ok {
		return
	}
	check := func(name, c string) {
		if col, ok := style.ParseHex(c); ok {
			style.WarnContrast(s.warn, name, col, bg)
		}
	}
	if s.gradient != style.GradientNone {
		check("gradient start color", s.gradFrom)
		check("gradient end color", s.gradTo)
	} else {
		check("module color", s.dark)
	}
	if s.eyeFrame != "" {
		check("eye frame color", s.eyeFrame)
	}
	if s.eyeBall != "" {
		check("eye ball color", s.eyeBall)
	}
}

// writeGradient emits the <defs> block for the module gradient, spanning the
// whole image in user space so all three paint groups share one ramp.
func (s SVG) writeGradient(sb *strings.Builder, size int) {
	sb.WriteString(`<defs>`)
	if s.gradient == style.GradientRadial {
		// Radius reaches the corners so no module clamps to the end stop early.
		r := float64(size) * math.Sqrt2 / 2
		fmt.Fprintf(sb, `<radialGradient id="qrgo-gradient" gradientUnits="userSpaceOnUse" cx="%d" cy="%d" r="%s">`,
			size/2, size/2, fmtNum(r))
		fmt.Fprintf(sb, `<stop offset="0" stop-color="%s"/><stop offset="1" stop-color="%s"/></radialGradient>`, escapeAttr(s.gradFrom), escapeAttr(s.gradTo))
	} else {
		rad := s.gradAngle * math.Pi / 180
		c := float64(size) / 2
		fmt.Fprintf(sb, `<linearGradient id="qrgo-gradient" gradientUnits="userSpaceOnUse" x1="%s" y1="%s" x2="%s" y2="%s">`,
			fmtNum(c*(1-math.Cos(rad))), fmtNum(c*(1-math.Sin(rad))),
			fmtNum(c*(1+math.Cos(rad))), fmtNum(c*(1+math.Sin(rad))))
		fmt.Fprintf(sb, `<stop offset="0" stop-color="%s"/><stop offset="1" stop-color="%s"/></linearGradient>`, escapeAttr(s.gradFrom), escapeAttr(s.gradTo))
	}
	sb.WriteString(`</defs>`)
}

// svgPath writes style.Path commands as SVG path data, mapping module units
// to pixels: px = (v + quiet) * module.
type svgPath struct {
	sb         *strings.Builder
	scale, off float64
}

func (p *svgPath) t(v float64) string { return fmtNum((v + p.off) * p.scale) }

func (p *svgPath) MoveTo(x, y float64) { p.sb.WriteString("M" + p.t(x) + " " + p.t(y)) }
func (p *svgPath) LineTo(x, y float64) { p.sb.WriteString("L" + p.t(x) + " " + p.t(y)) }
func (p *svgPath) CubeTo(c1x, c1y, c2x, c2y, x, y float64) {
	p.sb.WriteString("C" + p.t(c1x) + " " + p.t(c1y) + " " + p.t(c2x) + " " + p.t(c2y) + " " + p.t(x) + " " + p.t(y))
}
func (p *svgPath) Close() { p.sb.WriteString("Z") }

// fmtNum renders a coordinate with at most two decimals and no trailing zeros.
func fmtNum(v float64) string {
	return strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64)
}

// drawLogo clears a module-aligned square region and embeds the logo inside it.
func (s SVG) drawLogo(sb *strings.Builder, size, quiet, module, mods int) error {
	// Cleared region, expressed in whole modules and snapped to the grid.
	start := (size - mods) / 2
	x0 := (quiet + start) * module
	y0 := (quiet + start) * module
	region := mods * module
	fmt.Fprintf(sb, `<rect x="%d" y="%d" width="%d" height="%d" fill="%s"/>`, x0, y0, region, region, escapeAttr(s.light))

	// The browser fits and centres the image within the box (by default a
	// span-dependent slice of the cleared region, so the logo never touches
	// the surrounding modules), preserving its aspect ratio.
	box := render.LogoBoxWithWarnings(region, mods, s.logoScale, s.warn)

	// Embed at 2x the drawn box (for hi-dpi screens), never above the source
	// resolution. Embedding the source as-is would put a full-resolution data
	// URI in the markup: a phone photo becomes tens of MB of SVG.
	uri, err := pngDataURI(scaleToFit(s.logo, 2*box))
	if err != nil {
		return fmt.Errorf("svg: encode logo: %w", err)
	}

	bx := x0 + (region-box)/2
	by := y0 + (region-box)/2
	fmt.Fprintf(sb, `<image x="%d" y="%d" width="%d" height="%d" preserveAspectRatio="xMidYMid meet" href="%s"/>`, bx, by, box, box, escapeAttr(uri))
	return nil
}

func escapeAttr(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// scaleToFit shrinks img so its longest side is max pixels, preserving the
// aspect ratio. Images already within the limit are returned untouched.
func scaleToFit(img image.Image, max int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	longest := w
	if h > w {
		longest = h
	}
	if longest <= max {
		return img
	}
	tw, th := w*max/longest, h*max/longest
	if tw < 1 {
		tw = 1
	}
	if th < 1 {
		th = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Src, nil)
	return dst
}

func pngDataURI(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
