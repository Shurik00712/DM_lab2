package index

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"
)

const (
	treeMagic      = "BTR1"
	treeVersion    = 1
	treeHeaderSize = 64
	pageHeaderSize = 16
	maxPageSize    = 1 << 20
)

const (
	pageTypeFree     = 0
	pageTypeLeaf     = 1
	pageTypeInternal = 2
)

type Tree struct {
	f          *os.File
	keyMaxSize int
	pageSize   uint32
	maxKeys    int  

	rootPage   uint64
	height     uint32
	entryCount uint64
	freeHead   int64

	mu sync.Mutex
}

// ---------- create / open ----------

func computePageSize(keyMaxSize int) (uint32, error) {
	entrySize := keyMaxSize + 8
	minSize := pageHeaderSize + 4*entrySize
	if minSize < 512 {
		minSize = 512
	}
	ps := uint32(512)
	for int(ps) < minSize {
		ps *= 2
	}
	if int(ps) > maxPageSize {
		return 0, fmt.Errorf("key too large: %d", keyMaxSize)
	}
	return ps, nil
}

func CreateTree(path string, keyMaxSize int) (*Tree, error) {
	if keyMaxSize <= 0 || keyMaxSize > maxPageSize-8 {
		return nil, fmt.Errorf("invalid keyMaxSize: %d", keyMaxSize)
	}
	pageSize, err := computePageSize(keyMaxSize)
	if err != nil {
		return nil, err
	}
	entrySize := keyMaxSize + 8
	capacity := (int(pageSize) - pageHeaderSize) / entrySize
	maxKeys := capacity - 1
	if maxKeys < 2 {
		return nil, fmt.Errorf("btree: pageSize=%d too small for keyMaxSize=%d", pageSize, keyMaxSize)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	t := &Tree{
		f:          f,
		keyMaxSize: keyMaxSize,
		pageSize:   pageSize,
		maxKeys:    maxKeys,
		freeHead:   -1,
	}
	if err := f.Truncate(int64(pageSize)); err != nil {
		f.Close()
		os.Remove(path)
		return nil, err
	}
	if err := t.writeHeader(); err != nil {
		f.Close()
		os.Remove(path)
		return nil, err
	}
	return t, nil
}

func OpenTree(path string, keyMaxSize int) (*Tree, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, treeHeaderSize)
	if _, err := f.ReadAt(buf, 0); err != nil {
		f.Close()
		return nil, err
	}
	if string(buf[0:4]) != treeMagic {
		f.Close()
		return nil, errors.New("not a btree index file")
	}
	ver := binary.LittleEndian.Uint32(buf[4:8])
	if ver != treeVersion {
		f.Close()
		return nil, fmt.Errorf("unsupported btree version %d", ver)
	}
	ps := binary.LittleEndian.Uint32(buf[8:12])
	root := binary.LittleEndian.Uint64(buf[12:20])
	height := binary.LittleEndian.Uint32(buf[20:24])
	count := binary.LittleEndian.Uint64(buf[24:32])
	freeHead := int64(binary.LittleEndian.Uint64(buf[32:40]))
	kms := int(binary.LittleEndian.Uint32(buf[40:44]))
	if kms != keyMaxSize {
		f.Close()
		return nil, fmt.Errorf("keyMaxSize mismatch: file=%d expected=%d", kms, keyMaxSize)
	}
	entrySize := keyMaxSize + 8
	capacity := (int(ps) - pageHeaderSize) / entrySize
	maxKeys := capacity - 1
	if maxKeys < 2 {
		f.Close()
		return nil, fmt.Errorf("btree: pageSize=%d too small for keyMaxSize=%d", ps, keyMaxSize)
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if info.Size() < int64(ps) {
		f.Close()
		return nil, errors.New("btree file too small")
	}
	return &Tree{
		f:          f,
		keyMaxSize: keyMaxSize,
		pageSize:   ps,
		maxKeys:    maxKeys,
		rootPage:   root,
		height:     height,
		entryCount: count,
		freeHead:   freeHead,
	}, nil
}

func (t *Tree) writeHeader() error {
	buf := make([]byte, treeHeaderSize)
	copy(buf[0:4], treeMagic)
	binary.LittleEndian.PutUint32(buf[4:8], treeVersion)
	binary.LittleEndian.PutUint32(buf[8:12], t.pageSize)
	binary.LittleEndian.PutUint64(buf[12:20], t.rootPage)
	binary.LittleEndian.PutUint32(buf[20:24], t.height)
	binary.LittleEndian.PutUint64(buf[24:32], t.entryCount)
	binary.LittleEndian.PutUint64(buf[32:40], uint64(t.freeHead))
	binary.LittleEndian.PutUint32(buf[40:44], uint32(t.keyMaxSize))
	_, err := t.f.WriteAt(buf, 0)
	return err
}

// ---------- page primitives ----------

type page struct {
	id       uint64
	typ      uint8
	keyCount uint16
	extra    uint64
	data     []byte
}

func (t *Tree) newPage() *page {
	return &page{data: make([]byte, t.pageSize)}
}

func (t *Tree) readPage(pageID uint64) (*page, error) {
	if pageID == 0 {
		return nil, errors.New("cannot read page 0 (header)")
	}
	buf := make([]byte, t.pageSize)
	off := int64(pageID) * int64(t.pageSize)
	n, err := t.f.ReadAt(buf, off)
	if err != nil {
		return nil, err
	}
	if n != int(t.pageSize) {
		return nil, errors.New("short page read")
	}
	p := &page{id: pageID, data: buf}
	p.typ = buf[0]
	p.keyCount = binary.LittleEndian.Uint16(buf[2:4])
	p.extra = binary.LittleEndian.Uint64(buf[4:12])
	return p, nil
}

func (t *Tree) writePage(p *page) error {
	p.data[0] = p.typ
	binary.LittleEndian.PutUint16(p.data[2:4], p.keyCount)
	binary.LittleEndian.PutUint64(p.data[4:12], p.extra)
	off := int64(p.id) * int64(t.pageSize)
	_, err := t.f.WriteAt(p.data, off)
	return err
}

func (t *Tree) entryOffset(idx int) int {
	return pageHeaderSize + idx*(t.keyMaxSize+8)
}

func (t *Tree) leafKey(p *page, idx int) []byte {
	off := t.entryOffset(idx)
	return p.data[off : off+t.keyMaxSize]
}
func (t *Tree) leafSlot(p *page, idx int) int64 {
	off := t.entryOffset(idx) + t.keyMaxSize
	return int64(binary.LittleEndian.Uint64(p.data[off : off+8]))
}
func (t *Tree) setLeafEntry(p *page, idx int, key []byte, slot int64) {
	off := t.entryOffset(idx)
	copy(p.data[off:off+t.keyMaxSize], key)
	binary.LittleEndian.PutUint64(p.data[off+t.keyMaxSize:off+t.keyMaxSize+8], uint64(slot))
}

func (t *Tree) intChild(p *page, idx int) uint64 {
	off := t.entryOffset(idx)
	return binary.LittleEndian.Uint64(p.data[off : off+8])
}
func (t *Tree) intKey(p *page, idx int) []byte {
	off := t.entryOffset(idx) + 8
	return p.data[off : off+t.keyMaxSize]
}
func (t *Tree) setIntEntry(p *page, idx int, child uint64, key []byte) {
	off := t.entryOffset(idx)
	binary.LittleEndian.PutUint64(p.data[off:off+8], child)
	copy(p.data[off+8:off+8+t.keyMaxSize], key)
}

func compareKeys(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	switch {
	case len(a) < len(b):
		return -1
	case len(a) > len(b):
		return 1
	}
	return 0
}

func (t *Tree) compareComposite(k1 []byte, s1 int64, k2 []byte, s2 int64) int {
	c := compareKeys(k1, k2)
	if c != 0 {
		return c
	}
	switch {
	case s1 < s2:
		return -1
	case s1 > s2:
		return 1
	}
	return 0
}

// ---------- alloc ----------

func (t *Tree) allocPage() (uint64, error) {
	if t.freeHead != -1 {
		id := uint64(t.freeHead)
		p, err := t.readPage(id)
		if err != nil {
			return 0, err
		}
		t.freeHead = int64(p.extra)
		return id, nil
	}
	info, err := t.f.Stat()
	if err != nil {
		return 0, err
	}
	id := uint64(info.Size() / int64(t.pageSize))
	if id == 0 {
		return 0, errors.New("btree: file has no page 0")
	}
	newSize := int64(id+1) * int64(t.pageSize)
	if info.Size() < newSize {
		if err := t.f.Truncate(newSize); err != nil {
			return 0, err
		}
	}
	return id, nil
}

// ---------- descent ----------

type pathEntry struct {
	pageID   uint64
	childIdx int
}

func (t *Tree) findChildIdx(p *page, key []byte) int {
	lo, hi := 0, int(p.keyCount)
	for lo < hi {
		mid := (lo + hi) / 2
		if compareKeys(key, t.intKey(p, mid)) <= 0 {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	return lo
}

func (t *Tree) descend(key []byte) (*page, []pathEntry, error) {
	pageID := t.rootPage
	var path []pathEntry
	for pageID != 0 {
		p, err := t.readPage(pageID)
		if err != nil {
			return nil, nil, err
		}
		switch p.typ {
		case pageTypeLeaf:
			return p, path, nil
		case pageTypeInternal:
			idx := t.findChildIdx(p, key)
			path = append(path, pathEntry{pageID: pageID, childIdx: idx})
			if idx < int(p.keyCount) {
				pageID = t.intChild(p, idx)
			} else {
				pageID = p.extra
			}
		default:
			return nil, nil, fmt.Errorf("unexpected page type %d at id %d", p.typ, pageID)
		}
	}
	return nil, path, nil
}

func (t *Tree) descendLeftmost() (*page, error) {
	pageID := t.rootPage
	for pageID != 0 {
		p, err := t.readPage(pageID)
		if err != nil {
			return nil, err
		}
		if p.typ == pageTypeLeaf {
			return p, nil
		}
		if p.keyCount == 0 {
			pageID = p.extra
		} else {
			pageID = t.intChild(p, 0)
		}
	}
	return nil, nil
}

func (t *Tree) findLeafPos(p *page, key []byte, slot int64) int {
	lo, hi := 0, int(p.keyCount)
	for lo < hi {
		mid := (lo + hi) / 2
		if t.compareComposite(t.leafKey(p, mid), t.leafSlot(p, mid), key, slot) < 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

// ---------- public API ----------

func (t *Tree) Insert(key []byte, slotID int64) error {
	if len(key) != t.keyMaxSize {
		return fmt.Errorf("btree key length %d != %d", len(key), t.keyMaxSize)
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.rootPage == 0 {
		id, err := t.allocPage()
		if err != nil {
			return err
		}
		leaf := t.newPage()
		leaf.id = id
		leaf.typ = pageTypeLeaf
		leaf.keyCount = 1
		leaf.extra = 0
		t.setLeafEntry(leaf, 0, key, slotID)
		if err := t.writePage(leaf); err != nil {
			return err
		}
		t.rootPage = id
		t.height = 1
		t.entryCount = 1
		return t.writeHeader()
	}

	leaf, path, err := t.descend(key)
	if err != nil {
		return err
	}
	pos := t.findLeafPos(leaf, key, slotID)
	if pos < int(leaf.keyCount) &&
		t.compareComposite(t.leafKey(leaf, pos), t.leafSlot(leaf, pos), key, slotID) == 0 {
		return errors.New("btree: duplicate (key, slotID)")
	}

	for i := int(leaf.keyCount); i > pos; i-- {
		t.setLeafEntry(leaf, i, t.leafKey(leaf, i-1), t.leafSlot(leaf, i-1))
	}
	t.setLeafEntry(leaf, pos, key, slotID)
	leaf.keyCount++
	t.entryCount++

	if int(leaf.keyCount) > t.maxKeys {
		if err := t.splitLeaf(leaf, path); err != nil {
			return err
		}
	} else if err := t.writePage(leaf); err != nil {
		return err
	}
	return t.writeHeader()
}

func (t *Tree) splitLeaf(leaf *page, path []pathEntry) error {
	mid := int(leaf.keyCount) / 2
	rightID, err := t.allocPage()
	if err != nil {
		return err
	}
	right := t.newPage()
	right.id = rightID
	right.typ = pageTypeLeaf
	rightCount := int(leaf.keyCount) - mid
	right.keyCount = uint16(rightCount)
	for i := 0; i < rightCount; i++ {
		t.setLeafEntry(right, i, t.leafKey(leaf, mid+i), t.leafSlot(leaf, mid+i))
	}
	right.extra = leaf.extra
	leaf.keyCount = uint16(mid)
	leaf.extra = rightID

	sepKey := make([]byte, t.keyMaxSize)
	copy(sepKey, t.leafKey(right, 0))

	if err := t.writePage(leaf); err != nil {
		return err
	}
	if err := t.writePage(right); err != nil {
		return err
	}
	return t.propagateSplit(path, sepKey, rightID, leaf.id)
}

func (t *Tree) splitInternal(node *page) ([]byte, uint64, error) {
	N := int(node.keyCount)
	mid := N / 2
	sepKey := make([]byte, t.keyMaxSize)
	copy(sepKey, t.intKey(node, mid))

	rightID, err := t.allocPage()
	if err != nil {
		return nil, 0, err
	}
	right := t.newPage()
	right.id = rightID
	right.typ = pageTypeInternal
	rightCount := N - mid - 1
	right.keyCount = uint16(rightCount)
	for i := 0; i < rightCount; i++ {
		t.setIntEntry(right, i, t.intChild(node, mid+1+i), t.intKey(node, mid+1+i))
	}
	right.extra = node.extra

	leftExtra := t.intChild(node, mid)
	node.keyCount = uint16(mid)
	node.extra = leftExtra

	if err := t.writePage(node); err != nil {
		return nil, 0, err
	}
	if err := t.writePage(right); err != nil {
		return nil, 0, err
	}
	return sepKey, rightID, nil
}

func (t *Tree) propagateSplit(path []pathEntry, sepKey []byte, newRightID, leftID uint64) error {
	for {
		if len(path) == 0 {
			newRootID, err := t.allocPage()
			if err != nil {
				return err
			}
			newRoot := t.newPage()
			newRoot.id = newRootID
			newRoot.typ = pageTypeInternal
			newRoot.keyCount = 1
			t.setIntEntry(newRoot, 0, leftID, sepKey)
			newRoot.extra = newRightID
			if err := t.writePage(newRoot); err != nil {
				return err
			}
			t.rootPage = newRootID
			t.height++
			return nil
		}
		pe := path[len(path)-1]
		path = path[:len(path)-1]

		parent, err := t.readPage(pe.pageID)
		if err != nil {
			return err
		}
		j := pe.childIdx
		N := int(parent.keyCount)

		if j < N {
			oldC := t.intChild(parent, j)
			oldK := make([]byte, t.keyMaxSize)
			copy(oldK, t.intKey(parent, j))
			for i := N; i > j+1; i-- {
				t.setIntEntry(parent, i, t.intChild(parent, i-1), t.intKey(parent, i-1))
			}
			t.setIntEntry(parent, j+1, newRightID, oldK)
			t.setIntEntry(parent, j, oldC, sepKey)
			parent.keyCount = uint16(N + 1)
		} else {
			oldC := parent.extra
			t.setIntEntry(parent, N, oldC, sepKey)
			parent.extra = newRightID
			parent.keyCount = uint16(N + 1)
		}

		if int(parent.keyCount) <= t.maxKeys {
			return t.writePage(parent)
		}
		newSep, rightID, err := t.splitInternal(parent)
		if err != nil {
			return err
		}
		sepKey, newRightID, leftID = newSep, rightID, parent.id
	}
}

func (t *Tree) Lookup(key []byte) ([]int64, error) {
	if len(key) != t.keyMaxSize {
		return nil, fmt.Errorf("btree key length %d != %d", len(key), t.keyMaxSize)
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.rootPage == 0 {
		return nil, nil
	}
	leaf, _, err := t.descend(key)
	if err != nil {
		return nil, err
	}
	var out []int64
	for leaf != nil {
		for i := 0; i < int(leaf.keyCount); i++ {
			c := compareKeys(t.leafKey(leaf, i), key)
			if c < 0 {
				continue
			}
			if c > 0 {
				return out, nil
			}
			out = append(out, t.leafSlot(leaf, i))
		}
		if leaf.extra == 0 {
			return out, nil
		}
		next, err := t.readPage(leaf.extra)
		if err != nil {
			return out, err
		}
		leaf = next
	}
	return out, nil
}

func (t *Tree) RangeLookup(lo, hi []byte) ([]int64, error) {
	if lo != nil && len(lo) != t.keyMaxSize {
		return nil, fmt.Errorf("lo length %d != %d", len(lo), t.keyMaxSize)
	}
	if hi != nil && len(hi) != t.keyMaxSize {
		return nil, fmt.Errorf("hi length %d != %d", len(hi), t.keyMaxSize)
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.rootPage == 0 {
		return nil, nil
	}
	var leaf *page
	var err error
	if lo == nil {
		leaf, err = t.descendLeftmost()
	} else {
		leaf, _, err = t.descend(lo)
	}
	if err != nil {
		return nil, err
	}
	var out []int64
	for leaf != nil {
		for i := 0; i < int(leaf.keyCount); i++ {
			k := t.leafKey(leaf, i)
			if lo != nil && compareKeys(k, lo) < 0 {
				continue
			}
			if hi != nil && compareKeys(k, hi) > 0 {
				return out, nil
			}
			out = append(out, t.leafSlot(leaf, i))
		}
		if leaf.extra == 0 {
			return out, nil
		}
		next, err := t.readPage(leaf.extra)
		if err != nil {
			return out, err
		}
		leaf = next
	}
	return out, nil
}

func (t *Tree) Delete(key []byte, slotID int64) error {
	if len(key) != t.keyMaxSize {
		return fmt.Errorf("btree key length %d != %d", len(key), t.keyMaxSize)
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.rootPage == 0 {
		return errors.New("btree: tree is empty")
	}
	leaf, _, err := t.descend(key)
	if err != nil {
		return err
	}
	for leaf != nil {
		for i := 0; i < int(leaf.keyCount); i++ {
			c := compareKeys(t.leafKey(leaf, i), key)
			if c < 0 {
				continue
			}
			if c > 0 {
				return fmt.Errorf("btree: entry not found (key=%x slot=%d)", key, slotID)
			}
			if t.leafSlot(leaf, i) == slotID {
				for j := i; j < int(leaf.keyCount)-1; j++ {
					t.setLeafEntry(leaf, j, t.leafKey(leaf, j+1), t.leafSlot(leaf, j+1))
				}
				leaf.keyCount--
				t.entryCount--
				return t.writePage(leaf)
			}
		}
		if leaf.extra == 0 {
			return fmt.Errorf("btree: entry not found (key=%x slot=%d)", key, slotID)
		}
		next, err := t.readPage(leaf.extra)
		if err != nil {
			return err
		}
		leaf = next
	}
	return fmt.Errorf("btree: entry not found (key=%x slot=%d)", key, slotID)
}

func (t *Tree) Count() uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.entryCount
}

func (t *Tree) Sync() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.f == nil {
		return nil
	}
	return t.f.Sync()
}

func (t *Tree) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.f == nil {
		return nil
	}
	serr := t.f.Sync()
	cerr := t.f.Close()
	t.f = nil
	if serr != nil {
		return serr
	}
	return cerr
}

var _ = math.Float64bits