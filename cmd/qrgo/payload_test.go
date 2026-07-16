package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/nachop51/qr-go/content"
)

func TestBuildPayload(t *testing.T) {
	cases := []struct {
		name       string
		typ        string
		opts       options
		args       []string
		stdin      []byte
		wantText   string
		wantBinary []byte
		wantErr    bool
	}{
		{
			name:     "text positional joined",
			typ:      "text",
			args:     []string{"HELLO", "WORLD"},
			wantText: "HELLO WORLD",
		},
		{
			name:     "text empty type defaults to text",
			typ:      "",
			args:     []string{"hi"},
			wantText: "hi",
		},
		{
			name:     "text from stdin trims newline",
			typ:      "text",
			stdin:    []byte("piped value\n"),
			wantText: "piped value",
		},
		{
			name:     "url positional",
			typ:      "url",
			args:     []string{"https://example.com"},
			wantText: content.URL("https://example.com"),
		},
		{
			name:     "tel positional",
			typ:      "tel",
			args:     []string{"+15551234567"},
			wantText: content.Tel("+15551234567"),
		},
		{
			name:     "sms number from stdin",
			typ:      "sms",
			opts:     options{message: "later"},
			stdin:    []byte("+15551234567\n"),
			wantText: content.SMS("+15551234567", "later"),
		},
		{
			name:     "wifi ssid from stdin",
			typ:      "wifi",
			stdin:    []byte("CoffeeShop\n"),
			wantText: content.WiFi{SSID: "CoffeeShop"}.String(),
		},
		{
			name:     "vcard full name from stdin",
			typ:      "vcard",
			stdin:    []byte("Jane Doe\n"),
			wantText: content.VCard{FullName: "Jane Doe"}.String(),
		},
		{
			name:     "event summary from stdin",
			typ:      "event",
			stdin:    []byte("Launch\n"),
			wantText: content.Event{Summary: "Launch"}.String(),
		},
		{
			name:     "geo coordinates from stdin",
			typ:      "geo",
			stdin:    []byte("48.8584, 2.2945\n"),
			wantText: content.Geo(48.8584, 2.2945),
		},
		{
			name:    "geo stdin out of range",
			typ:     "geo",
			stdin:   []byte("999 2.2945\n"),
			wantErr: true,
		},
		{
			name:     "sms number and positional message",
			typ:      "sms",
			args:     []string{"+15551234567", "call", "me"},
			wantText: content.SMS("+15551234567", "call me"),
		},
		{
			name:     "sms message flag",
			typ:      "sms",
			opts:     options{message: "later"},
			args:     []string{"+15551234567"},
			wantText: content.SMS("+15551234567", "later"),
		},
		{
			name:     "email with subject and body",
			typ:      "email",
			opts:     options{subject: "Hi", body: "Hello there"},
			args:     []string{"a@b.test"},
			wantText: content.Email("a@b.test", "Hi", "Hello there"),
		},
		{
			name:     "geo positional",
			typ:      "geo",
			args:     []string{"48.8584", "2.2945"},
			wantText: content.Geo(48.8584, 2.2945),
		},
		{
			name:     "geo flags",
			typ:      "geo",
			opts:     options{lat: 40.7128, lng: -74.006},
			wantText: content.Geo(40.7128, -74.006),
		},
		{
			name:    "geo invalid latitude",
			typ:     "geo",
			args:    []string{"north", "2.0"},
			wantErr: true,
		},
		{
			name:    "geo latitude out of range",
			typ:     "geo",
			args:    []string{"90.1", "2.0"},
			wantErr: true,
		},
		{
			name:    "geo longitude out of range",
			typ:     "geo",
			args:    []string{"48.85", "-180.5"},
			wantErr: true,
		},
		{
			name:    "geo NaN rejected",
			typ:     "geo",
			args:    []string{"NaN", "2.0"},
			wantErr: true,
		},
		{
			name:     "wifi flags",
			typ:      "wifi",
			opts:     options{ssid: "home", pass: "s3cr3t"},
			wantText: content.WiFi{SSID: "home", Pass: "s3cr3t"}.String(),
		},
		{
			name:     "wifi ssid positional",
			typ:      "wifi",
			opts:     options{pass: "s3cr3t"},
			args:     []string{"home"},
			wantText: content.WiFi{SSID: "home", Pass: "s3cr3t"}.String(),
		},
		{
			name:     "wifi auth wep",
			typ:      "wifi",
			opts:     options{ssid: "legacy", pass: "abc", auth: "wep"},
			wantText: content.WiFi{SSID: "legacy", Pass: "abc", Auth: content.WiFiWEP}.String(),
		},
		{
			name:    "wifi missing ssid",
			typ:     "wifi",
			opts:    options{pass: "x"},
			wantErr: true,
		},
		{
			name:     "vcard flags",
			typ:      "vcard",
			opts:     options{first: "Jane", last: "Doe", email: "jane@acme.test"},
			wantText: content.VCard{First: "Jane", Last: "Doe", Email: "jane@acme.test"}.String(),
		},
		{
			name:     "vcard positional full name",
			typ:      "vcard",
			args:     []string{"Jane Doe"},
			wantText: content.VCard{FullName: "Jane Doe"}.String(),
		},
		{
			name:    "vcard empty",
			typ:     "vcard",
			wantErr: true,
		},
		{
			name:     "event summary flag",
			typ:      "event",
			opts:     options{summary: "Launch", location: "HQ"},
			wantText: content.Event{Summary: "Launch", Location: "HQ"}.String(),
		},
		{
			name:    "event bad time",
			typ:     "event",
			opts:    options{summary: "x", start: "not-a-date"},
			wantErr: true,
		},
		{
			name:       "binary from args",
			typ:        "binary",
			args:       []string{"raw bytes"},
			wantBinary: []byte("raw bytes"),
		},
		{
			name:       "binary from stdin",
			typ:        "binary",
			opts:       options{input: "-"},
			stdin:      []byte{0x00, 0x01, 0xff},
			wantBinary: []byte{0x00, 0x01, 0xff},
		},
		{
			name:    "unknown type",
			typ:     "bogus",
			args:    []string{"x"},
			wantErr: true,
		},
		{
			name:    "text no content",
			typ:     "text",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := tc.opts
			text, data, binary, err := buildPayload(&opts, tc.typ, tc.args, tc.stdin)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got text=%q data=%q", text, data)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantBinary != nil {
				if !binary {
					t.Fatalf("want binary output, got text %q", text)
				}
				if !bytes.Equal(data, tc.wantBinary) {
					t.Fatalf("binary mismatch: want %x got %x", tc.wantBinary, data)
				}
				return
			}
			if binary {
				t.Fatalf("want text output, got binary %x", data)
			}
			if text != tc.wantText {
				t.Fatalf("text mismatch:\n want %q\n got  %q", tc.wantText, text)
			}
		})
	}
}

func TestBuildPayloadBinaryFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.bin")
	want := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	opts := options{input: path}
	_, data, binary, err := buildPayload(&opts, "binary", nil, nil)
	if err != nil {
		t.Fatalf("buildPayload: %v", err)
	}
	if !binary || !bytes.Equal(data, want) {
		t.Fatalf("binary file mismatch: want %x got %x (binary=%v)", want, data, binary)
	}
}

func TestParseEventTime(t *testing.T) {
	cases := []struct {
		in           string
		wantErr      bool
		wantDateOnly bool
	}{
		{in: "2026-07-05T14:30:00Z"},
		{in: "2026-07-05 14:30"},
		{in: "2026-07-05T14:30"},
		{in: "2026-07-05", wantDateOnly: true},
		{in: "nonsense", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			_, dateOnly, err := parseEventTime(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error for %q", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dateOnly != tc.wantDateOnly {
				t.Fatalf("dateOnly: want %v got %v", tc.wantDateOnly, dateOnly)
			}
		})
	}
}
