package service

import ("fmt"
		"time")

func (s *Service) AddMessage(values map[string]any) error {
	authorType, err := requireOneOf(values, "author_type", "seeker", "employer")
	if err != nil {
		return fmt.Errorf("AddMessage: %w", err)
	}
	if _, err := requireString(values, "text"); err != nil {
		return fmt.Errorf("AddMessage: %w", err)
	}
	authorID, ok := values["author_id"]
	if !ok || authorID == nil {
		return fmt.Errorf("AddMessage: author_id required")
	}

	switch authorType {
	case "seeker":
		if err := s.checkFK(s.Seekers, authorID, "seeker"); err != nil {
			return fmt.Errorf("AddMessage: %w", err)
		}
	case "employer":
		if err := s.checkFK(s.Employers, authorID, "employer"); err != nil {
			return fmt.Errorf("AddMessage: %w", err)
		}
	}

	if _, ok := values["is_read"]; !ok {
		values["is_read"] = false
	}
	if _, ok := values["created_at"]; !ok || values["created_at"] == nil {
		values["created_at"] = time.Now().Unix()
	}
	return s.Messages.Add(values)
}

func (s *Service) GetMessagesByAuthor(authorType string, authorID any) ([]map[string]any, error) {
	if authorType != "seeker" && authorType != "employer" {
		return nil, fmt.Errorf("invalid author_type %q", authorType)
	}
	rows, err := s.Messages.FindBy("author_id", authorID)
	if err != nil {
		return nil, err
	}
	out := rows[:0]
	for _, m := range rows {
		if valuesEqual(m["author_type"], authorType) {
			out = append(out, m)
		}
	}
	return out, nil
}

func (s *Service) MarkMessageRead(id any) error {
	m, err := s.Messages.GetByPK(id)
	if err != nil {
		return err
	}
	m["is_read"] = true
	return s.Messages.Update(id, m)
}