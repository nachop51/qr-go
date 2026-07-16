package qr

import (
	"errors"
	"strings"
	"testing"

	"github.com/nachop51/qr-go/internal/matrix"
	"github.com/nachop51/qr-go/internal/spec"
	"github.com/nachop51/qr-go/render/terminal"
	reference "rsc.io/qr/coding"
)

func TestVersionMetadataExactPlacement(t *testing.T) {
	want := map[int]uint32{7: 0x07c94, 16: 0x10b78, 32: 0x209d5, 40: 0x28c69}
	for version, bits := range want {
		m := createQrBase(version)
		placeMetadata(m, version, 0, CorrectionLevelMedium)
		for _, pos := range spec.VersionModules(version) {
			got := m.Get(pos.X, pos.Y) == matrix.Black
			expected := (bits>>uint(pos.Bit))&1 != 0
			if got != expected {
				t.Fatalf("version %d module (%d,%d), bit %d = %v, want %v", version, pos.X, pos.Y, pos.Bit, got, expected)
			}
		}
	}
}

// TestConformanceEveryVersionLevelAndMask compares the complete emitted
// matrix against an independent QR implementation for all 1,280 public
// version/error-correction/mask combinations. A binary payload deliberately
// fixes both implementations to byte mode without ECI or segmentation choices.
func TestConformanceEveryVersionLevelAndMask(t *testing.T) {
	payload := []byte("x")
	levels := []struct {
		ours CorrectionLevel
		ref  reference.Level
	}{
		{CorrectionLevelLow, reference.L},
		{CorrectionLevelMedium, reference.M},
		{CorrectionLevelQuartile, reference.Q},
		{CorrectionLevelHigh, reference.H},
	}

	for version := 1; version <= 40; version++ {
		for _, level := range levels {
			for mask := 0; mask < 8; mask++ {
				ours, err := NewBinaryBuilder(payload).
					SetVersion(version).
					SetMask(mask).
					SetErrorCorrectionLevel(level.ours).
					Build()
				if err != nil {
					t.Fatalf("build v%d/%v/m%d: %v", version, level.ours, mask, err)
				}

				plan, err := reference.NewPlan(reference.Version(version), level.ref, reference.Mask(mask))
				if err != nil {
					t.Fatalf("reference plan v%d/%v/m%d: %v", version, level.ours, mask, err)
				}
				want, err := plan.Encode(reference.String(string(payload)))
				if err != nil {
					t.Fatalf("reference encode v%d/%v/m%d: %v", version, level.ours, mask, err)
				}
				if ours.Size() != want.Size {
					t.Fatalf("v%d/%v/m%d size=%d, want %d", version, level.ours, mask, ours.Size(), want.Size)
				}
				for y := 0; y < ours.Size(); y++ {
					for x := 0; x < ours.Size(); x++ {
						if got, expected := ours.IsDark(x, y), want.Black(x, y); got != expected {
							t.Fatalf("v%d/%v/m%d module (%d,%d)=%v, want %v", version, level.ours, mask, x, y, got, expected)
						}
					}
				}
			}
		}
	}
}

// TestConformanceNumericAndAlphanumeric compares numeric and alphanumeric
// bit-packing against the reference implementation. The payloads are chosen so
// the segmenter keeps a single segment of the intended mode, making the
// matrices directly comparable. Versions cover both sides of the 9/10 and
// 26/27 boundaries, where the character-count indicator widens.
func TestConformanceNumericAndAlphanumeric(t *testing.T) {
	modes := []struct {
		mode    string
		payload string
		ref     func(string) (reference.Encoding, error)
	}{
		{"Numeric", "0123456789", func(s string) (reference.Encoding, error) { return reference.Num(s), nil }},
		{"Alphanumeric", "A $%*+-./:", func(s string) (reference.Encoding, error) { return reference.Alpha(s), nil }},
	}
	levels := []struct {
		ours CorrectionLevel
		ref  reference.Level
	}{
		{CorrectionLevelLow, reference.L},
		{CorrectionLevelMedium, reference.M},
		{CorrectionLevelQuartile, reference.Q},
		{CorrectionLevelHigh, reference.H},
	}

	for _, mc := range modes {
		for _, version := range []int{1, 9, 10, 26, 27, 40} {
			for _, level := range levels {
				for mask := 0; mask < 8; mask++ {
					ours, err := NewTextBuilder(mc.payload).
						SetVersion(version).
						SetMask(mask).
						SetErrorCorrectionLevel(level.ours).
						Build()
					if err != nil {
						t.Fatalf("%s build v%d/%v/m%d: %v", mc.mode, version, level.ours, mask, err)
					}
					if segs := ours.Segments(); len(segs) != 1 || segs[0].Mode() != mc.mode {
						t.Fatalf("%s v%d: segmented as %+v, want single %s segment", mc.mode, version, segs, mc.mode)
					}

					plan, err := reference.NewPlan(reference.Version(version), level.ref, reference.Mask(mask))
					if err != nil {
						t.Fatalf("reference plan v%d/%v/m%d: %v", version, level.ours, mask, err)
					}
					enc, err := mc.ref(mc.payload)
					if err != nil {
						t.Fatalf("reference encoding %s: %v", mc.mode, err)
					}
					want, err := plan.Encode(enc)
					if err != nil {
						t.Fatalf("reference encode %s v%d/%v/m%d: %v", mc.mode, version, level.ours, mask, err)
					}
					for y := 0; y < ours.Size(); y++ {
						for x := 0; x < ours.Size(); x++ {
							if got, expected := ours.IsDark(x, y), want.Black(x, y); got != expected {
								t.Fatalf("%s v%d/%v/m%d module (%d,%d)=%v, want %v", mc.mode, version, level.ours, mask, x, y, got, expected)
							}
						}
					}
				}
			}
		}
	}
}

func TestMaximumCapacityBoundaries(t *testing.T) {
	cases := []struct {
		name, unit string
		max        int
	}{
		{"numeric", "1", 7089},
		{"alphanumeric", "A", 4296},
		{"byte", "a", 2953},
		{"kanji", "漢", 1817},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, err := NewTextBuilder(strings.Repeat(tc.unit, tc.max)).SetErrorCorrectionLevel(CorrectionLevelLow).Build()
			if err != nil {
				t.Fatalf("maximum payload rejected: %v", err)
			}
			if code.Version() != 40 {
				t.Fatalf("version = %d, want 40", code.Version())
			}
			_, err = NewTextBuilder(strings.Repeat(tc.unit, tc.max+1)).SetErrorCorrectionLevel(CorrectionLevelLow).Build()
			if !errors.Is(err, ErrDataTooLong) {
				t.Fatalf("over-capacity error = %v", err)
			}
		})
	}
	if _, err := NewTextBuilder(strings.Repeat("a", 100)).SetVersion(1).Build(); !errors.Is(err, ErrVersionTooSmall) {
		t.Fatalf("undersized forced version error = %v", err)
	}
}

func TestBuildRejectsInvalidPublicValues(t *testing.T) {
	cases := []struct {
		name string
		b    *Builder
	}{
		{"zero correction level", NewTextBuilder("x").SetErrorCorrectionLevel(0)},
		{"unknown correction level", NewTextBuilder("x").SetErrorCorrectionLevel(255)},
		{"unknown ECI policy", NewTextBuilder("x").SetTextECIPolicy(99)},
		{"zero forced version", NewTextBuilder("x").SetVersion(0)},
		{"high version", NewTextBuilder("x").SetVersion(41)},
		{"low mask", NewTextBuilder("x").SetMask(-2)},
		{"high mask", NewTextBuilder("x").SetMask(8)},
		{"nil renderer", NewTextBuilder("x").SetRenderer(nil)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.b.Build(); !errors.Is(err, ErrInvalidOptions) {
				t.Fatalf("Build error = %v, want ErrInvalidOptions", err)
			}
		})
	}
	if _, err := NewTextBuilder(string(make([]byte, 7090))).SetRenderer(terminal.New()).Build(); !errors.Is(err, ErrDataTooLong) {
		t.Fatalf("oversized text error = %v, want ErrDataTooLong", err)
	}
}

func TestCodeAccessorsReturnCopiesAndBoundsAreLight(t *testing.T) {
	code, err := NewTextBuilder("hello").Build()
	if err != nil {
		t.Fatal(err)
	}
	segments := code.Segments()
	data := segments[0].Bytes()
	data[0] ^= 0xff
	if code.Segments()[0].Data() != "hello" {
		t.Fatal("segment data mutated through accessor")
	}
	for _, p := range [][2]int{{-1, 0}, {0, -1}, {code.Size(), 0}, {0, code.Size()}} {
		if code.IsDark(p[0], p[1]) {
			t.Fatalf("out-of-bounds module %v reported dark", p)
		}
	}
}
