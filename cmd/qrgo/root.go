package main

import (
	"errors"
	"io"
	"os"

	qr "github.com/nachop51/qr-go"
	"github.com/spf13/cobra"
)

// version is stamped by the release build via
// -ldflags "-X main.version=v0.1.0"; plain "go install" builds show "dev".
var version = "dev"

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
	logoScale   int
	noECI       bool
	info        bool

	// Styling (PNG/SVG only)
	shape         string
	moduleShape   string
	eyeShape      string
	eyeFrameShape string
	eyeBallShape  string
	eyeFrame      string
	eyeBall       string
	gradient      string
}

// styleConfigured reports whether any style flag departs from the default
// square look; the terminal renderer rejects these.
func (o *options) styleConfigured() bool {
	return (o.shape != "" && o.shape != "square") ||
		(o.moduleShape != "" && o.moduleShape != "square") ||
		o.eyeShape != "" || o.eyeFrameShape != "" || o.eyeBallShape != "" ||
		o.eyeFrame != "" || o.eyeBall != "" || o.gradient != ""
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

// helpTemplate renders the root's help as usage/commands/flags first and the
// long description last (the description lives at the end of usageTemplate).
// Leaf subcommands keep cobra's default description-first layout.
const helpTemplate = `{{if .HasAvailableSubCommands}}{{.UsageString}}{{else}}{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}{{end}}`

// usageTemplate is cobra's default with two changes: for commands with
// subcommands (only the root here) the long description renders between the
// flags and the final help hint (helpTemplate skips its own description for
// those commands), and on subcommands the inherited output/render flags are
// replaced by a one-line pointer to the root help.
const usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

The shared output/render flags also apply; see "{{.Root.CommandPath}} help".{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}{{with .Long}}

{{. | trimTrailingWhitespaces}}{{end}}

Use "{{.CommandPath}} help <command>" for more information about a command.{{end}}
`

func newRootCmd() *cobra.Command {
	o := &options{}

	cmd := &cobra.Command{
		Use:           "qrgo [text...] | qrgo <type> [args]",
		Short:         "Generate QR codes from the terminal",
		Version:       version,
		Long:          longDescription,
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE:          o.runRoot,
	}
	cmd.SetHelpTemplate(helpTemplate)
	cmd.SetUsageTemplate(usageTemplate)

	// Render/output flags are persistent so every content subcommand inherits
	// them; they stay out of each subcommand's own (content-specific) flag list.
	f := cmd.PersistentFlags()
	f.SortFlags = false // keep the logical grouping below

	f.StringVarP(&o.output, "output", "o", "", "output file (format inferred from its extension)")
	f.StringVarP(&o.format, "format", "f", "", "output format: terminal, png, or svg")
	f.StringVarP(&o.ecc, "ecc", "e", "M", "error-correction level: L, M, Q, or H (H when --logo is set)")
	f.IntVarP(&o.quiet, "quiet", "q", -1, "quiet-zone width in modules (default: per-renderer)")
	f.StringVar(&o.dark, "dark", "#000000", "foreground color for PNG/SVG (hex for PNG; any CSS color for SVG)")
	f.StringVar(&o.light, "light", "#ffffff", "background color for PNG/SVG")
	f.IntVar(&o.width, "width", 800, "PNG canvas width in px")
	f.IntVar(&o.height, "height", 800, "PNG canvas height in px")
	f.IntVar(&o.size, "size", 0, "PNG square size in px (sets both width and height)")
	f.IntVar(&o.scale, "scale", 10, "SVG module size in px")
	f.BoolVar(&o.invert, "invert", false, "terminal: swap dark/light (for dark backgrounds)")
	f.BoolVar(&o.block, "block", false, "terminal: classic full-block style")
	f.StringVarP(&o.logo, "logo", "l", "", "overlay a logo image (PNG/JPEG/GIF/WebP/SVG) on PNG/SVG output")
	f.IntVar(&o.logoModules, "logo-modules", 0, "logo span in modules (0 = max the EC level allows)")
	f.IntVar(&o.logoScale, "logo-scale", 0, "percent of the logo area the image fills, up to 100 (default 70-80 by logo span)")
	f.StringVarP(&o.shape, "shape", "s", "", "PNG/SVG shape for modules and eyes: square, rounded, or circle (specific shape flags override)")
	f.StringVar(&o.moduleShape, "module-shape", "", "PNG/SVG module shape: square, rounded, or dot (default square)")
	f.StringVar(&o.eyeShape, "eye-shape", "", "PNG/SVG finder eye shape (frame and ball): square, rounded, or circle")
	f.StringVar(&o.eyeFrameShape, "eye-frame-shape", "", "finder frame shape (overrides --eye-shape)")
	f.StringVar(&o.eyeBallShape, "eye-ball-shape", "", "finder ball shape (overrides --eye-shape)")
	f.StringVar(&o.eyeFrame, "eye-frame", "", "finder frame color (default: --dark)")
	f.StringVar(&o.eyeBall, "eye-ball", "", "finder ball color (default: --dark)")
	f.StringVar(&o.gradient, "gradient", "", "module gradient: linear:<from>:<to>[:angle] or radial:<from>:<to>")
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

	return reportUsageError(cmd, o.encode(cmd, "text", args, stdin))
}

// runContent returns the RunE for a content subcommand of the given type.
func (o *options) runContent(typ string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return reportUsageError(cmd, o.encode(cmd, typ, args, o.maybeReadStdin(cmd, args)))
	}
}

// reportUsageError prints a usageError followed by the command's usage, then
// keeps cobra from repeating the bare error line. The root's SilenceUsage
// blocks cobra's own usage-on-error for every subcommand, which is right for
// runtime errors but not for a bad invocation. Other errors pass through.
func reportUsageError(cmd *cobra.Command, err error) error {
	var ue usageError
	if !errors.As(err, &ue) {
		return err
	}
	cmd.PrintErrln(cmd.ErrPrefix(), err.Error())
	cmd.PrintErrln()
	cmd.PrintErr(cmd.UsageString())
	cmd.SilenceErrors = true
	return err
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
	// A logo covers modules, so unless the user picked a level themselves,
	// raise the default to the highest error correction. That also maximizes
	// the module budget a logo may clear (--logo-modules 0 spans it fully).
	if o.logo != "" && !cmd.Flags().Changed("ecc") {
		level = qr.CorrectionLevelHigh
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
