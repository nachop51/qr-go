package png

import (
	"github.com/nachop51/qr-go/render"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"
)

type PNG struct {
	w      io.Writer
	dark   color.Color
	white  color.Color
	width  int
	height int
	quiet  int
}

func New() PNG {
	return PNG{dark: color.Black, white: color.White, quiet: 4, width: 800, height: 800}
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

func (p PNG) drawPixel(img *image.RGBA, x, y, pixelSize int, dark bool) {

	startX := x * pixelSize
	startY := y * pixelSize

	for i := range pixelSize {
		for j := range pixelSize {
			if dark {
				img.Set(startX+i, startY+j, p.dark)
			} else {
				img.Set(startX+i, startY+j, p.white)
			}
		}
	}
}

func (p PNG) drawQuiet(img *image.RGBA) {
	draw.Draw(img, img.Bounds(), &image.Uniform{C: p.white}, image.Point{}, draw.Src)
}

func (p PNG) Render(g render.Grid) error {
	size := max(p.width, p.height)

	pixelSize := size / (g.Size() + 2*p.quiet)

	img := image.NewRGBA(image.Rect(0, 0, p.width, p.height))
	p.drawQuiet(img)

	for y := 0; y < g.Size(); y++ {
		for x := 0; x < g.Size(); x++ {
			dark := g.IsDark(x, y)
			p.drawPixel(img, x+p.quiet, y+p.quiet, pixelSize, dark)
		}
	}

	w := p.w
	if w == nil {
		f, err := os.Create("test.png")
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}

	if err := png.Encode(w, img); err != nil {
		return err
	}

	return nil
}
