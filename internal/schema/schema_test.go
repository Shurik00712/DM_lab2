package schema_test

import (
	"bytes"
	"testing"

	"file_db/internal/schema"
)

func testSchema(t *testing.T) *schema.Schema {
	t.Helper()
	s := &schema.Schema{
		Fields: []schema.Field{
			{Name: "id",    Type: schema.TypeInt64, Size: 8, Flags: schema.FlagPK | schema.FlagAutoInc},
			{Name: "name",  Type: schema.TypeStr,   Size: 32, Flags: schema.FlagNotNull},
			{Name: "score", Type: schema.TypeFloat64, Size: 8},
			{Name: "active", Type: schema.TypeBool, Size: 1},
			{Name: "date",  Type: schema.TypeDate, Size: 8},
		},
	}
	if err := schema.Validate(s); err != nil {
		t.Fatalf("validate: %v", err)
	}
	return s
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	s := testSchema(t)
	values := map[string]any{
		"id":     int64(42),
		"name":   "Go",
		"score":  3.14,
		"active": true,
		"date":   int64(1700000000),
	}

	buf, err := schema.Encode(s, values)
	if err != nil { t.Fatalf("encode: %v", err) }
	if len(buf) != s.RecordSize {
		t.Fatalf("len = %d, want %d", len(buf), s.RecordSize)
	}

	got, err := schema.Decode(s, buf)
	if err != nil { t.Fatalf("decode: %v", err) }

	for k, want := range values {
		if got[k] != want {
			t.Errorf("%s: got %v, want %v", k, got[k], want)
		}
	}
}

func TestEncodeSentinelForNil(t *testing.T) {
	s := testSchema(t)
	buf, err := schema.Encode(s, map[string]any{"id": int64(1)})
	if err != nil { t.Fatalf("encode: %v", err) }

	got, _ := schema.Decode(s, buf)
	if got["name"] != "" { t.Errorf("name: got %v, want \"\"", got["name"]) }
	if got["score"] != float64(0) { t.Errorf("score: got %v, want 0", got["score"]) }
	if got["active"] != false { t.Errorf("active: got %v, want false", got["active"]) }
}

func TestEncodeStringTooLong(t *testing.T) {
	s := testSchema(t)
	values := map[string]any{"id": int64(1), "name": string(bytes.Repeat([]byte("x"), 33))}
	if _, err := schema.Encode(s, values); err == nil {
		t.Fatal("expected error for string longer than field size")
	}
}

func TestEncodeStringWithNUL(t *testing.T) {
	s := testSchema(t)
	values := map[string]any{"id": int64(1), "name": "a\x00b"}
	if _, err := schema.Encode(s, values); err == nil {
		t.Fatal("expected error for NUL in string")
	}
}

func TestValidateNoPK(t *testing.T) {
	s := &schema.Schema{
		Fields: []schema.Field{{Name: "name", Type: schema.TypeStr, Size: 32}},
	}
	if err := schema.Validate(s); err == nil {
		t.Fatal("expected error for schema without PK")
	}
}

func TestHeaderRoundTrip(t *testing.T) {
	s := testSchema(t)
	buf, err := schema.EncodeHeader(s, 123, -1)
	if err != nil { t.Fatalf("encode header: %v", err) }

	got, rc, fh, err := schema.DecodeHeader(buf)
	if err != nil { t.Fatalf("decode header: %v", err) }
	if rc != 123 { t.Errorf("recordCount: got %d, want 123", rc) }
	if fh != -1 { t.Errorf("freeHead: got %d, want -1", fh) }
	if got.RecordSize != s.RecordSize { t.Errorf("recordSize: got %d, want %d", got.RecordSize, s.RecordSize) }
	if len(got.Fields) != len(s.Fields) { t.Errorf("fields: got %d, want %d", len(got.Fields), len(s.Fields)) }
}

func TestDecodeHeaderTruncated(t *testing.T) {
	s := testSchema(t)
	buf, _ := schema.EncodeHeader(s, 0, -1)
	_, _, _, err := schema.DecodeHeader(buf[:10])
	if err == nil { t.Fatal("expected error on truncated header") }
}

func TestDecodeHeaderBadMagic(t *testing.T) {
	s := testSchema(t)
	buf, _ := schema.EncodeHeader(s, 0, -1)
	buf[0] = 'X'
	_, _, _, err := schema.DecodeHeader(buf)
	if err == nil { t.Fatal("expected error on bad magic") }
}