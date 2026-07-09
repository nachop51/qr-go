package main

import (
	"bytes"
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
