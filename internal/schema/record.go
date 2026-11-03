package schema

import (
	"fmt"
	"encoding/binary"
	"math"
	"strings"
	"strconv"
	"encoding/json"
)




func ToInt64(v any) (int64, error) {
    switch x := v.(type) {
    case int:     return int64(x), nil
    case int8:    return int64(x), nil
    case int16:   return int64(x), nil
    case int32:   return int64(x), nil
    case int64:   return x, nil
    case uint:    if uint64(x) > math.MaxInt64 { return 0, errOverflow }; return int64(x), nil
    case uint8:   return int64(x), nil
    case uint16:  return int64(x), nil
    case uint32:  return int64(x), nil
    case uint64:  if x > math.MaxInt64 { return 0, errOverflow }; return int64(x), nil
    case float64:
        if x != math.Trunc(x) {
            return 0, fmt.Errorf("got non-integer float %v", x)
        }
        if x > math.MaxInt64 || x < math.MinInt64 {
            return 0, errOverflow
        }
        return int64(x), nil
    case float32:
        f := float64(x)
        if f != math.Trunc(f) {
            return 0, fmt.Errorf("got non-integer float %v", x)
        }
        return int64(f), nil
	case string:                                  
        n, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
        if err != nil {
            return 0, fmt.Errorf("got non-numeric string %q", x)
        }
        return n, nil
	case json.Number:                             
        n, err := x.Int64()
        if err != nil {
            return 0, fmt.Errorf("got non-integer json.Number %q", x)
        }
        return n, nil
    default:
        return 0, fmt.Errorf("got %T, want integer", v)
    }
}

func Encode(schema *Schema, values map[string]any) ([]byte, error) {
	buf := make([]byte, schema.RecordSize)
	offset := 0
	for _, field := range schema.Fields {
		v, ok := values[field.Name]
		if !ok || v == nil {
			for i := offset; i < offset+field.Size; i++ {
				buf[i] = 0
			}
			offset += int(field.Size)
			continue
		}
		switch field.Type {
			case TypeInt64, TypeDate:
				n, err := ToInt64(v)
				if err != nil {
					return nil, fmt.Errorf("field %q: %w", field.Name, err)
				}
				binary.LittleEndian.PutUint64(buf[offset:offset+8], uint64(n))
			case TypeFloat64:
				f, ok := v.(float64)
				if !ok {
					if f32, ok := v.(float32); ok {
						f = float64(f32)
					} else {
						return nil, fmt.Errorf("field %q: got %T, want float", field.Name, v)
					}
				}
				binary.LittleEndian.PutUint64(buf[offset:offset+8], math.Float64bits(f))
			case TypeStr:
				var s string
				switch x := v.(type) {
				case string: s = x
				case []byte: s = string(x)
				default:     return nil, fmt.Errorf("field %q: got %T, want string/[]byte", field.Name, v)
				}

				if len(s) > field.Size {
					return nil, fmt.Errorf("field %q: string length %d exceeds size %d", field.Name, len(s), field.Size)
				}
				if strings.IndexByte(s, 0) >= 0 {
					return nil, fmt.Errorf("field %q: string contains NUL byte", field.Name)
				}
				copy(buf[offset:offset+field.Size], s)
				for i := offset + len(s); i < offset+field.Size; i++ {
					buf[i] = 0
				}
			case TypeBool:
				b, ok := v.(bool)
				if !ok {
					return nil, fmt.Errorf("field %q: got %T, want bool", field.Name, v)
				}
				if b { buf[offset] = 1 } else { buf[offset] = 0 }
			default:
				return nil, fmt.Errorf("field %q: unknown type %d", field.Name, field.Type)
		}
		offset += int(field.Size)
	}
	return buf, nil
}

func Decode(schema *Schema, data []byte) (map[string]any, error) {
	if len(data) != schema.RecordSize {
        return nil, fmt.Errorf("payload length %d != recordSize %d", len(data), schema.RecordSize)
	}
	values := make(map[string]any, len(schema.Fields))
	offset := 0
	for _, field := range schema.Fields {
		if offset+field.Size > len(data) {
            return nil, fmt.Errorf("field %q: truncated", field.Name)
        }
		val := data[offset:offset+field.Size]
		switch field.Type {
			case TypeInt64, TypeDate:
				n := int64(binary.LittleEndian.Uint64(val))
				values[field.Name] = n
			case TypeBool:
    			values[field.Name] = val[0] != 0
			case TypeStr:
				n := field.Size
				for i := 0; i < field.Size; i++ {
					if val[i] == 0{
						n = i
						break
					}
						
				}
				values[field.Name] = string(val[:n])
			case TypeFloat64:
				bits := binary.LittleEndian.Uint64(val)   
    			values[field.Name] = math.Float64frombits(bits)
			default:
				return nil, fmt.Errorf("field %q: unknown type %d", field.Name, field.Type)
		}
		
		offset += int(field.Size)
	}
	if offset != schema.RecordSize {
        return nil, fmt.Errorf("internal: offset %d != recordSize %d", offset, schema.RecordSize)
    }
	return values, nil
}