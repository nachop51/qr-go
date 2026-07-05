package qr_test

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	_ "image/png"
	"testing"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"

	qr "github.com/nachop51/qr-go"
	pngr "github.com/nachop51/qr-go/render/png"
)

// A code with a centred logo must still decode at a high error-correction
// level. A span of size/4 modules sits within High's proven scannable budget.
func TestLogoStillScannable(t *testing.T) {
	const want = "https://github.com/nachop51/qr-go"

	logo := image.NewRGBA(image.Rect(0, 0, 256, 256))
	draw.Draw(logo, logo.Bounds(), &image.Uniform{C: color.RGBA{200, 30, 30, 255}}, image.Point{}, draw.Src)

	code, err := qr.NewTextBuilder(want).SetErrorCorrectionLevel(qr.CorrectionLevelHigh).Build()
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := pngr.New().Writer(&buf).Logo(logo).LogoModules(code.Size() / 4).Render(code); err != nil {
		t.Fatal(err)
	}

	img, _, err := image.Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		t.Fatal(err)
	}
	res, err := qrcode.NewQRCodeReader().Decode(bmp, nil)
	if err != nil {
		t.Fatalf("logo'd code did not decode: %v", err)
	}
	if res.GetText() != want {
		t.Fatalf("decoded %q, want %q", res.GetText(), want)
	}
}

func TestMaxLogoModules(t *testing.T) {
	for _, c := range []struct {
		level qr.CorrectionLevel
		div   int
	}{
		{qr.CorrectionLevelHigh, 3},
		{qr.CorrectionLevelQuartile, 4},
		{qr.CorrectionLevelMedium, 5},
		{qr.CorrectionLevelLow, 6},
	} {
		code, err := qr.NewTextBuilder("HELLO WORLD").SetErrorCorrectionLevel(c.level).Build()
		if err != nil {
			t.Fatal(err)
		}
		if got, want := code.MaxLogoModules(), code.Size()/c.div; got != want {
			t.Errorf("div %d: MaxLogoModules = %d, want %d", c.div, got, want)
		}
	}
}
