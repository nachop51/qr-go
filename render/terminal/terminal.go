// Package terminal renders a QR grid as text. Zero dependencies beyond the
// render contract: the lightweight, output-anywhere default.
//
// By default it uses Unicode half-block glyphs (▀ ▄ █), packing two vertical
// modules into each character cell. Because a terminal cell is about twice as
// tall as it is wide, this makes every module roughly square and halves both
// the width and the height compared with the classic full-block style.
package terminal

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nachop51/qr-go/render"
)

// Half-block glyphs: each character cell stacks a top and a bottom module.
const (
	glyphFull  = '█' // top dark, bottom dark
	glyphUpper = '▀' // top dark, bottom light
	glyphLower = '▄' // top light, bottom dark
	glyphEmpty = ' ' // top light, bottom light
)

// Terminal writes the QR as text to its writer. Construct with New; fields are
// unexported so defaults always apply and the value can't be half-configured.
type Terminal struct {
	w      io.Writer
	dark   string // block-mode dark fill
	light  string // block-mode light fill
	quiet  int
	half   bool // half-block mode (the default)
	invert bool
}

// New returns a Terminal with defaults: os.Stdout, half-block modules, quiet 2.
func New() Terminal {
	return Terminal{w: os.Stdout, dark: "██", light: "  ", quiet: 2, half: true}
}

// Fluent overrides (value receiver → chainable, immutable).
func (t Terminal) Writer(w io.Writer) Terminal { t.w = w; return t }
func (t Terminal) Quiet(n int) Terminal        { t.quiet = n; return t }

// HalfBlock selects the compact half-block style (the default).
func (t Terminal) HalfBlock() Terminal { t.half = true; return t }

// Block selects the classic full-block style: two character cells per module,
// using the Dark and Light fill strings.
func (t Terminal) Block() Terminal { t.half = false; return t }

// Dark sets the full-block dark fill and switches to Block style, since custom
// fill strings only apply there.
func (t Terminal) Dark(s string) Terminal { t.dark = s; t.half = false; return t }

// Light sets the full-block light fill and switches to Block style.
func (t Terminal) Light(s string) Terminal { t.light = s; t.half = false; return t }

// Invert swaps dark and light. Useful when the terminal theme would otherwise
// render the code with reversed contrast (e.g. a dark background).
func (t Terminal) Invert() Terminal { t.invert = true; return t }

func (t Terminal) Render(g render.Grid) error {
	w := t.w
	if w == nil {
		w = os.Stdout
	}

	text, err := t.text(g)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, text)
	return err
}

// Bytes returns the rendered QR as UTF-8 text.
func (t Terminal) Bytes(g render.Grid) ([]byte, error) {
	text, err := t.text(g)
	return []byte(text), err
}

// validate rejects bad inputs before rendering iterates over them, mirroring
// the SVG and PNG renderers.
func (t Terminal) validate(g render.Grid) error {
	if err := render.ValidateGrid(g); err != nil {
		return err
	}
	if t.quiet < 0 || t.quiet > 256 {
		return fmt.Errorf("terminal: quiet zone must be between 0 and 256")
	}
	return nil
}

func (t Terminal) text(g render.Grid) (string, error) {
	if err := t.validate(g); err != nil {
		return "", err
	}
	if t.half {
		return t.renderHalf(g), nil
	}
	return t.renderBlock(g), nil
}

// ink reports whether a character cell at padded coordinate (x, y) should be
// drawn as a dark module. The quiet zone and anything outside the grid counts
// as light. Invert flips the result.
func (t Terminal) ink(g render.Grid, x, y int) bool {
	q := max(t.quiet, 0)
	total := g.Size() + 2*q

	// Anything past the padded symbol is terminal background and must never be
	// inverted. This matters for the trailing half-block filler row that appears
	// when total is odd (QR sizes always are): inverting it would draw an extra
	// dark module below the quiet zone, making the bottom border a module taller
	// than the top.
	if x < 0 || y < 0 || x >= total || y >= total {
		return false
	}

	mx, my := x-q, y-q
	dark := mx >= 0 && my >= 0 && mx < g.Size() && my < g.Size() && g.IsDark(mx, my)
	return dark != t.invert
}

func (t Terminal) renderHalf(g render.Grid) string {
	q := max(t.quiet, 0)
	total := g.Size() + 2*q

	var sb strings.Builder
	for y := 0; y < total; y += 2 {
		for x := range total {
			top := t.ink(g, x, y)
			bottom := t.ink(g, x, y+1) // one past the grid reads as light
			sb.WriteRune(halfGlyph(top, bottom))
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func halfGlyph(top, bottom bool) rune {
	switch {
	case top && bottom:
		return glyphFull
	case top:
		return glyphUpper
	case bottom:
		return glyphLower
	default:
		return glyphEmpty
	}
}

func (t Terminal) renderBlock(g render.Grid) string {
	q := max(t.quiet, 0)
	total := g.Size() + 2*q

	var sb strings.Builder
	for y := range total {
		for x := range total {
			if t.ink(g, x, y) {
				sb.WriteString(t.dark)
			} else {
				sb.WriteString(t.light)
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}
