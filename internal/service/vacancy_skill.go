package service

import ("fmt"
		"file_db/internal/schema")

func (s *Service) AddVacancySkill(vacancyID, skillID any) error {
	vid, err := schema.ToInt64(vacancyID)
	if err != nil { return fmt.Errorf("AddVacancySkill: vacancy_id: %w", err) }
	sid, err := schema.ToInt64(skillID)
	if err != nil { return fmt.Errorf("AddVacancySkill: skill_id: %w", err) }

	if err := s.checkFK(s.Vacancies, vid, "vacancy"); err != nil {
		return fmt.Errorf("AddVacancySkill: %w", err)
	}
	if err := s.checkFK(s.Skills, sid, "skill"); err != nil {
		return fmt.Errorf("AddVacancySkill: %w", err)
	}

	links, err := s.VacancySkills.FindBy("vacancy_id", vid)
	if err != nil {
		return err
	}
	for _, link := range links {
		if valuesEqual(link["skill_id"], sid) {
			return fmt.Errorf("AddVacancySkill: vacancy %v already requires skill %v", vacancyID, skillID)
		}
	}

	return s.VacancySkills.Add(map[string]any{
		"vacancy_id": vacancyID,
		"skill_id":   skillID,
	})
}

func (s *Service) GetVacancySkills(vacancyID any) ([]map[string]any, error) {
	vid, err := schema.ToInt64(vacancyID)
	if err != nil { return nil, fmt.Errorf("GetVacancySkills: vacancy_id: %w", err) }
	links, err := s.VacancySkills.FindBy("vacancy_id", vid)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(links))
	for _, link := range links {
		skill, err := s.Skills.GetByPK(link["skill_id"])
		if err != nil {
			continue
		}
		out = append(out, skill)
	}
	return out, nil
}

func (s *Service) RemoveVacancySkill(vacancyID, skillID any) error {
	vid, err := schema.ToInt64(vacancyID)
	if err != nil {
		return fmt.Errorf("RemoveVacancySkill: vacancy_id: %w", err)
	}
	sid, err := schema.ToInt64(skillID)
	if err != nil {
		return fmt.Errorf("RemoveVacancySkill: skill_id: %w", err)
	}

	links, err := s.VacancySkills.FindBy("vacancy_id", vid)
	if err != nil {
		return err
	}
	for _, link := range links {
		if valuesEqual(link["skill_id"], sid) {
			return s.VacancySkills.DeleteByPK(link["id"])
		}
	}
	return fmt.Errorf("RemoveVacancySkill: pair not found")
}