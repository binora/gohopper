package storage

import (
	"encoding/binary"
	"fmt"

	"gohopper/core/util"
)

const (
	kvEmptyPointer  = 0
	kvStartPointer  = 1
)

// KVStorage is an append-only key-value store using two DataAccess objects.
// Used for edge key-value attributes (street names, etc.).
// Note: In Java this lives in com.graphhopper.search; moved here to avoid import cycles.
type KVStorage struct {
	keys        DataAccess
	vals        DataAccess
	keyToIndex  map[string]int
	indexToKey  []string
	indexToType []byte // type short name: 'S'=string, 'i'=int, 'f'=float, 'l'=long, 'd'=double, '['=bytes
	bytePointer int64
}

// NewKVStorage creates a KVStorage for edges (edge=true) or nodes (edge=false).
func NewKVStorage(dir Directory, edge bool) *KVStorage {
	var keysName, valsName string
	if edge {
		keysName = "edgekv_keys"
		valsName = "edgekv_vals"
	} else {
		keysName = "nodekv_keys"
		valsName = "nodekv_vals"
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
	if !kv.vals.LoadExisting() {
		return false
	}
	if !kv.keys.LoadExisting() {
		return false
	}
	low := kv.vals.GetHeader(0 * 4)
	high := kv.vals.GetHeader(1 * 4)
	kv.bytePointer = util.BitLE.ToLongFromInts(low, high)

	version := kv.vals.GetHeader(2 * 4)
	kvCheckVersion("edgekv_vals", util.VersionKVStorage, int(version))

	pos := int64(0)
	countBytes := make([]byte, 2)
	kv.keys.GetBytes(pos, countBytes, 2)
	count := int(binary.BigEndian.Uint16(countBytes))
	pos += 2

	kv.indexToKey = make([]string, 0, count)
	kv.indexToType = make([]byte, 0, count)
	kv.keyToIndex = make(map[string]int, count)

	for i := 0; i < count; i++ {
		lenBytes := make([]byte, 2)
		kv.keys.GetBytes(pos, lenBytes, 2)
		keyLen := int(binary.BigEndian.Uint16(lenBytes))
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
	count := len(kv.indexToKey)
	totalSize := 2
	for _, key := range kv.indexToKey {
		totalSize += 2 + len(key) + 1
	}
	kv.keys.EnsureCapacity(int64(totalSize))
	pos := int64(0)
	countBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(countBytes, uint16(count))
	kv.keys.SetBytes(pos, countBytes, 2)
	pos += 2

	for i, key := range kv.indexToKey {
		lenBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lenBytes, uint16(len(key)))
		kv.keys.SetBytes(pos, lenBytes, 2)
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

// GetAll returns all key-value pairs at the given entry pointer.
func (kv *KVStorage) GetAll(entryPointer int64) map[string]interface{} {
	if entryPointer == kvEmptyPointer {
		return nil
	}
	result := make(map[string]interface{})
	count := int(kv.vals.GetByte(entryPointer))
	pos := entryPointer + 1
	for i := 0; i < count; i++ {
		keyIdxBytes := make([]byte, 2)
		kv.vals.GetBytes(pos, keyIdxBytes, 2)
		rawIdx := binary.BigEndian.Uint16(keyIdxBytes)
		keyIdx := int(rawIdx >> 2)
		pos += 2

		if keyIdx >= len(kv.indexToKey) {
			break
		}
		key := kv.indexToKey[keyIdx]
		classType := kv.indexToType[keyIdx]

		var value interface{}
		switch classType {
		case 'S':
			vLen := int(kv.vals.GetByte(pos))
			pos++
			vBytes := make([]byte, vLen)
			if vLen > 0 {
				kv.vals.GetBytes(pos, vBytes, vLen)
			}
			pos += int64(vLen)
			value = string(vBytes)
		case '[':
			vLen := int(kv.vals.GetByte(pos))
			pos++
			vBytes := make([]byte, vLen)
			if vLen > 0 {
				kv.vals.GetBytes(pos, vBytes, vLen)
			}
			pos += int64(vLen)
			value = vBytes
		case 'i':
			value = int(kv.vals.GetInt(pos))
			pos += 4
		case 'f':
			b := make([]byte, 4)
			kv.vals.GetBytes(pos, b, 4)
			value = util.BitLE.ToFloat(b, 0)
			pos += 4
		case 'l':
			b := make([]byte, 8)
			kv.vals.GetBytes(pos, b, 8)
			value = util.BitLE.ToLong(b, 0)
			pos += 8
		case 'd':
			b := make([]byte, 8)
			kv.vals.GetBytes(pos, b, 8)
			value = util.BitLE.ToDouble(b, 0)
			pos += 8
		}
		result[key] = value
	}
	return result
}

func kvCheckVersion(name string, expected, actual int) {
	if expected != actual {
		panic(fmt.Sprintf("cannot load %s - expected version %d, got %d", name, expected, actual))
	}
}
