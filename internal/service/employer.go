package service

import "fmt"

func (s *Service) AddEmployer(values map[string]any) error {
	if _, err := requireString(values, "email"); err != nil {
		return fmt.Errorf("AddEmployer: %w", err)
	}
	if _, err := requireString(values, "company_name"); err != nil {
		return fmt.Errorf("AddEmployer: %w", err)
	}
	return s.Employers.Add(values)
}

func (s *Service) GetEmployer(id any) (map[string]any, error) {
	return s.Employers.GetByPK(id)
}

func (s *Service) DeleteEmployer(id any) error {
	return s.deleteEmployerCascade(id)
}

func (s *Service) ApproveEmployer(moderatorID, employerID any) error {
	if err := s.checkFK(s.Moderators, moderatorID, "moderator"); err != nil {
		return err
	}
	emp, err := s.Employers.GetByPK(employerID)
	if err != nil {
		return err
	}
	emp["moderator_id"] = moderatorID
	emp["verification_status"] = "approved"
	return s.Employers.Update(employerID, emp)
}