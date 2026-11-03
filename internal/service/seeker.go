package service

import "fmt"

func (s *Service) AddSeeker(values map[string]any) error {
	if _, err := requireString(values, "email"); err != nil {
		return fmt.Errorf("AddSeeker: %w", err)
	}
	if _, err := requireString(values, "full_name"); err != nil {
		return fmt.Errorf("AddSeeker: %w", err)
	}
	return s.Seekers.Add(values)
}

func (s *Service) GetSeeker(id any) (map[string]any, error) {
	return s.Seekers.GetByPK(id)
}

func (s *Service) DeleteSeeker(id any) error {
	return s.deleteSeekerCascade(id)
}

type SeekerFilter struct {
	City   string
	Status string
}

func (s *Service) SearchSeekers(f SeekerFilter) ([]map[string]any, error) {
	base := "city"
	val := f.City
	if val == "" {
		base, val = "status", f.Status
	}
	if val == "" {
		return s.Seekers.FindAll()
	}
	rows, err := s.Seekers.FindBy(base, val)
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	for _, r := range rows {
		if f.City != "" && !valuesEqual(r["city"], f.City) { continue }
		if f.Status != "" && !valuesEqual(r["status"], f.Status) { continue }
		out = append(out, r)
	}
	return out, nil
}