package logo

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

func solid(w, h int, c color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
	return img
}

func redish(t *testing.T, img image.Image, x, y int) {
	t.Helper()
	r, g, b, _ := img.At(x, y).RGBA()
	if r>>8 < 150 || g>>8 > 90 || b>>8 > 90 {
		t.Errorf("pixel (%d,%d) not red: r=%d g=%d b=%d", x, y, r>>8, g>>8, b>>8)
	}
}

func TestDecodePNG(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, solid(20, 20, color.RGBA{200, 30, 30, 255})); err != nil {
		t.Fatal(err)
	}
	img, err := DecodeBytes(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 20 || img.Bounds().Dy() != 20 {
		t.Fatalf("bounds = %v, want 20x20", img.Bounds())
	}
	redish(t, img, 10, 10)
}

func TestDecodeJPEG(t *testing.T) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, solid(24, 16, color.RGBA{200, 30, 30, 255}), nil); err != nil {
		t.Fatal(err)
	}
	img, err := DecodeBytes(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 24 || img.Bounds().Dy() != 16 {
		t.Fatalf("bounds = %v, want 24x16", img.Bounds())
	}
}

func TestDecodeGIF(t *testing.T) {
	var buf bytes.Buffer
	if err := gif.Encode(&buf, solid(12, 9, color.RGBA{200, 30, 30, 255}), nil); err != nil {
		t.Fatal(err)
	}
	img, err := DecodeBytes(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 12 || img.Bounds().Dy() != 9 {
		t.Fatalf("bounds = %v, want 12x9", img.Bounds())
	}
}

func TestDecodeWebP(t *testing.T) {
	// A small lossless WebP fixture keeps this test independent of external
	// tools; x/image/webp intentionally provides a decoder but no encoder.
	data, err := base64.StdEncoding.DecodeString("UklGRrIBAABXRUJQVlA4TKUBAAAvSsAYAA8w//M///MfeJAkbXvaSG7m8Q3GfYSBJekwQztm/IcZlgwnmWImn2BK7aFmBtnVir6q//8VOkFE/xm4baTIu8c48ArEo6+B3zFKYln3pqClSCKX0begFTAXFOLXHSyF8cCNcZEG4OywuA4KVVfJCiArU7GAgJI8+lJP/OKMT/fBAjevg1cYB7YVkFuWga2lyPi5I0HFy5YTpWIHg0RZpkniRVW9odHAKOwosWuOGdxIyn2OvaCDvhg/we6TwadPBPbqBV58MsLmMJ8yZnOWk8SRz4N+QoyPL+MnamzMvcE1rHNEr91F9GKZPVUcS9w7PhhH36suB9qPeYb/oLk6cuTiJ0wOK3m5h1cKjW6EVZCYMK7dxcKCBdgP9HkKr9gkAO2P8GKZGWVdIAatQa+1IDpt6qyorVwdy01xdW8Jkfk6xjEXmVQQ+HQdFr6OKhIN34dXWq0+0qr6EJSCeeVLH9+gvGTLyqM65PQ44ihzlTXxQKjKbAvshXgir7Lil9w4L2bvMycmjQcqXaMCO6BlY28i+FOLzbfI1vEqxAhotocAAA==")
	if err != nil {
		t.Fatal(err)
	}
	img, err := DecodeBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 75 || img.Bounds().Dy() != 100 {
		t.Fatalf("bounds = %v, want 75x100", img.Bounds())
	}
}

func TestDecodeSVG(t *testing.T) {
	svg := []byte(`<?xml version="1.0"?>
<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100">
<rect width="100" height="100" fill="#c81e1e"/></svg>`)

	img, err := DecodeBytes(svg)
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	// A 100px viewBox is scaled up so its long edge is at least svgTarget.
	if m := max(b.Dx(), b.Dy()); m < svgTarget {
		t.Fatalf("rasterized long edge %d, want >= %d", m, svgTarget)
	}
	redish(t, img, b.Dx()/2, b.Dy()/2)
}

func TestIsSVG(t *testing.T) {
	cases := map[string]bool{
		`<svg xmlns="http://www.w3.org/2000/svg">`: true,
		`<?xml version="1.0"?>` + "\n" + `<svg>`:   true,
		"\x89PNG\r\n\x1a\n":                        false,
		"just some text":                           false,
		"binary payload with <svg later":           false,
	}
	for in, want := range cases {
		if got := isSVG([]byte(in)); got != want {
			t.Errorf("isSVG(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestDecodeLimitsAndPathologicalSVG(t *testing.T) {
	if _, err := DecodeBytes(make([]byte, MaxEncodedBytes+1)); err == nil {
		t.Fatal("expected oversized encoded input to be rejected")
	}
	bad := []string{
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10000001 1"/>`,
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1000000 1"/>`,
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 NaN 1"/>`,
	}
	for _, input := range bad {
		if _, err := DecodeBytes([]byte(input)); err == nil {
			t.Errorf("expected SVG %q to be rejected", input)
		}
	}
}

func TestDecodeGarbage(t *testing.T) {
	if _, err := DecodeBytes([]byte("not an image at all")); err == nil {
		t.Error("expected an error decoding non-image bytes")
	}
}
