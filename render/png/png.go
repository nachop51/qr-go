package png

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"

	xdraw "golang.org/x/image/draw"

	"github.com/nachop51/qr-go/render"
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
// and the reduction is reported through render.Warnf.
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

func (p PNG) Filename(name string) PNG {
	p.filename = name
	return p
}

func (p PNG) Writer(w io.Writer) PNG {
	p.w = w
	return p
}

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

// buildImage renders the code (and optional logo) into an RGBA image.
func (p PNG) buildImage(g render.Grid) *image.RGBA {
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

	for y := range size {
		for x := range size {
			if g.IsDark(x, y) {
				p.drawModule(img, offX+(x+p.quiet)*scale, offY+(y+p.quiet)*scale, scale)
			}
		}
	}

	if p.logo != nil {
		if mods := render.ResolveLogo(g, p.logoModules); mods > 0 {
			p.drawLogo(img, offX, offY, size, scale, mods)
		}
	}

	return img
}

func (p PNG) drawLogo(img *image.RGBA, offX, offY, size, scale, mods int) {
	// Cleared region, expressed in whole modules and snapped to the grid.
	start := (size - mods) / 2
	x0 := offX + (p.quiet+start)*scale
	y0 := offY + (p.quiet+start)*scale
	region := mods * scale
	draw.Draw(img, image.Rect(x0, y0, x0+region, y0+region),
		&image.Uniform{C: p.white}, image.Point{}, draw.Src)

	// Fit the logo inside the region, leaving a one-module white ring on every
	// side so it never touches the surrounding modules, preserving the aspect
	// ratio, centred on the region.
	box := region - 2*scale
	if box < scale {
		box = region - scale
	}
	if box < 1 {
		box = region
	}
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
	if err := png.Encode(&buf, p.buildImage(g)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p PNG) Render(g render.Grid) error {
	img := p.buildImage(g)

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
