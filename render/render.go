// Package render defines the renderer contract. Zero dependencies, so both the
// core qr package and every renderer can import it without a cycle and without
// pulling anything heavy.
package render

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
