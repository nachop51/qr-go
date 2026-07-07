package main

import (
	"io"
	"os"

	qr "github.com/nachop51/qr-go"
	"github.com/spf13/cobra"
)

// options holds every flag value. The render/output fields are bound to the
// root's persistent flags (shared by every subcommand); the content fields are
// bound to the flags of the subcommand they belong to.
type options struct {
	// Wi-Fi
	ssid, pass, auth string
	hidden           bool

	// vCard
	name, first, last, org, title, phone, email, url, address string

	// Calendar event
	summary, location, description, start, end string
	allDay                                     bool

	// Email / SMS
	subject, body, message string

	// Geo
	lat, lng float64

	// Binary
	input string // file to read raw bytes from ("-" = stdin)

	// Output / rendering (persistent shared by all commands)
	output      string
	format      string
	ecc         string
	quiet       int // -1 == unset (keep the renderer's own default)
	dark        string
	light       string
	width       int
	height      int
	size        int
	scale       int
	invert      bool
	block       bool
	logo        string
	logoModules int
	noECI       bool
	info        bool
}

const longDescription = `Generate QR codes from the terminal.

With no subcommand the arguments are encoded as plain text and printed to the
terminal:

  qrgo "HELLO WORLD"

Structured content types are subcommands, run "qrgo help <type>" (e.g.
"qrgo help wifi") for that type's own options:

  qrgo url https://example.com -o code.png
  qrgo tel +15551234567 -f svg
  qrgo geo 48.8584 2.2945
  qrgo wifi --ssid CoffeeShop --pass latte123 --ecc H -o wifi.svg
  qrgo vcard --name "Jane Doe" --email jane@acme.test -o card.png

The output format is inferred from -o's extension (.png/.svg), or set with
-f/--format. With no -o the code goes to stdout. Content can also arrive on
stdin: echo "https://example.com" | qrgo -o code.png`

func newRootCmd() *cobra.Command {
	o := &options{}

	cmd := &cobra.Command{
		Use:           "qrgo [text...] | qrgo <type> [args]",
		Short:         "Generate QR codes from the terminal",
		Long:          longDescription,
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE:          o.runRoot,
	}

	// Render/output flags are persistent so every content subcommand inherits
	// them; they stay out of each subcommand's own (content-specific) flag list.
	f := cmd.PersistentFlags()
	f.SortFlags = false // keep the logical grouping below

	f.StringVarP(&o.output, "output", "o", "", "output file (format inferred from its extension)")
	f.StringVarP(&o.format, "format", "f", "", "output format: terminal, png, or svg")
	f.StringVarP(&o.ecc, "ecc", "e", "M", "error-correction level: L, M, Q, or H")
	f.IntVarP(&o.quiet, "quiet", "q", -1, "quiet-zone width in modules (default: per-renderer)")
	f.StringVar(&o.dark, "dark", "#000000", "foreground color for PNG/SVG (hex for PNG; any CSS color for SVG)")
	f.StringVar(&o.light, "light", "#ffffff", "background color for PNG/SVG")
	f.IntVar(&o.width, "width", 800, "PNG canvas width in px")
	f.IntVar(&o.height, "height", 800, "PNG canvas height in px")
	f.IntVar(&o.size, "size", 0, "PNG square size in px (sets both width and height)")
	f.IntVar(&o.scale, "scale", 10, "SVG module size in px")
	f.BoolVar(&o.invert, "invert", false, "terminal: swap dark/light (for dark backgrounds)")
	f.BoolVar(&o.block, "block", false, "terminal: classic full-block style")
	f.StringVar(&o.logo, "logo", "", "overlay a logo image (PNG/JPEG/GIF/WebP/SVG) on PNG/SVG output")
	f.IntVar(&o.logoModules, "logo-modules", 0, "logo span in modules (0 = default size/5)")
	f.BoolVar(&o.noECI, "no-eci", false, "disable automatic ECI for text")
	f.BoolVarP(&o.info, "info", "i", false, "print the encoding outcome (version, mask, segments) to stdout")

	cmd.AddCommand(contentCommands(o)...)

	return cmd
}

// runRoot handles the default (text) path and the bare-invocation help.
func (o *options) runRoot(cmd *cobra.Command, args []string) error {
	stdin := o.maybeReadStdin(cmd, args)

	// A bare invocation: no arguments, nothing piped, no flags. Has nothing to
	// encode and no expressed intent: show the full help instead of an error.
	if len(args) == 0 && stdin == nil && cmd.Flags().NFlag() == 0 {
		return cmd.Help()
	}

	return o.encode(cmd, "text", args, stdin)
}

// runContent returns the RunE for a content subcommand of the given type.
func (o *options) runContent(typ string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return o.encode(cmd, typ, args, o.maybeReadStdin(cmd, args))
	}
}

// maybeReadStdin returns piped stdin when it could be the payload source.
func (o *options) maybeReadStdin(cmd *cobra.Command, args []string) []byte {
	if o.input == "-" || (o.input == "" && len(args) == 0) {
		if data, ok := readPipedStdin(cmd.InOrStdin()); ok {
			return data
		}
	}
	return nil
}

// encode builds the payload for typ, then renders it to the resolved output.
func (o *options) encode(cmd *cobra.Command, typ string, args []string, stdin []byte) error {
	format, err := resolveFormat(o.format, o.output)
	if err != nil {
		return err
	}

	text, data, binary, err := buildPayload(o, typ, args, stdin)
	if err != nil {
		return err
	}

	// Resolve the output sink: a file if -o is set, else the command's stdout.
	out := cmd.OutOrStdout()
	if o.output != "" {
		file, err := os.Create(o.output)
		if err != nil {
			return err
		}
		defer file.Close()
		out = file
	}

	renderer, err := buildRenderer(format, o, out)
	if err != nil {
		return err
	}

	var b *qr.Builder
	if binary {
		b = qr.NewBinaryBuilder(data)
	} else {
		b = qr.NewTextBuilder(text)
	}

	level, err := parseECC(o.ecc)
	if err != nil {
		return err
	}
	b.SetErrorCorrectionLevel(level)
	if o.noECI {
		b.SetTextECIPolicy(qr.TextECIPolicyDisabled)
	}
	b.SetRenderer(renderer)

	code, err := b.Build()
	if err != nil {
		return err
	}

	// --info goes to stdout. When output is a file, that's the terminal; when the
	// code itself renders to stdout (terminal format) it prints just before it.
	if o.info {
		printInfo(cmd.OutOrStdout(), code)
	}

	return code.Render()
}

// readPipedStdin returns stdin's contents when it is piped or redirected. For a
// real *os.File it checks the char-device bit so an interactive terminal isn't
// read from; any other reader (e.g. a test buffer) is read directly.
func readPipedStdin(r io.Reader) ([]byte, bool) {
	if f, ok := r.(*os.File); ok {
		fi, err := f.Stat()
		if err != nil || fi.Mode()&os.ModeCharDevice != 0 {
			return nil, false
		}
	}
	data, err := io.ReadAll(r)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data, true
}
