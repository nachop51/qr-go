package qr_test

import (
	"fmt"
	"log"
	"os"

	qr "github.com/nachop51/qr-go"
	"github.com/nachop51/qr-go/logo"
	"github.com/nachop51/qr-go/render/png"
)

func Example() {
	code, err := qr.NewTextBuilder("HELLO WORLD").
		SetErrorCorrectionLevel(qr.CorrectionLevelHigh).
		Build()
	if err != nil {
		panic(err)
	}

	fmt.Printf("version=%d size=%d segments=%d\n", code.Version(), code.Size(), len(code.Segments()))
	// Output: version=2 size=25 segments=1
}

func ExampleNewBinaryBuilder() {
	code, err := qr.NewBinaryBuilder([]byte{0x00, 0x01, 0x02, 0xff}).Build()
	if err != nil {
		panic(err)
	}

	segments := code.Segments()
	fmt.Printf("segments=%d mode=%s\n", len(segments), segments[0].Mode())
	// Output: segments=1 mode=Byte
}

// Overlay a logo loaded from any image format. logo.Decode handles PNG, JPEG,
// GIF, WebP, and SVG (SVG is rasterized), returning the image.Image that both
// the PNG and SVG renderers accept via Logo. Pair a logo with a high
// error-correction level so the covered modules can still be recovered.
func Example_logo() {
	f, err := os.Open("brand.svg") // or .png / .jpg / .webp / .gif
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	img, err := logo.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	code, err := qr.NewTextBuilder("https://example.com").
		SetErrorCorrectionLevel(qr.CorrectionLevelHigh).
		SetRenderer(png.New().Logo(img)). // or svg.New().Logo(img)
		Build()
	if err != nil {
		log.Fatal(err)
	}

	if err := code.Render(); err != nil { // writes image.png, logo centered
		log.Fatal(err)
	}
}
