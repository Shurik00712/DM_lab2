package service

import "fmt"

func (s *Service) AddVacancy(values map[string]any) error {
	if err := s.checkFK(s.Employers, values["employer_id"], "employer"); err != nil {
		return fmt.Errorf("AddVacancy: %w", err)
	}
	if _, err := requireString(values, "title"); err != nil {
		return fmt.Errorf("AddVacancy: %w", err)
	}
	return s.Vacancies.Add(values)
}

func (s *Service) GetVacancy(id any) (map[string]any, error) {
	return s.Vacancies.GetByPK(id)
}

func (s *Service) UpdateVacancy(id any, values map[string]any) error {
	old, err := s.Vacancies.GetByPK(id)
	if err != nil { return err }
	if v, ok := values["employer_id"]; ok && !valuesEqual(v, old["employer_id"]) {
		return fmt.Errorf("UpdateVacancy: employer_id cannot change")
	}
	return s.Vacancies.Update(id, values)
}

func (s *Service) DeleteVacancy(id any) error {
	return s.deleteVacancyCascade(id)
}

type VacancyFilter struct {
    City           string
    SalaryFrom     int64
    EmploymentType string
    Status         string
    IsRemote       *bool   // nil = not filter, false = office, true = online
}


func (s *Service) SearchVacancies(f VacancyFilter) ([]map[string]any, error) {
	status := f.Status
	if status == "" { status = "active" }

	rows, err := s.Vacancies.FindBy("status", status)
	if err != nil { return nil, err }

	out := rows[:0]
	for _, v := range rows {
		if f.City != "" && !valuesEqual(v["city"], f.City) { continue }
		if f.EmploymentType != "" && !valuesEqual(v["employment_type"], f.EmploymentType) { continue }
		if f.IsRemote != nil {
			schedule, _ := v["schedule"].(string)
			isRemote := schedule == "remote"
			if *f.IsRemote != isRemote { continue }
		}
		if f.SalaryFrom > 0 {
			sf, _ := v["salary_from"].(int64)
			if sf < f.SalaryFrom { continue }
		}
		out = append(out, v)
	}
	return out, nil
}