package service

import ("fmt"
		"time")

func (s *Service) AddReview(values map[string]any) error {
	if err := s.checkFK(s.Seekers, values["seeker_id"], "seeker"); err != nil {
		return fmt.Errorf("AddReview: %w", err)
	}
	if err := s.checkFK(s.Employers, values["employer_id"], "employer"); err != nil {
		return fmt.Errorf("AddReview: %w", err)
	}
	if _, err := requireInt64(values, "rating", 1, 5); err != nil {
		return fmt.Errorf("AddReview: %w", err)
	}
	if _, ok := values["is_published"]; !ok {
		values["is_published"] = false
	}
	if _, ok := values["created_at"]; !ok || values["created_at"] == nil {
		values["created_at"] = time.Now().Unix()
	}
	return s.Reviews.Add(values)
}

func (s *Service) GetReview(id any) (map[string]any, error) {
	return s.Reviews.GetByPK(id)
}

func (s *Service) GetReviewsByEmployer(employerID any) ([]map[string]any, error) {
	return s.Reviews.FindBy("employer_id", employerID)
}

func (s *Service) GetPublishedReviewsByEmployer(employerID any) ([]map[string]any, error) {
	rows, err := s.Reviews.FindBy("employer_id", employerID)
	if err != nil {
		return nil, err
	}
	out := rows[:0]
	for _, r := range rows {
		if valuesEqual(r["is_published"], true) {
			out = append(out, r)
		}
	}
	return out, nil
}

func (s *Service) DeleteReview(id any) error {
	return s.Reviews.DeleteByPK(id)
}