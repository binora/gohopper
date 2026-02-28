package storage

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/bits"
	"os"
	"path/filepath"

	"gohopper/core/util"
)

const (
	segmentSizeMin     = 1 << 7
	segmentSizeDefault = 1 << 20
	headerOffset       = 20*4 + 20
	headerCount        = (headerOffset - 20) / 4
)

// dataAccessBase holds shared state for DataAccess implementations (mirrors AbstractDataAccess).
type dataAccessBase struct {
	name               string
	location           string
	header             [headerCount]int32
	segmentSizeInBytes int
	segmentSizePower   int
	indexDivisor       int
	closed             bool
}

func newDataAccessBase(name, location string, segmentSize int) dataAccessBase {
	if location != "" && location[len(location)-1] != '/' {
		panic("Create DataAccess object via its corresponding Directory!")
	}
	if segmentSize < 0 {
		segmentSize = segmentSizeDefault
	}
	b := dataAccessBase{name: name, location: location}
	b.setSegmentSize(segmentSize)
	return b
}

func (b *dataAccessBase) fullName() string {
	return b.location + b.name
}

func (b *dataAccessBase) setSegmentSize(size int) {
	if size > 0 {
		power := bits.Len(uint(size)) - 1
		b.segmentSizeInBytes = max(1<<power, segmentSizeMin)
	}
	b.segmentSizePower = bits.TrailingZeros(uint(b.segmentSizeInBytes))
	b.indexDivisor = b.segmentSizeInBytes - 1
}

func (b *dataAccessBase) writeHeader(f *os.File, length int64, segmentSize int) error {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	buf := make([]byte, headerOffset)
	binary.BigEndian.PutUint16(buf[0:2], 2)
	buf[2] = 'G'
	buf[3] = 'H'
	binary.BigEndian.PutUint64(buf[4:12], uint64(length))
	binary.BigEndian.PutUint32(buf[12:16], uint32(segmentSize))
	for i := 0; i < headerCount; i++ {
		binary.BigEndian.PutUint32(buf[16+i*4:20+i*4], uint32(b.header[i]))
	}
	_, err := f.Write(buf)
	return err
}

func (b *dataAccessBase) readHeader(f *os.File) (int64, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return -1, err
	}
	info, err := f.Stat()
	if err != nil {
		return -1, err
	}
	if info.Size() == 0 {
		return -1, nil
	}
	buf := make([]byte, headerOffset)
	if _, err := f.Read(buf); err != nil {
		return -1, err
	}
	utfLen := binary.BigEndian.Uint16(buf[0:2])
	marker := string(buf[2 : 2+utfLen])
	if marker != "GH" {
		return -1, fmt.Errorf("not a GraphHopper file %s! Expected 'GH' as file marker but was %s", b.fullName(), marker)
	}
	byteCount := int64(binary.BigEndian.Uint64(buf[4:12]))
	b.setSegmentSize(int(binary.BigEndian.Uint32(buf[12:16])))
	for i := 0; i < headerCount; i++ {
		b.header[i] = int32(binary.BigEndian.Uint32(buf[16+i*4 : 20+i*4]))
	}
	return byteCount, nil
}

// RAMDataAccess is an in-memory byte-based DataAccess with optional disk persistence.
type RAMDataAccess struct {
	dataAccessBase
	segments [][]byte
	store    bool
}

func NewRAMDataAccess(name, location string, store bool, segmentSize int) *RAMDataAccess {
	return &RAMDataAccess{
		dataAccessBase: newDataAccessBase(name, location, segmentSize),
		store:          store,
	}
}

func (r *RAMDataAccess) Name() string { return r.name }

func (r *RAMDataAccess) Create(bytes int64) DataAccess {
	if len(r.segments) > 0 {
		panic("already created")
	}
	r.EnsureCapacity(max(40, bytes))
	return r
}

func (r *RAMDataAccess) EnsureCapacity(bytes int64) bool {
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
		r.segments = append(r.segments, make([]byte, r.segmentSizeInBytes))
	}
	return true
}

func (r *RAMDataAccess) LoadExisting() bool {
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
	r.segments = make([][]byte, n)
	for i := range r.segments {
		seg := make([]byte, r.segmentSizeInBytes)
		nr, err := f.Read(seg)
		if nr <= 0 || err != nil {
			panic(fmt.Sprintf("segment %d is empty? %s", i, r.fullName()))
		}
		r.segments[i] = seg
	}
	return true
}

func (r *RAMDataAccess) Flush() {
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
		panic(fmt.Sprintf("couldn't store bytes to %s: %v", r.fullName(), err))
	}
	defer f.Close()

	if err := r.writeHeader(f, r.Capacity(), r.segmentSizeInBytes); err != nil {
		panic(fmt.Sprintf("couldn't store bytes to %s: %v", r.fullName(), err))
	}
	if _, err := f.Seek(int64(headerOffset), io.SeekStart); err != nil {
		panic(fmt.Sprintf("couldn't store bytes to %s: %v", r.fullName(), err))
	}
	for _, seg := range r.segments {
		if _, err := f.Write(seg); err != nil {
			panic(fmt.Sprintf("couldn't store bytes to %s: %v", r.fullName(), err))
		}
	}
}

func (r *RAMDataAccess) SetInt(bytePos int64, value int32) {
	seg := int(uint64(bytePos) >> uint(r.segmentSizePower))
	idx := int(bytePos) & r.indexDivisor
	if idx+3 >= r.segmentSizeInBytes {
		b1 := r.segments[seg]
		b2 := r.segments[seg+1]
		switch {
		case idx+1 >= r.segmentSizeInBytes:
			util.BitLE.FromUInt3(b2, value>>8, 0)
			b1[idx] = byte(value)
		case idx+2 >= r.segmentSizeInBytes:
			util.BitLE.FromShort(b2, int16(value>>16), 0)
			util.BitLE.FromShort(b1, int16(value), idx)
		default:
			b2[0] = byte(value >> 24)
			util.BitLE.FromUInt3(b1, value, idx)
		}
	} else {
		binary.LittleEndian.PutUint32(r.segments[seg][idx:], uint32(value))
	}
}

func (r *RAMDataAccess) GetInt(bytePos int64) int32 {
	seg := int(uint64(bytePos) >> uint(r.segmentSizePower))
	idx := int(bytePos) & r.indexDivisor
	if idx+3 >= r.segmentSizeInBytes {
		b1 := r.segments[seg]
		b2 := r.segments[seg+1]
		switch {
		case idx+1 >= r.segmentSizeInBytes:
			return int32(b2[2])<<24 | int32(b2[1])<<16 | int32(b2[0])<<8 | int32(b1[idx])
		case idx+2 >= r.segmentSizeInBytes:
			return int32(b2[1])<<24 | int32(b2[0])<<16 | int32(b1[idx+1])<<8 | int32(b1[idx])
		default:
			return int32(b2[0])<<24 | int32(b1[idx+2])<<16 | int32(b1[idx+1])<<8 | int32(b1[idx])
		}
	}
	return int32(binary.LittleEndian.Uint32(r.segments[seg][idx:]))
}

func (r *RAMDataAccess) SetShort(bytePos int64, value int16) {
	seg := int(uint64(bytePos) >> uint(r.segmentSizePower))
	idx := int(bytePos) & r.indexDivisor
	if idx+1 >= r.segmentSizeInBytes {
		r.segments[seg][idx] = byte(value)
		r.segments[seg+1][0] = byte(value >> 8)
	} else {
		binary.LittleEndian.PutUint16(r.segments[seg][idx:], uint16(value))
	}
}

func (r *RAMDataAccess) GetShort(bytePos int64) int16 {
	seg := int(uint64(bytePos) >> uint(r.segmentSizePower))
	idx := int(bytePos) & r.indexDivisor
	if idx+1 >= r.segmentSizeInBytes {
		return int16(r.segments[seg+1][0])<<8 | int16(r.segments[seg][idx])
	}
	return int16(binary.LittleEndian.Uint16(r.segments[seg][idx:]))
}

func (r *RAMDataAccess) SetBytes(bytePos int64, values []byte, length int) {
	seg := int(uint64(bytePos) >> uint(r.segmentSizePower))
	idx := int(bytePos) & r.indexDivisor
	overflow := idx + length - r.segmentSizeInBytes
	if overflow > 0 {
		first := length - overflow
		copy(r.segments[seg][idx:], values[:first])
		copy(r.segments[seg+1], values[first:length])
	} else {
		copy(r.segments[seg][idx:], values[:length])
	}
}

func (r *RAMDataAccess) GetBytes(bytePos int64, values []byte, length int) {
	seg := int(uint64(bytePos) >> uint(r.segmentSizePower))
	idx := int(bytePos) & r.indexDivisor
	overflow := idx + length - r.segmentSizeInBytes
	if overflow > 0 {
		first := length - overflow
		copy(values[:first], r.segments[seg][idx:])
		copy(values[first:length], r.segments[seg+1])
	} else {
		copy(values[:length], r.segments[seg][idx:])
	}
}

func (r *RAMDataAccess) SetByte(bytePos int64, value byte) {
	seg := int(uint64(bytePos) >> uint(r.segmentSizePower))
	idx := int(bytePos) & r.indexDivisor
	r.segments[seg][idx] = value
}

func (r *RAMDataAccess) GetByte(bytePos int64) byte {
	seg := int(uint64(bytePos) >> uint(r.segmentSizePower))
	idx := int(bytePos) & r.indexDivisor
	return r.segments[seg][idx]
}

func (r *RAMDataAccess) SetHeader(bytePos int, value int32) {
	r.header[bytePos>>2] = value
}

func (r *RAMDataAccess) GetHeader(bytePos int) int32 {
	return r.header[bytePos>>2]
}

func (r *RAMDataAccess) Close() {
	r.segments = nil
	r.closed = true
}

func (r *RAMDataAccess) IsClosed() bool { return r.closed }

func (r *RAMDataAccess) Capacity() int64 {
	return int64(len(r.segments)) * int64(r.segmentSizeInBytes)
}

func (r *RAMDataAccess) SegmentSize() int { return r.segmentSizeInBytes }
func (r *RAMDataAccess) Segments() int    { return len(r.segments) }

func (r *RAMDataAccess) Type() DAType {
	if r.store {
		return DATypeRAMStore
	}
	return DATypeRAM
}
