package png

import (
	"image/color"
	"testing"
)

func FuzzPNGRendererOptions(f *testing.F) {
	f.Add(256, 256, 4)
	f.Add(0, -1, -1)
	f.Add(8193, 8193, 257)
	f.Fuzz(func(t *testing.T, width, height, quiet int) {
		// Exercise boundary validation without allowing random valid dimensions
		// to allocate tens of megapixels during a fuzz campaign.
		if width > 512 && width <= 8192 {
			width = 8193
		}
		if height > 512 && height <= 8192 {
			height = 8193
		}
		_, _ = New().Width(width).Height(height).Quiet(quiet).
			Dark(color.Black).White(color.White).Bytes(budgetGrid{n: 21, budget: 3})
	})
}
