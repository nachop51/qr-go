package qr

import (
	"github.com/nachop51/qr-go/internal/spec"
)

type DataKind int

const (
	DataKindText DataKind = iota
	DataKindBinary
)

type TextECIPolicy int

const (
	TextECIPolicyAuto TextECIPolicy = iota
	TextECIPolicyDisabled
)

// CorrectionLevel controls the amount of error correction in a code. Its
// numeric representation is intentionally opaque; use the named constants.
type CorrectionLevel uint8

const (
	CorrectionLevelLow CorrectionLevel = iota + 1
	CorrectionLevelMedium
	CorrectionLevelQuartile
	CorrectionLevelHigh
)

func (c CorrectionLevel) valid() bool {
	return c >= CorrectionLevelLow && c <= CorrectionLevelHigh
}

// tableIndex maps the public value to the ordering used by the QR capacity
// tables (L, M, Q, H).
func (c CorrectionLevel) tableIndex() int {
	switch c {
	case CorrectionLevelLow:
		return 0
	case CorrectionLevelMedium:
		return 1
	case CorrectionLevelQuartile:
		return 2
	case CorrectionLevelHigh:
		return 3
	default:
		return -1
	}
}

// formatBits maps the level to the ISO/IEC 18004 format-information bits.
func (c CorrectionLevel) formatBits() int {
	switch c {
	case CorrectionLevelLow:
		return 0b01
	case CorrectionLevelMedium:
		return 0b00
	case CorrectionLevelQuartile:
		return 0b11
	case CorrectionLevelHigh:
		return 0b10
	default:
		return -1
	}
}

type EncodingMode = spec.EncodingMode

const (
	EncodingModeNumeric      = spec.EncodingModeNumeric
	EncodingModeAlphanumeric = spec.EncodingModeAlphanumeric
	EncodingModeByte         = spec.EncodingModeByte
	EncodingModeKanji        = spec.EncodingModeKanji
)

type Segment struct {
	mode EncodingMode
	data []byte
}

func (s Segment) Data() string {
	return string(s.data)
}

// Bytes returns a copy of the segment's encoded source bytes.
func (s Segment) Bytes() []byte { return append([]byte(nil), s.data...) }

func (s Segment) Mode() string {
	switch s.mode {
	case EncodingModeNumeric:
		return "Numeric"
	case EncodingModeAlphanumeric:
		return "Alphanumeric"
	case EncodingModeByte:
		return "Byte"
	case EncodingModeKanji:
		return "Kanji"
	default:
		return "Unknown"
	}
}
