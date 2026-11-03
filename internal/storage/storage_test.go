package storage_test

import (
	"path/filepath"
	"testing"
	"os"
	"file_db/internal/schema"
	"file_db/internal/storage"
)

func testSchema(t *testing.T) *schema.Schema {
	t.Helper()
	s := &schema.Schema{
		Fields: []schema.Field{
			{Name: "id",   Type: schema.TypeInt64, Size: 8, Flags: schema.FlagPK | schema.FlagAutoInc},
			{Name: "name", Type: schema.TypeStr,   Size: 32},
		},
	}
	if err := schema.Validate(s); err != nil { t.Fatalf("validate: %v", err) }
	return s
}

func newTestFile(t *testing.T) (*storage.File, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	f, err := storage.Create(path, testSchema(t))
	if err != nil { t.Fatalf("create: %v", err) }
	t.Cleanup(func() { f.Close() })
	return f, path
}

func TestCreateOpen(t *testing.T) {
	f, path := newTestFile(t)
	f.Close()

	f2, err := storage.Open(path)
	if err != nil { t.Fatalf("open: %v", err) }
	defer f2.Close()
	if f2.RecordCount() != 0 { t.Fatalf("recordCount: got %d, want 0", f2.RecordCount()) }
}

func TestAllocWriteRead(t *testing.T) {
	f, _ := newTestFile(t)

	slot, err := f.Alloc()
	if err != nil { t.Fatalf("alloc: %v", err) }
	if slot != 0 { t.Fatalf("slot: got %d, want 0", slot) }

	if err := f.Write(slot, map[string]any{"id": int64(1), "name": "Alice"}); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := f.Read(slot)
	if err != nil { t.Fatalf("read: %v", err) }
	if got["name"] != "Alice" { t.Errorf("name: got %v", got["name"]) }
}

func TestReadFreeSlot(t *testing.T) {
	f, _ := newTestFile(t)
	if _, err := f.Read(0); err == nil { t.Fatal("expected error reading non-existent slot") }
}

func TestFreeReuse(t *testing.T) {
	f, _ := newTestFile(t)

	s0, _ := f.Alloc()
	_ = f.Write(s0, map[string]any{"id": int64(1)})
	if err := f.Free(s0); err != nil { t.Fatalf("free: %v", err) }

	s1, _ := f.Alloc()
	if s1 != s0 { t.Fatalf("got slot %d, want reused %d", s1, s0) }
	if f.RecordCount() != 1 { t.Fatalf("recordCount: got %d, want 1", f.RecordCount()) }
}

func TestDoubleFree(t *testing.T) {
	f, _ := newTestFile(t)
	s, _ := f.Alloc()
	_ = f.Free(s)
	if err := f.Free(s); err == nil { t.Fatal("expected error on double free") }
}

func TestOpenCorrupted(t *testing.T) {
	f, path := newTestFile(t)
	_ = f.Close()

	if err := os.Truncate(path, 5); err != nil { t.Fatal(err) }

	if _, err := storage.Open(path); err == nil {
		t.Fatal("expected error opening corrupted file")
	}
}

func TestClear(t *testing.T) {
	f, _ := newTestFile(t)
	for i := 0; i < 3; i++ {
		s, _ := f.Alloc()
		_ = f.Write(s, map[string]any{"id": int64(i + 1)})
	}

	if err := f.Clear(); err != nil { t.Fatalf("clear: %v", err) }
	if f.RecordCount() != 0 { t.Fatalf("recordCount after clear: got %d", f.RecordCount()) }

	if _, err := f.Alloc(); err != nil { t.Fatalf("alloc after clear: %v", err) }
}