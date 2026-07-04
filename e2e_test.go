package qr

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"

	"github.com/nachop51/qr-go/internal/spec"
	"github.com/nachop51/qr-go/render/png"
)

// ---------------------------------------------------------------------------
// e2e helpers: generate a PNG with the library, decode it back with gozxing
// (a pure-Go ZXing port) and compare against the original input. This is a
// real round-trip: if the generated matrix is wrong in any way a standard
// decoder cannot read, these tests fail.
// ---------------------------------------------------------------------------

// renderPNG renders a built code to a PNG in the test's temp dir at the given
// module scale, returning the file path.
func renderPNG(t *testing.T, code *Code, scale int) string {
	t.Helper()

	const quiet = 4

	path := filepath.Join(t.TempDir(), "qr.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()

	size := (code.Size() + 2*quiet) * scale
	r := png.New().Writer(f).Quiet(quiet).Width(size).Height(size)
	if err := r.Render(code); err != nil {
		t.Fatalf("png.Render failed: %v", err)
	}
	return path
}

// buildPNG builds the QR and renders it to a PNG, returning the file path.
// Fails the test on any builder/IO error.
func buildPNG(t *testing.T, b *Builder) string {
	t.Helper()

	code, err := b.Build()
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}
	return renderPNG(t, code, 8)
}

// readResult decodes the QR PNG at path and returns the gozxing result.
func readResult(t *testing.T, path string) *gozxing.Result {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatalf("decode png %s: %v", path, err)
	}

	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		t.Fatalf("bitmap from image: %v", err)
	}

	// PURE_BARCODE selects gozxing's detector for clean, axis-aligned,
	// quiet-zoned images (exactly what this library renders). Without it the
	// generic detector misreads small/high-version pristine codes — a known
	// decoder quirk, reproducible even with reference generators like
	// rsc.io/qr — not a defect in the code under test.
	hints := map[gozxing.DecodeHintType]any{
		gozxing.DecodeHintType_PURE_BARCODE: true,
		gozxing.DecodeHintType_TRY_HARDER:   true,
	}
	res, err := qrcode.NewQRCodeReader().Decode(bmp, hints)
	if err != nil {
		t.Fatalf("QR decode failed (the generated code is unreadable): %v", err)
	}
	return res
}

// decodeText returns the textual payload a standard decoder reads back.
func decodeText(t *testing.T, path string) string {
	t.Helper()
	return readResult(t, path).GetText()
}

// decodeBytes reconstructs the raw byte payload from the decoder's byte
// segment metadata, so binary round-trips can be compared byte-for-byte.
func decodeBytes(t *testing.T, path string) []byte {
	t.Helper()

	res := readResult(t, path)
	md := res.GetResultMetadata()
	segs, ok := md[gozxing.ResultMetadataType_BYTE_SEGMENTS].([][]byte)
	if !ok || len(segs) == 0 {
		t.Fatalf("no BYTE_SEGMENTS metadata; cannot recover raw bytes")
	}
	var out []byte
	for _, s := range segs {
		out = append(out, s...)
	}
	return out
}

// ---------------------------------------------------------------------------
// Text round-trip across every error correction level
// ---------------------------------------------------------------------------

func TestE2E_TextRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{"numeric_short", "8675309"},
		{"numeric_long", strings.Repeat("0123456789", 8)},
		{"alnum_hello", "HELLO WORLD"},
		{"alnum_url_upper", "HTTPS://EXAMPLE.COM/PATH-1"},
		{"alnum_symbols", "ABC $%*+-./:1234"},
		{"byte_lower", "Hola mundo!"},
		{"byte_url", "https://github.com/github.com/nachop51/qr-go-go"},
		{"byte_json", `{"id":42,"ok":true}`},
		{"mixed_alnum_num", "HELLO123456789012345"},
		{"mixed_byte_num", "Order #100200300"},
		{"unicode_accents", "Café crème brûlée"},
		{"unicode_kanji", "日本語のテスト"},
		{"unicode_mixed", "ABC日本123"},
		{"unicode_emoji", "café ☕ 北京 🚀"},
		{"long_text", strings.Repeat("The quick brown fox 1234. ", 12)},
		{"very_long_text", strings.Repeat("Lorem ipsum dolor sit amet 9. ", 30)},
	}

	levels := []struct {
		name  string
		level CorrectionLevel
	}{
		{"L", CorrectionLevelLow},
		{"M", CorrectionLevelMedium},
		{"Q", CorrectionLevelQuartile},
		{"H", CorrectionLevelHigh},
	}

	for _, tc := range cases {
		for _, lv := range levels {
			t.Run(fmt.Sprintf("%s/%s", tc.name, lv.name), func(t *testing.T) {
				path := buildPNG(t, NewTextBuilder(tc.text).SetErrorCorrectionLevel(lv.level))

				got := decodeText(t, path)
				if got != tc.text {
					t.Fatalf("round-trip mismatch\n  want: %q\n  got:  %q", tc.text, got)
				}
			})
		}
	}
}

// ---------------------------------------------------------------------------
// Binary round-trip: raw bytes must come back byte-for-byte
// ---------------------------------------------------------------------------

func TestE2E_BinaryRoundTrip(t *testing.T) {
	mkrand := func(n int) []byte {
		b := make([]byte, n)
		if _, err := rand.Read(b); err != nil {
			t.Fatalf("rand: %v", err)
		}
		return b
	}

	cases := []struct {
		name    string
		payload []byte
	}{
		{"fixed_small", []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x01, 0x02}},
		{"all_zero", make([]byte, 16)},
		{"all_ff", bytes.Repeat([]byte{0xFF}, 16)},
		{"full_byte_range", func() []byte {
			b := make([]byte, 256)
			for i := range b {
				b[i] = byte(i)
			}
			return b
		}()},
		{"rand_32", mkrand(32)},
		{"rand_128", mkrand(128)},
		{"rand_512", mkrand(512)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := buildPNG(t, NewBinaryBuilder(tc.payload).SetErrorCorrectionLevel(CorrectionLevelMedium))

			got := decodeBytes(t, path)
			if !bytes.Equal(got, tc.payload) {
				t.Fatalf("binary round-trip mismatch\n  want (%d bytes): %x\n  got  (%d bytes): %x",
					len(tc.payload), tc.payload, len(got), got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Same payload, every ECC level decodes to the same content
// ---------------------------------------------------------------------------

func TestE2E_ECCLevelsConsistent(t *testing.T) {
	const payload = "Round-trip across all ECC levels: ABC日本 123 https://x.io"

	levels := []CorrectionLevel{
		CorrectionLevelLow,
		CorrectionLevelMedium,
		CorrectionLevelQuartile,
		CorrectionLevelHigh,
	}

	for i, lv := range levels {
		t.Run(fmt.Sprintf("level_%d", i), func(t *testing.T) {
			path := buildPNG(t, NewTextBuilder(payload).SetErrorCorrectionLevel(lv))
			if got := decodeText(t, path); got != payload {
				t.Fatalf("ECC level %d mismatch: want %q got %q", i, payload, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Version scaling: growing payloads push higher versions, all must decode
// ---------------------------------------------------------------------------

func TestE2E_VersionScaling(t *testing.T) {
	seenVersions := map[int]bool{}

	for _, n := range []int{1, 10, 25, 50, 100, 200, 400, 700} {
		t.Run(fmt.Sprintf("digits_%d", n), func(t *testing.T) {
			payload := strings.Repeat("1234567890", n/10+1)[:n]

			code, err := NewTextBuilder(payload).
				SetErrorCorrectionLevel(CorrectionLevelLow).
				Build()
			if err != nil {
				t.Fatalf("build n=%d: %v", n, err)
			}
			seenVersions[code.Version] = true

			path := renderPNG(t, code, 8)
			if got := decodeText(t, path); got != payload {
				t.Fatalf("version scaling n=%d (v%d) mismatch:\n want %q\n got  %q",
					n, code.Version, payload, got)
			}
		})
	}

	if len(seenVersions) < 3 {
		t.Errorf("expected several distinct versions exercised, got %v", seenVersions)
	}
}

// ---------------------------------------------------------------------------
// Render scale variation: the same code stays readable at different module
// pixel sizes (rendering is decoupled from the builder in the clean API).
// ---------------------------------------------------------------------------

func TestE2E_CustomScales(t *testing.T) {
	const payload = "Dimensions test 日本 123"

	for _, scale := range []int{4, 8, 12, 16} {
		t.Run(fmt.Sprintf("scale_%d", scale), func(t *testing.T) {
			code, err := NewTextBuilder(payload).Build()
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			path := renderPNG(t, code, scale)
			if got := decodeText(t, path); got != payload {
				t.Fatalf("scale %d mismatch: want %q got %q", scale, payload, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Optimal segmentation should pick the expected modes for known inputs
// ---------------------------------------------------------------------------

func TestSegmentModeDetection(t *testing.T) {
	cases := []struct {
		name  string
		text  string
		modes []EncodingMode
	}{
		{"pure_numeric", "123456", []EncodingMode{EncodingModeNumeric}},
		{"pure_alnum", "HELLO", []EncodingMode{EncodingModeAlphanumeric}},
		{"pure_byte", "hello", []EncodingMode{EncodingModeByte}},
		{"pure_kanji", "日本語", []EncodingMode{EncodingModeKanji}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, err := NewTextBuilder(tc.text).Build()
			if err != nil {
				t.Fatalf("build: %v", err)
			}

			got := make([]EncodingMode, len(code.Segments))
			for i, s := range code.Segments {
				got[i] = s.mode
			}
			if len(got) != len(tc.modes) {
				t.Fatalf("segment count: want %v got %v", tc.modes, got)
			}
			for i := range got {
				if got[i] != tc.modes[i] {
					t.Fatalf("segment %d mode: want %v got %v (all: %v)", i, tc.modes[i], got[i], got)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Builder error paths (no decode needed)
// ---------------------------------------------------------------------------

func TestBuildErrors(t *testing.T) {
	t.Run("invalid_utf8", func(t *testing.T) {
		_, err := NewTextBuilder(string([]byte{0xff, 0xfe, 0xfd})).Build()
		if err != spec.ErrInvalidUTF8Text {
			t.Fatalf("want ErrInvalidUTF8Text, got %v", err)
		}
	})

	t.Run("data_too_long", func(t *testing.T) {
		// Beyond version-40 capacity at the highest ECC level.
		huge := strings.Repeat("A", 5000)
		_, err := NewTextBuilder(huge).
			SetErrorCorrectionLevel(CorrectionLevelHigh).
			Build()
		if err != spec.ErrDataTooLong {
			t.Fatalf("want ErrDataTooLong, got %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// ECI policy: disabling ECI for non-ASCII text still produces a code that
// decodes to the same raw bytes (interpretation may differ, bytes must not).
// ---------------------------------------------------------------------------

func TestE2E_ECIDisabledBytesPreserved(t *testing.T) {
	// All-byte-mode payload (accented latin, no kanji/numeric runs) so the
	// whole thing lands in a single byte segment we can compare exactly.
	const payload = "Mañana café ñoño €5"
	want := []byte(payload)

	path := buildPNG(t, NewTextBuilder(payload).
		SetTextECIPolicy(TextECIPolicyDisabled))

	// With ECI disabled the decoder may not know the charset, but the raw
	// byte segment must still equal the original UTF-8 bytes.
	got := decodeBytes(t, path)
	if !bytes.Equal(got, want) {
		t.Fatalf("ECI-disabled byte mismatch:\n want %x\n got  %x", want, got)
	}
}
