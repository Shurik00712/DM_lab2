package storage

import ("os"
		"errors"
		"io"
		"sync")


const (
	defaultPageSize = 16 * 1024
	defaultCapacity = 64
)


type pageID int64

type frame struct {
	id pageID
	data []byte
	prev, next *frame
	dirty bool
	pinCount int 
}

type bufferPool struct {
	file *os.File
	pageSize int
	capacity int
	fileSize int64
	frames map[pageID]*frame
	head, tail *frame
	mu sync.Mutex
}

func newBufferPool(f *os.File, pageSize, capacity int, fileSize int64) *bufferPool {
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if capacity <= 0 {
		capacity = defaultCapacity
	}
	return &bufferPool{
		file:     f,
		pageSize: pageSize,
		capacity: capacity,
		fileSize: fileSize,
		frames:   make(map[pageID]*frame, capacity),
	}
}

func (bp *bufferPool) SetFileSize(size int64) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.fileSize = size
}

func (bp *bufferPool) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, errors.New("negative offset")
	}

	bp.mu.Lock()
	defer bp.mu.Unlock()
	total := 0

	for total < len(p) {
		curoff := int64(total) + off
		pid := pageID(curoff / int64(bp.pageSize))
		offinpage := int(curoff % int64(bp.pageSize))
		fr, err := bp.fetchPageLocked(pid)
		if err != nil {
			return total, err
		}
		n := copy(p[total:], fr.data[offinpage:])
		total += n
		bp.unpinLocked(fr)
		if (n == 0) {
			break
		}
	}
	return total, nil
}

func (bp *bufferPool) unpinLocked(fr *frame) {
	if fr.pinCount > 0 {
		fr.pinCount--
	}
}

func (bp *bufferPool) Flush() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	for _, fr := range bp.frames {
		if fr.dirty {
			if err := bp.flushFrameLocked(fr); err != nil {
				return err
			}
		}
	}
	return nil
}

func (bp *bufferPool) Reset() {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.frames = make(map[pageID]*frame, bp.capacity)
	bp.head = nil
	bp.tail = nil
}

func (bp *bufferPool) WriteAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, errors.New("negative offset")
	}

	bp.mu.Lock()
	defer bp.mu.Unlock()
	total := 0

	for total < len(p) {
		curoff := int64(total) + off
		pid := pageID(curoff / int64(bp.pageSize))
		offinpage := int(curoff % int64(bp.pageSize))
		fr, err := bp.fetchPageLocked(pid)
		if err != nil {
			return total, err
		}
		n := copy(fr.data[offinpage:], p[total:])
		fr.dirty = true
		total += n
		bp.unpinLocked(fr)
		if (n == 0) {
			break
		}
	}
	end := off + int64(len(p))
	if end > bp.fileSize {
		bp.fileSize = end
	}
	return total, nil
}

func (bp *bufferPool) fetchPageLocked(pid pageID) (*frame, error) {
	if fr, ok := bp.frames[pid]; ok {
		bp.moveToHeadLocked(fr)
		return fr, nil
	}

	var fr *frame
	if len(bp.frames) > bp.capacity {
		fr = bp.evictLocked()
		if fr == nil {
			return nil, errors.New("buffer pool: all frames pinned")
		}
	} else {
		fr = &frame{data: make([]byte, bp.pageSize)}
	}
	off := int64(pid) * int64(bp.pageSize)
	n, err := bp.file.ReadAt(fr.data, off)
	if err != nil && err != io.EOF {
		return nil, err
	}
	for i := n; i < bp.pageSize; i++ {
		fr.data[i] = 0
	}
	fr.id = pid
	fr.dirty = false
	fr.pinCount = 1
	bp.frames[pid] = fr
	bp.pushHeadLocked(fr)
	return fr, nil
}

func (bp *bufferPool) evictLocked() *frame {
	for fr := bp.tail; fr != nil; fr = fr.prev {
		if fr.pinCount > 0 {
			continue
		}
		delete(bp.frames, fr.id)
		bp.removeLocked(fr)
		if fr.dirty {
			_ = bp.flushFrameLocked(fr)
		}
		return fr
	}
	return nil
}

func (bp *bufferPool) flushFrameLocked(fr *frame) error {
    off := int64(fr.id) * int64(bp.pageSize)
    n := bp.pageSize
    if off+int64(n) > bp.fileSize {
        n = int(bp.fileSize - off)  
    }
    if n <= 0 { fr.dirty = false; return nil }
    _, err := bp.file.WriteAt(fr.data[:n], off)
    fr.dirty = false
    return err
}

func (bp *bufferPool) removeLocked(fr *frame) {
    if fr.prev != nil { fr.prev.next = fr.next } else { bp.head = fr.next }
    if fr.next != nil { fr.next.prev = fr.prev } else { bp.tail = fr.prev }
    fr.prev = nil
    fr.next = nil
}

func (bp *bufferPool) pushHeadLocked(fr *frame) {
    fr.prev = nil
    fr.next = bp.head
    if bp.head != nil { bp.head.prev = fr }
    bp.head = fr
    if bp.tail == nil { bp.tail = fr }
}

func (bp *bufferPool) moveToHeadLocked(fr *frame) {
	if bp.head == fr {
    	return              
	}
	bp.removeLocked(fr)     
	bp.pushHeadLocked(fr) 
}

