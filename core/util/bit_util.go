package util

import (
	"encoding/binary"
	"math"
	"math/bits"
)

// BitLE provides little-endian byte conversion utilities (mirrors GraphHopper's BitUtil.LITTLE).
var BitLE = bitUtil{}

type bitUtil struct{}

func (bitUtil) ToShort(b []byte, offset int) int16 {
	return int16(binary.LittleEndian.Uint16(b[offset:]))
}

func (bitUtil) FromShort(b []byte, value int16, offset int) {
	binary.LittleEndian.PutUint16(b[offset:], uint16(value))
}

func (bitUtil) ToInt(b []byte, offset int) int32 {
	return int32(binary.LittleEndian.Uint32(b[offset:]))
}

func (bitUtil) FromInt(b []byte, value int32, offset int) {
	binary.LittleEndian.PutUint32(b[offset:], uint32(value))
}

func (bitUtil) ToUInt3(b []byte, offset int) int32 {
	return int32(b[offset+2])<<16 | int32(b[offset+1])<<8 | int32(b[offset])
}

func (bitUtil) FromUInt3(b []byte, value int32, offset int) {
	b[offset+2] = byte(value >> 16)
	b[offset+1] = byte(value >> 8)
	b[offset] = byte(value)
}

func (bitUtil) ToLong(b []byte, offset int) int64 {
	return int64(binary.LittleEndian.Uint64(b[offset:]))
}

func (bitUtil) ToLongFromInts(low, high int32) int64 {
	return int64(high)<<32 | int64(uint32(low))
}

func (bitUtil) FromLong(b []byte, value int64, offset int) {
	binary.LittleEndian.PutUint64(b[offset:], uint64(value))
}

func (bitUtil) ToFloat(b []byte, offset int) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b[offset:]))
}

func (bitUtil) FromFloat(b []byte, value float32, offset int) {
	binary.LittleEndian.PutUint32(b[offset:], math.Float32bits(value))
}

func (bitUtil) ToDouble(b []byte, offset int) float64 {
	return math.Float64frombits(binary.LittleEndian.Uint64(b[offset:]))
}

func (bitUtil) FromDouble(b []byte, value float64, offset int) {
	binary.LittleEndian.PutUint64(b[offset:], math.Float64bits(value))
}

func (bitUtil) GetIntLow(v int64) int32 {
	return int32(v)
}

func (bitUtil) GetIntHigh(v int64) int32 {
	return int32(v >> 32)
}

// CountBitValue returns the number of bits needed to represent n.
func CountBitValue(n int) int {
	if n < 0 {
		panic("CountBitValue: value cannot be negative")
	}
	return bits.Len(uint(n))
}

// ToSignedInt truncates an int64 to int32.
func ToSignedInt(x int64) int32 {
	return int32(x)
}

func (bitUtil) ToBitString(value int64, nBits int) string {
	buf := make([]byte, nBits)
	uv := uint64(value)
	mask := uint64(1) << 63
	for i := range buf {
		if uv&mask == 0 {
			buf[i] = '0'
		} else {
			buf[i] = '1'
		}
		uv <<= 1
	}
	return string(buf)
}

func (bitUtil) ToLastBitString(value int64, nBits int) string {
	buf := make([]byte, nBits)
	uv := uint64(value)
	mask := uint64(1) << uint(nBits-1)
	for i := range buf {
		if uv&mask == 0 {
			buf[i] = '0'
		} else {
			buf[i] = '1'
		}
		uv <<= 1
	}
	return string(buf)
}

func (bitUtil) ToBitStringBytes(data []byte) string {
	buf := make([]byte, len(data)*8)
	idx := 0
	for i := len(data) - 1; i >= 0; i-- {
		b := data[i]
		for bit := 7; bit >= 0; bit-- {
			if b&(1<<uint(bit)) == 0 {
				buf[idx] = '0'
			} else {
				buf[idx] = '1'
			}
			idx++
		}
	}
	return string(buf)
}

func (bitUtil) FromBitString(str string) []byte {
	n := len(str)
	bLen := (n + 7) / 8
	out := make([]byte, bLen)
	ci := 0
	for b := bLen - 1; b >= 0; b-- {
		var v byte
		for i := 0; i < 8; i++ {
			v <<= 1
			if ci < n && str[ci] != '0' {
				v |= 1
			}
			ci++
		}
		out[b] = v
	}
	return out
}
