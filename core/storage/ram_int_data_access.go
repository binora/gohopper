package storage

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/bits"
	"os"
	"path/filepath"
)

// RAMIntDataAccess is an in-memory int-based DataAccess optimized for integer access.
// SetBytes/GetBytes/SetByte/GetByte are not supported.
type RAMIntDataAccess struct {
	dataAccessBase
	segments           [][]int32
	store              bool
	segmentSizeIntsPow int
}

func NewRAMIntDataAccess(name, location string, store bool, segmentSize int) *RAMIntDataAccess {
	da := &RAMIntDataAccess{
		dataAccessBase: newDataAccessBase(name, location, segmentSize),
		store:          store,
	}
	da.updateIntsPower()
	return da
}

func (r *RAMIntDataAccess) updateIntsPower() {
	intsPerSeg := r.segmentSizeInBytes / 4
	r.segmentSizeIntsPow = bits.TrailingZeros(uint(intsPerSeg))
	r.indexDivisor = intsPerSeg - 1
}

func (r *RAMIntDataAccess) Name() string { return r.name }

func (r *RAMIntDataAccess) Create(bytes int64) DataAccess {
	if len(r.segments) > 0 {
		panic("already created")
	}
	r.EnsureCapacity(max(40, bytes))
	return r
}

func (r *RAMIntDataAccess) EnsureCapacity(bytes int64) bool {
	if bytes < 0 {
		panic("new capacity has to be strictly positive")
	}
	need := bytes - r.Capacity()
	if need <= 0 {
		return false
	}
	n := int(need) / r.segmentSizeInBytes
	if int(need)%r.segmentSizeInBytes != 0 {
		n++
	}
	for i := 0; i < n; i++ {
		r.segments = append(r.segments, make([]int32, 1<<r.segmentSizeIntsPow))
	}
	return true
}

func (r *RAMIntDataAccess) LoadExisting() bool {
	if len(r.segments) > 0 {
		panic("already initialized")
	}
	if r.closed {
		panic("already closed")
	}
	if !r.store {
		return false
	}
	path := r.fullName()
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		panic(fmt.Sprintf("problem while loading %s: %v", path, err))
	}
	defer f.Close()

	byteCount, err := r.readHeader(f)
	if err != nil {
		panic(fmt.Sprintf("problem while loading %s: %v", path, err))
	}
	r.updateIntsPower()
	byteCount -= int64(headerOffset)
	if byteCount < 0 {
		return false
	}
	if _, err := f.Seek(int64(headerOffset), io.SeekStart); err != nil {
		panic(fmt.Sprintf("problem while loading %s: %v", path, err))
	}
	n := int(byteCount) / r.segmentSizeInBytes
	if int(byteCount)%r.segmentSizeInBytes != 0 {
		n++
	}
	r.segments = make([][]int32, n)
	rawBytes := make([]byte, r.segmentSizeInBytes)
	for i := range r.segments {
		nr, _ := f.Read(rawBytes)
		intCount := nr / 4
		area := make([]int32, intCount)
		for j := 0; j < intCount; j++ {
			area[j] = int32(binary.LittleEndian.Uint32(rawBytes[j*4:]))
		}
		r.segments[i] = area
	}
	return true
}

func (r *RAMIntDataAccess) Flush() {
	if r.closed {
		panic("already closed")
	}
	if !r.store {
		return
	}
	dir := filepath.Dir(r.fullName())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(fmt.Sprintf("couldn't create directory %s: %v", dir, err))
	}
	f, err := os.Create(r.fullName())
	if err != nil {
		panic(fmt.Sprintf("couldn't store ints to %s: %v", r.fullName(), err))
	}
	defer f.Close()

	if err := r.writeHeader(f, r.Capacity(), r.segmentSizeInBytes); err != nil {
		panic(fmt.Sprintf("couldn't store ints to %s: %v", r.fullName(), err))
	}
	if _, err := f.Seek(int64(headerOffset), io.SeekStart); err != nil {
		panic(fmt.Sprintf("couldn't store ints to %s: %v", r.fullName(), err))
	}
	for _, area := range r.segments {
		buf := make([]byte, len(area)*4)
		for i, v := range area {
			binary.LittleEndian.PutUint32(buf[i*4:], uint32(v))
		}
		if _, err := f.Write(buf); err != nil {
			panic(fmt.Sprintf("couldn't store ints to %s: %v", r.fullName(), err))
		}
	}
}

func (r *RAMIntDataAccess) SetInt(bytePos int64, value int32) {
	pos := bytePos >> 2
	seg := int(uint64(pos) >> uint(r.segmentSizeIntsPow))
	idx := int(pos) & r.indexDivisor
	r.segments[seg][idx] = value
}

func (r *RAMIntDataAccess) GetInt(bytePos int64) int32 {
	pos := bytePos >> 2
	seg := int(uint64(pos) >> uint(r.segmentSizeIntsPow))
	idx := int(pos) & r.indexDivisor
	return r.segments[seg][idx]
}

func (r *RAMIntDataAccess) SetShort(bytePos int64, value int16) {
	if bytePos%4 != 0 && bytePos%4 != 2 {
		panic(fmt.Sprintf("bytePos of wrong multiple for RAMInt %d", bytePos))
	}
	pos := bytePos >> 2
	seg := int(uint64(pos) >> uint(r.segmentSizeIntsPow))
	idx := int(pos) & r.indexDivisor
	old := r.segments[seg][idx]
	if pos*4 == bytePos {
		r.segments[seg][idx] = old&^0x0000FFFF | int32(value)&0x0000FFFF
	} else {
		r.segments[seg][idx] = old&0x0000FFFF | int32(value)<<16
	}
}

func (r *RAMIntDataAccess) GetShort(bytePos int64) int16 {
	if bytePos%4 != 0 && bytePos%4 != 2 {
		panic(fmt.Sprintf("bytePos of wrong multiple for RAMInt %d", bytePos))
	}
	pos := bytePos >> 2
	seg := int(uint64(pos) >> uint(r.segmentSizeIntsPow))
	idx := int(pos) & r.indexDivisor
	if pos*4 == bytePos {
		return int16(r.segments[seg][idx] & 0x0000FFFF)
	}
	return int16(r.segments[seg][idx] >> 16)
}

func (r *RAMIntDataAccess) SetBytes(bytePos int64, values []byte, length int) {
	panic(fmt.Sprintf("%s does not support byte based access. Use RAMDataAccess instead", r.name))
}

func (r *RAMIntDataAccess) GetBytes(bytePos int64, values []byte, length int) {
	panic(fmt.Sprintf("%s does not support byte based access. Use RAMDataAccess instead", r.name))
}

func (r *RAMIntDataAccess) SetByte(bytePos int64, value byte) {
	panic(fmt.Sprintf("%s does not support byte based access. Use RAMDataAccess instead", r.name))
}

func (r *RAMIntDataAccess) GetByte(bytePos int64) byte {
	panic(fmt.Sprintf("%s does not support byte based access. Use RAMDataAccess instead", r.name))
}

func (r *RAMIntDataAccess) SetHeader(bytePos int, value int32) {
	r.header[bytePos>>2] = value
}

func (r *RAMIntDataAccess) GetHeader(bytePos int) int32 {
	return r.header[bytePos>>2]
}

func (r *RAMIntDataAccess) Close() {
	r.segments = nil
	r.closed = true
}

func (r *RAMIntDataAccess) IsClosed() bool { return r.closed }

func (r *RAMIntDataAccess) Capacity() int64 {
	return int64(len(r.segments)) * int64(r.segmentSizeInBytes)
}

func (r *RAMIntDataAccess) SegmentSize() int { return r.segmentSizeInBytes }
func (r *RAMIntDataAccess) Segments() int    { return len(r.segments) }

func (r *RAMIntDataAccess) Type() DAType {
	if r.store {
		return DATypeRAMIntStore
	}
	return DATypeRAMInt
}
