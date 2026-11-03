package engine_test

import (
	"path/filepath"
	"testing"

	"file_db/internal/engine"
	"file_db/internal/schema"
	"file_db/internal/storage"
)

func testSchema(t *testing.T) *schema.Schema {
	t.Helper()
	s := &schema.Schema{
		Fields: []schema.Field{
			{Name: "id",    Type: schema.TypeInt64, Size: 8,  Flags: schema.FlagPK | schema.FlagAutoInc},
			{Name: "email", Type: schema.TypeStr,   Size: 64, Flags: schema.FlagUnique},
			{Name: "city",  Type: schema.TypeStr,   Size: 32, Flags: schema.FlagHash},
		},
	}
	if err := schema.Validate(s); err != nil { t.Fatalf("validate: %v", err) }
	return s
}

func setupEngine(t *testing.T) *engine.Engine {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	s := testSchema(t)

	db, err := storage.Create(path, s)
	if err != nil { t.Fatalf("create: %v", err) }

	eng, err := engine.New(db, s, "test", path)
	if err != nil {
		db.Close()
		t.Fatalf("engine: %v", err)
	}
	t.Cleanup(func() { _ = eng.Close() })   
	return eng
}

func TestAddAutoInc(t *testing.T) {
	e := setupEngine(t)

	if err := e.Add(map[string]any{"email": "a@x.com", "city": "Kazan"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := e.Add(map[string]any{"email": "b@x.com", "city": "Moscow"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	v, err := e.GetByPK(int64(1))
	if err != nil { t.Fatalf("get 1: %v", err) }
	if v["email"] != "a@x.com" { t.Errorf("id=1: email = %v", v["email"]) }

	v, err = e.GetByPK(int64(2))
	if err != nil { t.Fatalf("get 2: %v", err) }
	if v["email"] != "b@x.com" { t.Errorf("id=2: email = %v", v["email"]) }
}

func TestAddDuplicatePK(t *testing.T) {
	e := setupEngine(t)
	_ = e.Add(map[string]any{"email": "a@x.com"})
	err := e.Add(map[string]any{"id": int64(1), "email": "b@x.com"})
	if err == nil { t.Fatal("expected error on duplicate PK") }
}

func TestAddDuplicateUnique(t *testing.T) {
	e := setupEngine(t)
	_ = e.Add(map[string]any{"email": "same@x.com"})
	err := e.Add(map[string]any{"email": "same@x.com"})
	if err == nil { t.Fatal("expected error on duplicate unique") }
}

func TestAddEmptyUniqueAllowed(t *testing.T) {
	e := setupEngine(t)
	if err := e.Add(map[string]any{"city": "A"}); err != nil { t.Fatalf("add 1: %v", err) }
	if err := e.Add(map[string]any{"city": "B"}); err != nil { t.Fatalf("add 2: %v", err) }
}

func TestGetNotFound(t *testing.T) {
	e := setupEngine(t)
	if _, err := e.GetByPK(int64(999)); err == nil {
		t.Fatal("expected error on not found")
	}
}

func TestUpdate(t *testing.T) {
	e := setupEngine(t)
	_ = e.Add(map[string]any{"email": "a@x.com", "city": "Old"})

	if err := e.Update(int64(1), map[string]any{"email": "a@x.com", "city": "New"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	v, _ := e.GetByPK(int64(1))
	if v["city"] != "New" { t.Errorf("city: got %v, want New", v["city"]) }
}

func TestDeleteByPK(t *testing.T) {
	e := setupEngine(t)
	_ = e.Add(map[string]any{"email": "a@x.com"})
	_ = e.Add(map[string]any{"email": "b@x.com"})

	if err := e.DeleteByPK(int64(1)); err != nil { t.Fatalf("delete: %v", err) }
	if _, err := e.GetByPK(int64(1)); err == nil { t.Fatal("expected not found after delete") }

	n, _ := e.Count()
	if n != 1 { t.Errorf("count: got %d, want 1", n) }
}

func TestDeleteByNonKey(t *testing.T) {
	e := setupEngine(t)
	_ = e.Add(map[string]any{"email": "a@x.com", "city": "Kazan"})
	_ = e.Add(map[string]any{"email": "b@x.com", "city": "Kazan"})
	_ = e.Add(map[string]any{"email": "c@x.com", "city": "Moscow"})

	n, err := e.DeleteBy("city", "Kazan")
	if err != nil { t.Fatalf("deleteBy: %v", err) }
	if n != 2 { t.Errorf("deleted: got %d, want 2", n) }

	remaining, _ := e.Count()
	if remaining != 1 { t.Errorf("remaining: got %d, want 1", remaining) }
}

func TestFindByAllMatches(t *testing.T) {
	e := setupEngine(t)
	_ = e.Add(map[string]any{"email": "a@x.com", "city": "Kazan"})
	_ = e.Add(map[string]any{"email": "b@x.com", "city": "Kazan"})
	_ = e.Add(map[string]any{"email": "c@x.com", "city": "Moscow"})

	rows, err := e.FindBy("city", "Kazan")
	if err != nil { t.Fatalf("findBy: %v", err) }
	if len(rows) != 2 { t.Errorf("got %d rows, want 2", len(rows)) }
}

func TestPersistenceNextID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	s := testSchema(t)

	db, err := storage.Create(path, s)
	if err != nil { t.Fatalf("create: %v", err) }
	e, err := engine.New(db, s, "test", path)
	if err != nil {
		db.Close()
		t.Fatalf("engine: %v", err)
	}

	_ = e.Add(map[string]any{"email": "a@x.com"})
	_ = e.Add(map[string]any{"email": "b@x.com"})
	_ = e.Add(map[string]any{"email": "c@x.com"})

	if err := e.Close(); err != nil {             
		t.Fatalf("close e: %v", err)
	}

	db2, err := storage.Open(path)
	if err != nil { t.Fatalf("open: %v", err) }
	e2, err := engine.New(db2, s, "test", path)
	if err != nil {
		db2.Close()
		t.Fatalf("engine: %v", err)
	}
	defer e2.Close()                              

	_ = e2.Add(map[string]any{"email": "d@x.com"})
	v, err := e2.GetByPK(int64(4))
	if err != nil { t.Fatalf("get id=4: %v", err) }
	if v["email"] != "d@x.com" { t.Errorf("email = %v", v["email"]) }
}