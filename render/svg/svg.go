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
// defaults to size/5 modules — safe at any error-correction level — and can be
// changed with LogoModules. The overlay covers whole modules snapped to the
// grid, so it never slices one.
//
// A logo hides the modules it covers; a span wider than the code's
// error-correction budget (see Code.MaxLogoModules) is capped to that budget so
// the result still scans, and the reduction is reported through render.Warnf.
func (s SVG) Logo(img image.Image) SVG { s.logo = img; return s }

// LogoModules sets how many modules across the logo spans. A value <= 0 restores
// the default of size/5. A span wider than the code's error-correction budget
// (roughly size/3 at High, size/4 at Quartile, size/5 at Medium) is capped to
// that budget so the code stays scannable.
func (s SVG) LogoModules(n int) SVG { s.logoModules = n; return s }

func (s SVG) Render(g render.Grid) error {
	module := s.module
	if module <= 0 {
		module = 10
	}

	quiet := max(s.quiet, 0)

	w := s.w
	if w == nil {
		w = os.Stdout
	}

	totalModules := g.Size() + 2*quiet
	size := totalModules * module

	var sb strings.Builder
	fmt.Fprintf(&sb, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d" shape-rendering="crispEdges">`, size, size, size, size)
	fmt.Fprintf(&sb, `<rect width="%d" height="%d" fill="%s"/>`, size, size, s.light)

	for y := 0; y < g.Size(); y++ {
		for x := 0; x < g.Size(); x++ {
			if !g.IsDark(x, y) {
				continue
			}

			px := (x + quiet) * module
			py := (y + quiet) * module
			fmt.Fprintf(&sb, `<rect x="%d" y="%d" width="%d" height="%d" fill="%s"/>`, px, py, module, module, s.dark)
		}
	}

	if s.logo != nil {
		if mods := render.ResolveLogo(g, s.logoModules); mods > 0 {
			s.drawLogo(&sb, g.Size(), quiet, module, mods)
		}
	}

	sb.WriteString(`</svg>`)

	_, err := io.WriteString(w, sb.String())
	return err
}

// drawLogo clears a module-aligned square region and embeds the logo inside it.
func (s SVG) drawLogo(sb *strings.Builder, size, quiet, module, mods int) {
	// Cleared region, expressed in whole modules and snapped to the grid.
	start := (size - mods) / 2
	x0 := (quiet + start) * module
	y0 := (quiet + start) * module
	region := mods * module
	fmt.Fprintf(sb, `<rect x="%d" y="%d" width="%d" height="%d" fill="%s"/>`, x0, y0, region, region, s.light)

	uri, err := pngDataURI(s.logo)
	if err != nil {
		return // skip the logo rather than emit broken markup
	}

	// Leave a thin light ring; the browser fits and centres the image within
	// the box, preserving its aspect ratio.
	box := region - module
	if box < 1 {
		box = region
	}
	bx := x0 + (region-box)/2
	by := y0 + (region-box)/2
	fmt.Fprintf(sb, `<image x="%d" y="%d" width="%d" height="%d" preserveAspectRatio="xMidYMid meet" href="%s"/>`, bx, by, box, box, uri)
}

func pngDataURI(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
