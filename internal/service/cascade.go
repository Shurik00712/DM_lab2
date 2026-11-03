package service

import "fmt"

func (s *Service) deleteResumeCascade(resumeID any) error {
	if _, err := s.ResumeSkills.DeleteBy("resume_id", resumeID); err != nil {
		return fmt.Errorf("delete resume_skill: %w", err)
	}
	if _, err := s.Applications.DeleteBy("resume_id", resumeID); err != nil {
		return fmt.Errorf("delete applications: %w", err)
	}
	return s.Resumes.DeleteByPK(resumeID)
}

func (s *Service) deleteVacancyCascade(vacancyID any) error {
	if _, err := s.VacancySkills.DeleteBy("vacancy_id", vacancyID); err != nil {
		return fmt.Errorf("delete vacancy_skill: %w", err)
	}
	if _, err := s.Applications.DeleteBy("vacancy_id", vacancyID); err != nil {
		return fmt.Errorf("delete applications: %w", err)
	}
	return s.Vacancies.DeleteByPK(vacancyID)
}

func (s *Service) deleteSeekerCascade(seekerID any) error {
	resumes, err := s.Resumes.FindBy("seeker_id", seekerID)
	if err != nil {
		return err
	}
	for _, r := range resumes {
		if err := s.deleteResumeCascade(r["id"]); err != nil {
			return fmt.Errorf("delete resume %v: %w", r["id"], err)
		}
	}

	if _, err := s.Applications.DeleteBy("seeker_id", seekerID); err != nil {
		return fmt.Errorf("delete applications: %w", err)
	}

	if err := s.deleteMessagesByAuthor("seeker", seekerID); err != nil {
		return err
	}

	if _, err := s.Reviews.DeleteBy("seeker_id", seekerID); err != nil {
		return fmt.Errorf("delete reviews: %w", err)
	}

	return s.Seekers.DeleteByPK(seekerID)
}

func (s *Service) deleteEmployerCascade(employerID any) error {
	vacancies, err := s.Vacancies.FindBy("employer_id", employerID)
	if err != nil {
		return err
	}
	for _, v := range vacancies {
		if err := s.deleteVacancyCascade(v["id"]); err != nil {
			return fmt.Errorf("delete vacancy %v: %w", v["id"], err)
		}
	}

	if _, err := s.Reviews.DeleteBy("employer_id", employerID); err != nil {
		return fmt.Errorf("delete reviews: %w", err)
	}

	if err := s.deleteMessagesByAuthor("employer", employerID); err != nil {
		return err
	}

	return s.Employers.DeleteByPK(employerID)
}

func (s *Service) deleteMessagesByAuthor(authorType string, authorID any) error {
	msgs, err := s.Messages.FindBy("author_id", authorID)
	if err != nil {
		return err
	}
	for _, m := range msgs {
		if !valuesEqual(m["author_type"], authorType) {
			continue
		}
		if err := s.Messages.DeleteByPK(m["id"]); err != nil {
			return fmt.Errorf("delete message %v: %w", m["id"], err)
		}
	}
	return nil
}