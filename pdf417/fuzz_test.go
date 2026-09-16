package pdf417

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boombuler/barcode/utils"
)

// skipZXing opts out of the ZXingReader round-trip check; otherwise
// FuzzEncode requires it and fails fast if it's not on PATH.
var skipZXing = flag.Bool("pdf417.skipzxing", false, "skip the ZXingReader PDF417 round-trip decode check in FuzzEncode")

// maxFuzzInputLen keeps the fuzzer from wasting time on inputs so large
// they're trivially rejected by calcDimensions.
const maxFuzzInputLen = 4096

// The constants and formulas below come from the USS PDF417 symbology
// spec (AIM, "PDF417: Specification for Barcode Symbology") rather than
// this package's own dimensions.go/encoder.go, so a bug in those files
// can't also corrupt the check meant to catch it.

// Sec. 2.2.3: 3 to 90 rows, 1 to 30 data columns per row.
const (
	specMinRows = 3
	specMaxRows = 90
	specMinCols = 1
	specMaxCols = 30
)

// Sec. 2.1: data region codewords (excludes start/stop/row indicators)
// cannot exceed 928.
const specMaxDataRegionCodewords = 928

// Table 6: error correction codewords per level, 0 through 8.
var specECCodewordCount = [9]int{2, 4, 8, 16, 32, 64, 128, 256, 512}

var zxingReaderPath, _ = exec.LookPath("ZXingReader")

func FuzzEncode(f *testing.F) {
	if !*skipZXing && zxingReaderPath == "" {
		f.Fatalf("ZXingReader not found on PATH: install zxing-cpp (https://github.com/zxing-cpp/zxing-cpp) " +
			"so FuzzEncode can round-trip decode its output, or pass -pdf417.skipzxing to run without it")
	}

	// Seed with the literal cases already covered by the table-driven tests,
	// plus values that sit on known edge conditions in the encoder.
	seeds := []struct {
		data  string
		level byte
	}{
		{"", 0},
		{"0", 0},
		{"01234", 2},
		{"Super !", 4},
		{"ABC123", 0},
		{"123ABC", 8},
		{"alcool", 1},
		{"alcoolique", 1},
		// non-ASCII / byte-mode input
		{"héllo wörld \x00\x01\xff", 3},
		// long run of a single character, to stress run-length/repeat handling
		{strings.Repeat("A", 500), 5},
		{strings.Repeat("1", 500), 5},
		// boundary security levels
		{"test", 8},
		{"test", 9},   // invalid: should error, not panic
		{"test", 255}, // invalid: should error, not panic
	}
	for _, s := range seeds {
		f.Add(s.data, s.level)
	}

	f.Fuzz(func(t *testing.T, data string, securityLevel byte) {
		if len(data) > maxFuzzInputLen {
			t.Skip()
		}

		bc, err := Encode(data, securityLevel)

		if securityLevel >= 9 {
			if err == nil {
				t.Fatalf("Encode(%q, %d): expected error for invalid security level, got none", data, securityLevel)
			}
			return
		}

		if err != nil {
			return // oversized/unsupported input; not a bug
		}

		pb, ok := bc.(*pdfBarcode)
		if !ok {
			t.Fatalf("Encode(%q, %d): unexpected barcode implementation %T", data, securityLevel, bc)
		}

		meta := pb.Metadata()
		if meta.CodeKind != "PDF417" || meta.Dimensions != 2 {
			t.Fatalf("Encode(%q, %d): unexpected metadata %+v", data, securityLevel, meta)
		}

		width := pb.width
		totalBits := pb.code.Len()

		if width <= 0 || totalBits <= 0 {
			t.Fatalf("Encode(%q, %d): non-positive width=%d totalBits=%d", data, securityLevel, width, totalBits)
		}

		// Sec. 2.5: W = (17C + 69)X + 2Q. This renderer has no quiet zone
		// and one pixel per module (Q=0, X=1), leaving W = 17C + 69.
		if (width-69)%17 != 0 {
			t.Fatalf("Encode(%q, %d): width %d is not of the form 17*C+69 required by spec sec. 2.5", data, securityLevel, width)
		}
		columns := (width - 69) / 17
		if columns < specMinCols || columns > specMaxCols {
			t.Fatalf("Encode(%q, %d): columns %d outside the spec's sec. 2.2.3 range [%d,%d]", data, securityLevel, columns, specMinCols, specMaxCols)
		}

		// Every row renders to exactly `width` bits.
		if totalBits%width != 0 {
			t.Fatalf("Encode(%q, %d): total bits %d not a multiple of width %d", data, securityLevel, totalBits, width)
		}
		rows := totalBits / width
		if rows < specMinRows || rows > specMaxRows {
			t.Fatalf("Encode(%q, %d): rows %d outside the spec's sec. 2.2.3 range [%d,%d]", data, securityLevel, rows, specMinRows, specMaxRows)
		}

		// Sec. 2.1: data-region codeword count (excluding start/stop/row
		// indicators) cannot exceed 928.
		if dataRegionCodewords := rows * columns; dataRegionCodewords > specMaxDataRegionCodewords {
			t.Fatalf("Encode(%q, %d): data region has %d codewords, exceeding the spec sec. 2.1 limit of %d",
				data, securityLevel, dataRegionCodewords, specMaxDataRegionCodewords)
		}

		// Sec. 2.2.3 / Figure 3: each row's left/right row-indicator
		// codewords must match the spec's L_i/R_i formulas, checked here
		// by decoding the rendered bits directly rather than calling
		// getLeftCodeWord/getRightCodeWord (the functions under test).
		y := (rows - 1) / 3
		z := int(securityLevel)*3 + (rows-1)%3
		v := columns - 1
		for row := 0; row < rows; row++ {
			rowStart := row * width
			leftPattern := readBits(pb.code, rowStart+17, 17)
			rightPattern := readBits(pb.code, rowStart+34+columns*17, 17)

			cluster := row % 3
			leftVal, leftOK := codewordValuesByCluster[cluster][leftPattern]
			rightVal, rightOK := codewordValuesByCluster[cluster][rightPattern]
			if !leftOK || !rightOK {
				t.Fatalf("Encode(%q, %d): row %d row-indicator bit pattern isn't a valid cluster-%d symbol character (left=%#x ok=%v, right=%#x ok=%v)",
					data, securityLevel, row, cluster, leftPattern, leftOK, rightPattern, rightOK)
			}

			xi := row / 3
			var wantLeft, wantRight int
			switch cluster {
			case 0:
				wantLeft, wantRight = 30*xi+y, 30*xi+v
			case 1:
				wantLeft, wantRight = 30*xi+z, 30*xi+y
			case 2:
				wantLeft, wantRight = 30*xi+v, 30*xi+z
			}
			if leftVal != wantLeft || rightVal != wantRight {
				t.Fatalf("Encode(%q, %d): row %d row indicators (L=%d, R=%d) don't match spec sec. 2.2.3 formula (want L=%d, R=%d) for rows=%d cols=%d level=%d",
					data, securityLevel, row, leftVal, rightVal, wantLeft, wantRight, rows, columns, securityLevel)
			}
		}

		// Exercises GetBit at the edges; an off-by-one here panics instead
		// of silently returning a wrong pixel.
		bounds := pb.Bounds()
		if bounds.Dx() != width {
			t.Fatalf("Encode(%q, %d): Bounds().Dx()=%d, want %d", data, securityLevel, bounds.Dx(), width)
		}
		for _, pt := range []struct{ x, y int }{
			{bounds.Min.X, bounds.Min.Y},
			{bounds.Max.X - 1, bounds.Min.Y},
			{bounds.Min.X, bounds.Max.Y - 1},
			{bounds.Max.X - 1, bounds.Max.Y - 1},
		} {
			_ = pb.At(pt.x, pt.y)
		}

		if decoded, ok := decodeWithZXing(t, pb); ok {
			if string(decoded) != data {
				t.Fatalf("Encode(%q, %d): round-trip mismatch, ZXingReader decoded %q (% x)",
					data, securityLevel, decoded, decoded)
			}
		}
	})
}

// TestErrorCorrectionWordCountMatchesSpec checks the EC codeword formula
// against Table 6 directly, since FuzzEncode's checks only exercise the
// formula, not the table it's meant to reproduce.
func TestErrorCorrectionWordCountMatchesSpec(t *testing.T) {
	for level, want := range specECCodewordCount {
		if got := securitylevel(level).ErrorCorrectionWordCount(); got != want {
			t.Errorf("securitylevel(%d).ErrorCorrectionWordCount() = %d, want %d per spec Table 6", level, got, want)
		}
	}
}

// readBits reads `count` bits starting at `offset` out of bl as an
// unsigned integer, MSB first (matching BitList.AddBits).
func readBits(bl *utils.BitList, offset, count int) int {
	v := 0
	for i := 0; i < count; i++ {
		v <<= 1
		if bl.GetBit(offset + i) {
			v |= 1
		}
	}
	return v
}

// codewordValuesByCluster inverts the codewords table (Sec. 2.2.2 /
// Appendix H) so a rendered bit pattern can be mapped back to the value
// it represents, the way a decoder would.
var codewordValuesByCluster = buildCodewordValuesByCluster()

func buildCodewordValuesByCluster() []map[int]int {
	byCluster := make([]map[int]int, len(codewords))
	for cluster, patterns := range codewords {
		m := make(map[int]int, len(patterns))
		for value, pattern := range patterns {
			m[pattern] = value
		}
		byCluster[cluster] = m
	}
	return byCluster
}

// zxingResult holds the ZXingReader -json fields a round-trip check needs.
type zxingResult struct {
	Format string `json:"Format"`
	Error  string `json:"Error"`
	Bytes  string `json:"Bytes"`
}

// decodeWithZXing renders pb to a PNG and decodes it with ZXingReader.
// ok is false when no verdict could be reached: -pdf417.skipzxing was
// passed, or the symbol wasn't detected at all (very small barcodes can
// be below ZXing's detection floor even when correctly encoded) - either
// way the caller should skip, not fail.
func decodeWithZXing(t *testing.T, pb *pdfBarcode) (decoded []byte, ok bool) {
	if *skipZXing {
		return nil, false
	}

	pngPath := filepath.Join(t.TempDir(), "code.png")
	f, err := os.Create(pngPath)
	if err != nil {
		t.Fatalf("creating temp png: %v", err)
	}
	if err := png.Encode(f, pb); err != nil {
		f.Close()
		t.Fatalf("encoding png: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("closing temp png: %v", err)
	}

	out, err := exec.Command(zxingReaderPath, "-formats", "pdf417", "-single", "-errors", "-json", pngPath).Output()
	if err != nil {
		// A non-zero exit with -errors set can still mean "detected, but
		// with a problem" (see the Error field below); only bail out here
		// if ZXingReader itself couldn't run.
		if _, isExit := err.(*exec.ExitError); !isExit {
			t.Fatalf("running ZXingReader: %v", err)
		}
	}
	if len(strings.TrimSpace(string(out))) == 0 {
		return nil, false // not detected at all
	}

	var res zxingResult
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("parsing ZXingReader output %q: %v", out, err)
	}

	if res.Error != "" {
		t.Fatalf("ZXingReader detected the PDF417 symbol but failed to decode it (%s) for data=%q", res.Error, pb.data)
	}

	decoded, err = hex.DecodeString(strings.ReplaceAll(res.Bytes, " ", ""))
	if err != nil {
		t.Fatalf("parsing ZXingReader Bytes field %q: %v", res.Bytes, err)
	}
	return decoded, true
}
