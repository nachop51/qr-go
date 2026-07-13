package qr

import (
	"bytes"
	"image"
	_ "image/png"
	"testing"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"

	"github.com/nachop51/qr-go/render/png"
)

func TestBinaryRoundTrip(t *testing.T) {
	const want = "HELLO WORLD"

	var buf bytes.Buffer

	qr, err := NewBinaryBuilder([]byte(want)).
		SetRenderer(png.New().Writer(&buf)).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("version=%d mask=%d size=%d ecLevel=%d", qr.Version(), qr.Mask(), qr.Size(), qr.CorrectionLevel())

	cw := buildCodewords(qr.Segments(), qr.Version(), qr.CorrectionLevel(), qr.UsesECI())
	t.Logf("codewords (%d): % X", len(cw), cw)

	if err := qr.Render(); err != nil {
		t.Fatal(err)
	}

	img, _, err := image.Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	bmp, _ := gozxing.NewBinaryBitmapFromImage(img)
	res, err := qrcode.NewQRCodeReader().Decode(bmp, nil)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if res.GetText() != want {
		t.Fatalf("decoded %q, want %q", res.GetText(), want)
	}
}
