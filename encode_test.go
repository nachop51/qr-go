package qr

import "testing"

// bchRemainder reduces value modulo the GF(2) generator polynomial g, whose
// highest set bit is at position deg.
func bchRemainder(value uint32, g uint32, deg int) uint32 {
	for i := 31; i >= deg; i-- {
		if (value>>uint(i))&1 == 1 {
			value ^= g << uint(i-deg)
		}
	}
	return value
}

func TestEncodeFormat(t *testing.T) {
	// Data 0 has remainder 0, so the code is the bare XOR mask.
	if got := encodeFormat(0); got != 0x5412 {
		t.Errorf("encodeFormat(0) = %#x, want 0x5412", got)
	}
	for d := uint16(0); d < 32; d++ {
		code := encodeFormat(d)
		unmasked := uint32(code) ^ 0x5412
		if unmasked>>10 != uint32(d) {
			t.Errorf("encodeFormat(%#b): data bits = %#b, want %#b", d, unmasked>>10, d)
		}
		// A valid BCH(15,5) codeword is divisible by the generator.
		if rem := bchRemainder(unmasked, 0x537, 10); rem != 0 {
			t.Errorf("encodeFormat(%#b) = %#x: BCH remainder %#x, want 0", d, code, rem)
		}
	}
}

func TestEncodeVersion(t *testing.T) {
	// Known answer from ISO/IEC 18004 (also cross-checked by the metadata
	// placement test).
	if got := encodeVersion(7); got != 0x07C94 {
		t.Errorf("encodeVersion(7) = %#x, want 0x07c94", got)
	}
	for v := uint16(7); v <= 40; v++ {
		code := encodeVersion(v)
		if code>>12 != uint32(v) {
			t.Errorf("encodeVersion(%d): data bits = %d, want %d", v, code>>12, v)
		}
		if rem := bchRemainder(code, 0x1F25, 12); rem != 0 {
			t.Errorf("encodeVersion(%d) = %#x: BCH remainder %#x, want 0", v, code, rem)
		}
	}
}
