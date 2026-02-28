package storage

// DataAccess provides low-level byte-based storage with segmented memory layout.
//
// Life cycle: (1) object creation, (2) configuration (e.g. segment size),
// (3) Create or LoadExisting, (4) usage and calling EnsureCapacity if necessary, (5) Close
type DataAccess interface {
	Name() string

	SetInt(bytePos int64, value int32)
	GetInt(bytePos int64) int32

	SetShort(bytePos int64, value int16)
	GetShort(bytePos int64) int16

	SetBytes(bytePos int64, values []byte, length int)
	GetBytes(bytePos int64, values []byte, length int)

	SetByte(bytePos int64, value byte)
	GetByte(bytePos int64) byte

	SetHeader(bytePos int, value int32)
	GetHeader(bytePos int) int32

	Create(bytes int64) DataAccess
	Flush()
	Close()
	IsClosed() bool

	LoadExisting() bool

	Capacity() int64
	EnsureCapacity(bytes int64) bool

	SegmentSize() int
	Segments() int
	Type() DAType
}
