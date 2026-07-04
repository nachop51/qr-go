package png

import (
	"fmt"
	"image"
	"image/color"
	"testing"
)

func TestDrawQuietFillsBackgroundAndPreservesModules(t *testing.T) {
	p := New()
	img := image.NewRGBA(image.Rect(0, 0, 6, 6))

	p.drawQuiet(img)
	p.drawPixel(img, 1, 1, 2, true)

	wantWhite := color.RGBAModel.Convert(p.white).(color.RGBA)
	wantDark := color.RGBAModel.Convert(p.dark).(color.RGBA)

	for y := range 6 {
		for x := range 6 {
			want := wantWhite
			if x >= 2 && x < 4 && y >= 2 && y < 4 {
				want = wantDark
			}

			if got := img.RGBAAt(x, y); got != want {
				t.Fatalf("pixel (%d, %d): got %s, want %s", x, y, rgbaString(got), rgbaString(want))
			}
		}
	}
}

func rgbaString(c color.RGBA) string {
	return fmt.Sprintf("rgba(%d,%d,%d,%d)", c.R, c.G, c.B, c.A)
}
