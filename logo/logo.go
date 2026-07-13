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
	"encoding/xml"
	"fmt"
	"image"
	"io"
	"math"
	"strconv"
	"strings"

	_ "image/gif"  // register GIF decoder
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	_ "golang.org/x/image/webp" // register WebP decoder
)

// svgTarget is the minimum size, in pixels, an SVG is rasterized to on its
// longer edge, large enough to stay sharp after the renderer scales it.
const (
	svgTarget          = 512
	MaxEncodedBytes    = 16 << 20
	MaxSourceDimension = 4096
	MaxSourcePixels    = 16_000_000
	MaxSVGRasterEdge   = 2048
)

// Decode reads a logo from r, detecting the format from its content. It supports
// PNG, JPEG, GIF, WebP, and SVG.
func Decode(r io.Reader) (image.Image, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxEncodedBytes+1))
	if err != nil {
		return nil, fmt.Errorf("logo: read: %w", err)
	}
	return DecodeBytes(data)
}

// DecodeBytes is Decode over an in-memory buffer.
func DecodeBytes(data []byte) (image.Image, error) {
	if len(data) > MaxEncodedBytes {
		return nil, fmt.Errorf("logo: encoded input exceeds 16 MiB")
	}
	if isSVG(data) {
		return rasterizeSVG(data)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("logo: decode configuration: %w", err)
	}
	if err := validateDimensions(cfg.Width, cfg.Height); err != nil {
		return nil, err
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
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '<' {
		return false
	}
	d := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := d.Token()
		if err != nil {
			return false
		}
		if start, ok := tok.(xml.StartElement); ok {
			return strings.EqualFold(start.Name.Local, "svg") && (start.Name.Space == "" || start.Name.Space == "http://www.w3.org/2000/svg")
		}
	}
}

func rasterizeSVG(data []byte) (image.Image, error) {
	if err := validateSVGRoot(data); err != nil {
		return nil, err
	}
	icon, err := oksvg.ReadIconStream(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("logo: parse svg: %w", err)
	}

	vw, vh := icon.ViewBox.W, icon.ViewBox.H
	if math.IsNaN(vw) || math.IsNaN(vh) || math.IsInf(vw, 0) || math.IsInf(vh, 0) || vw < 0 || vh < 0 || vw > 1e7 || vh > 1e7 {
		return nil, fmt.Errorf("logo: invalid or pathological SVG viewBox")
	}
	w, h := int(math.Ceil(vw)), int(math.Ceil(vh))
	if w <= 0 || h <= 0 {
		w, h = svgTarget, svgTarget
	}
	if max(w, h)/max(min(w, h), 1) > 10_000 {
		return nil, fmt.Errorf("logo: SVG viewBox aspect ratio is too extreme")
	}
	// Scale up small viewBoxes so downscaling by the renderer stays crisp.
	if m := max(w, h); m < svgTarget {
		w = w * svgTarget / m
		h = h * svgTarget / m
	}
	if m := max(w, h); m > MaxSVGRasterEdge {
		w = max(1, w*MaxSVGRasterEdge/m)
		h = max(1, h*MaxSVGRasterEdge/m)
	}
	if err := validateDimensions(w, h); err != nil {
		return nil, err
	}

	icon.SetTarget(0, 0, float64(w), float64(h))
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	scanner := rasterx.NewScannerGV(w, h, img, img.Bounds())
	icon.Draw(rasterx.NewDasher(w, h, scanner), 1.0)
	return img, nil
}

func validateSVGRoot(data []byte) error {
	d := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := d.Token()
		if err != nil {
			return fmt.Errorf("logo: parse svg root: %w", err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if !strings.EqualFold(start.Name.Local, "svg") || (start.Name.Space != "" && start.Name.Space != "http://www.w3.org/2000/svg") {
			return fmt.Errorf("logo: document root is not svg")
		}
		for _, attr := range start.Attr {
			if !strings.EqualFold(attr.Name.Local, "viewBox") {
				continue
			}
			parts := strings.Fields(attr.Value)
			if len(parts) != 4 {
				return fmt.Errorf("logo: invalid SVG viewBox")
			}
			values := make([]float64, 4)
			for i, part := range parts {
				v, err := strconv.ParseFloat(part, 64)
				if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
					return fmt.Errorf("logo: invalid SVG viewBox")
				}
				values[i] = v
			}
			if values[2] <= 0 || values[3] <= 0 || values[2] > 1e7 || values[3] > 1e7 || max(values[2], values[3])/min(values[2], values[3]) > 10_000 {
				return fmt.Errorf("logo: invalid or pathological SVG viewBox")
			}
		}
		return nil
	}
}

func validateDimensions(w, h int) error {
	if w < 1 || h < 1 || w > MaxSourceDimension || h > MaxSourceDimension || int64(w)*int64(h) > MaxSourcePixels {
		return fmt.Errorf("logo: dimensions must be positive, at most 4096 pixels per edge and 16 megapixels")
	}
	return nil
}
