package storage

import ("file_db/internal/schema"
		"fmt"
		"errors")

func (f *File) Read(slotID int) (map[string]any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.f == nil {
		return nil, errors.New("file is closed")
	}
	if slotID < 0 || slotID >= int(f.recordCount) {
		return nil, fmt.Errorf("slotID %d out of range [0, %d)", slotID, f.recordCount)
	}

	offset := int64(f.headerSize) + int64(slotID)*int64(f.slotSize())
	head := make([]byte, slotHeaderSize)
	if _, err := f.pool.ReadAt(head, offset); err != nil {
		return nil, fmt.Errorf("read slot header: %w", err)
	}
	if head[0] == 0 {
		return nil, fmt.Errorf("slot %d is free", slotID)
	}

	payload := make([]byte, f.schema.RecordSize)
	if _, err := f.pool.ReadAt(payload, offset+slotHeaderSize); err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}

	return schema.Decode(f.schema, payload)
}

func (f *File) Write(slotID int, values map[string]any) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.f == nil {
		return errors.New("file is closed")
	}
	if slotID < 0 || slotID >= int(f.recordCount) {
		return fmt.Errorf("slotID %d out of range [0, %d)", slotID, f.recordCount)
	}

	offset := int64(f.headerSize) + int64(slotID)*int64(f.slotSize())
	head := make([]byte, 1)
	if _, err := f.pool.ReadAt(head, offset); err != nil {
		return fmt.Errorf("read flags: %w", err)
	}
	if head[0] == 0 {
		return fmt.Errorf("slot %d is free, use Alloc first", slotID)
	}

	payload, err := schema.Encode(f.schema, values)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	if _, err := f.pool.WriteAt(payload, offset+slotHeaderSize); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}
	return nil
}

func (f *File) Alloc() (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.f == nil {
		return 0, errors.New("file is closed")
	}

	var slotID int
	if f.freeHead != -1 {
		slotID = int(f.freeHead)
		offset := int64(f.headerSize) + int64(slotID)*int64(f.slotSize())
		next := make([]byte, 8)
		if _, err := f.pool.ReadAt(next, offset+1); err != nil {
			return 0, fmt.Errorf("read nextFree: %w", err)
		}
		f.freeHead = int64(schema.U64(next))
	} else {
		slotID = int(f.recordCount)
		f.recordCount++
		newSize := int64(f.headerSize) + int64(f.recordCount)*int64(f.slotSize())
		if err := f.f.Truncate(newSize); err != nil {
			f.recordCount--
			return 0, fmt.Errorf("extend file: %w", err)
		}
		f.pool.SetFileSize(newSize)
		if err := f.writeHeader(); err != nil {
			f.recordCount--
			_ = f.f.Truncate(int64(f.headerSize) + int64(f.recordCount)*int64(f.slotSize()))
			f.pool.SetFileSize(int64(f.headerSize) + int64(f.recordCount)*int64(f.slotSize()))
			return 0, fmt.Errorf("write header: %w", err)
		}
	}

	offset := int64(f.headerSize) + int64(slotID)*int64(f.slotSize())
	if _, err := f.pool.WriteAt([]byte{1}, offset); err != nil {
		return 0, fmt.Errorf("mark used: %w", err)
	}

	if err := f.writeHeader(); err != nil {
		return 0, fmt.Errorf("write header: %w", err)
	}
	return slotID, nil
}

func (f *File) Free(slotID int) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.f == nil {
		return errors.New("file is closed")
	}
	if slotID < 0 || slotID >= int(f.recordCount) {
		return fmt.Errorf("slotID %d out of range [0, %d)", slotID, f.recordCount)
	}

	offset := int64(f.headerSize) + int64(slotID)*int64(f.slotSize())
	head := make([]byte, 1)
	if _, err := f.pool.ReadAt(head, offset); err != nil {
		return fmt.Errorf("read flags: %w", err)
	}
	if head[0] == 0 {
		return fmt.Errorf("slot %d already free", slotID)
	}

	next := make([]byte, 8)
	schema.PutU64(next, uint64(f.freeHead))
	if _, err := f.pool.WriteAt(next, offset+1); err != nil {
		return fmt.Errorf("write nextFree: %w", err)
	}
	if _, err := f.pool.WriteAt([]byte{0}, offset); err != nil {
		return fmt.Errorf("mark free: %w", err)
	}
	f.freeHead = int64(slotID)

	if err := f.writeHeader(); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	return nil
}
