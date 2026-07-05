package logo_test

import (
	"fmt"

	"github.com/nachop51/qr-go/logo"
)

// Decode normalises any supported image format — PNG, JPEG, GIF, WebP, or SVG —
// into an image.Image the renderers can overlay. Here an inline SVG is
// rasterised; its 100-unit viewBox is scaled up so the result stays crisp when
// the renderer shrinks it.
func ExampleDecodeBytes() {
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">` +
		`<circle cx="50" cy="50" r="50" fill="#00ADD8"/></svg>`)

	img, err := logo.DecodeBytes(svg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%dx%d\n", img.Bounds().Dx(), img.Bounds().Dy())
	// Output: 512x512
}
