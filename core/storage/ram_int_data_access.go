package storage

import (
	"encoding/binary"
	"fmt"
	"math"
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
	r.segmentSizeIntsPow = int(math.Log(float64(r.segmentSizeInBytes/4)) / math.Log(2))
	r.indexDivisor = r.segmentSizeInBytes/4 - 1
}

func (r *RAMIntDataAccess) Name() string { return r.name }

func (r *RAMIntDataAccess) Create(bytes int64) DataAccess {
	if len(r.segments) > 0 {
		panic("already created")
	}
	r.EnsureCapacity(max(10*4, bytes))
	return r
}

func (r *RAMIntDataAccess) EnsureCapacity(bytes int64) bool {
	if bytes < 0 {
		panic("new capacity has to be strictly positive")
	}
	cap := r.Capacity()
	newBytes := bytes - cap
	if newBytes <= 0 {
		return false
	}
	segsToCreate := int(newBytes) / r.segmentSizeInBytes
	if int(newBytes)%r.segmentSizeInBytes != 0 {
		segsToCreate++
	}
	for i := 0; i < segsToCreate; i++ {
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
	if _, err := f.Seek(int64(headerOffset), 0); err != nil {
		panic(fmt.Sprintf("problem while loading %s: %v", path, err))
	}
	segCount := int(byteCount) / r.segmentSizeInBytes
	if int(byteCount)%r.segmentSizeInBytes != 0 {
		segCount++
	}
	r.segments = make([][]int32, segCount)
	rawBytes := make([]byte, r.segmentSizeInBytes)
	for s := 0; s < segCount; s++ {
		n, _ := f.Read(rawBytes)
		intCount := n / 4
		area := make([]int32, intCount)
		for j := 0; j < intCount; j++ {
			area[j] = int32(binary.LittleEndian.Uint32(rawBytes[j*4:]))
		}
		r.segments[s] = area
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

	length := r.Capacity()
	if err := r.writeHeader(f, length, r.segmentSizeInBytes); err != nil {
		panic(fmt.Sprintf("couldn't store ints to %s: %v", r.fullName(), err))
	}
	if _, err := f.Seek(int64(headerOffset), 0); err != nil {
		panic(fmt.Sprintf("couldn't store ints to %s: %v", r.fullName(), err))
	}
	for _, area := range r.segments {
		byteArea := make([]byte, len(area)*4)
		for i, v := range area {
			binary.LittleEndian.PutUint32(byteArea[i*4:], uint32(v))
		}
		if _, err := f.Write(byteArea); err != nil {
			panic(fmt.Sprintf("couldn't store ints to %s: %v", r.fullName(), err))
		}
	}
}

func (r *RAMIntDataAccess) SetInt(bytePos int64, value int32) {
	bytePos >>= 2
	bufIdx := int(uint64(bytePos) >> uint(r.segmentSizeIntsPow))
	idx := int(bytePos) & r.indexDivisor
	r.segments[bufIdx][idx] = value
}

func (r *RAMIntDataAccess) GetInt(bytePos int64) int32 {
	bytePos >>= 2
	bufIdx := int(uint64(bytePos) >> uint(r.segmentSizeIntsPow))
	idx := int(bytePos) & r.indexDivisor
	return r.segments[bufIdx][idx]
}

func (r *RAMIntDataAccess) SetShort(bytePos int64, value int16) {
	if bytePos%4 != 0 && bytePos%4 != 2 {
		panic(fmt.Sprintf("bytePos of wrong multiple for RAMInt %d", bytePos))
	}
	tmpIdx := bytePos >> 2
	bufIdx := int(uint64(tmpIdx) >> uint(r.segmentSizeIntsPow))
	idx := int(tmpIdx) & r.indexDivisor
	oldVal := r.segments[bufIdx][idx]
	if tmpIdx*4 == bytePos {
		r.segments[bufIdx][idx] = oldVal&^0x0000FFFF | int32(value)&0x0000FFFF
	} else {
		r.segments[bufIdx][idx] = oldVal&0x0000FFFF | int32(value)<<16
	}
}

func (r *RAMIntDataAccess) GetShort(bytePos int64) int16 {
	if bytePos%4 != 0 && bytePos%4 != 2 {
		panic(fmt.Sprintf("bytePos of wrong multiple for RAMInt %d", bytePos))
	}
	tmpIdx := bytePos >> 2
	bufIdx := int(uint64(tmpIdx) >> uint(r.segmentSizeIntsPow))
	idx := int(tmpIdx) & r.indexDivisor
	if tmpIdx*4 == bytePos {
		return int16(r.segments[bufIdx][idx] & 0x0000FFFF)
	}
	return int16(r.segments[bufIdx][idx] >> 16)
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
