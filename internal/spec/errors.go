package spec

import "errors"

var (
	ErrInvalidDataKind = errors.New("invalid data kind")
	ErrInvalidUTF8Text = errors.New("invalid utf8 text")
	ErrDataTooLong     = errors.New("Data too long for the selected version and error correction level")
)
