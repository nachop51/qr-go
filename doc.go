// Package qr encodes text or binary data into QR codes.
//
// Start with [NewTextBuilder] or [NewBinaryBuilder], tune the result with the
// fluent setters — [Builder.SetErrorCorrectionLevel], [Builder.SetRenderer],
// [Builder.SetTextECIPolicy] — then call [Builder.Build]:
//
//	code, err := qr.NewTextBuilder("HELLO WORLD").
//		SetErrorCorrectionLevel(qr.CorrectionLevelHigh).
//		Build()
//	if err != nil {
//		// handle err
//	}
//	code.Render() // uses the configured renderer (terminal by default)
//
// The builder picks the encoding segmentation, symbol version, and mask
// automatically. The resulting [Code] exposes the outcome ([Code.Version],
// [Code.Mask], [Code.Size], [Code.IsDark]) and the module matrix.
//
// Output is produced by a [github.com/nachop51/qr-go/render.Renderer]. Three
// implementations ship with the module:
//
//   - render/terminal — compact half-block text (the default)
//   - render/png — raster images
//   - render/svg — vector markup
//
// The PNG and SVG renderers can overlay a centred logo (their Logo method); the
// logo package decodes that logo from any common image format — PNG, JPEG, GIF,
// WebP, or SVG (SVG is rasterized) — into the image.Image they accept.
//
// The content package builds the specially formatted payloads that scanners
// turn into actions (Wi-Fi networks, contact cards, calendar events, and more).
package qr
