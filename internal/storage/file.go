package storage


import ("fmt"
		"os"
		"errors"
		"sync"
		"file_db/internal/schema")

const slotHeaderSize = 9
type File struct {
	f *os.File
	schema *schema.Schema
	headerSize int
	recordCount uint64
	freeHead int64
	pool *bufferPool
	mu sync.Mutex
}

func Create(path string, s *schema.Schema) (*File, error) {
	if err := schema.Validate(s); err != nil {
        return nil, fmt.Errorf("invalid schema: %w", err)
    }
	encodedHead, err := schema.EncodeHeader(s, 0, -1)
	if err != nil {
		return nil, fmt.Errorf("Failed encode: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	ok := false
    defer func() {
        if !ok {
            file.Close()
            os.Remove(path)
        }
    }()

	if _, err := file.WriteAt(encodedHead, 0); err != nil {
		return nil, fmt.Errorf("write header to file: %w", err)

	}

	if err := file.Sync(); err != nil {
        return nil, fmt.Errorf("sync: %w", err)
    }

	ok = true
    info, err := file.Stat()
    if err != nil { return nil, err }
    pool := newBufferPool(file, defaultPageSize, defaultCapacity, info.Size())

    ok = true
    return &File{
        f:           file,
        schema:      s,
        headerSize:  schema.HeaderSize(s),
        recordCount: 0,
        freeHead:    -1,
        pool:        pool,   
    }, nil
}



func Open(path string) (*File, error) {
    f, err := os.OpenFile(path, os.O_RDWR, 0)
    if err != nil {
        return nil, fmt.Errorf("open: %w", err)
    }

    ok := false
    defer func() {
        if !ok {
            f.Close()
        }
    }()
    info, err := f.Stat()
    if err != nil {
        return nil, fmt.Errorf("stat: %w", err)
    }
    fileSize := info.Size()
	fixedSize := schema.FixedHeaderSize()
	if fileSize < int64(fixedSize) {
    	return nil, fmt.Errorf("corrupted: file too small (%d < %d)", fileSize, fixedSize)
	}
	prefData := make([]byte, fixedSize)

	if _, err := f.ReadAt(prefData, 0); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

    totalSize, err := schema.HeaderSizeFromPrefix(prefData)
    if err != nil {
        return nil, fmt.Errorf("header prefix: %w", err)
    }
    full := make([]byte, totalSize)
    copy(full, prefData)
    if totalSize > fixedSize {
		if _, err := f.ReadAt(full[fixedSize:], int64(fixedSize)); err != nil {
			return nil, fmt.Errorf("read header: %w", err)
		}
	}

    sch, recordCount, freeHead, err := schema.DecodeHeader(full)
    if err != nil {
        return nil, fmt.Errorf("decode header: %w", err)
    }
	headerSize := schema.HeaderSize(sch)
    slotSize := int64(slotHeaderSize + sch.RecordSize)

    if recordCount > uint64(fileSize)/uint64(slotSize) {
        return nil, fmt.Errorf("corrupted: recordCount %d too large for file size %d", recordCount, fileSize)
    }
    expected := int64(headerSize) + int64(recordCount)*slotSize
    if expected != fileSize {
        if err := f.Truncate(expected); err != nil {
            return nil, fmt.Errorf("recover: truncate extra %d: %w", fileSize-expected, err)
        }
    }

    pool := newBufferPool(f, defaultPageSize, defaultCapacity, fileSize)

    ok = true
    return &File{
        f:           f,
        schema:      sch,
        headerSize:  headerSize,
        recordCount: recordCount,
        freeHead:    freeHead,
        pool:        pool,  
    }, nil
}

func (f *File) Sync() error {
    return f.syncLocked()
}

func (f *File) syncLocked() error {
    if f.f == nil {
        return errors.New("file is closed")
    }
	if err := f.pool.Flush(); err != nil {
		return err
	}
    return f.f.Sync()
}

func (f *File) writeHeader() error {
    buf, err := schema.EncodeHeader(f.schema, f.recordCount, f.freeHead)
    if err != nil { return err }
    _, err = f.pool.WriteAt(buf, 0)   
    return err
}

func (f *File) Close() error {
    f.mu.Lock()
    defer f.mu.Unlock()

    if f.f == nil {
        return nil
    }

    syncErr := f.syncLocked()     
    closeErr := f.f.Close()        

    if syncErr != nil {
        return fmt.Errorf("sync: %w", syncErr)
    }
    if closeErr != nil {
        return fmt.Errorf("close: %w", closeErr)
    }
    return nil
}

func (f *File) Clear() error {
    f.mu.Lock()
    defer f.mu.Unlock()

    if f.f == nil {
        return errors.New("file is closed")
    }

    f.pool.Reset()
    if err := f.f.Truncate(int64(f.headerSize)); err != nil {
        return fmt.Errorf("truncate: %w", err)
    }
    f.pool.SetFileSize(int64(f.headerSize))

    f.recordCount = 0
    f.freeHead = -1

    if err := f.writeHeader(); err != nil {
        return fmt.Errorf("write header: %w", err)
    }

    if err := f.syncLocked(); err != nil {             
        return fmt.Errorf("sync: %w", err)
    }
    return nil
}

func (f *File) slotSize() int {
	return slotHeaderSize + f.schema.RecordSize
}

func (f *File) RecordCount() int {
    f.mu.Lock()
    defer f.mu.Unlock()
    return int(f.recordCount)
}

