package qr

import (
	"fmt"
	"strings"
	"testing"

	"github.com/boombuler/barcode"
)

type test struct {
	Text   string
	Mode   Encoding
	ECL    ErrorCorrectionLevel
	Result string
}

var tests = []test{
	{
		Text: "hello world",
		Mode: Unicode,
		ECL:  H,
		Result: `
+++++++.+.+.+...+.+++++++
+.....+.++...+++..+.....+
+.+++.+.+.+.++.++.+.+++.+
+.+++.+....++.++..+.+++.+
+.+++.+..+...++.+.+.+++.+
+.....+.+..+..+++.+.....+
+++++++.+.+.+.+.+.+++++++
........++..+..+.........
..+++.+.+++.+.++++++..+++
+++..+..+...++.+...+..+..
+...+.++++....++.+..++.++
++.+.+.++...+...+.+....++
..+..+++.+.+++++.++++++++
+.+++...+..++..++..+..+..
+.....+..+.+.....+++++.++
+.+++.....+...+.+.+++...+
+.+..+++...++.+.+++++++..
........+....++.+...+.+..
+++++++......++++.+.+.+++
+.....+....+...++...++.+.
+.+++.+.+.+...+++++++++..
+.+++.+.++...++...+.++..+
+.+++.+.++.+++++..++.+..+
+.....+..+++..++.+.++...+
+++++++....+..+.+..+..+++`,
	},
}

func Test_GetUnknownEncoder(t *testing.T) {
	if unknownEncoding.getEncoder() != nil {
		t.Fail()
	}
}

func Test_EncodingStringer(t *testing.T) {
	tests := map[Encoding]string{
		Auto:            "Auto",
		Numeric:         "Numeric",
		AlphaNumeric:    "AlphaNumeric",
		Unicode:         "Unicode",
		unknownEncoding: "",
	}

	for enc, str := range tests {
		if enc.String() != str {
			t.Fail()
		}
	}
}

func Test_InvalidEncoding(t *testing.T) {
	_, err := Encode("hello world", H, Numeric)
	if err == nil {
		t.Fail()
	}
}

func Test_Encode(t *testing.T) {
	for _, tst := range tests {
		res, err := Encode(tst.Text, tst.ECL, tst.Mode)
		if err != nil {
			t.Error(err)
		}
		checkBarcode(t, res, strings.TrimSpace(tst.Result))
	}
}

// checkBarcode fails if the ascii representation of the barcode does
// not match the provided string.
func checkBarcode(tb testing.TB, bc barcode.Barcode, want string) {
	tb.Helper()
	if got := asciiBarcode(bc); want != got {
		tb.Errorf("incorrect barcode: (-got +want)\n%s", linediff(want, got))
	}
}

// asciiBarcode returns an ASCII art encoding of a barcode.
//
// Encoding assumes a black and white image with one pixel per QR module.
//
// Dark and light pixels are encoded as '+' and '.' respectively.
func asciiBarcode(bc barcode.Barcode) string {
	const dark = '+'
	const light = '.'

	// Handle optional colour scheme through interface promotion.
	foreground := barcode.ColorScheme16.Foreground
	if bc, ok := bc.(barcode.BarcodeColor); ok {
		foreground = bc.ColorScheme().Foreground
	}

	var sb strings.Builder
	for x := 0; x < bc.Bounds().Max.X; x++ {
		if x > 0 {
			sb.WriteByte('\n')
		}
		for y := 0; y < bc.Bounds().Max.Y; y++ {
			if bc.At(y, x) == foreground {
				sb.WriteByte(dark)
			} else {
				sb.WriteByte(light)
			}
		}

	}
	return sb.String()
}

func linediff(a, b string) string {
	const (
		want     = "- "
		got      = "+ "
		identity = "  "
	)

	sa := strings.Split(a, "\n")
	sb := strings.Split(b, "\n")

	var diff strings.Builder

	for len(sa) > 0 && len(sb) > 0 {
		if sa[0] == sb[0] {
			fmt.Fprintln(&diff, identity, sa[0])
		} else {
			fmt.Fprintln(&diff, want, sa[0])
			fmt.Fprintln(&diff, got, sb[0])
		}
		sa, sb = sa[1:], sb[1:]
	}
	for _, line := range sa {
		fmt.Fprintln(&diff, want, line)
	}
	for _, line := range sb {
		fmt.Fprintln(&diff, got, line)
	}

	return diff.String()
}
