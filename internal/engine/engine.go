package engine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"

	"file_db/internal/index"
	"file_db/internal/schema"
	"file_db/internal/storage"
)

type Engine struct {
	db        *storage.File
	schema    *schema.Schema
	pkIdx     int
	name      string
	indexBase string

	mu     sync.Mutex
	nextID int64

	pkHash      *index.Hash
	fieldHashes map[string]*index.Hash
	fieldTrees  map[string]*index.Tree
}

func New(db *storage.File, s *schema.Schema, name string, dataPath string) (*Engine, error) {
	pkIdx := -1
	for i, f := range s.Fields {
		if f.Flags&schema.FlagPK != 0 {
			pkIdx = i
			break
		}
	}
	if pkIdx < 0 {
		return nil, errors.New("schema has no PK")
	}
	e := &Engine{
		db:        db,
		schema:    s,
		pkIdx:     pkIdx,
		name:      name,
		indexBase: strings.TrimSuffix(dataPath, ".db"),
	}
	if err := e.rebuildIndexes(); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Engine) keyMaxSize(f schema.Field) int {
	switch f.Type {
	case schema.TypeInt64, schema.TypeDate:
		return 8
	case schema.TypeBool:
		return 1
	case schema.TypeStr:
		return f.Size
	}
	return 8
}

func (e *Engine) hashPath(field string) string  { return e.indexBase + "." + field + ".hash" }
func (e *Engine) treePath(field string) string  { return e.indexBase + "." + field + ".btree" }
func (e *Engine) encodeKey(f schema.Field, v any) []byte {
	switch f.Type {
	case schema.TypeInt64, schema.TypeDate:
		n, err := schema.ToInt64(v)
		if err != nil || n == 0 {
			return nil
		}
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(n))
		return buf
	case schema.TypeStr:
		s, _ := v.(string)
		if s == "" {
			return nil
		}
		return []byte(s)
	case schema.TypeBool:
		b, _ := v.(bool)
		if b {
			return []byte{1}
		}
		return nil
	}
	return nil
}

func (e *Engine) encodeTreeKey(f schema.Field, v any) []byte {
	if v == nil {
		return nil
	}
	switch f.Type {
	case schema.TypeInt64, schema.TypeDate:
		n, err := schema.ToInt64(v)
		if err != nil {
			return nil
		}
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(n))
		return buf
	case schema.TypeStr:
		s, ok := v.(string)
		if !ok {
			if b, ok2 := v.([]byte); ok2 {
				s = string(b)
			} else {
				return nil
			}
		}
		if len(s) > f.Size {
			return nil
		}
		buf := make([]byte, f.Size)
		copy(buf, s)
		return buf
	case schema.TypeBool:
		b, _ := v.(bool)
		if b {
			return []byte{1}
		}
		return []byte{0}
	case schema.TypeFloat64:
		fl, ok := v.(float64)
		if !ok {
			if f32, ok2 := v.(float32); ok2 {
				fl = float64(f32)
			} else {
				return nil
			}
		}
		bits := math.Float64bits(fl)
		if bits&0x8000000000000000 != 0 {
			bits = ^bits
		} else {
			bits |= 0x8000000000000000
		}
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, bits)
		return buf
	}
	return nil
}

func zeroValue(t schema.FieldType) any {
	switch t {
	case schema.TypeInt64, schema.TypeDate:
		return int64(0)
	case schema.TypeFloat64:
		return float64(0)
	case schema.TypeStr:
		return ""
	case schema.TypeBool:
		return false
	}
	return nil
}

func (e *Engine) rebuildIndexes() error {
	if e.pkHash != nil {
		e.pkHash.Close()
		e.pkHash = nil
	}
	for fname, h := range e.fieldHashes {
		h.Close()
		os.Remove(e.hashPath(fname))
	}
	for fname, tr := range e.fieldTrees {
		tr.Close()
		os.Remove(e.treePath(fname))
	}
	e.fieldHashes = make(map[string]*index.Hash)
	e.fieldTrees = make(map[string]*index.Tree)

	pkField := e.schema.Fields[e.pkIdx]
	pkHash, err := index.Create(e.hashPath("pk"), e.keyMaxSize(pkField))
	if err != nil {
		return fmt.Errorf("create pk hash: %w", err)
	}
	e.pkHash = pkHash

	for _, f := range e.schema.Fields {
		if f.Flags&schema.FlagPK != 0 {
			continue
		}
		isHash := f.Flags&schema.FlagHash != 0
		isTree := f.Flags&schema.FlagBTree != 0
		if isHash && isTree {
			return fmt.Errorf("field %q: both FlagHash and FlagBTree set", f.Name)
		}
		if isHash {
			h, err := index.Create(e.hashPath(f.Name), e.keyMaxSize(f))
			if err != nil {
				return fmt.Errorf("create %s hash: %w", f.Name, err)
			}
			e.fieldHashes[f.Name] = h
		}
		if isTree {
			tr, err := index.CreateTree(e.treePath(f.Name), e.keyMaxSize(f))
			if err != nil {
				return fmt.Errorf("create %s btree: %w", f.Name, err)
			}
			e.fieldTrees[f.Name] = tr
		}
	}

	var maxPK int64
	err = e.scan(func(slotID int, v map[string]any) error {
		pkKey := e.encodeKey(pkField, v[pkField.Name])
		if len(pkKey) > 0 {
			if err := pkHash.Insert(pkKey, int64(slotID)); err != nil {
				return err
			}
		}
		if pk, ok := v[pkField.Name].(int64); ok && pk > maxPK {
			maxPK = pk
		}
		for fname, h := range e.fieldHashes {
			f := e.fieldBy(fname)
			key := e.encodeKey(f, v[fname])
			if len(key) > 0 {
				if err := h.Insert(key, int64(slotID)); err != nil {
					return err
				}
			}
		}
		for fname, tr := range e.fieldTrees {
			f := e.fieldBy(fname)
			val := v[fname]
			if val == nil {
				val = zeroValue(f.Type)
			}
			key := e.encodeTreeKey(f, val)
			if len(key) > 0 {
				if err := tr.Insert(key, int64(slotID)); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	e.nextID = maxPK + 1
	return nil
}

func (e *Engine) scan(fn func(slotID int, values map[string]any) error) error {
	n := int(e.db.RecordCount())
	for slotID := 0; slotID < n; slotID++ {
		v, err := e.db.Read(slotID)
		if err != nil {
			continue
		}
		if err := fn(slotID, v); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) DB() *storage.File { return e.db }

func (e *Engine) Add(values map[string]any) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	pkField := e.schema.Fields[e.pkIdx]
	pkName := pkField.Name
	if pkField.Flags&schema.FlagAutoInc != 0 {
		if _, ok := values[pkName]; !ok || values[pkName] == nil {
			values[pkName] = e.nextID
		}
	}

	pkKey := e.encodeKey(pkField, values[pkName])
	if len(pkKey) > 0 {
		slots, err := e.pkHash.Lookup(pkKey)
		if err != nil {
			return err
		}
		if len(slots) > 0 {
			return fmt.Errorf("duplicate PK %v", values[pkName])
		}
	}

	for _, f := range e.schema.Fields {
		if f.Flags&schema.FlagUnique == 0 || f.Flags&schema.FlagPK != 0 {
			continue
		}
		val, ok := values[f.Name]
		if !ok || val == nil || isSentinel(val) {
			continue
		}
		if h, ok := e.fieldHashes[f.Name]; ok {
			key := e.encodeKey(f, val)
			if len(key) == 0 {
				continue
			}
			dups, err := h.Lookup(key)
			if err != nil {
				return err
			}
			if len(dups) > 0 {
				return fmt.Errorf("duplicate %s: %v", f.Name, val)
			}
		} else if tr, ok := e.fieldTrees[f.Name]; ok {
			key := e.encodeTreeKey(f, val)
			if len(key) == 0 {
				continue
			}
			dups, err := tr.Lookup(key)
			if err != nil {
				return err
			}
			if len(dups) > 0 {
				return fmt.Errorf("duplicate %s: %v", f.Name, val)
			}
		} else {
			var dup bool
			_ = e.scan(func(_ int, v map[string]any) error {
				if valuesEqual(v[f.Name], val) {
					dup = true
					return errStopIteration
				}
				return nil
			})
			if dup {
				return fmt.Errorf("duplicate %s: %v", f.Name, val)
			}
		}
	}

	slotID, err := e.db.Alloc()
	if err != nil {
		return fmt.Errorf("alloc: %w", err)
	}
	if err := e.db.Write(slotID, values); err != nil {
		_ = e.db.Free(slotID)
		return fmt.Errorf("write: %w", err)
	}

	if len(pkKey) > 0 {
		if err := e.pkHash.Insert(pkKey, int64(slotID)); err != nil {
			_ = e.db.Free(slotID)
			return fmt.Errorf("pk hash insert: %w", err)
		}
	}
	for fname, h := range e.fieldHashes {
		f := e.fieldBy(fname)
		key := e.encodeKey(f, values[fname])
		if len(key) == 0 {
			continue
		}
		if err := h.Insert(key, int64(slotID)); err != nil {
			if len(pkKey) > 0 {
				_ = e.pkHash.Delete(pkKey, int64(slotID))
			}
			_ = e.db.Free(slotID)
			return fmt.Errorf("%s hash insert: %w", fname, err)
		}
	}
	for fname, tr := range e.fieldTrees {
		f := e.fieldBy(fname)
		val := values[fname]
		if val == nil {
			val = zeroValue(f.Type)
		}
		key := e.encodeTreeKey(f, val)
		if len(key) == 0 {
			continue
		}
		if err := tr.Insert(key, int64(slotID)); err != nil {
			_ = e.db.Free(slotID)
			return fmt.Errorf("%s btree insert: %w", fname, err)
		}
	}
	if pkField.Flags&schema.FlagAutoInc != 0 {
		e.nextID++
	}
	return nil
}

func isSentinel(v any) bool {
	switch x := v.(type) {
	case string:
		return x == ""
	case int64:
		return x == 0
	case int:
		return x == 0
	case float64:
		return x == 0
	case bool:
		return !x
	}
	return false
}

func valuesEqual(a, b any) bool {
	switch av := a.(type) {
	case int64:
		switch bv := b.(type) {
		case int64:
			return av == bv
		case int:
			return av == int64(bv)
		case int32:
			return av == int64(bv)
		case uint64:
			return av == int64(bv)
		case float64:
			return bv == math.Trunc(bv) && av == int64(bv)
		case string:
			n, err := strconv.ParseInt(bv, 10, 64)
			return err == nil && av == n
		}
	case int:
		return valuesEqual(int64(av), b)
	case float64:
		switch bv := b.(type) {
		case float64:
			return av == bv
		case float32:
			return av == float64(bv)
		}
	case string:
		switch bv := b.(type) {
		case string:
			return av == bv
		case int64:
			n, err := strconv.ParseInt(av, 10, 64)
			return err == nil && n == bv
		}
	case bool:
		if bv, ok := b.(bool); ok {
			return av == bv
		}
	}
	return false
}

func (e *Engine) GetByPK(pk any) (map[string]any, error) {
	pkField := e.schema.Fields[e.pkIdx]
	pk = e.coercePK(pk)
	key := e.encodeKey(pkField, pk)
	if len(key) == 0 {
		return nil, fmt.Errorf("not found: %v", pk)
	}
	slots, err := e.pkHash.Lookup(key)
	if err != nil {
		return nil, err
	}
	if len(slots) == 0 {
		return nil, fmt.Errorf("not found: %v", pk)
	}
	return e.db.Read(int(slots[0]))
}

func (e *Engine) deleteSlotLocked(slotID int) error {
	v, err := e.db.Read(slotID)
	if err != nil {
		return err
	}
	pkField := e.schema.Fields[e.pkIdx]
	pkKey := e.encodeKey(pkField, v[pkField.Name])
	if len(pkKey) > 0 {
		if err := e.pkHash.Delete(pkKey, int64(slotID)); err != nil {
			return fmt.Errorf("pk hash delete: %w", err)
		}
	}
	for fname, h := range e.fieldHashes {
		f := e.fieldBy(fname)
		k := e.encodeKey(f, v[fname])
		if len(k) > 0 {
			if err := h.Delete(k, int64(slotID)); err != nil {
				return fmt.Errorf("%s hash delete: %w", fname, err)
			}
		}
	}
	for fname, tr := range e.fieldTrees {
		f := e.fieldBy(fname)
		val := v[fname]
		if val == nil {
			val = zeroValue(f.Type)
		}
		k := e.encodeTreeKey(f, val)
		if len(k) > 0 {
			if err := tr.Delete(k, int64(slotID)); err != nil {
				return fmt.Errorf("%s btree delete: %w", fname, err)
			}
		}
	}
	return e.db.Free(slotID)
}

func (e *Engine) DeleteByPK(pk any) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	pkField := e.schema.Fields[e.pkIdx]
	pk = e.coercePK(pk)
	key := e.encodeKey(pkField, pk)
	if len(key) == 0 {
		return fmt.Errorf("not found: %v", pk)
	}
	slots, err := e.pkHash.Lookup(key)
	if err != nil {
		return err
	}
	if len(slots) == 0 {
		return fmt.Errorf("not found: %v", pk)
	}
	return e.deleteSlotLocked(int(slots[0]))
}

func (e *Engine) DeleteBy(field string, value any) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.hasField(field) {
		return 0, fmt.Errorf("unknown field %q", field)
	}

	if field == e.PKFieldName() {
		pkField := e.schema.Fields[e.pkIdx]
		pk := e.coercePK(value)
		key := e.encodeKey(pkField, pk)
		if len(key) == 0 {
			return 0, nil
		}
		slots, err := e.pkHash.Lookup(key)
		if err != nil {
			return 0, err
		}
		deleted := 0
		for _, sid := range slots {
			if err := e.deleteSlotLocked(int(sid)); err != nil {
				return deleted, err
			}
			deleted++
		}
		return deleted, nil
	}

	if h, ok := e.fieldHashes[field]; ok {
		f := e.fieldBy(field)
		key := e.encodeKey(f, value)
		if len(key) == 0 {
			return e.deleteByScanLocked(field, value)
		}
		slots, err := h.Lookup(key)
		if err != nil {
			return 0, err
		}
		deleted := 0
		for _, sid := range slots {
			if err := e.deleteSlotLocked(int(sid)); err != nil {
				return deleted, err
			}
			deleted++
		}
		return deleted, nil
	}

	if tr, ok := e.fieldTrees[field]; ok {
		f := e.fieldBy(field)
		key := e.encodeTreeKey(f, value)
		if len(key) == 0 {
			return e.deleteByScanLocked(field, value)
		}
		slots, err := tr.Lookup(key)
		if err != nil {
			return 0, err
		}
		deleted := 0
		for _, sid := range slots {
			if err := e.deleteSlotLocked(int(sid)); err != nil {
				return deleted, err
			}
			deleted++
		}
		return deleted, nil
	}

	return e.deleteByScanLocked(field, value)
}

func (e *Engine) deleteByScanLocked(field string, value any) (int, error) {
	var slots []int
	_ = e.scan(func(slotID int, v map[string]any) error {
		if valuesEqual(v[field], value) {
			slots = append(slots, slotID)
		}
		return nil
	})
	deleted := 0
	for _, sid := range slots {
		if err := e.deleteSlotLocked(sid); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}

func (e *Engine) hasField(field string) bool {
	for _, f := range e.schema.Fields {
		if f.Name == field {
			return true
		}
	}
	return false
}

func (e *Engine) Update(pk any, values map[string]any) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	pkField := e.schema.Fields[e.pkIdx]
	pk = e.coercePK(pk)
	key := e.encodeKey(pkField, pk)
	if len(key) == 0 {
		return fmt.Errorf("not found: %v", pk)
	}
	slots, err := e.pkHash.Lookup(key)
	if err != nil {
		return err
	}
	if len(slots) == 0 {
		return fmt.Errorf("not found: %v", pk)
	}
	slotID := int(slots[0])

	old, err := e.db.Read(slotID)
	if err != nil {
		return err
	}
	merged := make(map[string]any, len(old))
	for k, v := range old {
		merged[k] = v
	}
	for k, v := range values {
		merged[k] = v
	}
	merged[pkField.Name] = pk
	values = merged

	for fname, h := range e.fieldHashes {
		f := e.fieldBy(fname)
		oldKey := e.encodeKey(f, old[fname])
		newKey := e.encodeKey(f, values[fname])
		if bytes.Equal(oldKey, newKey) {
			continue
		}
		if len(oldKey) > 0 {
			if err := h.Delete(oldKey, int64(slotID)); err != nil {
				return fmt.Errorf("hash %s delete: %w", fname, err)
			}
		}
		if len(newKey) > 0 {
			if err := h.Insert(newKey, int64(slotID)); err != nil {
				return fmt.Errorf("hash %s insert: %w", fname, err)
			}
		}
	}
	for fname, tr := range e.fieldTrees {
		f := e.fieldBy(fname)
		oldVal := old[fname]
		if oldVal == nil {
			oldVal = zeroValue(f.Type)
		}
		newVal := values[fname]
		if newVal == nil {
			newVal = zeroValue(f.Type)
		}
		oldKey := e.encodeTreeKey(f, oldVal)
		newKey := e.encodeTreeKey(f, newVal)
		if bytes.Equal(oldKey, newKey) {
			continue
		}
		if len(oldKey) > 0 {
			if err := tr.Delete(oldKey, int64(slotID)); err != nil {
				return fmt.Errorf("btree %s delete: %w", fname, err)
			}
		}
		if len(newKey) > 0 {
			if err := tr.Insert(newKey, int64(slotID)); err != nil {
				return fmt.Errorf("btree %s insert: %w", fname, err)
			}
		}
	}

	return e.db.Write(slotID, values)
}

func (e *Engine) fieldBy(name string) schema.Field {
	for _, f := range e.schema.Fields {
		if f.Name == name {
			return f
		}
	}
	return schema.Field{}
}

func (e *Engine) FieldByName(name string) (schema.Field, bool) {
	for _, f := range e.schema.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return schema.Field{}, false
}

func (e *Engine) FindBy(field string, value any) ([]map[string]any, error) {
	if !e.hasField(field) {
		return nil, fmt.Errorf("unknown field %q", field)
	}
	if field == e.PKFieldName() {
		pkField := e.schema.Fields[e.pkIdx]
		pk := e.coercePK(value)
		key := e.encodeKey(pkField, pk)
		if len(key) == 0 {
			return []map[string]any{}, nil
		}
		slots, err := e.pkHash.Lookup(key)
		if err != nil {
			return nil, err
		}
		if len(slots) == 0 {
			return []map[string]any{}, nil
		}
		v, err := e.db.Read(int(slots[0]))
		if err != nil {
			return nil, err
		}
		return []map[string]any{v}, nil
	}

	if h, ok := e.fieldHashes[field]; ok {
		f := e.fieldBy(field)
		key := e.encodeKey(f, value)
		if len(key) == 0 {
			return e.scanFind(field, value)
		}
		slots, err := h.Lookup(key)
		if err != nil {
			return nil, err
		}
		out := make([]map[string]any, 0, len(slots))
		for _, sid := range slots {
			if v, err := e.db.Read(int(sid)); err == nil {
				out = append(out, v)
			}
		}
		return out, nil
	}

	if tr, ok := e.fieldTrees[field]; ok {
		f := e.fieldBy(field)
		key := e.encodeTreeKey(f, value)
		if len(key) == 0 {
			return e.scanFind(field, value)
		}
		slots, err := tr.Lookup(key)
		if err != nil {
			return nil, err
		}
		out := make([]map[string]any, 0, len(slots))
		for _, sid := range slots {
			if v, err := e.db.Read(int(sid)); err == nil {
				out = append(out, v)
			}
		}
		return out, nil
	}
	return e.scanFind(field, value)
}

func (e *Engine) FindByRange(field string, lo, hi any) ([]map[string]any, error) {
	if !e.hasField(field) {
		return nil, fmt.Errorf("unknown field %q", field)
	}
	tr, ok := e.fieldTrees[field]
	if !ok {
		return nil, fmt.Errorf("field %q is not BTree-indexed", field)
	}
	f := e.fieldBy(field)
	var loKey, hiKey []byte
	if lo != nil {
		loKey = e.encodeTreeKey(f, lo)
	}
	if hi != nil {
		hiKey = e.encodeTreeKey(f, hi)
	}
	slots, err := tr.RangeLookup(loKey, hiKey)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(slots))
	for _, sid := range slots {
		if v, err := e.db.Read(int(sid)); err == nil {
			out = append(out, v)
		}
	}
	return out, nil
}

func (e *Engine) scanFind(field string, value any) ([]map[string]any, error) {
	var out []map[string]any
	err := e.scan(func(_ int, v map[string]any) error {
		if valuesEqual(v[field], value) {
			out = append(out, v)
		}
		return nil
	})
	return out, err
}

func (e *Engine) Count() (int, error) {
	n := 0
	err := e.scan(func(_ int, _ map[string]any) error {
		n++
		return nil
	})
	return n, err
}

func (e *Engine) FindAll() ([]map[string]any, error) {
	var out []map[string]any
	err := e.scan(func(_ int, v map[string]any) error {
		out = append(out, v)
		return nil
	})
	return out, err
}

func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, h := range e.fieldHashes {
		_ = h.Close()
	}
	for _, tr := range e.fieldTrees {
		_ = tr.Close()
	}
	if e.pkHash != nil {
		_ = e.pkHash.Close()
	}
	return e.db.Close()
}

func (e *Engine) Sync() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, h := range e.fieldHashes {
		_ = h.Sync()
	}
	for _, tr := range e.fieldTrees {
		_ = tr.Sync()
	}
	if e.pkHash != nil {
		_ = e.pkHash.Sync()
	}
	return e.db.Sync()
}

func (e *Engine) Reset() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.db.Clear(); err != nil {
		return err
	}
	return e.rebuildIndexes()
}

func (e *Engine) coercePK(pk any) any {
	f := e.schema.Fields[e.pkIdx]
	if f.Type != schema.TypeInt64 {
		return pk
	}
	switch x := pk.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		if x == math.Trunc(x) {
			return int64(x)
		}
	case string:
		if n, err := strconv.ParseInt(x, 10, 64); err == nil {
			return n
		}
	}
	return pk
}

func (e *Engine) Schema() *schema.Schema { return e.schema }

func (e *Engine) PKFieldName() string { return e.schema.Fields[e.pkIdx].Name }