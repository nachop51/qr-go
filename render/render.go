// Package render defines the renderer contract. Zero dependencies, so both the
// core qr package and every renderer can import it without a cycle and without
// pulling anything heavy.
package render

type Grid interface {
	Size() int
	IsDark(x, y int) bool
}

type Renderer interface {
	Render(g Grid) error
}
