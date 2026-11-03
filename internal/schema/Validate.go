package schema

import ("errors"
		"fmt")

func Validate(s *Schema) error {
	if s == nil { return errors.New("nil schema") }
	if len(s.Fields) == 0 { return errors.New("empty schema") }

	seen := make(map[string]struct{}, len(s.Fields))
	pkCount := 0
	totalSize := 0
    for i := range s.Fields {
        f := &s.Fields[i]
        if f.Name == "" { return errors.New("empty field name") }
		if len(f.Name) > 32 { return fmt.Errorf("field %q: name > 32 bytes", f.Name) }
		if _, ok := seen[f.Name]; ok { return fmt.Errorf("duplicate field %q", f.Name) }
		seen[f.Name] = struct{}{}

		switch f.Type {
		case TypeInt64, TypeFloat64, TypeDate:
			if f.Size != 8 { return fmt.Errorf("field %q: size %d, want 8", f.Name, f.Size) }
		case TypeBool:
			if f.Size != 1 { return fmt.Errorf("field %q: size %d, want 1", f.Name, f.Size) }
		case TypeStr:
			if f.Size <= 0 { return fmt.Errorf("field %q: size must be > 0", f.Name) }
		default:
			return fmt.Errorf("field %q: unknown type %d", f.Name, f.Type)
		}
        if f.Flags&FlagPK != 0 {
            f.Flags |= FlagNotNull  
            pkCount++
        }
        if f.Flags&FlagAutoInc != 0 && f.Flags&FlagPK == 0 {
			return fmt.Errorf("AutoInc on non-PK %q", f.Name)
		}

		totalSize += int(f.Size)
    }

	if pkCount != 1 { return fmt.Errorf("exactly one PK required, got %d", pkCount) }
	if totalSize > 64*1024 { return fmt.Errorf("RecordSize %d > 64KB", totalSize) }

	s.RecordSize = totalSize
	return nil
}