package qr

import (
	"bytes"
	"errors"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/makiuchi-d/gozxing"
	zqrcode "github.com/makiuchi-d/gozxing/qrcode"
)

func buildAndDraw(t *testing.T, builder *QrBuilder) *QrObject {
	t.Helper()

	code, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	code.Draw()
	return code
}

func decodeResult(t *testing.T, img image.Image) *gozxing.Result {
	t.Helper()

	bitmap, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		t.Fatalf("NewBinaryBitmapFromImage() error = %v", err)
	}

	reader := zqrcode.NewQRCodeReader()
	result, err := reader.Decode(bitmap, nil)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	return result
}

func decodeImage(t *testing.T, img image.Image) string {
	t.Helper()
	return decodeResult(t, img).GetText()
}

func TestQRCodeRoundTripDecode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		builder func() *QrBuilder
		want    string
	}{
		{
			name: "numeric_low",
			builder: func() *QrBuilder {
				return NewTextQrBuilder("1289421489").
					SetErrorCorrectionLevel(QrCorrectionLevelLow)
			},
			want: "1289421489",
		},
		{
			name: "alphanumeric_medium",
			builder: func() *QrBuilder {
				return NewTextQrBuilder("HELLO WORLD").
					SetErrorCorrectionLevel(QrCorrectionLevelMedium)
			},
			want: "HELLO WORLD",
		},
		{
			name: "kanji_quartile",
			builder: func() *QrBuilder {
				return NewTextQrBuilder("日本").
					SetErrorCorrectionLevel(QrCorrectionLevelQuartile)
			},
			want: "日本",
		},
		{
			name: "utf8_byte_mode_with_eci",
			builder: func() *QrBuilder {
				return NewTextQrBuilder("café").
					SetErrorCorrectionLevel(QrCorrectionLevelHigh)
			},
			want: "café",
		},
		{
			name: "binary_ascii_payload",
			builder: func() *QrBuilder {
				return NewBinaryQrBuilder([]byte("binary payload 123")).
					SetErrorCorrectionLevel(QrCorrectionLevelHigh)
			},
			want: "binary payload 123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := buildAndDraw(t, tt.builder())
			got := decodeImage(t, code.img)
			if got != tt.want {
				t.Fatalf("decoded text = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDecoderSeesRequestedErrorCorrectionLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		level QrCorrectionLevel
		want  string
	}{
		{name: "low", level: QrCorrectionLevelLow, want: "L"},
		{name: "medium", level: QrCorrectionLevelMedium, want: "M"},
		{name: "quartile", level: QrCorrectionLevelQuartile, want: "Q"},
		{name: "high", level: QrCorrectionLevelHigh, want: "H"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := buildAndDraw(t, NewTextQrBuilder("EC level "+tt.name).SetErrorCorrectionLevel(tt.level))
			result := decodeResult(t, code.img)

			metadata := result.GetResultMetadata()
			got, ok := metadata[gozxing.ResultMetadataType_ERROR_CORRECTION_LEVEL].(string)
			if !ok {
				t.Fatalf("decoder metadata missing error correction level: %#v", metadata)
			}
			if got != tt.want {
				t.Fatalf("decoded error correction level = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBinaryPayloadExposesByteSegments(t *testing.T) {
	t.Parallel()

	payload := []byte{0x00, 0x01, 0x02, 0x7F, 0x80, 0xFE, 0xFF}
	code := buildAndDraw(t, NewBinaryQrBuilder(payload).SetErrorCorrectionLevel(QrCorrectionLevelHigh))
	result := decodeResult(t, code.img)

	metadata := result.GetResultMetadata()
	segmentsValue, ok := metadata[gozxing.ResultMetadataType_BYTE_SEGMENTS]
	if !ok {
		t.Fatalf("decoder metadata missing byte segments: %#v", metadata)
	}

	segments, ok := segmentsValue.([][]byte)
	if !ok {
		t.Fatalf("byte segment metadata has unexpected type %T", segmentsValue)
	}
	if len(segments) != 1 {
		t.Fatalf("decoded byte segments = %d, want 1", len(segments))
	}
	if !bytes.Equal(segments[0], payload) {
		t.Fatalf("decoded byte segment = %v, want %v", segments[0], payload)
	}
}

func TestBuilderDefaults(t *testing.T) {
	t.Parallel()

	code, err := NewTextQrBuilder("defaults").Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if code.Filename != "image.png" {
		t.Fatalf("Filename = %q, want %q", code.Filename, "image.png")
	}
	if code.ErrorCorrectionLevel != QrCorrectionLevelMedium {
		t.Fatalf("ErrorCorrectionLevel = %#v, want %#v", code.ErrorCorrectionLevel, QrCorrectionLevelMedium)
	}
	if code.img.Bounds().Dx() != 400 || code.img.Bounds().Dy() != 400 {
		t.Fatalf("image bounds = %v, want 400x400", code.img.Bounds())
	}
	if code.Version <= 0 {
		t.Fatalf("Version = %d, want > 0", code.Version)
	}
	if code.Mask < 0 || code.Mask > 7 {
		t.Fatalf("Mask = %d, want between 0 and 7", code.Mask)
	}
	if len(code.Segments) == 0 {
		t.Fatal("Segments is empty, want at least one segment")
	}
}

func TestNonSquareCanvasStillDecodes(t *testing.T) {
	t.Parallel()

	want := "wide canvas"
	code := buildAndDraw(t, NewTextQrBuilder(want).SetWidth(640).SetHeight(320))

	if code.img.Bounds().Dx() != 640 || code.img.Bounds().Dy() != 320 {
		t.Fatalf("image bounds = %v, want 640x320", code.img.Bounds())
	}

	got := decodeImage(t, code.img)
	if got != want {
		t.Fatalf("decoded text = %q, want %q", got, want)
	}
}

func TestBuildErrors(t *testing.T) {
	t.Parallel()

	t.Run("invalid_dimensions", func(t *testing.T) {
		_, err := NewTextQrBuilder("hello").SetWidth(0).Build()
		if !errors.Is(err, ErrInvalidDimensions) {
			t.Fatalf("Build() error = %v, want %v", err, ErrInvalidDimensions)
		}
	})

	t.Run("invalid_utf8_text", func(t *testing.T) {
		_, err := NewTextQrBuilder(string([]byte{0xFF, 0xFE})).Build()
		if !errors.Is(err, ErrInvalidUTF8Text) {
			t.Fatalf("Build() error = %v, want %v", err, ErrInvalidUTF8Text)
		}
	})

	t.Run("data_too_long", func(t *testing.T) {
		_, err := NewTextQrBuilder(strings.Repeat("A", 5000)).Build()
		if !errors.Is(err, ErrDataTooLong) {
			t.Fatalf("Build() error = %v, want %v", err, ErrDataTooLong)
		}
	})
}

func TestSaveWritesDecodablePNG(t *testing.T) {
	t.Parallel()

	filename := filepath.Join(t.TempDir(), "saved.png")
	want := "save smoke test"

	code, err := NewTextQrBuilder(want).
		SetFilename(filename).
		SetErrorCorrectionLevel(QrCorrectionLevelMedium).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	code.Draw()
	if err := code.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	file, err := os.Open(filename)
	if err != nil {
		t.Fatalf("os.Open() error = %v", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		t.Fatalf("image.Decode() error = %v", err)
	}

	got := decodeImage(t, img)
	if got != want {
		t.Fatalf("decoded text = %q, want %q", got, want)
	}
}
