package qr

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"

	"github.com/nachop51/qr-go/render/png"
	"github.com/nachop51/qr-go/render/style"
	"github.com/nachop51/qr-go/render/svg"
)

// Styled SVG output must still decode: rasterize our own markup with oksvg
// and read it back with gozxing.
func TestStyledSVGRoundTrip(t *testing.T) {
	const want = "https://example.com/styled-round-trip"

	for name, renderer := range map[string]svg.SVG{
		"rounded-rounded": svg.New().
			ModuleShape(style.ModuleRounded).
			EyeShape(style.EyeRounded),
		"dot-circle-colored-eyes": svg.New().
			ModuleShape(style.ModuleDot).
			EyeShape(style.EyeCircle).
			EyeFrame("#1d4ed8").
			EyeBall("#b91c1c"),
		"rounded-gradient": svg.New().
			ModuleShape(style.ModuleRounded).
			EyeFrameShape(style.EyeRounded).
			EyeBallShape(style.EyeCircle).
			GradientLinear("#0f172a", "#7f1d1d", 45),
		"radial-gradient": svg.New().
			GradientRadial("#111111", "#1e3a8a"),
	} {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			code, err := NewTextBuilder(want).
				SetRenderer(renderer.Writer(&buf)).
				SetErrorCorrectionLevel(CorrectionLevelHigh).
				Build()
			if err != nil {
				t.Fatal(err)
			}
			if err := code.Render(); err != nil {
				t.Fatal(err)
			}

			img := rasterizeMarkup(t, buf.Bytes())
			bmp, _ := gozxing.NewBinaryBitmapFromImage(img)
			res, err := qrcode.NewQRCodeReader().Decode(bmp, nil)
			if err != nil {
				t.Fatalf("decode failed: %v\nmarkup:\n%s", err, buf.String())
			}
			if res.GetText() != want {
				t.Fatalf("decoded %q, want %q", res.GetText(), want)
			}
		})
	}
}

// Styled PNG output must still decode.
func TestStyledPNGRoundTrip(t *testing.T) {
	const want = "https://example.com/styled-round-trip"

	blue := color.RGBA{29, 78, 216, 255}
	red := color.RGBA{185, 28, 28, 255}
	navy := color.RGBA{15, 23, 42, 255}
	maroon := color.RGBA{127, 29, 29, 255}

	for name, renderer := range map[string]png.PNG{
		"rounded-rounded": png.New().
			ModuleShape(style.ModuleRounded).
			EyeShape(style.EyeRounded),
		"dot-circle-colored-eyes": png.New().
			ModuleShape(style.ModuleDot).
			EyeShape(style.EyeCircle).
			EyeFrame(blue).
			EyeBall(red),
		"rounded-linear-gradient": png.New().
			ModuleShape(style.ModuleRounded).
			GradientLinear(navy, maroon, 45),
		"radial-gradient": png.New().
			GradientRadial(color.Black, navy),
	} {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			code, err := NewTextBuilder(want).
				SetRenderer(renderer.Writer(&buf)).
				SetErrorCorrectionLevel(CorrectionLevelHigh).
				Build()
			if err != nil {
				t.Fatal(err)
			}
			if err := code.Render(); err != nil {
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
		})
	}
}

// Every module shape x eye frame shape x eye ball shape combination must
// survive a decode round-trip, at Quartile as well as High error correction.
// The full cross-product runs through the PNG renderer; SVG stays with the
// representative combos above (rasterizing markup per combo is much slower).
func TestStyledPNGShapeSweep(t *testing.T) {
	const want = "https://example.com/styled-shape-sweep"

	moduleShapes := []style.ModuleShape{style.ModuleSquare, style.ModuleRounded, style.ModuleDot}
	eyeShapes := []style.EyeShape{style.EyeSquare, style.EyeRounded, style.EyeCircle}
	levels := map[string]CorrectionLevel{
		"quartile": CorrectionLevelQuartile,
		"high":     CorrectionLevelHigh,
	}

	for levelName, level := range levels {
		for _, mod := range moduleShapes {
			for _, frame := range eyeShapes {
				for _, ball := range eyeShapes {
					name := levelName + "/" + mod.String() + "-" + frame.String() + "-" + ball.String()
					t.Run(name, func(t *testing.T) {
						var buf bytes.Buffer
						renderer := png.New().
							Writer(&buf).
							ModuleShape(mod).
							EyeFrameShape(frame).
							EyeBallShape(ball)
						code, err := NewTextBuilder(want).
							SetRenderer(renderer).
							SetErrorCorrectionLevel(level).
							Build()
						if err != nil {
							t.Fatal(err)
						}
						if err := code.Render(); err != nil {
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
					})
				}
			}
		}
	}
}

func rasterizeMarkup(t *testing.T, markup []byte) image.Image {
	t.Helper()
	icon, err := oksvg.ReadIconStream(bytes.NewReader(markup))
	if err != nil {
		t.Fatalf("parse svg: %v", err)
	}
	w, h := int(icon.ViewBox.W), int(icon.ViewBox.H)
	icon.SetTarget(0, 0, float64(w), float64(h))
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	scanner := rasterx.NewScannerGV(w, h, img, img.Bounds())
	icon.Draw(rasterx.NewDasher(w, h, scanner), 1.0)
	return img
}
