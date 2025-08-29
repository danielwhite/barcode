package qr

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
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

func imgStrToBools(str string) []bool {
	res := make([]bool, 0, len(str))
	for _, r := range str {
		switch r {
		case '+':
			res = append(res, true)
		case '.':
			res = append(res, false)
		}
	}
	return res
}

func Test_Encode(t *testing.T) {
	for _, tst := range tests {
		res, err := Encode(tst.Text, tst.ECL, tst.Mode)
		if err != nil {
			t.Error(err)
		}
		qrCode, ok := res.(*qrcode)
		if !ok {
			t.Fail()
		}
		testRes := imgStrToBools(tst.Result)
		if (qrCode.dimension * qrCode.dimension) != len(testRes) {
			t.Fail()
		}
		t.Logf("dim %d", qrCode.dimension)
		for i := 0; i < len(testRes); i++ {
			x := i % qrCode.dimension
			y := i / qrCode.dimension
			if qrCode.Get(x, y) != testRes[i] {
				t.Errorf("Failed at index %d", i)
			}
		}
	}
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
		Characters int
		// Function that randomly generates N characters to be encoded.
		DataFunc func(int) string
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

		name := fmt.Sprintf("%s/%s/%d", tc.Encoding, tc.ErrorCorrectionLevel, tc.Characters)
		b.Run(name, func(b *testing.B) {
			for b.Loop() {
				b.StopTimer()
				text := genNumeric(tc.Characters)
				b.StartTimer()

				_, err := Encode(text, tc.ErrorCorrectionLevel, tc.Encoding)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
