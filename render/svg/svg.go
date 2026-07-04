package svg

import (
	"fmt"
	"io"
	"os"
	"strings"

	"nachop51/qr/render"
)

type SVG struct {
	w      io.Writer
	dark   string
	light  string
	quiet  int
	module int
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

	sb.WriteString(`</svg>`)

	_, err := io.WriteString(w, sb.String())
	return err
}
