package qr

import (
	"image/color"
	"testing"
)

func Test_NewQRCode(t *testing.T) {
	bc := newBarcode(2)
	if bc == nil {
		t.Fail()
		return
	}
	if bc.data.Len() != 4 {
		t.Fail()
	}
	if bc.dimension != 2 {
		t.Fail()
	}
}

func Test_QRBasics(t *testing.T) {
	qr := newBarcode(10)
	if qr.ColorModel() != color.Gray16Model {
		t.Fail()
	}
	code, _ := Encode("test", L, Unicode)
	if code.Content() != "test" {
		t.Fail()
	}
	if code.Metadata().Dimensions != 2 {
		t.Fail()
	}
	bounds := code.Bounds()
	if bounds.Min.X != 0 || bounds.Min.Y != 0 || bounds.Max.X != 21 || bounds.Max.Y != 21 {
		t.Fail()
	}
	if code.At(0, 0) != color.Black || code.At(0, 7) != color.White {
		t.Fail()
	}
	qr = code.(*qrcode)
	if !qr.Get(0, 0) || qr.Get(0, 7) {
		t.Fail()
	}
	sum := qr.calcPenaltyRule1() + qr.calcPenaltyRule2() + qr.calcPenaltyRule3() + qr.calcPenaltyRule4()
	if qr.calcPenalty() != sum {
		t.Fail()
	}
}

func Test_Penalty1(t *testing.T) {
	qr := newBarcode(7)
	if qr.calcPenaltyRule1() != 70 {
		t.Fail()
	}
	qr.Set(0, 0, true)
	if qr.calcPenaltyRule1() != 68 {
		t.Fail()
	}
	qr.Set(0, 6, true)
	if qr.calcPenaltyRule1() != 66 {
		t.Fail()
	}
}

func Test_Penalty2(t *testing.T) {
	qr := newBarcode(3)
	if qr.calcPenaltyRule2() != 12 {
		t.Fail()
	}
	qr.Set(0, 0, true)
	qr.Set(1, 1, true)
	qr.Set(2, 0, true)
	if qr.calcPenaltyRule2() != 0 {
		t.Fail()
	}
	qr.Set(1, 1, false)
	if qr.calcPenaltyRule2() != 6 {
		t.Fail()
	}
}

func Test_Penalty3(t *testing.T) {
	runTest := func(content string, result uint) {
		code, _ := Encode(content, L, AlphaNumeric)
		qr := code.(*qrcode)
		if qr.calcPenaltyRule3() != result {
			t.Errorf("Failed Penalty Rule 3 for content %q got %d but expected %d", content, qr.calcPenaltyRule3(), result)
		}
	}
	runTest("A", 80)
	runTest("FOO", 40)
	runTest("0815", 0)
}

func Test_Penalty3_v2(t *testing.T) {
	pattern1 := []bool{true, false, true, true, true, false, true, false, false, false, false}
	pattern2 := []bool{false, false, false, false, true, false, true, true, true, false, true}

	setPatternY := func(qr *qrcode, x, y int, pattern []bool) {
		for i, bit := range pattern {
			qr.Set(x, y+i, bit)
		}
	}
	setPatternX := func(qr *qrcode, x, y int, pattern []bool) {
		for i, bit := range pattern {
			qr.Set(x+i, y, bit)
		}
	}
	checkPenalty := func(t *testing.T, qr *qrcode, want uint) {
		t.Helper()
		if got := qr.calcPenaltyRule3(); got != want {
			t.Errorf("calcPenaltyRule3() = %d, want %d", got, 40)
			t.Logf("qr(dim=%d): %b", qr.dimension, qr.data.GetBytes())
		}
	}

	vi := &versionInfo{Version: 1}

	// Horizontal pattern in top-left corner.
	qr := newBarcode(vi.modulWidth())
	setPatternX(qr, 0, 0, pattern1)
	checkPenalty(t, qr, 40)

	// Vertical pattern in top-left corner.
	qr = newBarcode(vi.modulWidth())
	setPatternY(qr, 0, 0, pattern1)
	checkPenalty(t, qr, 40)

	// Horizontal pattern in bottom-right corner.
	qr = newBarcode(vi.modulWidth())
	setPatternX(qr, vi.modulWidth()-len(pattern2), vi.modulWidth()-1, pattern2)
	checkPenalty(t, qr, 40)

	// Vertical pattern in bottom-right corner.
	qr = newBarcode(vi.modulWidth())
	setPatternY(qr, vi.modulWidth()-1, vi.modulWidth()-len(pattern2), pattern2)
	checkPenalty(t, qr, 40)

	// Matches of both patterns on the horizontal and vertical axes.
	qr = newBarcode(vi.modulWidth())
	setPatternX(qr, 5, 5, pattern1)
	setPatternY(qr, 14, 8, pattern1)
	checkPenalty(t, qr, 160)
}

func Test_Penalty4(t *testing.T) {
	qr := newBarcode(3)
	if qr.calcPenaltyRule4() != 100 {
		t.Fail()
	}
	qr.Set(0, 0, true)
	if qr.calcPenaltyRule4() != 70 {
		t.Fail()
	}
	qr.Set(0, 1, true)
	if qr.calcPenaltyRule4() != 50 {
		t.Fail()
	}
	qr.Set(0, 2, true)
	if qr.calcPenaltyRule4() != 30 {
		t.Fail()
	}
	qr.Set(1, 0, true)
	if qr.calcPenaltyRule4() != 10 {
		t.Fail()
	}
	qr.Set(1, 1, true)
	if qr.calcPenaltyRule4() != 10 {
		t.Fail()
	}
	qr = newBarcode(2)
	qr.Set(0, 0, true)
	qr.Set(1, 0, true)
	if qr.calcPenaltyRule4() != 0 {
		t.Fail()
	}
}
