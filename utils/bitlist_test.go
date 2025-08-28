package utils

import (
	"math/rand/v2"
	"testing"
)

func TestBitList_Count(t *testing.T) {
	// Random bit list with an uneven split of 1s and 0s that span
	// 2.5 words of backing storage.
	bl := NewBitList(0)
	for i := 0; i < 80; i++ {
		b := (rand.Int() % 3) == 0
		bl.AddBit(b)
	}

	// Naively count the number of bits.
	var want int
	for i := 0; i < bl.Len(); i++ {
		if bl.GetBit(i) {
			want++
		}
	}

	// Ensure count matches the naive count.
	if got := bl.Count(); want != got {
		t.Errorf("Count() = %d, want %d", got, want)
	}
}
