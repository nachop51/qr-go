package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Negative coordinates look like shorthand flags to pflag; geo's own argument
// pass must keep them as positionals (and as flag values) while still parsing
// the real flags.
func TestGeoNegativeCoordinates(t *testing.T) {
	cases := [][]string{
		{"geo", "-20.05", "57.52", "-i", "-f", "svg"},
		{"geo", "--lat", "-20.05", "--lng", "57.52", "-i", "-f", "svg"},
		{"geo", "-i", "-f", "svg", "--", "-20.05", "57.52"},
	}
	for _, args := range cases {
		cmd := newRootCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetIn(strings.NewReader(""))
		cmd.SetArgs(args)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if !strings.Contains(out.String(), `data="geo:-20.05,57.52"`) {
			t.Errorf("%v: payload info not found in output:\n%s", args, out.String())
		}
		if !strings.Contains(out.String(), "<svg") {
			t.Errorf("%v: -f svg not honored", args)
		}
	}
}

// A logo raises the default error correction to H; an explicit --ecc wins.
func TestLogoDefaultsECCHigh(t *testing.T) {
	dir := t.TempDir()

	logoPath := filepath.Join(dir, "logo.png")
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}
	img.Set(0, 0, color.Black)
	f, err := os.Create(logoPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()

	cases := []struct {
		args []string
		ecc  string
	}{
		{[]string{"hello", "--logo", logoPath}, "ecc=H"},
		{[]string{"hello", "--logo", logoPath, "--ecc", "L"}, "ecc=L"},
		{[]string{"hello"}, "ecc=M"},
	}
	for _, c := range cases {
		cmd := newRootCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetIn(strings.NewReader(""))
		cmd.SetArgs(append(c.args, "-i", "-o", filepath.Join(dir, "out.svg")))

		if err := cmd.Execute(); err != nil {
			t.Fatalf("%v: %v", c.args, err)
		}
		if !strings.Contains(out.String(), c.ecc) {
			t.Errorf("%v: want %s in info output, got:\n%s", c.args, c.ecc, out.String())
		}
	}
}

// geo requires both coordinates: two positionals, or --lat and --lng together.
func TestGeoRejectsIncompleteCoordinates(t *testing.T) {
	cases := [][]string{
		{"geo"},
		{"geo", "hello"},
		{"geo", "hello", "world"}, // two positionals, but not numbers
		{"geo", "-20.05"},
		{"geo", "-20.05", "57.52", "extra"},
		{"geo", "--lat", "-20.05"},
		{"geo", "-20.05", "--lng", "57.52"},
	}
	for _, args := range cases {
		cmd := newRootCmd()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetIn(strings.NewReader(""))
		cmd.SetArgs(args)

		if err := cmd.Execute(); err == nil {
			t.Errorf("%v: expected an error, got none", args)
		}
	}
}
