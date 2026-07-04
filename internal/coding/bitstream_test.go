package coding

import (
	"bytes"
	"testing"
)

func TestBitWriterPacksMSBFirst(t *testing.T) {
	w := &BitWriter{}
	w.AppendBits(0b101, 3) // 1 0 1
	w.AppendBits(0b11, 2)  // 1 1  -> bits so far: 1 0 1 1 1

	if w.BitLen() != 5 {
		t.Fatalf("BitLen = %d, want 5", w.BitLen())
	}
	// 10111000 = 0xB8
	if got := w.Data(); !bytes.Equal(got, []byte{0xB8}) {
		t.Fatalf("Data = %v, want [0xB8]", got)
	}
}

func TestBitWriterByteBoundary(t *testing.T) {
	w := &BitWriter{}
	w.AppendBits(0xFF, 8) // full byte
	w.AppendBits(0x01, 1) // spills into a second byte, MSB set

	if w.BitLen() != 9 {
		t.Fatalf("BitLen = %d, want 9", w.BitLen())
	}
	if got := w.Data(); !bytes.Equal(got, []byte{0xFF, 0x80}) {
		t.Fatalf("Data = %v, want [0xFF 0x80]", got)
	}
}

func TestBitWriterReaderRoundTrip(t *testing.T) {
	bits := []int{1, 0, 1, 1, 0, 0, 1, 0, 1, 1, 1}

	w := &BitWriter{}
	for _, b := range bits {
		w.AppendBits(b, 1)
	}

	r := NewBitReader(w.Data())
	for i, want := range bits {
		if got := r.Next(); got != want {
			t.Fatalf("bit %d: got %d, want %d", i, got, want)
		}
	}
	if r.Pos() != len(bits) {
		t.Fatalf("Pos = %d, want %d", r.Pos(), len(bits))
	}
}

// Past the end, Next must return 0 without advancing the position.
func TestBitReaderPastEnd(t *testing.T) {
	r := NewBitReader([]byte{0xFF}) // 8 readable bits
	for i := 0; i < 8; i++ {
		if got := r.Next(); got != 1 {
			t.Fatalf("bit %d: got %d, want 1", i, got)
		}
	}
	if r.Pos() != 8 {
		t.Fatalf("Pos = %d, want 8 after consuming all bits", r.Pos())
	}
	if got := r.Next(); got != 0 {
		t.Fatalf("past-end Next = %d, want 0", got)
	}
	if r.Pos() != 8 {
		t.Fatalf("Pos advanced past end: %d, want 8", r.Pos())
	}
}
