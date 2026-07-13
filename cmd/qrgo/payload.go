package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nachop51/qr-go/content"
)

// buildPayload turns the content type, positional args, and flags into the
// string (or raw bytes, for binary) to encode. When no value is given
// positionally or via flags it falls back to stdin (already read by the caller
// and passed as stdin, nil if unavailable).
func buildPayload(o *options, typ string, args []string, stdin []byte) (text string, data []byte, binary bool, err error) {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "", "text":
		s := strings.Join(args, " ")
		if s == "" {
			s = stdinString(stdin)
		}
		if s == "" {
			return "", nil, false, errNoContent("text")
		}
		return s, nil, false, nil

	case "url":
		s := firstArg(args)
		if s == "" {
			s = stdinString(stdin)
		}
		if s == "" {
			return "", nil, false, errNoContent("url")
		}
		return content.URL(s), nil, false, nil

	case "tel", "phone":
		s := firstArg(args)
		if s == "" {
			s = stdinString(stdin)
		}
		if s == "" {
			return "", nil, false, errNoContent("tel")
		}
		return content.Tel(s), nil, false, nil

	case "sms":
		number := firstArg(args)
		if number == "" {
			return "", nil, false, errNoContent("sms")
		}
		message := o.message
		if message == "" && len(args) >= 2 {
			message = strings.Join(args[1:], " ")
		}
		return content.SMS(number, message), nil, false, nil

	case "email", "mail":
		to := firstArg(args)
		if to == "" {
			to = stdinString(stdin)
		}
		if to == "" {
			return "", nil, false, errNoContent("email")
		}
		return content.Email(to, o.subject, o.body), nil, false, nil

	case "geo":
		lat, lng := o.lat, o.lng
		if len(args) >= 2 {
			if lat, err = strconv.ParseFloat(args[0], 64); err != nil {
				return "", nil, false, fmt.Errorf("geo: latitude %q is not a number", args[0])
			}
			if lng, err = strconv.ParseFloat(args[1], 64); err != nil {
				return "", nil, false, fmt.Errorf("geo: longitude %q is not a number", args[1])
			}
		}
		// The negated form also rejects NaN, which fails every comparison.
		if !(lat >= -90 && lat <= 90) {
			return "", nil, false, fmt.Errorf("geo: latitude %v is outside [-90, 90]", lat)
		}
		if !(lng >= -180 && lng <= 180) {
			return "", nil, false, fmt.Errorf("geo: longitude %v is outside [-180, 180]", lng)
		}
		return content.Geo(lat, lng), nil, false, nil

	case "wifi":
		ssid := o.ssid
		if ssid == "" {
			ssid = firstArg(args)
		}
		if ssid == "" {
			return "", nil, false, usageError{"wifi: an SSID is required (--ssid or a positional argument)"}
		}
		w := content.WiFi{SSID: ssid, Pass: o.pass, Hidden: o.hidden}
		if o.auth != "" {
			auth, err := parseWiFiAuth(o.auth)
			if err != nil {
				return "", nil, false, err
			}
			w.Auth = auth
		}
		return w.String(), nil, false, nil

	case "vcard", "contact":
		v := content.VCard{
			FullName: o.name,
			First:    o.first,
			Last:     o.last,
			Org:      o.org,
			Title:    o.title,
			Phone:    o.phone,
			Email:    o.email,
			URL:      o.url,
			Address:  o.address,
		}
		if v.FullName == "" && v.First == "" && v.Last == "" {
			v.FullName = firstArg(args)
		}
		if v == (content.VCard{}) {
			return "", nil, false, usageError{"vcard: provide at least a name or one field (--name, --email, --phone, ...)"}
		}
		return v.String(), nil, false, nil

	case "event":
		e := content.Event{
			Summary:     o.summary,
			Location:    o.location,
			Description: o.description,
		}
		var startDate, endDate bool
		if e.Summary == "" {
			e.Summary = firstArg(args)
		}
		if o.start != "" {
			t, dateOnly, err := parseEventTime(o.start)
			if err != nil {
				return "", nil, false, fmt.Errorf("event --start: %w", err)
			}
			e.Start = t
			startDate = dateOnly
		}
		if o.end != "" {
			t, dateOnly, err := parseEventTime(o.end)
			if err != nil {
				return "", nil, false, fmt.Errorf("event --end: %w", err)
			}
			e.End = t
			endDate = dateOnly
		}
		if !e.Start.IsZero() && !e.End.IsZero() && startDate != endDate {
			return "", nil, false, fmt.Errorf("event: start and end must both be dates or both be date-times")
		}
		e.AllDay = o.allDay || startDate || endDate
		if e.AllDay && (e.Start.IsZero() || e.End.IsZero() || !startDate || !endDate) {
			return "", nil, false, fmt.Errorf("event: all-day events require date-only start and end values")
		}
		if e.Summary == "" && e.Location == "" && e.Description == "" && e.Start.IsZero() && e.End.IsZero() {
			return "", nil, false, usageError{"event: provide at least a summary (--summary or a positional argument)"}
		}
		return e.String(), nil, false, nil

	case "binary", "bytes":
		var raw []byte
		switch {
		case o.input == "-":
			raw = stdin
		case o.input != "":
			if raw, err = os.ReadFile(o.input); err != nil {
				return "", nil, false, fmt.Errorf("binary --input: %w", err)
			}
		case len(args) > 0:
			raw = []byte(strings.Join(args, " "))
		default:
			raw = stdin
		}
		if len(raw) == 0 {
			return "", nil, false, errNoContent("binary")
		}
		return "", raw, true, nil

	default:
		return "", nil, false, fmt.Errorf("unknown type %q (want text, url, tel, sms, email, geo, wifi, vcard, event, or binary)", typ)
	}
}

func firstArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
}

// stdinString trims a single trailing newline so `echo foo | qrgo` encodes
// "foo", not "foo\n".
func stdinString(stdin []byte) string {
	return strings.TrimRight(string(stdin), "\r\n")
}

// usageError marks an invocation that misses required content; the command
// reports it followed by its own usage instead of a bare error line.
type usageError struct{ msg string }

func (e usageError) Error() string { return e.msg }

func errNoContent(typ string) error {
	return usageError{fmt.Sprintf("no content for %q: pass it as an argument, a flag, or on stdin", typ)}
}

func parseWiFiAuth(s string) (content.WiFiAuth, error) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "WPA", "WPA2", "WPA3":
		return content.WiFiWPA, nil
	case "WEP":
		return content.WiFiWEP, nil
	case "NOPASS", "NONE", "OPEN":
		return content.WiFiNone, nil
	default:
		return "", fmt.Errorf("invalid wifi auth %q (want WPA, WEP, or nopass)", s)
	}
}

// parseEventTime accepts RFC3339, "2006-01-02 15:04", "2006-01-02T15:04", and a
// bare "2006-01-02" (which reports dateOnly = true so the event is all-day).
func parseEventTime(s string) (t time.Time, dateOnly bool, err error) {
	s = strings.TrimSpace(s)
	if t, err = time.Parse(time.RFC3339, s); err == nil {
		return t, false, nil
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02 15:04", "2006-01-02T15:04"} {
		if t, err = time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, false, nil
		}
	}
	if t, err = time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		return t, true, nil
	}
	return time.Time{}, false, fmt.Errorf("invalid time %q (use RFC3339, \"2006-01-02 15:04\", or \"2006-01-02\")", s)
}
