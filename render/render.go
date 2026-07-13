// Package render defines the renderer contract. Zero dependencies, so both the
// core qr package and every renderer can import it without a cycle and without
// pulling anything heavy.
package render

import "fmt"

type Grid interface {
	Size() int
	IsDark(x, y int) bool
}

type Renderer interface {
	// Render writes the code to the renderer's configured sink (writer or file).
	Render(g Grid) error
	// Bytes returns the rendered output instead of writing it: UTF-8 text for
	// terminal, markup for SVG, an encoded image for PNG.
	Bytes(g Grid) ([]byte, error)
}

// ValidateGrid rejects nil or pathological grids before renderers iterate or
// allocate from values supplied by third-party Grid implementations.
func ValidateGrid(g Grid) error {
	if g == nil {
		return fmt.Errorf("render: grid is nil")
	}
	n := g.Size()
	if n < 1 || n > 177 {
		return fmt.Errorf("render: grid size must be between 1 and 177 (got %d)", n)
	}
	return nil
}
