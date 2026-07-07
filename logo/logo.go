// Package logo decodes a QR-code logo from any supported image format into an
// image.Image that the renderers can overlay.
//
// Raster formats (PNG, JPEG, GIF, WebP) are decoded directly. SVG is
// rasterized (it has no pixels of its own) at a resolution derived from its
// viewBox, so it stays crisp when the renderer scales it down.
//
//	f, _ := os.Open("brand.svg")
//	img, err := logo.Decode(f)
//	// then: png.New().Logo(img)  or  svg.New().Logo(img)
package logo

import (
	"bytes"
	"fmt"
	"image"
	"io"

	_ "image/gif"  // register GIF decoder
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	_ "golang.org/x/image/webp" // register WebP decoder
)

// svgTarget is the minimum size, in pixels, an SVG is rasterized to on its
// longer edge, large enough to stay sharp after the renderer scales it.
const svgTarget = 512

// Decode reads a logo from r, detecting the format from its content. It supports
// PNG, JPEG, GIF, WebP, and SVG.
func Decode(r io.Reader) (image.Image, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("logo: read: %w", err)
	}
	return DecodeBytes(data)
}

// DecodeBytes is Decode over an in-memory buffer.
func DecodeBytes(data []byte) (image.Image, error) {
	if isSVG(data) {
		return rasterizeSVG(data)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("logo: decode: %w", err)
	}
	return img, nil
}

// isSVG reports whether data looks like SVG markup (a bare <svg or an XML
// prolog followed by one).
func isSVG(data []byte) bool {
	return bytes.Contains(data, []byte("<svg")) || bytes.Contains(data, []byte("<SVG"))
}

func rasterizeSVG(data []byte) (image.Image, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("logo: parse svg: %w", err)
	}

	w, h := int(icon.ViewBox.W), int(icon.ViewBox.H)
	if w <= 0 || h <= 0 {
		w, h = svgTarget, svgTarget
	}
	// Scale up small viewBoxes so downscaling by the renderer stays crisp.
	if m := max(w, h); m < svgTarget {
		w = w * svgTarget / m
		h = h * svgTarget / m
	}

	icon.SetTarget(0, 0, float64(w), float64(h))
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	scanner := rasterx.NewScannerGV(w, h, img, img.Bounds())
	icon.Draw(rasterx.NewDasher(w, h, scanner), 1.0)
	return img, nil
}
