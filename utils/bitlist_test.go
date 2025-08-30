package utils

import (
	"bytes"
	"math/rand/v2"
	"testing"
)

func TestBitList_AddBit(t *testing.T) {
	bits := [...]bool{
		0: true,
		1: true,
		3: true,
		7: true,
		8: true,
	}
	bl := NewBitList(0)
	bl.AddBit(bits[:]...)

	for i, want := range bits {
		if got := bl.GetBit(i); want != got {
			t.Errorf("GetBit(%d) = %t, want %t", i, got, want)
		}
	}
}

func TestBitList_SetBit(t *testing.T) {
	bl := NewBitList(wordSize * 10)

	// Randomly flip bits, ensuring that the expected bit is toggled.
	for n := 0; n < bl.Len()*10; n++ {
		i := rand.IntN(bl.Len())
		want := !bl.GetBit(i)
		bl.SetBit(i, want)
		if got := bl.GetBit(i); want != got {
			t.Errorf("GetBit(%d) = %t, want %t", i, got, want)
		}
	}
}

func TestBitList_GetBytes(t *testing.T) {
	want := []byte{1, 3, 5, 9, 13}

	bl := NewBitList(0)
	for _, x := range want {
		bl.AddByte(x)
	}

	if got := bl.GetBytes(); !bytes.Equal(want, got) {
		t.Errorf("GetBytes() = %b, want %b", got, want)
	}
}

func TestBitList_GetWords(t *testing.T) {
	want := []byte{1, 3, 5, 9, 13, 0, 0, 0}

	bl := NewBitList(0)
	for _, x := range want {
		bl.AddByte(x)
	}

	var got []byte
	for _, x := range bl.GetWords() {
		for s := wordSize - 8; s >= 0; s -= 8 {
			got = append(got, byte(x>>s))
		}
	}
	if !bytes.Equal(want, got) {
		t.Errorf("GetWords() = %b, want %b", got, want)
	}
}

func TestBitList_IterateBytes(t *testing.T) {
	data := []byte{1, 3, 5, 9, 13}
	bl := NewBitList(0)
	for _, x := range data {
		bl.AddByte(x)
	}

	var got []byte
	for x := range bl.IterateBytes() {
		got = append(got, x)
	}
	if want := bl.GetBytes(); !bytes.Equal(want, got) {
		t.Errorf("GetBytes() = %b\nwant: %b", got, want)
	}
}

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
