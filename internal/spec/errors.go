package spec

import "errors"

var (
	ErrInvalidDataKind = errors.New("invalid data kind")
	ErrInvalidUTF8Text = errors.New("invalid utf8 text")
	ErrDataTooLong     = errors.New("data too long for the selected version and error correction level")
	ErrInvalidVersion  = errors.New("invalid version")
	ErrInvalidMask     = errors.New("invalid mask")
	ErrVersionTooSmall = errors.New("version too small for the data and error correction level")
)
