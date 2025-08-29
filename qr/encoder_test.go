package qr

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/boombuler/barcode"
)

type test struct {
	Name   string
	Text   string
	Mode   Encoding
	ECL    ErrorCorrectionLevel
	Result string
}

var tests = []test{
	// Example from ISO/IEC 18004:2015(E), Annex I.
	//
	// The standard has mask 2 as the correctly result, but the
	// calculation of penalties appear to be correct and choose
	// mask 0 instead.
	{
		Name: "Numeric: Version 1",
		Text: "01234567",
		Mode: Numeric,
		ECL:  M,
		Result: `
+++++++...+++.+++++++
+.....+.+++...+.....+
+.+++.+..++...+.+++.+
+.+++.+..+.++.+.+++.+
+.+++.+.++.++.+.+++.+
+.....+....+..+.....+
+++++++.+.+.+.+++++++
.....................
+.+.+.+...+.+...+..+.
++.+....+.++.+.+...+.
...++.+++.++.+++.+++.
++..++.+.+.+++.++..+.
..+..+++.+++.+++....+
........+.+...+....+.
+++++++.....+...+...+
+.....+...+...+..+.++
+.+++.+.+++.+.+.+++.+
+.+++.+..+.+.+.+.+++.
+.+++.+.++.+.+++..+.+
+.....+....+++.+++...
+++++++.+..+.+++..+.+`,
	},
	{
		Name: "Tiny Text",
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
		t.Run(tst.Name, func(t *testing.T) {
			res, err := Encode(tst.Text, tst.ECL, tst.Mode)
			if err != nil {
				t.Error(err)
			}
			checkBarcode(t, res, strings.TrimSpace(tst.Result))
		})
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

func BenchmarkEncode(b *testing.B) {
	source := new(rand.ChaCha8)
	rand := rand.New(source)

	genNumeric := func(n int) string {
		var sb strings.Builder
		for i := 0; i < n; i++ {
			sb.WriteByte(byte('0' + rand.IntN(10)))
		}
		return sb.String()
	}
	genAlphaNumeric := func(n int) string {
		const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ $%*+-./:"

		var sb strings.Builder
		for i := 0; i < n; i++ {
			sb.WriteByte(charset[rand.IntN(len(charset))])
		}
		return sb.String()
	}
	genBytes := func(n int) string {
		b := make([]byte, n)
		n, err := source.Read(b)
		if n != len(b) || err != nil {
			panic("read error")
		}
		return string(b)
	}

	benchmarks := []struct {
		Encoding Encoding
		ErrorCorrectionLevel
		// Number of characters to be encoded in the benchmark.
		Size int
		// Function that randomly generates N characters to be encoded.
		GenerateFunc func(int) string
	}{
		// Approximate the size of relatively small URLs as a common QR use-case.
		{Numeric, L, 100, genNumeric},
		{AlphaNumeric, L, 100, genAlphaNumeric},
		{Unicode, L, 100, genBytes},

		// Maximum size of encodings with minimal error correction.
		{Numeric, L, 7089, genNumeric},
		{AlphaNumeric, L, 4296, genAlphaNumeric},
		{Unicode, L, 2953, genBytes},
		// Maximum size of encodings with maximal error correction.
		{Numeric, H, 1852, genNumeric},
		{AlphaNumeric, H, 1273, genAlphaNumeric},
		{Unicode, H, 784, genBytes},
	}
	for _, tc := range benchmarks {
		// Data for benchmark is random, but consistent across runs.
		source.Seed([32]byte{0xde, 0xad, 0xbe, 0xef})

		name := fmt.Sprintf("mode=%s/ecl=%s/size=%d", tc.Encoding, tc.ErrorCorrectionLevel, tc.Size)
		b.Run(name, func(b *testing.B) {
			for b.Loop() {
				b.StopTimer()
				text := genNumeric(tc.Size)
				b.StartTimer()

				_, err := Encode(text, tc.ErrorCorrectionLevel, tc.Encoding)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
