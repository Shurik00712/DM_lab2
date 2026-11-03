package service

import "fmt"

func (s *Service) AddSkill(name string) error {
	if name == "" {
		return fmt.Errorf("AddSkill: name required")
	}
	return s.Skills.Add(map[string]any{"name": name})
}

func (s *Service) GetSkill(id any) (map[string]any, error) {
	return s.Skills.GetByPK(id)
}

func (s *Service) FindSkillByName(name string) (map[string]any, error) {
	rows, err := s.Skills.FindBy("name", name)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("skill %q: not found", name)
	}
	return rows[0], nil
}