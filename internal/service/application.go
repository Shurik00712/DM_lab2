package service

import (
	"fmt"
	"errors"
	"time"
)


func (s *Service) AddApplication(values map[string]any) error {
	seekerID, ok := values["seeker_id"]
	if !ok || seekerID == nil {
		return fmt.Errorf("AddApplication: seeker_id required")
	}
	resumeID, ok := values["resume_id"]
	if !ok || resumeID == nil {
		return fmt.Errorf("AddApplication: resume_id required")
	}
	vacancyID, ok := values["vacancy_id"]
	if !ok || vacancyID == nil {
		return fmt.Errorf("AddApplication: vacancy_id required")
	}

	if err := s.checkFK(s.Seekers, seekerID, "seeker"); err != nil {
		return fmt.Errorf("AddApplication: %w", err)
	}
	if err := s.checkFK(s.Resumes, resumeID, "resume"); err != nil {
		return fmt.Errorf("AddApplication: %w", err)
	}
	if err := s.checkFK(s.Vacancies, vacancyID, "vacancy"); err != nil {
		return fmt.Errorf("AddApplication: %w", err)
	}

	resume, err := s.Resumes.GetByPK(resumeID)
	if err != nil {
		return fmt.Errorf("AddApplication: %w", err)
	}
	if !valuesEqual(resume["seeker_id"], seekerID) {
		return errors.New("AddApplication: resume does not belong to this seeker")
	}

	vacancy, err := s.Vacancies.GetByPK(vacancyID)
	if err != nil {
		return fmt.Errorf("AddApplication: %w", err)
	}
	if !valuesEqual(vacancy["status"], "active") {
		return errors.New("AddApplication: vacancy is not active")
	}

	existing, err := s.Applications.FindBy("vacancy_id", vacancyID)
	if err != nil {
		return err
	}
	for _, a := range existing {
		if valuesEqual(a["seeker_id"], seekerID) {
			return fmt.Errorf("AddApplication: %w", errApplicationExists)
		}
	}

	if _, ok := values["initiator"]; !ok {
		values["initiator"] = "seeker"
	}
	if _, err := requireOneOf(values, "initiator", "seeker", "employer"); err != nil {
		return fmt.Errorf("AddApplication: %w", err)
	}

	if _, ok := values["status"]; !ok {
		values["status"] = "pending"
	}

	if _, ok := values["created_at"]; !ok || values["created_at"] == nil {
		values["created_at"] = time.Now().Unix()
	}

	return s.Applications.Add(values)
}

func (s *Service) GetApplication(id any) (map[string]any, error) {
	return s.Applications.GetByPK(id)
}

func (s *Service) UpdateApplicationStatus(id any, status string) error {
	app, err := s.Applications.GetByPK(id)
	if err != nil {
		return err
	}
	app["status"] = status
	return s.Applications.Update(id, app)
}

func (s *Service) GetApplicationsByVacancy(vacancyID any) ([]map[string]any, error) {
	return s.Applications.FindBy("vacancy_id", vacancyID)
}

func (s *Service) GetApplicationsBySeeker(seekerID any) ([]map[string]any, error) {
	return s.Applications.FindBy("seeker_id", seekerID)
}

func (s *Service) DeleteApplication(id any) error {
	return s.Applications.DeleteByPK(id)
}