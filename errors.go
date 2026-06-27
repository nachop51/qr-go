package qr

import "errors"

var (
	ErrInvalidDimensions = errors.New("invalid dimensions")
	ErrInvalidDataKind   = errors.New("invalid data kind")
	ErrInvalidUTF8Text   = errors.New("invalid utf8 text")
	ErrInvalidMask       = errors.New("invalid mask: a number between 0 and 7 must be provided")
	ErrDataTooLong       = errors.New("Data too long for the selected version and error correction level")
)
