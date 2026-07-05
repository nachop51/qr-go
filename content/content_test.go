package content

import (
	"strings"
	"testing"
	"time"

	qr "github.com/nachop51/qr-go"
)

func TestWiFi(t *testing.T) {
	cases := []struct {
		name string
		in   WiFi
		want string
	}{
		{"wpa default", WiFi{SSID: "home", Pass: "s3cr3t"}, "WIFI:S:home;T:WPA;P:s3cr3t;;"},
		{"open", WiFi{SSID: "cafe"}, "WIFI:S:cafe;T:nopass;;"},
		{"explicit wep", WiFi{SSID: "old", Pass: "abc", Auth: WiFiWEP}, "WIFI:S:old;T:WEP;P:abc;;"},
		{"hidden", WiFi{SSID: "ghost", Pass: "x", Hidden: true}, "WIFI:S:ghost;T:WPA;P:x;H:true;;"},
		{"escaping", WiFi{SSID: "my;net", Pass: `a:b\c`}, `WIFI:S:my\;net;T:WPA;P:a\:b\\c;;`},
	}
	for _, c := range cases {
		if got := c.in.String(); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestVCard(t *testing.T) {
	got := VCard{
		First: "Jane", Last: "Doe",
		Org: "Acme", Title: "CEO",
		Phone: "+15551234567", Email: "jane@acme.test",
		URL: "https://acme.test",
	}.String()

	want := strings.Join([]string{
		"BEGIN:VCARD",
		"VERSION:3.0",
		"N:Doe;Jane;;;",
		"FN:Jane Doe",
		"ORG:Acme",
		"TITLE:CEO",
		"TEL:+15551234567",
		"EMAIL:jane@acme.test",
		"URL:https://acme.test",
		"END:VCARD",
	}, "\n")

	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestVCardFullNameFallback(t *testing.T) {
	got := VCard{First: "Ann", Last: "Lee"}.String()
	if !strings.Contains(got, "FN:Ann Lee") {
		t.Errorf("expected derived FN, got:\n%s", got)
	}
}

func TestEventTimed(t *testing.T) {
	start := time.Date(2026, 7, 4, 9, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 4, 10, 30, 0, 0, time.UTC)
	got := Event{Summary: "Launch", Location: "HQ", Start: start, End: end}.String()

	want := strings.Join([]string{
		"BEGIN:VEVENT",
		"SUMMARY:Launch",
		"LOCATION:HQ",
		"DTSTART:20260704T090000Z",
		"DTEND:20260704T103000Z",
		"END:VEVENT",
	}, "\n")

	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestEventAllDay(t *testing.T) {
	day := time.Date(2026, 12, 25, 0, 0, 0, 0, time.UTC)
	got := Event{Summary: "Holiday", Start: day, AllDay: true}.String()
	if !strings.Contains(got, "DTSTART;VALUE=DATE:20261225") {
		t.Errorf("expected date-only DTSTART, got:\n%s", got)
	}
}

func TestEventTimezoneNormalisation(t *testing.T) {
	// A non-UTC time must be normalised to UTC in the payload.
	loc := time.FixedZone("UTC+2", 2*3600)
	start := time.Date(2026, 7, 4, 11, 0, 0, 0, loc) // == 09:00Z
	got := Event{Summary: "x", Start: start}.String()
	if !strings.Contains(got, "DTSTART:20260704T090000Z") {
		t.Errorf("expected UTC-normalised time, got:\n%s", got)
	}
}

func TestSimpleHelpers(t *testing.T) {
	cases := map[string]string{
		Tel("+15551234567"):        "tel:+15551234567",
		SMS("+15551234567", "hi"):  "SMSTO:+15551234567:hi",
		SMS("+15551234567", ""):    "SMSTO:+15551234567",
		Geo(48.8584, 2.2945):       "geo:48.8584,2.2945",
		URL("https://example.com"): "https://example.com",
		Email("a@b.test", "", ""):  "mailto:a@b.test",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

func TestEmailWithSubjectAndBody(t *testing.T) {
	got := Email("a@b.test", "Hi there", "line one")
	// url.Values.Encode sorts keys: body before subject; spaces become %20.
	want := "mailto:a@b.test?body=line%20one&subject=Hi%20there"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// The payloads must actually encode through the QR pipeline without error.
func TestHelpersEncode(t *testing.T) {
	payloads := []string{
		WiFi{SSID: "home", Pass: "s3cr3t"}.String(),
		VCard{First: "Jane", Last: "Doe", Email: "jane@acme.test"}.String(),
		Event{Summary: "Launch", Start: time.Date(2026, 7, 4, 9, 0, 0, 0, time.UTC)}.String(),
		Tel("+15551234567"),
		Geo(48.8584, 2.2945),
		Email("a@b.test", "Hi", "Body text"),
	}
	for _, p := range payloads {
		if _, err := qr.NewTextBuilder(p).Build(); err != nil {
			t.Errorf("Build failed for %q: %v", p, err)
		}
	}
}
