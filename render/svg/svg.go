package svg

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"strings"

	xdraw "golang.org/x/image/draw"

	"github.com/nachop51/qr-go/render"
)

type SVG struct {
	w           io.Writer
	dark        string
	light       string
	quiet       int
	module      int
	logo        image.Image
	logoModules int
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

func (s SVG) Writer(w io.Writer) SVG { s.w = w; return s }
func (s SVG) Dark(c string) SVG      { s.dark = c; return s }
func (s SVG) Light(c string) SVG     { s.light = c; return s }
func (s SVG) Quiet(n int) SVG        { s.quiet = n; return s }
func (s SVG) Module(n int) SVG       { s.module = n; return s }

// Logo overlays img centred on the code, embedded as a data URI. Its span
// defaults to the largest size the code's error-correction level can afford
// to lose (see Code.MaxLogoModules) and can be lowered with LogoModules. The
// overlay covers whole modules snapped to the grid, so it never slices one,
// and the logo is inset one module inside the cleared region so it never
// touches the surrounding modules.
//
// A logo hides the modules it covers; a span wider than the code's
// error-correction budget is capped to that budget so the result still scans,
// and the reduction is reported through render.Warnf.
func (s SVG) Logo(img image.Image) SVG { s.logo = img; return s }

// LogoModules sets how many modules across the logo spans. A value <= 0 restores
// the default: the code's full error-correction budget (roughly size/3 at High,
// size/4 at Quartile, size/5 at Medium, size/6 at Low). A span wider than that
// budget is capped to it so the code stays scannable.
func (s SVG) LogoModules(n int) SVG { s.logoModules = n; return s }

func (s SVG) Render(g render.Grid) error {
	w := s.w
	if w == nil {
		w = os.Stdout
	}

	_, err := io.WriteString(w, s.markup(g))
	return err
}

// Bytes returns the rendered QR as SVG markup.
func (s SVG) Bytes(g render.Grid) ([]byte, error) {
	return []byte(s.markup(g)), nil
}

func (s SVG) markup(g render.Grid) string {
	module := s.module
	if module <= 0 {
		module = 10
	}

	quiet := max(s.quiet, 0)

	totalModules := g.Size() + 2*quiet
	size := totalModules * module

	var sb strings.Builder
	fmt.Fprintf(&sb, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d" shape-rendering="crispEdges">`, size, size, size, size)
	fmt.Fprintf(&sb, `<rect width="%d" height="%d" fill="%s"/>`, size, size, s.light)

	// All dark modules as a single <path>: one DOM node instead of one <rect>
	// per module, with horizontal runs merged so a solid row is one command.
	sb.WriteString(`<path fill="` + s.dark + `" d="`)
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

	if s.logo != nil {
		if mods := render.ResolveLogo(g, s.logoModules); mods > 0 {
			s.drawLogo(&sb, g.Size(), quiet, module, mods)
		}
	}

	sb.WriteString(`</svg>`)
	return sb.String()
}

// drawLogo clears a module-aligned square region and embeds the logo inside it.
func (s SVG) drawLogo(sb *strings.Builder, size, quiet, module, mods int) {
	// Cleared region, expressed in whole modules and snapped to the grid.
	start := (size - mods) / 2
	x0 := (quiet + start) * module
	y0 := (quiet + start) * module
	region := mods * module
	fmt.Fprintf(sb, `<rect x="%d" y="%d" width="%d" height="%d" fill="%s"/>`, x0, y0, region, region, s.light)

	// Leave a one-module light ring on every side so the logo never touches
	// the surrounding modules; the browser fits and centres the image within
	// the box, preserving its aspect ratio.
	box := region - 2*module
	if box < module {
		box = region - module
	}
	if box < 1 {
		box = region
	}

	// Embed at 2x the drawn box (for hi-dpi screens), never above the source
	// resolution. Embedding the source as-is would put a full-resolution data
	// URI in the markup: a phone photo becomes tens of MB of SVG.
	uri, err := pngDataURI(scaleToFit(s.logo, 2*box))
	if err != nil {
		return // skip the logo rather than emit broken markup
	}

	bx := x0 + (region-box)/2
	by := y0 + (region-box)/2
	fmt.Fprintf(sb, `<image x="%d" y="%d" width="%d" height="%d" preserveAspectRatio="xMidYMid meet" href="%s"/>`, bx, by, box, box, uri)
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
