package qr

import (
	"nachop51/qr/internal/spec"
)

type QrDataKind int

const (
	QrDataKindText QrDataKind = iota
	QrDataKindBinary
)

type QrTextECIPolicy int

const (
	QrTextECIPolicyAuto QrTextECIPolicy = iota
	QrTextECIPolicyDisabled
)

type QrCorrectionLevel struct {
	level int
	value int
}

var (
	QrCorrectionLevelLow      = QrCorrectionLevel{level: 0, value: 0b01}
	QrCorrectionLevelMedium   = QrCorrectionLevel{level: 1, value: 0b00}
	QrCorrectionLevelQuartile = QrCorrectionLevel{level: 2, value: 0b11}
	QrCorrectionLevelHigh     = QrCorrectionLevel{level: 3, value: 0b10}
)

type QrColor int

const (
	QrWhite QrColor = iota
	QrBlack
)

type QrPoint struct {
	x         int
	y         int
	col       QrColor
	protected bool
	drawn     bool
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
