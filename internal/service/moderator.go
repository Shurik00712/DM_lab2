package service

import "fmt"

func (s *Service) AddModerator(values map[string]any) error {
	if _, err := requireString(values, "email"); err != nil {
		return fmt.Errorf("AddModerator: %w", err)
	}
	if _, err := requireString(values, "full_name"); err != nil {
		return fmt.Errorf("AddModerator: %w", err)
	}
	return s.Moderators.Add(values)
}

func (s *Service) GetModerator(id any) (map[string]any, error) {
	return s.Moderators.GetByPK(id)
}

func (s *Service) ApproveVacancy(moderatorID, vacancyID any) error {
	if err := s.checkFK(s.Moderators, moderatorID, "moderator"); err != nil {
		return err
	}
	v, err := s.Vacancies.GetByPK(vacancyID)
	if err != nil {
		return err
	}
	v["moderator_id"] = moderatorID
	v["status"] = "active"
	return s.Vacancies.Update(vacancyID, v)
}

func (s *Service) RejectVacancy(moderatorID, vacancyID any) error {
	if err := s.checkFK(s.Moderators, moderatorID, "moderator"); err != nil {
		return err
	}
	v, err := s.Vacancies.GetByPK(vacancyID)
	if err != nil {
		return err
	}
	v["moderator_id"] = moderatorID
	v["status"] = "rejected"
	return s.Vacancies.Update(vacancyID, v)
}

func (s *Service) ApproveReview(moderatorID, reviewID any) error {
	if err := s.checkFK(s.Moderators, moderatorID, "moderator"); err != nil {
		return err
	}
	r, err := s.Reviews.GetByPK(reviewID)
	if err != nil {
		return err
	}
	r["moderator_id"] = moderatorID
	r["is_published"] = true
	return s.Reviews.Update(reviewID, r)
}

func (s *Service) RejectReview(moderatorID, reviewID any) error {
	if err := s.checkFK(s.Moderators, moderatorID, "moderator"); err != nil {
		return err
	}
	r, err := s.Reviews.GetByPK(reviewID)
	if err != nil {
		return err
	}
	r["moderator_id"] = moderatorID
	r["is_published"] = false
	return s.Reviews.Update(reviewID, r)
}