package service

import ("file_db/internal/engine"
        "fmt"
        "errors"
		"math"
		"strconv"
		"encoding/json")


type Engines struct {
    Seekers       *engine.Engine
    Employers     *engine.Engine
    Moderators    *engine.Engine
    Resumes       *engine.Engine
    Vacancies     *engine.Engine
    Skills        *engine.Engine
    ResumeSkills  *engine.Engine
    VacancySkills *engine.Engine
    Applications  *engine.Engine
    Messages      *engine.Engine
    Reviews       *engine.Engine
}

type Service struct {
    *Engines
}

func New(e *Engines) (*Service, error) {
	if e == nil {
		return nil, errors.New("nil engines")
	}
	if err := e.validate(); err != nil {
		return nil, err
	}
	return &Service{Engines: e}, nil
}

func (e *Engines) validate() error {
	items := []struct {
		name string
		eng  *engine.Engine
	}{
		{"seekers", e.Seekers},
		{"employers", e.Employers},
		{"moderators", e.Moderators},
		{"resumes", e.Resumes},
		{"vacancies", e.Vacancies},
		{"skills", e.Skills},
		{"resume_skills", e.ResumeSkills},
		{"vacancy_skills", e.VacancySkills},
		{"applications", e.Applications},
		{"messages", e.Messages},
		{"reviews", e.Reviews},
	}
	for _, it := range items {
		if it.eng == nil {
			return fmt.Errorf("engine %q is nil", it.name)
		}
	}
	return nil
}

func (s *Service) checkFK(eng *engine.Engine, id any, what string) error {
	if id == nil {
		return fmt.Errorf("%s: id is required", what)
	}
	if _, err := eng.GetByPK(id); err != nil {
		return fmt.Errorf("%s %v: %w", what, id, err)
	}
	return nil
}

func requireString(values map[string]any, name string) (string, error) {
	v, ok := values[name]
	if !ok || v == nil {
		return "", fmt.Errorf("%s: required", name)
	}
	str, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%s: expected string, got %T", name, v)
	}
	if str == "" {
		return "", fmt.Errorf("%s: empty", name)
	}
	return str, nil
}

func requireOneOf(values map[string]any, name string, allowed ...string) (string, error) {
	str, err := requireString(values, name)
	if err != nil {
		return "", err
	}
	for _, a := range allowed {
		if str == a {
			return str, nil
		}
	}
	return "", fmt.Errorf("%s: %q not in %v", name, str, allowed)
}

func compareInt64(a int64, b any) bool {
	switch bv := b.(type) {
	case int:     return a == int64(bv)
	case int8:    return a == int64(bv)
	case int16:   return a == int64(bv)
	case int32:   return a == int64(bv)
	case int64:   return a == bv
	case uint:    return bv <= uint(^uint64(0)>>1) && a == int64(bv)
	case uint64:  return bv <= uint64(^uint64(0)>>1) && a == int64(bv)
	case float32:                     
		f := float64(bv)
		return f == math.Trunc(f) && a == int64(f)
	case float64:                    
		return bv == math.Trunc(bv) && a == int64(bv)
	}
	return false
}

func asInt64(name string, v any) (int64, error) {
    if v == nil {
        return 0, fmt.Errorf("%s: required", name)
    }
    switch x := v.(type) {
    case int:     return int64(x), nil
    case int8:    return int64(x), nil
    case int16:   return int64(x), nil
    case int32:   return int64(x), nil
    case int64:   return x, nil
    case uint:    return int64(x), nil
    case uint8:   return int64(x), nil
    case uint16:  return int64(x), nil
    case uint32:  return int64(x), nil
    case uint64:
        if x > math.MaxInt64 {
            return 0, fmt.Errorf("%s: %d overflows int64", name, x)
        }
        return int64(x), nil
    case float32:
        f := float64(x)
        if f != math.Trunc(f) {
            return 0, fmt.Errorf("%s: %v is not an integer", name, x)
        }
        return int64(f), nil
    case float64:
        if x != math.Trunc(x) {
            return 0, fmt.Errorf("%s: %v is not an integer", name, x)
        }
        return int64(x), nil
    case string:
        n, err := strconv.ParseInt(x, 10, 64)
        if err != nil {
            return 0, fmt.Errorf("%s: %q is not an integer", name, x)
        }
        return n, nil
    case json.Number:
        n, err := x.Int64()
        if err != nil {
            return 0, fmt.Errorf("%s: %q is not an integer", name, x)
        }
        return n, nil
    }
    return 0, fmt.Errorf("%s: unsupported type %T", name, v)
}

func compareFloat64(a float64, b any) bool {
	switch bv := b.(type) {
	case float32: return a == float64(bv)
	case float64: return a == bv
	case int:                                  
		return a == math.Trunc(a) && int64(a) == int64(bv)
	case int8:
		return a == math.Trunc(a) && int64(a) == int64(bv)
	case int16:
		return a == math.Trunc(a) && int64(a) == int64(bv)
	case int32:
		return a == math.Trunc(a) && int64(a) == int64(bv)
	case int64:
		return a == math.Trunc(a) && int64(a) == bv
	case uint:
		return a >= 0 && a == math.Trunc(a) && uint64(a) == uint64(bv)
	case uint64:
		return a >= 0 && a == math.Trunc(a) && uint64(a) == bv
	}
	return false
}


func requireInt64(values map[string]any, name string, min, max int64) (int64, error) {
	v, ok := values[name]
	if !ok || v == nil {
		return 0, fmt.Errorf("%s: required", name)
	}
	var n int64
	switch x := v.(type) {
	case int:
		n = int64(x)
	case int8:
		n = int64(x)
	case int16:
		n = int64(x)
	case int32:
		n = int64(x)
	case int64:
		n = x
	default:
		return 0, fmt.Errorf("%s: expected int, got %T", name, v)
	}
	if n < min || n > max {
		return 0, fmt.Errorf("%s: %d out of [%d, %d]", name, n, min, max)
	}
	return n, nil
}


func valuesEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	switch av := a.(type) {
	case int:
		return compareInt64(int64(av), b)
	case int8:
		return compareInt64(int64(av), b)
	case int16:
		return compareInt64(int64(av), b)
	case int32:
		return compareInt64(int64(av), b)
	case int64:
		return compareInt64(av, b)
	case uint:
		return compareUint64(uint64(av), b)
	case uint8:
		return compareUint64(uint64(av), b)
	case uint16:
		return compareUint64(uint64(av), b)
	case uint32:
		return compareUint64(uint64(av), b)
	case uint64:
		return compareUint64(av, b)
	case float32:
		return compareFloat64(float64(av), b)
	case float64:
		return compareFloat64(av, b)
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	}
	return false
}


func compareUint64(a uint64, b any) bool {
	switch bv := b.(type) {
	case uint:
		return a == uint64(bv)
	case uint64:
		return a == bv
	case int:
		return bv >= 0 && a == uint64(bv)
	case int64:
		return bv >= 0 && a == uint64(bv)
	}
	return false
}

