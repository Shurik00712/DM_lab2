package index

import (
	"encoding/binary"
	"fmt"
	"path/filepath"
	"sort"
	"testing"
)

func k(n int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(n))
	return b
}

func TestTreeBasic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.btree")
	tr, err := CreateTree(path, 8)
	if err != nil { t.Fatal(err) }
	defer tr.Close()

	if err := tr.Insert(k(10), 100); err != nil { t.Fatal(err) }
	if err := tr.Insert(k(20), 200); err != nil { t.Fatal(err) }
	if err := tr.Insert(k(10), 101); err != nil { t.Fatal(err) }

	got, _ := tr.Lookup(k(10))
	sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
	if len(got) != 2 || got[0] != 100 || got[1] != 101 {
		t.Fatalf("lookup 10: %v", got)
	}
	got, _ = tr.Lookup(k(20))
	if len(got) != 1 || got[0] != 200 { t.Fatalf("lookup 20: %v", got) }

	if err := tr.Delete(k(10), 100); err != nil { t.Fatal(err) }
	got, _ = tr.Lookup(k(10))
	if len(got) != 1 || got[0] != 101 { t.Fatalf("after delete: %v", got) }
}

func TestTreeMany(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.btree")
	tr, err := CreateTree(path, 8)
	if err != nil { t.Fatal(err) }
	defer tr.Close()

	const N = 2000
	for i := 0; i < N; i++ {
		if err := tr.Insert(k(int64(i)), int64(i)); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}
	for i := 0; i < N; i++ {
		got, err := tr.Lookup(k(int64(i)))
		if err != nil { t.Fatalf("lookup %d: %v", i, err) }
		if len(got) != 1 || got[0] != int64(i) {
			t.Fatalf("lookup %d: %v", i, got)
		}
	}

	got, err := tr.RangeLookup(k(500), k(600))
	if err != nil { t.Fatal(err) }
	if len(got) != 101 { t.Fatalf("range size %d, want 101", len(got)) }

	for i := 0; i < N; i += 2 {
		if err := tr.Delete(k(int64(i)), int64(i)); err != nil {
			t.Fatalf("delete %d: %v", i, err)
		}
	}
	for i := 0; i < N; i++ {
		got, _ := tr.Lookup(k(int64(i)))
		want := 1
		if i%2 == 0 { want = 0 }
		if len(got) != want { t.Fatalf("after delete %d: got %v want %d", i, got, want) }
	}
	got, _ = tr.RangeLookup(k(500), k(600))
	if len(got) != 50 { t.Fatalf("range after delete: %d, want 50", len(got)) }
}

func TestTreePersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.btree")
	tr, err := CreateTree(path, 8)
	if err != nil { t.Fatal(err) }
	for i := 0; i < 100; i++ {
		_ = tr.Insert(k(int64(i)), int64(i))
	}
	_ = tr.Close()

	tr2, err := OpenTree(path, 8)
	if err != nil { t.Fatal(err) }
	defer tr2.Close()
	got, _ := tr2.Lookup(k(50))
	if len(got) != 1 || got[0] != 50 { t.Fatalf("persist: %v", got) }
	if err := tr2.Insert(k(500), 500); err != nil { t.Fatal(err) }
	got, _ = tr2.Lookup(k(500))
	if len(got) != 1 { t.Fatalf("after reopen insert: %v", got) }
}

func TestTreeDuplicateKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.btree")
	tr, err := CreateTree(path, 8)
	if err != nil { t.Fatal(err) }
	defer tr.Close()
	for i := 0; i < 500; i++ {
		if err := tr.Insert(k(42), int64(i)); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}
	got, err := tr.Lookup(k(42))
	if err != nil { t.Fatal(err) }
	if len(got) != 500 { t.Fatalf("lookup dup: got %d, want 500", len(got)) }
	seen := make(map[int64]bool)
	for _, s := range got { seen[s] = true }
	for i := 0; i < 500; i++ {
		if !seen[int64(i)] { t.Fatalf("missing slot %d", i) }
	}
}

var _ = fmt.Sprintf