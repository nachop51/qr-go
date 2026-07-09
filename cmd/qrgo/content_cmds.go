package main

import (
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// contentCommands builds one subcommand per content type. Only content-specific
// flags live here; the render/output flags are inherited from the root's
// persistent flags, keeping each subcommand's help focused on its own type.
func contentCommands(o *options) []*cobra.Command {
	text := &cobra.Command{
		Use:     "text [text...]",
		Short:   "Encode plain text (the default with no subcommand)",
		Example: "  qrgo text \"HELLO WORLD\"",
		Args:    cobra.ArbitraryArgs,
		RunE:    o.runContent("text"),
	}

	url := &cobra.Command{
		Use:     "url <url>",
		Short:   "Encode a URL",
		Example: "  qrgo url https://example.com -o code.png",
		Args:    cobra.ArbitraryArgs,
		RunE:    o.runContent("url"),
	}

	tel := &cobra.Command{
		Use:     "tel <number>",
		Short:   "Encode a phone number as a tel: URI (dial on scan)",
		Example: "  qrgo tel +15551234567 -f svg",
		Args:    cobra.ArbitraryArgs,
		RunE:    o.runContent("tel"),
	}

	sms := &cobra.Command{
		Use:     "sms <number> [message...]",
		Short:   "Encode a pre-filled SMS (SMSTO:)",
		Example: "  qrgo sms +15551234567 \"call me\"",
		Args:    cobra.ArbitraryArgs,
		RunE:    o.runContent("sms"),
	}
	sms.Flags().StringVar(&o.message, "message", "", "message body (or give it positionally)")

	email := &cobra.Command{
		Use:     "email <address>",
		Short:   "Encode a mailto: link (compose on scan)",
		Example: "  qrgo email a@b.test --subject Hi --body \"Hello there\"",
		Args:    cobra.ArbitraryArgs,
		RunE:    o.runContent("email"),
	}
	email.Flags().StringVar(&o.subject, "subject", "", "subject line")
	email.Flags().StringVar(&o.body, "body", "", "body text")

	// pflag reads a negative coordinate ("-20.05") as a run of shorthand
	// flags, so geo parses its own arguments: number-shaped tokens become
	// positionals, everything else goes through the normal flag parser.
	geo := &cobra.Command{
		Use:                "geo <lat> <lng>",
		Short:              "Encode geographic coordinates as a geo: URI",
		Example:            "  qrgo geo 48.8584 2.2945\n  qrgo geo -20.0566 57.5250 -o mru.png",
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			pos, err := parseFlagsKeepingNegatives(cmd, args)
			if err != nil {
				return err
			}
			if help, _ := cmd.Flags().GetBool("help"); help {
				return cmd.Help()
			}
			bothFlags := cmd.Flags().Changed("lat") && cmd.Flags().Changed("lng")
			if !(len(pos) == 2 || (len(pos) == 0 && bothFlags)) {
				return reportUsageError(cmd, usageError{"geo: give both coordinates, either positionally (qrgo geo <lat> <lng>) or with --lat and --lng"})
			}
			return o.runContent("geo")(cmd, pos)
		},
	}
	geo.Flags().Float64Var(&o.lat, "lat", 0, "latitude (when not given positionally)")
	geo.Flags().Float64Var(&o.lng, "lng", 0, "longitude (when not given positionally)")

	wifi := &cobra.Command{
		Use:     "wifi [ssid]",
		Short:   "Encode Wi-Fi credentials (join the network on scan)",
		Example: "  qrgo wifi --ssid CoffeeShop --pass latte123 --ecc H -o wifi.svg",
		Args:    cobra.ArbitraryArgs,
		RunE:    o.runContent("wifi"),
	}
	wifi.Flags().StringVar(&o.ssid, "ssid", "", "network name (or give it positionally)")
	wifi.Flags().StringVar(&o.pass, "pass", "", "network password")
	wifi.Flags().StringVar(&o.auth, "auth", "", "authentication: WPA, WEP, or nopass")
	wifi.Flags().BoolVar(&o.hidden, "hidden", false, "the network is hidden")

	vcard := &cobra.Command{
		Use:     "vcard [full name]",
		Short:   "Encode a contact card (vCard; save on scan)",
		Example: "  qrgo vcard --name \"Jane Doe\" --email jane@acme.test -o card.png",
		Args:    cobra.ArbitraryArgs,
		RunE:    o.runContent("vcard"),
	}
	vcard.Flags().StringVar(&o.name, "name", "", "full name (or give it positionally)")
	vcard.Flags().StringVar(&o.first, "first", "", "first name")
	vcard.Flags().StringVar(&o.last, "last", "", "last name")
	vcard.Flags().StringVar(&o.org, "org", "", "organization")
	vcard.Flags().StringVar(&o.title, "title", "", "job title")
	vcard.Flags().StringVar(&o.phone, "phone", "", "phone number")
	vcard.Flags().StringVar(&o.email, "email", "", "email address")
	vcard.Flags().StringVar(&o.url, "url", "", "website")
	vcard.Flags().StringVar(&o.address, "address", "", "street address")

	event := &cobra.Command{
		Use:     "event [summary]",
		Short:   "Encode a calendar event (iCalendar VEVENT)",
		Example: "  qrgo event --summary Launch --start \"2026-07-05 14:30\" -f svg",
		Args:    cobra.ArbitraryArgs,
		RunE:    o.runContent("event"),
	}
	event.Flags().StringVar(&o.summary, "summary", "", "title/summary (or give it positionally)")
	event.Flags().StringVar(&o.location, "location", "", "location")
	event.Flags().StringVar(&o.description, "description", "", "description")
	event.Flags().StringVar(&o.start, "start", "", "start (RFC3339, \"2006-01-02 15:04\", or \"2006-01-02\")")
	event.Flags().StringVar(&o.end, "end", "", "end (same formats as --start)")
	event.Flags().BoolVar(&o.allDay, "all-day", false, "all-day (date only, no time)")

	binary := &cobra.Command{
		Use:     "binary [bytes...]",
		Short:   "Encode raw bytes (single byte segment, no ECI)",
		Example: "  qrgo binary --input payload.bin -o code.png",
		Args:    cobra.ArbitraryArgs,
		RunE:    o.runContent("binary"),
	}
	binary.Flags().StringVar(&o.input, "input", "", "read raw bytes from a file (- for stdin)")

	return []*cobra.Command{text, url, tel, sms, email, geo, wifi, vcard, event, binary}
}

// negNumber matches a token that is a negative decimal number ("-20", "-.5",
// "-20.0566"); such a token can never be a flag here, no flag has a numeric
// shorthand.
var negNumber = regexp.MustCompile(`^-(\d+(\.\d*)?|\.\d+)$`)

// parseFlagsKeepingNegatives is the flag pass for a command with
// DisableFlagParsing: it routes negative-number tokens to the returned
// positionals and parses the remaining arguments as flags.
func parseFlagsKeepingNegatives(cmd *cobra.Command, args []string) ([]string, error) {
	// cmd.ParseFlags is a no-op under DisableFlagParsing, so parse the flag
	// set directly. InheritedFlags first: its side effect merges the
	// persistent flags into cmd.Flags() for the lookups and parse below.
	cmd.InheritedFlags()
	fs := cmd.Flags()

	var flagArgs, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			pos = append(pos, args[i+1:]...)
			i = len(args)
		case negNumber.MatchString(a):
			pos = append(pos, a)
		case len(a) > 1 && strings.HasPrefix(a, "-"):
			flagArgs = append(flagArgs, a)
			if flagWantsNextArg(fs, a) && i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
		default:
			pos = append(pos, a)
		}
	}

	return pos, fs.Parse(flagArgs)
}

// flagWantsNextArg reports whether the flag token consumes the following
// argument as its value (so a negative number there stays with its flag).
func flagWantsNextArg(fs *pflag.FlagSet, token string) bool {
	if strings.Contains(token, "=") {
		return false
	}
	if strings.HasPrefix(token, "--") {
		f := fs.Lookup(token[2:])
		return f != nil && f.NoOptDefVal == ""
	}
	// Shorthand run ("-eM", "-io"): only a value flag in last position takes
	// the next argument; earlier ones consume the rest of the token itself.
	body := token[1:]
	for i := 0; i < len(body); i++ {
		f := fs.ShorthandLookup(string(body[i]))
		if f == nil {
			return false
		}
		if f.NoOptDefVal != "" {
			continue // boolean-style: consumes nothing
		}
		return i == len(body)-1
	}
	return false
}
