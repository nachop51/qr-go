// Package content builds the specially-formatted payload strings that QR code
// scanners recognise as actionable content: Wi-Fi networks, contact cards,
// calendar events, phone numbers, and so on.
//
// Every helper returns a plain string ready to hand to the encoder:
//
//	qr.NewTextBuilder(content.WiFi{SSID: "home", Pass: "s3cr3t"}.String()).Build()
//	qr.NewTextBuilder(content.Tel("+15551234567")).Build()
//
// The multi-field types (WiFi, VCard, Event) implement fmt.Stringer; the
// simple ones are plain functions.
package content

import (
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// WiFiAuth is a Wi-Fi network's authentication type.
type WiFiAuth string

const (
	WiFiWPA  WiFiAuth = "WPA"    // WPA / WPA2 / WPA3
	WiFiWEP  WiFiAuth = "WEP"    // legacy WEP
	WiFiNone WiFiAuth = "nopass" // open network
)

// WiFi encodes credentials for joining a wireless network. Scanning it prompts
// the device to connect. When Auth is empty it defaults to WPA if a password
// is set, otherwise to an open (nopass) network.
type WiFi struct {
	SSID   string
	Pass   string
	Auth   WiFiAuth
	Hidden bool
}

// String renders the WIFI: payload, e.g. WIFI:S:home;T:WPA;P:s3cr3t;;
func (w WiFi) String() string {
	auth := w.Auth
	if auth == "" {
		if w.Pass == "" {
			auth = WiFiNone
		} else {
			auth = WiFiWPA
		}
	}

	var b strings.Builder
	b.WriteString("WIFI:S:")
	b.WriteString(wifiEscape(w.SSID))
	b.WriteString(";T:")
	b.WriteString(string(auth))
	b.WriteByte(';')
	if auth != WiFiNone && w.Pass != "" {
		b.WriteString("P:")
		b.WriteString(wifiEscape(w.Pass))
		b.WriteByte(';')
	}
	if w.Hidden {
		b.WriteString("H:true;")
	}
	b.WriteByte(';')
	return b.String()
}

// VCard encodes a contact as a vCard 3.0 record. Empty fields are omitted.
type VCard struct {
	FullName string // FN, e.g. "Jane Doe" (derived from First/Last if empty)
	First    string
	Last     string
	Org      string
	Title    string
	Phone    string
	Email    string
	URL      string
	Address  string // single-line street address
}

// String renders the BEGIN:VCARD … END:VCARD record.
func (v VCard) String() string {
	lines := []string{"BEGIN:VCARD", "VERSION:3.0"}

	if v.First != "" || v.Last != "" {
		lines = append(lines, "N:"+vcEscape(v.Last)+";"+vcEscape(v.First)+";;;")
	}
	fn := v.FullName
	if fn == "" {
		fn = strings.TrimSpace(v.First + " " + v.Last)
	}
	if fn != "" {
		lines = append(lines, "FN:"+vcEscape(fn))
	}
	for _, f := range []struct{ tag, val string }{
		{"ORG", v.Org},
		{"TITLE", v.Title},
		{"TEL", v.Phone},
		{"EMAIL", v.Email},
		{"URL", v.URL},
	} {
		if f.val != "" {
			lines = append(lines, f.tag+":"+vcEscape(f.val))
		}
	}
	if v.Address != "" {
		lines = append(lines, "ADR:;;"+vcEscape(v.Address)+";;;;")
	}

	lines = append(lines, "END:VCARD")
	return foldContentLines(lines)
}

// Event encodes a calendar entry as an iCalendar VEVENT. A zero Start or End is
// omitted; set AllDay for date-only (no time) events.
type Event struct {
	Summary     string
	Location    string
	Description string
	Start       time.Time
	End         time.Time
	AllDay      bool
}

// String renders the BEGIN:VEVENT … END:VEVENT record.
func (e Event) String() string {
	lines := []string{"BEGIN:VEVENT"}
	for _, f := range []struct{ tag, val string }{
		{"SUMMARY", e.Summary},
		{"LOCATION", e.Location},
		{"DESCRIPTION", e.Description},
	} {
		if f.val != "" {
			lines = append(lines, f.tag+":"+vcEscape(f.val))
		}
	}
	if !e.Start.IsZero() {
		lines = append(lines, "DTSTART"+icalTime(e.Start, e.AllDay))
	}
	if !e.End.IsZero() {
		lines = append(lines, "DTEND"+icalTime(e.End, e.AllDay))
	}
	lines = append(lines, "END:VEVENT")
	return foldContentLines(lines)
}

// URL returns the address unchanged; a URL QR code is simply the URL text.
// Provided for discoverability alongside the other helpers.
func URL(u string) string { return u }

// Tel encodes a phone number as a tel: URI. Scanning it starts a call.
func Tel(number string) string { return "tel:" + number }

// SMS encodes a pre-filled text message using the widely supported SMSTO form.
func SMS(number, message string) string {
	if message == "" {
		return "SMSTO:" + number
	}
	return "SMSTO:" + number + ":" + message
}

// Geo encodes geographic coordinates as a geo: URI.
func Geo(lat, lng float64) string {
	return "geo:" + strconv.FormatFloat(lat, 'f', -1, 64) +
		"," + strconv.FormatFloat(lng, 'f', -1, 64)
}

// Email encodes a mailto: link with an optional pre-filled subject and body.
func Email(to, subject, body string) string {
	to = mailtoEscape(to)
	if subject == "" && body == "" {
		return "mailto:" + to
	}
	vals := url.Values{}
	if subject != "" {
		vals.Set("subject", subject)
	}
	if body != "" {
		vals.Set("body", body)
	}
	// mailto clients expect %20 for spaces rather than '+'.
	q := strings.ReplaceAll(vals.Encode(), "+", "%20")
	return "mailto:" + to + "?" + q
}

// mailtoEscape percent-encodes the characters that would terminate or corrupt
// the recipient part of a mailto: URI. '@' and the RFC 6068 multi-recipient
// separator ',' stay literal.
func mailtoEscape(s string) string {
	return mailtoReplacer.Replace(s)
}

var mailtoReplacer = strings.NewReplacer(
	"%", "%25",
	" ", "%20",
	"?", "%3F",
	"#", "%23",
	"&", "%26",
	"\r", "%0D",
	"\n", "%0A",
	"\t", "%09",
	`"`, "%22",
	"<", "%3C",
	">", "%3E",
)

func icalTime(t time.Time, allDay bool) string {
	if allDay {
		return ";VALUE=DATE:" + t.Format("20060102")
	}
	return ":" + t.UTC().Format("20060102T150405Z")
}

// wifiEscape backslash-escapes the characters reserved in a WIFI: payload.
func wifiEscape(s string) string {
	return wifiReplacer.Replace(s)
}

var wifiReplacer = strings.NewReplacer(
	`\`, `\\`,
	`;`, `\;`,
	`,`, `\,`,
	`:`, `\:`,
	`"`, `\"`,
)

// vcEscape backslash-escapes the characters reserved in vCard/iCalendar text.
func vcEscape(s string) string {
	return vcReplacer.Replace(s)
}

var vcReplacer = strings.NewReplacer(
	`\`, `\\`,
	`;`, `\;`,
	`,`, `\,`,
	"\r\n", `\n`,
	"\r", `\n`,
	"\n", `\n`,
)

// foldContentLines emits CRLF-terminated vCard/iCalendar lines and folds at
// 75 UTF-8 octets. Continuation lines begin with one space, which counts
// toward their 75-octet limit.
func foldContentLines(lines []string) string {
	var out strings.Builder
	for _, line := range lines {
		first := true
		for len(line) > 0 {
			limit := 75
			if !first {
				out.WriteByte(' ')
				limit--
			}
			cut := utf8PrefixLen(line, limit)
			out.WriteString(line[:cut])
			out.WriteString("\r\n")
			line = line[cut:]
			first = false
		}
		if first {
			out.WriteString("\r\n")
		}
	}
	return out.String()
}

func utf8PrefixLen(s string, maxBytes int) int {
	if len(s) <= maxBytes {
		return len(s)
	}
	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	if cut == 0 {
		// Invalid UTF-8: a run of continuation bytes longer than the limit has
		// no rune boundary to back up to. Hard-cut at the byte limit so the
		// caller always makes progress instead of looping forever.
		return maxBytes
	}
	return cut
}
