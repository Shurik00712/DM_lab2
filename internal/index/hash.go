package index

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sync"
)

const (
	hashMagic         = "HSH1"
	hashVersion       = 2        
	headerSize        = 32
	bucketHeaderSize  = 2
	defaultBucketSize = 2048
	maxBucketSizeCap  = 1 << 20 
	maxFieldBuckets   = 512
)

type Hash struct {
	f           *os.File
	keyMaxSize  int
	bucketSize  uint32
	bucketCount uint32
	mu          sync.Mutex
}

// ---------- создание / открытие ----------

func Create(path string, keyMaxSize int) (*Hash, error) {
	entrySize := 8 + 2 + keyMaxSize
	minSize := uint32(bucketHeaderSize + 4*entrySize)
	bs := uint32(defaultBucketSize)
	if minSize > bs {
		bs = minSize
	}
	if int(bs) >= maxBucketSizeCap {
		return nil, fmt.Errorf("key too large: %d", keyMaxSize)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	h := &Hash{
		f:           f,
		keyMaxSize:  keyMaxSize,
		bucketSize:  bs,
		bucketCount: maxFieldBuckets,
	}
	if err := h.writeHeader(0); err != nil {
		f.Close()
		os.Remove(path)
		return nil, err
	}
	fullSize := int64(headerSize) + int64(h.bucketCount)*int64(h.bucketSize)
	if err := f.Truncate(fullSize); err != nil {
		f.Close()
		os.Remove(path)
		return nil, err
	}
	return h, nil
}

func Open(path string, keyMaxSize int) (*Hash, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, headerSize)
	if _, err := f.ReadAt(buf, 0); err != nil {
		f.Close()
		return nil, err
	}
	if string(buf[0:4]) != hashMagic {
		f.Close()
		return nil, errors.New("not a hash index file")
	}
	ver := binary.LittleEndian.Uint32(buf[4:8])
	var bs uint32
	switch ver {
	case 1:
		bs = defaultBucketSize
	case 2:
		bs = binary.LittleEndian.Uint32(buf[24:28])
		if bs == 0 {
			bs = defaultBucketSize
		}
	default:
		f.Close()
		return nil, errors.New("unsupported hash version")
	}
	bc := binary.LittleEndian.Uint32(buf[8:12])
	kms := int(binary.LittleEndian.Uint32(buf[20:24]))
	if kms != keyMaxSize {
		f.Close()
		return nil, fmt.Errorf("keyMaxSize mismatch: file=%d expected=%d", kms, keyMaxSize)
	}
	return &Hash{
		f:           f,
		keyMaxSize:  keyMaxSize,
		bucketSize:  bs,
		bucketCount: bc,
	}, nil
}

func (h *Hash) writeHeader(count uint64) error {
	buf := make([]byte, headerSize)
	copy(buf[0:4], hashMagic)
	binary.LittleEndian.PutUint32(buf[4:8], hashVersion)
	binary.LittleEndian.PutUint32(buf[8:12], h.bucketCount)
	binary.LittleEndian.PutUint64(buf[12:20], count)
	binary.LittleEndian.PutUint32(buf[20:24], uint32(h.keyMaxSize))
	binary.LittleEndian.PutUint32(buf[24:28], h.bucketSize)
	_, err := h.f.WriteAt(buf, 0)
	return err
}

// ---------- низкоуровневое ----------

func (h *Hash) hashKey(key []byte) uint32 {
	var x uint32 = 2166136261
	for _, b := range key {
		x ^= uint32(b)
		x *= 16777619
	}
	return x % h.bucketCount
}

func (h *Hash) bucketOffset(bucket uint32) int64 {
	return int64(headerSize) + int64(bucket)*int64(h.bucketSize)
}

func (h *Hash) entrySize() int { return 8 + 2 + h.keyMaxSize }

func (h *Hash) maxEntries() int {
	return (int(h.bucketSize) - bucketHeaderSize) / h.entrySize()
}

func (h *Hash) readBucket(bucket uint32) ([]byte, error) {
	buf := make([]byte, h.bucketSize)
	_, err := h.f.ReadAt(buf, h.bucketOffset(bucket))
	return buf, err
}

func (h *Hash) writeBucket(bucket uint32, buf []byte) error {
	_, err := h.f.WriteAt(buf, h.bucketOffset(bucket))
	return err
}

// ---------- публичный API ----------

func (h *Hash) Lookup(key []byte) ([]int64, error) {
	if len(key) > h.keyMaxSize {
		return nil, fmt.Errorf("key too long: %d > %d", len(key), h.keyMaxSize)
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	bucket := h.hashKey(key)
	buf, err := h.readBucket(bucket)
	if err != nil {
		return nil, err
	}
	count := int(binary.LittleEndian.Uint16(buf[0:2]))
	es := h.entrySize()
	var out []int64
	for i := 0; i < count; i++ {
		off := bucketHeaderSize + i*es
		sid := int64(binary.LittleEndian.Uint64(buf[off : off+8]))
		kl := int(binary.LittleEndian.Uint16(buf[off+8 : off+10]))
		if kl == len(key) && string(buf[off+10:off+10+kl]) == string(key) {
			out = append(out, sid)
		}
	}
	return out, nil
}

func (h *Hash) Insert(key []byte, slotID int64) error {
	if len(key) > h.keyMaxSize {
		return fmt.Errorf("key too long: %d > %d", len(key), h.keyMaxSize)
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	for attempt := 0; attempt < 8; attempt++ {
		ok, err := h.tryInsertLocked(key, slotID)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		if err := h.growLocked(); err != nil {
			return fmt.Errorf("grow: %w", err)
		}
	}
	return errors.New("hash: bucket full after multiple grows")
}

func (h *Hash) tryInsertLocked(key []byte, slotID int64) (bool, error) {
	bucket := h.hashKey(key)
	buf, err := h.readBucket(bucket)
	if err != nil {
		return false, err
	}
	count := int(binary.LittleEndian.Uint16(buf[0:2]))
	if count >= h.maxEntries() {
		return false, nil
	}
	es := h.entrySize()
	off := bucketHeaderSize + count*es
	binary.LittleEndian.PutUint64(buf[off:off+8], uint64(slotID))
	binary.LittleEndian.PutUint16(buf[off+8:off+10], uint16(len(key)))
	copy(buf[off+10:off+10+h.keyMaxSize], key)
	for j := off + 10 + len(key); j < off+es; j++ {
		buf[j] = 0
	}
	binary.LittleEndian.PutUint16(buf[0:2], uint16(count+1))
	return true, h.writeBucket(bucket, buf)
}

func (h *Hash) growLocked() error {
	newSize := h.bucketSize * 2
	if newSize > maxBucketSizeCap {
		newSize = maxBucketSizeCap
	}
	if newSize == h.bucketSize {
		return fmt.Errorf("bucket size cap reached: %d", h.bucketSize)
	}

	type rec struct {
		key  []byte
		slot int64
	}
	es := h.entrySize()
	var recs []rec
	for b := uint32(0); b < h.bucketCount; b++ {
		buf := make([]byte, h.bucketSize)
		if _, err := h.f.ReadAt(buf, h.bucketOffset(b)); err != nil {
			return err
		}
		cnt := int(binary.LittleEndian.Uint16(buf[0:2]))
		for i := 0; i < cnt; i++ {
			off := bucketHeaderSize + i*es
			slot := int64(binary.LittleEndian.Uint64(buf[off : off+8]))
			kl := int(binary.LittleEndian.Uint16(buf[off+8 : off+10]))
			k := make([]byte, kl)
			copy(k, buf[off+10:off+10+kl])
			recs = append(recs, rec{key: k, slot: slot})
		}
	}

	h.bucketSize = newSize
	totalSize := int64(headerSize) + int64(h.bucketCount)*int64(newSize)
	if err := h.f.Truncate(totalSize); err != nil {
		return err
	}
	zeros := make([]byte, newSize)
	for b := uint32(0); b < h.bucketCount; b++ {
		if _, err := h.f.WriteAt(zeros, h.bucketOffset(b)); err != nil {
			return err
		}
	}
		if err := h.writeHeader(0); err != nil {
			return err
		}

	for _, r := range recs {
		ok, err := h.tryInsertLocked(r.key, r.slot)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("grow: internal inconsistency (bucket still full after growth)")
		}
	}
	return nil
}

func (h *Hash) Delete(key []byte, slotID int64) error {
	if len(key) > h.keyMaxSize {
		return fmt.Errorf("key too long")
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	bucket := h.hashKey(key)
	buf, err := h.readBucket(bucket)
	if err != nil {
		return err
	}
	count := int(binary.LittleEndian.Uint16(buf[0:2]))
	es := h.entrySize()
	for i := 0; i < count; i++ {
		off := bucketHeaderSize + i*es
		sid := int64(binary.LittleEndian.Uint64(buf[off : off+8]))
		kl := int(binary.LittleEndian.Uint16(buf[off+8 : off+10]))
		if sid == slotID && kl == len(key) && string(buf[off+10:off+10+kl]) == string(key) {
			lastOff := bucketHeaderSize + (count-1)*es
			if off != lastOff {
				copy(buf[off:off+es], buf[lastOff:lastOff+es])
			}
			for j := lastOff; j < lastOff+es; j++ {
				buf[j] = 0
			}
			binary.LittleEndian.PutUint16(buf[0:2], uint16(count-1))
			return h.writeBucket(bucket, buf)
		}
	}
	return fmt.Errorf("hash entry not found: slotID=%d key=%q", slotID, key)
}

func (h *Hash) Sync() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.f == nil {
		return nil
	}
	return h.f.Sync()
}

func (h *Hash) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.f == nil {
		return nil
	}
	err := h.f.Sync()
	cerr := h.f.Close()
	h.f = nil
	if err != nil {
		return err
	}
	return cerr
}