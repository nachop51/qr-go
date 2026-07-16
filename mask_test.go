package qr

import (
	"testing"

	"github.com/nachop51/qr-go/internal/matrix"
)

// parsePattern builds a matrix from rows of 'X' (dark) and '.' (light).
func parsePattern(t *testing.T, rows []string) *matrix.Matrix {
	t.Helper()
	m := matrix.New(len(rows))
	for y, row := range rows {
		if len(row) != len(rows) {
			t.Fatalf("row %d has %d cells, want %d", y, len(row), len(rows))
		}
		for x, c := range row {
			if c == 'X' {
				m.Set(x, y, matrix.Black)
			}
		}
	}
	return m
}

func fillPattern(size int, c byte) []string {
	rows := make([]string, size)
	for i := range rows {
		row := make([]byte, size)
		for j := range row {
			row[j] = c
		}
		rows[i] = string(row)
	}
	return rows
}

func TestMaskPenalty1(t *testing.T) {
	cases := []struct {
		name string
		rows []string
		want int
	}{
		// No run reaches 5 modules.
		{"checkerboard", []string{
			"X.X.X",
			".X.X.",
			"X.X.X",
			".X.X.",
			"X.X.X",
		}, 0},
		// Every row and column is one 5-run: (3+0) * 10.
		{"all dark 5x5", fillPattern(5, 'X'), 30},
		{"all light 5x5", fillPattern(5, '.'), 30},
		// Runs of 4 never score.
		{"all dark 4x4", fillPattern(4, 'X'), 0},
		// A 6-run scores 3+1: rows 4*6, columns 6*4.
		{"all dark 6x6", fillPattern(6, 'X'), 48},
	}
	for _, tc := range cases {
		if got := maskPenalty1(parsePattern(t, tc.rows)); got != tc.want {
			t.Errorf("%s: N1 = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestMaskPenalty2(t *testing.T) {
	cases := []struct {
		name string
		rows []string
		want int
	}{
		{"checkerboard", []string{
			"X.X",
			".X.",
			"X.X",
		}, 0},
		// Four overlapping 2x2 blocks * 3.
		{"all dark 3x3", fillPattern(3, 'X'), 12},
		{"all light 2x2", fillPattern(2, '.'), 3},
		{"single block in corner", []string{
			"XX.",
			"XX.",
			"..X",
		}, 3},
	}
	for _, tc := range cases {
		if got := maskPenalty2(parsePattern(t, tc.rows)); got != tc.want {
			t.Errorf("%s: N2 = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestMaskPenalty3(t *testing.T) {
	cases := []struct {
		name string
		rows []string
		want int
	}{
		// Finder-like run followed by 4 light modules.
		{"pattern with light after", []string{
			"X.XXX.X....",
			"...........",
			"...........",
			"...........",
			"...........",
			"...........",
			"...........",
			"...........",
			"...........",
			"...........",
			"...........",
		}, 40},
		// Light on both sides still counts once, not twice.
		{"pattern flanked both sides", []string{
			"....X.XXX.X....",
			"...............",
			"...............",
			"...............",
			"...............",
			"...............",
			"...............",
			"...............",
			"...............",
			"...............",
			"...............",
			"...............",
			"...............",
			"...............",
			"...............",
		}, 40},
		// Dark modules adjacent on both sides: no light run, no penalty.
		{"pattern embedded in dark", []string{
			"XX.XXX.XXXX",
			"...........",
			"...........",
			"...........",
			"...........",
			"...........",
			"...........",
			"...........",
			"...........",
			"...........",
			"...........",
		}, 0},
		// The vertical orientation scores the same as the horizontal one.
		{"vertical pattern with light after", []string{
			"X..........",
			"...........",
			"X..........",
			"X..........",
			"X..........",
			"...........",
			"X..........",
			"...........",
			"...........",
			"...........",
			"...........",
		}, 40},
	}
	for _, tc := range cases {
		if got := maskPenalty3(parsePattern(t, tc.rows)); got != tc.want {
			t.Errorf("%s: N3 = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestMaskPenalty4(t *testing.T) {
	cases := []struct {
		name string
		rows []string
		want int
	}{
		// 100% dark: 50 points of deviation -> (50/5)*10.
		{"all dark", fillPattern(5, 'X'), 100},
		{"all light", fillPattern(5, '.'), 100},
		// 13/25 = 52%: brackets 50 and 55, min(0, 10) = 0.
		{"near half", []string{
			"X.X.X",
			".X.X.",
			"X.X.X",
			".X.X.",
			"X.X.X",
		}, 0},
		// 21/25 = 84%: brackets 80 and 85, min(60, 70) = 60.
		{"84 percent dark", []string{
			"XXXXX",
			"XXXXX",
			"XXXXX",
			"XXXXX",
			"X....",
		}, 60},
	}
	for _, tc := range cases {
		if got := maskPenalty4(parsePattern(t, tc.rows)); got != tc.want {
			t.Errorf("%s: N4 = %d, want %d", tc.name, got, tc.want)
		}
	}
}
