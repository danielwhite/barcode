package utils

import (
	"encoding/binary"
	"math/bits"
)

const (
	// wordSize of a word in the bit list data
	wordSize = bits.UintSize
	// wordBytes is the bytes of a word in the bit list data
	wordBytes = wordSize / 8
	// wordMask is used for bit indexing a word
	wordMask = wordSize - 1
)

// BitList is a list that contains bits
type BitList struct {
	count int
	data  []uint
}

// NewBitList returns a new BitList with the given length
// all bits are initialize with false
func NewBitList(capacity int) *BitList {
	bl := new(BitList)
	bl.count = capacity
	bl.data = make([]uint, wordsRequired(capacity))
	return bl
}

// Len returns the number of contained bits
func (bl *BitList) Len() int {
	return bl.count
}

func (bl *BitList) grow() {
	growBy := len(bl.data)
	if growBy < 128 {
		growBy = 128
	} else if growBy >= 1024 {
		growBy = 1024
	}

	nd := make([]uint, len(bl.data)+growBy)
	copy(nd, bl.data)
	bl.data = nd
}

// AddBit appends the given bits to the end of the list
func (bl *BitList) AddBit(bits ...bool) {
	for _, bit := range bits {
		itmIndex := bl.count / wordSize
		for itmIndex >= len(bl.data) {
			bl.grow()
		}
		bl.SetBit(bl.count, bit)
		bl.count++
	}
}

// SetBit sets the bit at the given index to the given value
func (bl *BitList) SetBit(index int, value bool) {
	itmIndex := index / wordSize
	itmBitShift := wordMask - (index % wordSize)
	if value {
		bl.data[itmIndex] = bl.data[itmIndex] | 1<<uint(itmBitShift)
	} else {
		bl.data[itmIndex] = bl.data[itmIndex] & ^(1 << uint(itmBitShift))
	}
}

// GetBit returns the bit at the given index
func (bl *BitList) GetBit(index int) bool {
	itmIndex := index / wordSize
	itmBitShift := wordMask - (index % wordSize)
	return (bl.data[itmIndex] & (1 << itmBitShift)) != 0
}

// AddByte appends all 8 bits of the given byte to the end of the list
func (bl *BitList) AddByte(b byte) {
	for i := 7; i >= 0; i-- {
		bl.AddBit(((b >> uint(i)) & 1) == 1)
	}
}

// AddBits appends the last (LSB) 'count' bits of 'b' the the end of the list
func (bl *BitList) AddBits(b int, count byte) {
	for i := int(count) - 1; i >= 0; i-- {
		bl.AddBit(((b >> uint(i)) & 1) == 1)
	}
}

// GetBytes returns all bits of the BitList as a []byte.
//
// Bits are ordered in each byte using MSb 0 bit numbering where the
// MSB is first and LSB is last.
func (bl *BitList) GetBytes() []byte {
	result := make([]byte, 0, bytesRequired(bl.count))
	for i := range bl.data {
		if wordSize == 32 {
			result = binary.BigEndian.AppendUint32(result, uint32(bl.data[i]))
		} else {
			result = binary.BigEndian.AppendUint64(result, uint64(bl.data[i]))
		}
	}
	return result[:bytesRequired(bl.count)]
}

// IterateBytes iterates through all bytes contained in the BitList
func (bl *BitList) IterateBytes() <-chan byte {
	res := make(chan byte)

	go func() {
		c := bl.count
		shift := (wordBytes - 1) * 8
		i := 0
		for c > 0 {
			res <- byte((bl.data[i] >> uint(shift)) & 0xFF)
			shift -= 8
			if shift < 0 {
				shift = (wordBytes - 1) * 8
				i++
			}
			c -= 8
		}
		close(res)
	}()

	return res
}

// Count returns the number of one bits ("population count") in the BitList.
func (bl *BitList) Count() int {
	n := 0
	for _, x := range bl.data {
		n += bits.OnesCount(x)
	}
	return n
}

// wordsRequired returns the minimum number of words required to store n bits.
func wordsRequired(n int) int {
	return (n + wordMask) / wordSize
}

// bytesRequired returns the minimum number of words required to store n bits.
func bytesRequired(n int) int {
	return (n + 7) / 8
}
