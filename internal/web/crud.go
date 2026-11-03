package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"file_db/internal/schema"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
}

func readJSON(r *http.Request) (map[string]any, error) {
	var v map[string]any
	err := json.NewDecoder(r.Body).Decode(&v)
	return v, err
}

func crudListCreate(ops entityOps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			rows, err := ops.FindAll()
			if err != nil {
				writeErr(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"fields": ops.Fields, "rows": rows})
		case http.MethodPost:
			v, err := readJSON(r)
			if err != nil {
				writeErr(w, err)
				return
			}
			if err := ops.Add(v); err != nil {
				writeErr(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func crudByID(ops entityOps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/"+ops.Name+"/")
		if id == "" {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			v, err := ops.Get(id)
			if err != nil {
				writeErr(w, err)
				return
			}
			writeJSON(w, http.StatusOK, v)
		case http.MethodPut, http.MethodPatch:
			v, err := readJSON(r)
			if err != nil {
				writeErr(w, err)
				return
			}
			if err := ops.Update(id, v); err != nil {
				writeErr(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		case http.MethodDelete:
			if err := ops.Delete(id); err != nil {
				writeErr(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func (s *Server) handleSearch(ops entityOps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		field := r.URL.Query().Get("field")
		valueStr := r.URL.Query().Get("value")
		if field == "" {
			http.Error(w, "field required", http.StatusBadRequest)
			return
		}
		eng := s.engineOf(ops.Name)
		if eng == nil {
			http.Error(w, "unknown entity", http.StatusBadRequest)
			return
		}
		f, ok := eng.FieldByName(field)
		if !ok {
			writeErr(w, fmt.Errorf("unknown field %q", field))
			return
		}
		value, err := coerceQueryValue(f, valueStr)
		if err != nil {
			writeErr(w, err)
			return
		}
		rows, err := eng.FindBy(field, value)
		if err != nil {
			writeErr(w, err)
			return
		}
		if rows == nil {
			rows = []map[string]any{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"rows": rows})
	}
}

// GET /api/{entity}/range?field=...&lo=...&hi=... 
func (s *Server) handleRange(ops entityOps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		field := r.URL.Query().Get("field")
		if field == "" {
			http.Error(w, "field required", http.StatusBadRequest)
			return
		}
		eng := s.engineOf(ops.Name)
		if eng == nil {
			http.Error(w, "unknown entity", http.StatusBadRequest)
			return
		}
		f, ok := eng.FieldByName(field)
		if !ok {
			writeErr(w, fmt.Errorf("unknown field %q", field))
			return
		}
		if f.Flags&schema.FlagBTree == 0 {
			writeErr(w, fmt.Errorf("field %q is not BTree-indexed", field))
			return
		}
		var loVal, hiVal any
		if s := r.URL.Query().Get("lo"); s != "" {
			v, err := coerceQueryValue(f, s)
			if err != nil {
				writeErr(w, err)
				return
			}
			loVal = v
		}
		if s := r.URL.Query().Get("hi"); s != "" {
			v, err := coerceQueryValue(f, s)
			if err != nil {
				writeErr(w, err)
				return
			}
			hiVal = v
		}
		rows, err := eng.FindByRange(field, loVal, hiVal)
		if err != nil {
			writeErr(w, err)
			return
		}
		if rows == nil {
			rows = []map[string]any{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"rows": rows})
	}
}

func (s *Server) handleDeleteByField(ops entityOps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete && r.Method != http.MethodPost {
			http.Error(w, "DELETE or POST only", http.StatusMethodNotAllowed)
			return
		}
		field := r.URL.Query().Get("field")
		valueStr := r.URL.Query().Get("value")
		if field == "" {
			http.Error(w, "field required", http.StatusBadRequest)
			return
		}
		eng := s.engineOf(ops.Name)
		if eng == nil {
			http.Error(w, "unknown entity", http.StatusBadRequest)
			return
		}
		f, ok := eng.FieldByName(field)
		if !ok {
			writeErr(w, fmt.Errorf("unknown field %q", field))
			return
		}
		value, err := coerceQueryValue(f, valueStr)
		if err != nil {
			writeErr(w, err)
			return
		}
		rows, err := eng.FindBy(field, value)
		if err != nil {
			writeErr(w, err)
			return
		}
		deleted := 0
		var errs []string
		for _, row := range rows {
			if err := ops.Delete(row["id"]); err != nil {
				errs = append(errs, err.Error())
				continue
			}
			deleted++
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": deleted, "errors": errs})
	}
}

func coerceQueryValue(f schema.Field, s string) (any, error) {
	switch f.Type {
	case schema.TypeInt64, schema.TypeDate:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("field %q: %q is not a valid integer", f.Name, s)
		}
		return n, nil
	case schema.TypeFloat64:
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, fmt.Errorf("field %q: %q is not a valid number", f.Name, s)
		}
		return v, nil
	case schema.TypeBool:
		switch s {
		case "true", "1", "да":
			return true, nil
		case "false", "0", "нет":
			return false, nil
		}
		return nil, fmt.Errorf("field %q: %q is not a bool", f.Name, s)
	case schema.TypeStr:
		return s, nil
	}
	return nil, fmt.Errorf("field %q: unsupported type", f.Name)
}