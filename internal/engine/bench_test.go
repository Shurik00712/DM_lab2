package engine_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"file_db/internal/engine"
	"file_db/internal/schema"
	"file_db/internal/storage"
)

func benchSchema(b *testing.B) *schema.Schema {
	b.Helper()
	s := &schema.Schema{
		Fields: []schema.Field{
			{Name: "id", Type: schema.TypeInt64, Size: 8, Flags: schema.FlagPK | schema.FlagAutoInc},
			{Name: "email", Type: schema.TypeStr, Size: 64, Flags: schema.FlagHash},
			{Name: "score", Type: schema.TypeInt64, Size: 8, Flags: schema.FlagBTree},
			{Name: "city", Type: schema.TypeStr, Size: 32},
		},
	}
	if err := schema.Validate(s); err != nil {
		b.Fatalf("schema: %v", err)
	}
	return s
}

func benchSetup(b *testing.B) (*engine.Engine, func()) {
	b.Helper()
	dir := b.TempDir()
	path := filepath.Join(dir, "bench.db")
	s := benchSchema(b)
	db, err := storage.Create(path, s)
	if err != nil {
		b.Fatalf("create: %v", err)
	}
	eng, err := engine.New(db, s, "bench", path)
	if err != nil {
		db.Close()
		b.Fatalf("engine: %v", err)
	}
	return eng, func() { _ = eng.Close() }
}

func benchPopulate(b *testing.B, eng *engine.Engine, n int) {
	b.Helper()
	for i := 0; i < n; i++ {
		if err := eng.Add(map[string]any{
			"email": fmt.Sprintf("u%d@x.com", i),
			"score": int64(i),
			"city":  fmt.Sprintf("c_%d", i),
		}); err != nil {
			b.Fatalf("populate %d: %v", i, err)
		}
	}
}

var benchSizes = []int{1000, 10000, 100000}

// ============================================================
// ADD
// ============================================================

func BenchmarkAdd(b *testing.B) {
	for _, size := range benchSizes {
		size := size
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			eng, cleanup := benchSetup(b)
			defer cleanup()
			benchPopulate(b, eng, size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := eng.Add(map[string]any{
					"email": fmt.Sprintf("new%d@x.com", i),
					"score": int64(size + i),
					"city":  "Novosibirsk",
				}); err != nil {
					b.Fatalf("add: %v", err)
				}
			}
		})
	}
}

// ============================================================
// READ
// ============================================================

func BenchmarkGetByPK(b *testing.B) {
	for _, size := range benchSizes {
		size := size
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			eng, cleanup := benchSetup(b)
			defer cleanup()
			benchPopulate(b, eng, size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := eng.GetByPK(int64(i%size + 1)); err != nil {
					b.Fatalf("get: %v", err)
				}
			}
		})
	}
}

func BenchmarkFindByHash(b *testing.B) {
	for _, size := range benchSizes {
		size := size
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			eng, cleanup := benchSetup(b)
			defer cleanup()
			benchPopulate(b, eng, size)
			keys := make([]string, size)
			for i := range keys {
				keys[i] = fmt.Sprintf("u%d@x.com", i)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := eng.FindBy("email", keys[i%size]); err != nil {
					b.Fatalf("find: %v", err)
				}
			}
		})
	}
}

func BenchmarkFindByBTree(b *testing.B) {
	for _, size := range benchSizes {
		size := size
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			eng, cleanup := benchSetup(b)
			defer cleanup()
			benchPopulate(b, eng, size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := eng.FindBy("score", int64(i%size)); err != nil {
					b.Fatalf("find: %v", err)
				}
			}
		})
	}
}

func BenchmarkFindByScan(b *testing.B) {
	for _, size := range benchSizes {
		size := size
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			eng, cleanup := benchSetup(b)
			defer cleanup()
			benchPopulate(b, eng, size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := eng.FindBy("city", fmt.Sprintf("c_%d", i%size)); err != nil {
					b.Fatalf("find: %v", err)
				}
			}
		})
	}
}

func BenchmarkFindByRangeBTree(b *testing.B) {
	for _, size := range benchSizes {
		size := size
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			eng, cleanup := benchSetup(b)
			defer cleanup()
			benchPopulate(b, eng, size)
			span := int64(size / 10)
			if span < 1 {
				span = 1
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				lo := int64(i) % int64(size)
				if _, err := eng.FindByRange("score", lo, lo+span); err != nil {
					b.Fatalf("range: %v", err)
				}
			}
		})
	}
}

// ============================================================
// UPDATE
// ============================================================

func BenchmarkUpdateByPK(b *testing.B) {
	for _, size := range benchSizes {
		size := size
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			eng, cleanup := benchSetup(b)
			defer cleanup()
			benchPopulate(b, eng, size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				id := int64(i%size + 1)
				if err := eng.Update(id, map[string]any{
					"email": fmt.Sprintf("u%d@x.com", id-1),
					"score": int64(size + i),
					"city":  "Updated",
				}); err != nil {
					b.Fatalf("update: %v", err)
				}
			}
		})
	}
}

// ============================================================
// DELETE
// ============================================================

func BenchmarkDeleteByPK(b *testing.B) {
	for _, size := range benchSizes {
		size := size
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			eng, cleanup := benchSetup(b)
			defer cleanup()
			benchPopulate(b, eng, size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				id := int64(i%size + 1)
				if err := eng.DeleteByPK(id); err != nil {
					b.Fatalf("delete: %v", err)
				}
				b.StopTimer()
				if err := eng.Add(map[string]any{
					"id":    id,
					"email": fmt.Sprintf("u%d@x.com", id-1),
					"score": int64(id - 1),
					"city":  fmt.Sprintf("c_%d", id-1),
				}); err != nil {
					b.Fatalf("re-add: %v", err)
				}
				b.StartTimer()
			}
		})
	}
}

func BenchmarkDeleteByHash(b *testing.B) {
	for _, size := range benchSizes {
		size := size
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			eng, cleanup := benchSetup(b)
			defer cleanup()
			benchPopulate(b, eng, size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				idx := i % size
				email := fmt.Sprintf("u%d@x.com", idx)
				n, err := eng.DeleteBy("email", email)
				if err != nil || n != 1 {
					b.Fatalf("delete: n=%d err=%v", n, err)
				}
				b.StopTimer()
				id := int64(idx + 1)
				if err := eng.Add(map[string]any{
					"id":    id,
					"email": email,
					"score": int64(id - 1),
					"city":  fmt.Sprintf("c_%d", id-1),
				}); err != nil {
					b.Fatalf("re-add: %v", err)
				}
				b.StartTimer()
			}
		})
	}
}

func BenchmarkDeleteByBTree(b *testing.B) {
	for _, size := range benchSizes {
		size := size
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			eng, cleanup := benchSetup(b)
			defer cleanup()
			benchPopulate(b, eng, size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				idx := i % size
				score := int64(idx)
				n, err := eng.DeleteBy("score", score)
				if err != nil || n != 1 {
					b.Fatalf("delete: n=%d err=%v", n, err)
				}
				b.StopTimer()
				id := int64(idx + 1)
				if err := eng.Add(map[string]any{
					"id":    id,
					"email": fmt.Sprintf("u%d@x.com", idx),
					"score": score,
					"city":  fmt.Sprintf("c_%d", idx),
				}); err != nil {
					b.Fatalf("re-add: %v", err)
				}
				b.StartTimer()
			}
		})
	}
}

func BenchmarkDeleteByScan(b *testing.B) {
	for _, size := range benchSizes {
		size := size
		b.Run(fmt.Sprintf("N=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			eng, cleanup := benchSetup(b)
			defer cleanup()
			benchPopulate(b, eng, size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				idx := i % size
				city := fmt.Sprintf("c_%d", idx)
				n, err := eng.DeleteBy("city", city)
				if err != nil || n != 1 {
					b.Fatalf("delete: n=%d err=%v", n, err)
				}
				b.StopTimer()
				id := int64(idx + 1)
				if err := eng.Add(map[string]any{
					"id":    id,
					"email": fmt.Sprintf("u%d@x.com", idx),
					"score": int64(idx),
					"city":  city,
				}); err != nil {
					b.Fatalf("re-add: %v", err)
				}
				b.StartTimer()
			}
		})
	}
}