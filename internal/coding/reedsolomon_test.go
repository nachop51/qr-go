package coding

import (
	"bytes"
	"testing"
)

// TestReedSolomonKnownAnswer checks the encoder against the canonical
// "HELLO WORLD" version 1-M example (16 data codewords, 10 EC codewords)
// published in the Thonky QR tutorial and widely used as a reference vector.
func TestReedSolomonKnownAnswer(t *testing.T) {
	data := []byte{32, 91, 11, 120, 209, 114, 220, 77, 67, 64, 236, 17, 236, 17, 236, 17}
	want := []byte{196, 35, 39, 119, 235, 215, 231, 226, 93, 23}

	enc := NewRSEncoder(len(want))
	got := ReedSolomon(data, len(want), enc)

	if !bytes.Equal(got, want) {
		t.Fatalf("ReedSolomon mismatch:\n got  %v\n want %v", got, want)
	}
}

func TestReedSolomonLength(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	for _, ecCount := range []int{7, 10, 13, 17, 30} {
		enc := NewRSEncoder(ecCount)
		got := ReedSolomon(data, ecCount, enc)
		if len(got) != ecCount {
			t.Errorf("ecCount=%d: got %d EC codewords, want %d", ecCount, len(got), ecCount)
		}
	}
}

// TestReedSolomonDeterministic guards against hidden state in the shared
// encoder: encoding the same data twice must yield identical EC codewords.
func TestReedSolomonDeterministic(t *testing.T) {
	data := []byte{10, 20, 30, 40, 50, 60}
	enc := NewRSEncoder(8)
	a := ReedSolomon(data, 8, enc)
	b := ReedSolomon(data, 8, enc)
	if !bytes.Equal(a, b) {
		t.Fatalf("non-deterministic EC output: %v vs %v", a, b)
	}
}

// TestInterleave checks the column-major interleaving of data blocks followed
// by column-major interleaving of the (equal-length) EC blocks.
func TestInterleave(t *testing.T) {
	dataBlocks := [][]byte{{1, 2, 3}, {4, 5}}
	ecBlocks := [][]byte{{10, 11}, {12, 13}}

	// data: col0 -> 1,4 ; col1 -> 2,5 ; col2 -> 3 (block 2 exhausted)
	// ec:   col0 -> 10,12 ; col1 -> 11,13
	want := []byte{1, 4, 2, 5, 3, 10, 12, 11, 13}

	got := Interleave(dataBlocks, ecBlocks)
	if !bytes.Equal(got, want) {
		t.Fatalf("Interleave mismatch:\n got  %v\n want %v", got, want)
	}
}

func TestInterleaveEqualLengthBlocks(t *testing.T) {
	dataBlocks := [][]byte{{1, 2}, {3, 4}}
	ecBlocks := [][]byte{{9}, {8}}
	want := []byte{1, 3, 2, 4, 9, 8}

	got := Interleave(dataBlocks, ecBlocks)
	if !bytes.Equal(got, want) {
		t.Fatalf("Interleave mismatch:\n got  %v\n want %v", got, want)
	}
}
