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

type CorrectionLevel struct {
	level int
	value int
}

var (
	CorrectionLevelLow      = CorrectionLevel{level: 0, value: 0b01}
	CorrectionLevelMedium   = CorrectionLevel{level: 1, value: 0b00}
	CorrectionLevelQuartile = CorrectionLevel{level: 2, value: 0b11}
	CorrectionLevelHigh     = CorrectionLevel{level: 3, value: 0b10}
)

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
		return "Numeric"
	}
}
