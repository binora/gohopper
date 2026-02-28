package storage

import (
	"encoding/binary"

	"gohopper/core/util"
)

const (
	kvEmptyPointer = 0
	kvStartPointer = 1
)

// KVStorage is an append-only key-value store using two DataAccess objects.
// Used for edge key-value attributes (street names, etc.).
type KVStorage struct {
	keys        DataAccess
	vals        DataAccess
	keyToIndex  map[string]int
	indexToKey  []string
	indexToType []byte
	bytePointer int64
}

func NewKVStorage(dir Directory, edge bool) *KVStorage {
	keysName, valsName := "nodekv_keys", "nodekv_vals"
	if edge {
		keysName, valsName = "edgekv_keys", "edgekv_vals"
	}
	return &KVStorage{
		keys:        dir.CreateWithSegmentSize(keysName, 10*1024),
		vals:        dir.Create(valsName),
		keyToIndex:  make(map[string]int),
		bytePointer: kvStartPointer,
	}
}

func (kv *KVStorage) Create(initBytes int64) {
	kv.keys.Create(initBytes)
	kv.vals.Create(initBytes)
	kv.keyToIndex[""] = 0
	kv.indexToKey = append(kv.indexToKey, "")
	kv.indexToType = append(kv.indexToType, 'S')
}

func (kv *KVStorage) LoadExisting() bool {
	if !kv.vals.LoadExisting() || !kv.keys.LoadExisting() {
		return false
	}
	low := kv.vals.GetHeader(0 * 4)
	high := kv.vals.GetHeader(1 * 4)
	kv.bytePointer = util.BitLE.ToLongFromInts(low, high)

	checkDAVersion("edgekv_vals", util.VersionKVStorage, int(kv.vals.GetHeader(2*4)))

	pos := int64(0)
	buf2 := make([]byte, 2)
	kv.keys.GetBytes(pos, buf2, 2)
	count := int(binary.BigEndian.Uint16(buf2))
	pos += 2

	kv.indexToKey = make([]string, 0, count)
	kv.indexToType = make([]byte, 0, count)
	kv.keyToIndex = make(map[string]int, count)

	for i := range count {
		kv.keys.GetBytes(pos, buf2, 2)
		keyLen := int(binary.BigEndian.Uint16(buf2))
		pos += 2

		keyBytes := make([]byte, keyLen)
		if keyLen > 0 {
			kv.keys.GetBytes(pos, keyBytes, keyLen)
		}
		pos += int64(keyLen)

		classByte := kv.keys.GetByte(pos)
		pos++

		key := string(keyBytes)
		kv.keyToIndex[key] = i
		kv.indexToKey = append(kv.indexToKey, key)
		kv.indexToType = append(kv.indexToType, classByte)
	}
	return true
}

func (kv *KVStorage) Flush() {
	totalSize := 2
	for _, key := range kv.indexToKey {
		totalSize += 2 + len(key) + 1
	}
	kv.keys.EnsureCapacity(int64(totalSize))

	pos := int64(0)
	buf2 := make([]byte, 2)
	binary.BigEndian.PutUint16(buf2, uint16(len(kv.indexToKey)))
	kv.keys.SetBytes(pos, buf2, 2)
	pos += 2

	for i, key := range kv.indexToKey {
		binary.BigEndian.PutUint16(buf2, uint16(len(key)))
		kv.keys.SetBytes(pos, buf2, 2)
		pos += 2
		if len(key) > 0 {
			kv.keys.SetBytes(pos, []byte(key), len(key))
			pos += int64(len(key))
		}
		kv.keys.SetByte(pos, kv.indexToType[i])
		pos++
	}

	kv.vals.SetHeader(0*4, util.BitLE.GetIntLow(kv.bytePointer))
	kv.vals.SetHeader(1*4, util.BitLE.GetIntHigh(kv.bytePointer))
	kv.vals.SetHeader(2*4, int32(util.VersionKVStorage))

	kv.keys.Flush()
	kv.vals.Flush()
}

func (kv *KVStorage) Close() {
	kv.keys.Close()
	kv.vals.Close()
}

func (kv *KVStorage) IsClosed() bool {
	return kv.vals.IsClosed() && kv.keys.IsClosed()
}

// Add stores key-value pairs and returns a pointer to retrieve them later.
// All values are stored as both forward and backward (direction-agnostic).
// Supported value types: string, []byte, int, int32, float32, float64, int64.
func (kv *KVStorage) Add(entries map[string]any) int64 {
	if entries == nil {
		panic("specified entries must not be nil")
	}
	if len(entries) == 0 {
		return kvEmptyPointer
	}

	entryPointer := kv.bytePointer
	kv.vals.EnsureCapacity(kv.bytePointer + 1)
	kv.vals.SetByte(kv.bytePointer, byte(len(entries)))
	kv.bytePointer++

	buf2 := make([]byte, 2)
	for key, value := range entries {
		keyIdx, ok := kv.keyToIndex[key]
		if !ok {
			keyIdx = len(kv.indexToKey)
			kv.keyToIndex[key] = keyIdx
			kv.indexToKey = append(kv.indexToKey, key)
			kv.indexToType = append(kv.indexToType, classTypeOf(value))
		}

		// encode keyIndex<<2 | fwd(2) | bwd(1) = both directions
		raw := uint16(keyIdx<<2 | 3)
		binary.BigEndian.PutUint16(buf2, raw)
		kv.vals.EnsureCapacity(kv.bytePointer + 2)
		kv.vals.SetBytes(kv.bytePointer, buf2, 2)
		kv.bytePointer += 2

		kv.writeValue(value, kv.indexToType[keyIdx])
	}
	return entryPointer
}

func (kv *KVStorage) writeValue(value any, classType byte) {
	switch classType {
	case 'S':
		s, _ := value.(string)
		b := []byte(s)
		kv.vals.EnsureCapacity(kv.bytePointer + 1 + int64(len(b)))
		kv.vals.SetByte(kv.bytePointer, byte(len(b)))
		kv.bytePointer++
		if len(b) > 0 {
			kv.vals.SetBytes(kv.bytePointer, b, len(b))
			kv.bytePointer += int64(len(b))
		}
	case '[':
		b, _ := value.([]byte)
		kv.vals.EnsureCapacity(kv.bytePointer + 1 + int64(len(b)))
		kv.vals.SetByte(kv.bytePointer, byte(len(b)))
		kv.bytePointer++
		if len(b) > 0 {
			kv.vals.SetBytes(kv.bytePointer, b, len(b))
			kv.bytePointer += int64(len(b))
		}
	case 'i':
		var v int32
		switch val := value.(type) {
		case int:
			v = int32(val)
		case int32:
			v = val
		}
		kv.vals.EnsureCapacity(kv.bytePointer + 4)
		kv.vals.SetInt(kv.bytePointer, v)
		kv.bytePointer += 4
	case 'f':
		var b [4]byte
		util.BitLE.FromFloat(b[:], value.(float32), 0)
		kv.vals.EnsureCapacity(kv.bytePointer + 4)
		kv.vals.SetBytes(kv.bytePointer, b[:], 4)
		kv.bytePointer += 4
	case 'l':
		var b [8]byte
		util.BitLE.FromLong(b[:], value.(int64), 0)
		kv.vals.EnsureCapacity(kv.bytePointer + 8)
		kv.vals.SetBytes(kv.bytePointer, b[:], 8)
		kv.bytePointer += 8
	case 'd':
		var b [8]byte
		util.BitLE.FromDouble(b[:], value.(float64), 0)
		kv.vals.EnsureCapacity(kv.bytePointer + 8)
		kv.vals.SetBytes(kv.bytePointer, b[:], 8)
		kv.bytePointer += 8
	}
}

func classTypeOf(v any) byte {
	switch v.(type) {
	case string:
		return 'S'
	case []byte:
		return '['
	case int, int32:
		return 'i'
	case float32:
		return 'f'
	case int64:
		return 'l'
	case float64:
		return 'd'
	default:
		panic("unsupported KV value type")
	}
}

// Get returns a single value by key at the given entry pointer.
// The reverse parameter selects direction (false=forward, true=backward).
func (kv *KVStorage) Get(entryPointer int64, key string, reverse bool) any {
	if entryPointer == kvEmptyPointer {
		return nil
	}
	keyIdx, ok := kv.keyToIndex[key]
	if !ok {
		return nil
	}

	count := int(kv.vals.GetByte(entryPointer))
	pos := entryPointer + 1
	buf2 := make([]byte, 2)

	// Direction bits: bit 1 = bwd, bit 2 = fwd.
	// When reverse=false we check fwd (bit 1), when reverse=true we check bwd (bit 0).
	dirBit := 2 // fwd
	if reverse {
		dirBit = 1 // bwd
	}

	for range count {
		kv.vals.GetBytes(pos, buf2, 2)
		raw := int(binary.BigEndian.Uint16(buf2))
		curKeyIdx := raw >> 2
		pos += 2

		if curKeyIdx >= len(kv.indexToKey) {
			break
		}
		classType := kv.indexToType[curKeyIdx]
		if curKeyIdx == keyIdx && raw&dirBit != 0 {
			return kv.readValue(pos, classType)
		}
		pos += kv.valueLength(pos, classType)
	}
	return nil
}

func (kv *KVStorage) readValue(pos int64, classType byte) any {
	switch classType {
	case 'S':
		vLen := int(kv.vals.GetByte(pos))
		pos++
		if vLen == 0 {
			return ""
		}
		b := make([]byte, vLen)
		kv.vals.GetBytes(pos, b, vLen)
		return string(b)
	case '[':
		vLen := int(kv.vals.GetByte(pos))
		pos++
		b := make([]byte, vLen)
		if vLen > 0 {
			kv.vals.GetBytes(pos, b, vLen)
		}
		return b
	case 'i':
		return int(kv.vals.GetInt(pos))
	case 'f':
		b := make([]byte, 4)
		kv.vals.GetBytes(pos, b, 4)
		return util.BitLE.ToFloat(b, 0)
	case 'l':
		b := make([]byte, 8)
		kv.vals.GetBytes(pos, b, 8)
		return util.BitLE.ToLong(b, 0)
	case 'd':
		b := make([]byte, 8)
		kv.vals.GetBytes(pos, b, 8)
		return util.BitLE.ToDouble(b, 0)
	}
	return nil
}

func (kv *KVStorage) valueLength(pos int64, classType byte) int64 {
	switch classType {
	case 'S', '[':
		return 1 + int64(kv.vals.GetByte(pos))
	case 'i', 'f':
		return 4
	case 'l', 'd':
		return 8
	}
	return 0
}

// GetAll returns all key-value pairs at the given entry pointer.
func (kv *KVStorage) GetAll(entryPointer int64) map[string]any {
	if entryPointer == kvEmptyPointer {
		return nil
	}
	count := int(kv.vals.GetByte(entryPointer))
	pos := entryPointer + 1
	result := make(map[string]any, count)
	buf2 := make([]byte, 2)

	for range count {
		kv.vals.GetBytes(pos, buf2, 2)
		keyIdx := int(binary.BigEndian.Uint16(buf2)) >> 2
		pos += 2

		if keyIdx >= len(kv.indexToKey) {
			break
		}
		classType := kv.indexToType[keyIdx]
		result[kv.indexToKey[keyIdx]] = kv.readValue(pos, classType)
		pos += kv.valueLength(pos, classType)
	}
	return result
}
