package service

import "fmt"

func (s *Service) AddResume(values map[string]any) error {
	if err := s.checkFK(s.Seekers, values["seeker_id"], "seeker"); err != nil {
		return fmt.Errorf("AddResume: %w", err)
	}
	if _, err := requireString(values, "position"); err != nil {
		return fmt.Errorf("AddResume: %w", err)
	}
	return s.Resumes.Add(values)
}

func (s *Service) GetResume(id any) (map[string]any, error) {
	return s.Resumes.GetByPK(id)
}


func (s *Service) UpdateResume(id any, values map[string]any) error {
	old, err := s.Resumes.GetByPK(id)
	if err != nil {
		return err
	}
	if v, ok := values["seeker_id"]; ok && !valuesEqual(v, old["seeker_id"]) {
		return fmt.Errorf("UpdateResume: seeker_id cannot change")
	}
	return s.Resumes.Update(id, values)
}

func (s *Service) DeleteResume(id any) error {
	return s.deleteResumeCascade(id)
}

type ResumeFilter struct {
	SeekerID any
	Position string
	City     string
	Status   string
}

func (s *Service) SearchResumes(f ResumeFilter) ([]map[string]any, error) {
	var rows []map[string]any
	var err error

	switch {
	case f.SeekerID != nil:
		rows, err = s.Resumes.FindBy("seeker_id", f.SeekerID)
	case f.Position != "":
		rows, err = s.Resumes.FindBy("position", f.Position)
	case f.Status != "":
		rows, err = s.Resumes.FindBy("status", f.Status)
	default:
		rows, err = s.Resumes.FindAll()
	}
	if err != nil {
		return nil, err
	}
	if f.City == "" && f.Status == "" && f.Position == "" && f.SeekerID != nil {
		return rows, nil
	}

	out := rows[:0]
	for _, r := range rows {
		if f.City != "" && !valuesEqual(r["city"], f.City) { continue }
		if f.Status != "" && !valuesEqual(r["status"], f.Status) { continue }
		if f.Position != "" && !valuesEqual(r["position"], f.Position) { continue }
		out = append(out, r)
	}
	return out, nil
}