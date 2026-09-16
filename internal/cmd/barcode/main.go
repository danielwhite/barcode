// Command barcode is a general-purpose CLI for exercising the symbologies
// implemented by this module. It encodes CLI-supplied data with any of the
// library's Encode functions and writes the result as a PNG, so ad-hoc or
// edge-case barcodes can be produced (e.g. to reproduce a specific encoder
// input) without writing a bespoke Go program for each one.
package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"
	"strconv"
	"strings"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/aztec"
	"github.com/boombuler/barcode/codabar"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/code39"
	"github.com/boombuler/barcode/code93"
	"github.com/boombuler/barcode/datamatrix"
	"github.com/boombuler/barcode/ean"
	"github.com/boombuler/barcode/pdf417"
	"github.com/boombuler/barcode/qr"
	"github.com/boombuler/barcode/twooffive"
)

// formats lists the accepted values for the format positional argument, in
// the order shown in -help output.
var formats = []string{
	"aztec", "codabar", "code128", "code39", "code93",
	"datamatrix", "ean", "pdf417", "qr", "2of5",
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "barcode:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("barcode", flag.ContinueOnError)
	fs.Usage = func() { usage(fs) }

	if len(args) == 0 || args[0] == "-h" || args[0] == "-help" || args[0] == "--help" {
		fs.Usage()
		if len(args) == 0 {
			return fmt.Errorf("format is required")
		}
		return nil
	}
	if strings.HasPrefix(args[0], "-") {
		fs.Usage()
		return fmt.Errorf("format must come first, got %q", args[0])
	}
	format, args := args[0], args[1:]

	optionsFlag := fs.String("options", "", "comma-separated key=value list of format-specific options (see -help)")
	scaleFlag := fs.String("scale", "", "scale output to WIDTHxHEIGHT pixels, or NxN if given a single number (default: native size)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	positional := fs.Args()
	if len(positional) == 0 {
		fs.Usage()
		return fmt.Errorf("text is required")
	}
	if len(positional) > 2 {
		return fmt.Errorf("too many arguments after flags: %v (did you forget to quote <text>?)", positional)
	}
	text := positional[0]
	out := ""
	if len(positional) == 2 {
		out = positional[1]
	}

	opts := parseOptions(*optionsFlag)

	bc, err := build(format, text, opts)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	width, height, err := parseScale(*scaleFlag)
	if err != nil {
		return err
	}
	if width > 0 && height > 0 {
		if bc, err = barcode.Scale(bc, width, height); err != nil {
			return fmt.Errorf("scale: %w", err)
		}
	}

	quiet, err := optBool(opts, "quiet", false)
	if err != nil {
		return err
	}
	if !quiet {
		printSummary(bc)
	}

	return writePNG(bc, out)
}

// parseOptions splits a comma-separated key=value list into a map. A bare
// key with no "=" is treated as key=true, so boolean options can be written
// as e.g. "-options checksum,fullascii" instead of "checksum=true".
func parseOptions(s string) map[string]string {
	opts := map[string]string{}
	for _, kv := range strings.Split(s, ",") {
		kv = strings.TrimSpace(kv)
		if kv == "" {
			continue
		}
		key, value, ok := strings.Cut(kv, "=")
		if !ok {
			key, value = kv, "true"
		}
		opts[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
	}
	return opts
}

func optString(opts map[string]string, key, def string) string {
	if v, ok := opts[key]; ok {
		return v
	}
	return def
}

func optInt(opts map[string]string, key string, def int) (int, error) {
	v, ok := opts[key]
	if !ok {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("option %s: invalid integer %q", key, v)
	}
	return n, nil
}

func optBool(opts map[string]string, key string, def bool) (bool, error) {
	v, ok := opts[key]
	if !ok {
		return def, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("option %s: invalid boolean %q", key, v)
	}
	return b, nil
}

// parseScale parses the -scale flag: "" means no scaling, "N" means an NxN
// target size, and "WxH" (or "WxH" with an uppercase X) means an explicit
// width and height.
func parseScale(s string) (w, h int, err error) {
	if s == "" {
		return 0, 0, nil
	}
	if i := strings.IndexAny(s, "xX"); i >= 0 {
		w, err = strconv.Atoi(strings.TrimSpace(s[:i]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid -scale %q: %w", s, err)
		}
		h, err = strconv.Atoi(strings.TrimSpace(s[i+1:]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid -scale %q: %w", s, err)
		}
		return w, h, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid -scale %q: want WIDTHxHEIGHT or N", s)
	}
	return n, n, nil
}

func colorScheme(name string) (barcode.ColorScheme, error) {
	switch strings.ToLower(name) {
	case "mono", "mono8":
		return barcode.ColorScheme8, nil
	case "gray16", "":
		return barcode.ColorScheme16, nil
	case "rgb", "rgb24":
		return barcode.ColorScheme24, nil
	case "rgba", "rgba32":
		return barcode.ColorScheme32, nil
	}
	return barcode.ColorScheme{}, fmt.Errorf("unknown color %q (want mono, gray16, rgb or rgba)", name)
}

// build dispatches to the Encode function for the requested format, reading
// any format-specific settings out of opts (see usage() for the keys each
// format understands).
func build(format, content string, opts map[string]string) (barcode.Barcode, error) {
	scheme, err := colorScheme(optString(opts, "color", "gray16"))
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(strings.TrimSpace(format)) {
	case "aztec":
		minECC, err := optInt(opts, "minecc", 33)
		if err != nil {
			return nil, err
		}
		layers, err := optInt(opts, "layers", 0)
		if err != nil {
			return nil, err
		}
		return aztec.EncodeWithColor([]byte(content), minECC, layers, scheme)
	case "codabar":
		return codabar.EncodeWithColor(content, scheme)
	case "code128":
		checksum, err := optBool(opts, "checksum", true)
		if err != nil {
			return nil, err
		}
		if checksum {
			return code128.EncodeWithColor(content, scheme)
		}
		return code128.EncodeWithoutChecksumWithColor(content, scheme)
	case "code39":
		checksum, err := optBool(opts, "checksum", true)
		if err != nil {
			return nil, err
		}
		fullASCII, err := optBool(opts, "fullascii", false)
		if err != nil {
			return nil, err
		}
		return code39.EncodeWithColor(content, checksum, fullASCII, scheme)
	case "code93":
		checksum, err := optBool(opts, "checksum", true)
		if err != nil {
			return nil, err
		}
		fullASCII, err := optBool(opts, "fullascii", false)
		if err != nil {
			return nil, err
		}
		return code93.EncodeWithColor(content, checksum, fullASCII, scheme)
	case "datamatrix":
		return datamatrix.EncodeWithColor(content, scheme)
	case "ean":
		return ean.EncodeWithColor(content, scheme)
	case "pdf417":
		level, err := optInt(opts, "level", 0)
		if err != nil {
			return nil, err
		}
		if level < 0 || level > 8 {
			return nil, fmt.Errorf("option level must be between 0 and 8, got %d", level)
		}
		return pdf417.EncodeWithColor(content, byte(level), scheme)
	case "qr":
		lvl, err := parseQRLevel(optString(opts, "level", "L"))
		if err != nil {
			return nil, err
		}
		mode, err := parseQRMode(optString(opts, "mode", "Auto"))
		if err != nil {
			return nil, err
		}
		return qr.EncodeWithColor(content, lvl, mode, scheme)
	case "2of5", "twooffive":
		interleaved, err := optBool(opts, "interleaved", false)
		if err != nil {
			return nil, err
		}
		return twooffive.EncodeWithColor(content, interleaved, scheme)
	}
	return nil, fmt.Errorf("unknown format %q (want one of: %s)", format, strings.Join(formats, ", "))
}

func parseQRLevel(s string) (qr.ErrorCorrectionLevel, error) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "L":
		return qr.L, nil
	case "M":
		return qr.M, nil
	case "Q":
		return qr.Q, nil
	case "H":
		return qr.H, nil
	}
	return 0, fmt.Errorf("option level: unknown qr level %q (want L, M, Q or H)", s)
}

func parseQRMode(s string) (qr.Encoding, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "auto", "":
		return qr.Auto, nil
	case "numeric":
		return qr.Numeric, nil
	case "alphanumeric":
		return qr.AlphaNumeric, nil
	case "unicode":
		return qr.Unicode, nil
	}
	return 0, fmt.Errorf("option mode: unknown qr mode %q (want Auto, Numeric, AlphaNumeric or Unicode)", s)
}

// printSummary writes a one-line description of the produced barcode to
// stderr, so callers piping the PNG bytes on stdout still get feedback
// about what was actually encoded (row/column counts, checksum, etc. are
// implied by the image size; the content and kind are made explicit here).
func printSummary(bc barcode.Barcode) {
	md := bc.Metadata()
	b := bc.Bounds()
	fmt.Fprintf(os.Stderr, "format=%s dimensions=%dD size=%dx%d content=%q\n",
		md.CodeKind, md.Dimensions, b.Dx(), b.Dy(), bc.Content())
	if cs, ok := bc.(barcode.BarcodeIntCS); ok {
		fmt.Fprintf(os.Stderr, "checksum=%d\n", cs.CheckSum())
	}
}

func writePNG(bc barcode.Barcode, path string) error {
	if path == "" || path == "-" {
		return png.Encode(os.Stdout, bc)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	defer f.Close()
	if err := png.Encode(f, bc); err != nil {
		f.Close()
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return f.Close()
}

func usage(fs *flag.FlagSet) {
	fmt.Fprintf(os.Stderr, `barcode encodes CLI-supplied data using this module's barcode symbologies
and writes the result as a PNG. It exists so ad-hoc or edge-case barcodes
can be produced without writing a bespoke Go program for each one.

Usage:
  barcode <format> [-options "key=value,..."] [-scale=] <text> [output]

If output is omitted, the PNG is written to stdout.

Formats: %s

Format-specific -options keys:
  aztec       minecc=<percent>                         (default: 33)
              layers=<n>                               (default: 0, auto-select)
  code128     checksum=<bool>                          (default: true)
  code39      checksum=<bool>                          (default: true)
              fullascii=<bool>                         (default: false)
  code93      checksum=<bool>                          (default: true)
              fullascii=<bool>                         (default: false)
  pdf417      level=<0-8>                              (default: 0)
  qr          level=<L|M|Q|H>                          (default: L)
              mode=<Auto|Numeric|AlphaNumeric|Unicode> (default: Auto)
  2of5        interleaved=<bool>                       (default: false)

General -options keys (any format):
  color=<mono|gray16|rgb|rgba> (default: gray16)
  quiet=<bool>                 (default: false; suppress the summary written to stderr)

Examples:
  # QR code at error correction level M, written to a file
  barcode qr -options "level=M" "Hello World" hello.png

  # PDF417 at error correction level 2, written to stdout
  barcode pdf417 -options "level=2" "some data" > out.png

  # scale a QR code up to a fixed pixel size
  barcode qr -scale=300 "Hello World" hello.png

Flags:
`, strings.Join(formats, ", "))
	fs.PrintDefaults()
}
