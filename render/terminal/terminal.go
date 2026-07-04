// Package terminal renders a QR grid as text. Zero dependencies beyond the
// render contract — the lightweight, output-anywhere default.
package terminal

import (
	"io"
	"os"
	"strings"

	"github.com/nachop51/qr-go/render"
)

// Terminal writes the QR as text to its writer. Construct with New; fields are
// unexported so defaults always apply and the value can't be half-configured.
type Terminal struct {
	w     io.Writer
	dark  string
	light string
	quiet int
}

// New returns a Terminal with defaults: os.Stdout, block modules, quiet 2.
func New() Terminal {
	return Terminal{w: os.Stdout, dark: "██", light: "  ", quiet: 2}
}

// Fluent overrides (value receiver → chainable, immutable).
func (t Terminal) Writer(w io.Writer) Terminal { t.w = w; return t }
func (t Terminal) Dark(s string) Terminal      { t.dark = s; return t }
func (t Terminal) Light(s string) Terminal     { t.light = s; return t }
func (t Terminal) Quiet(n int) Terminal        { t.quiet = n; return t }

func (t Terminal) Render(g render.Grid) error {
	var sb strings.Builder

	quiet := max(t.quiet, 0)
	total := g.Size() + 2*quiet

	writeModules := func(s string, n int) {
		for range n {
			sb.WriteString(s)
		}
	}
	writeQuietRow := func() {
		writeModules(t.light, total)
		sb.WriteString("\n")
	}

	for range quiet {
		writeQuietRow()
	}

	for y := range g.Size() {
		writeModules(t.light, quiet)
		for x := range g.Size() {
			if g.IsDark(x, y) {
				sb.WriteString(t.dark)
			} else {
				sb.WriteString(t.light)
			}
		}
		writeModules(t.light, quiet)
		sb.WriteString("\n")
	}

	for range quiet {
		writeQuietRow()
	}

	_, err := t.w.Write([]byte(sb.String()))
	return err
}
