package service

import ("fmt"
		"file_db/internal/schema")

func (s *Service) AddResumeSkill(resumeID, skillID any) error {
    rid, err := asInt64("resume_id", resumeID)
    if err != nil {
        return fmt.Errorf("AddResumeSkill: %w", err)
    }
    sid, err := asInt64("skill_id", skillID)
    if err != nil {
        return fmt.Errorf("AddResumeSkill: %w", err)
    }
    if err := s.checkFK(s.Resumes, rid, "resume"); err != nil {
        return fmt.Errorf("AddResumeSkill: %w", err)
    }
    if err := s.checkFK(s.Skills, sid, "skill"); err != nil {
        return fmt.Errorf("AddResumeSkill: %w", err)
    }
    links, err := s.ResumeSkills.FindBy("resume_id", rid)
    if err != nil {
        return err
    }
    for _, link := range links {
        if valuesEqual(link["skill_id"], sid) {
            return fmt.Errorf("AddResumeSkill: resume %d already has skill %d", rid, sid)
        }
    }
    return s.ResumeSkills.Add(map[string]any{
        "resume_id": rid,
        "skill_id":  sid,
    })
}

func (s *Service) GetResumeSkills(resumeID any) ([]map[string]any, error) {
    rid, err := asInt64("resume_id", resumeID)
    if err != nil {
        return nil, fmt.Errorf("GetResumeSkills: %w", err)
    }
    links, err := s.ResumeSkills.FindBy("resume_id", rid)
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

func (s *Service) RemoveResumeSkill(resumeID, skillID any) error {
	rid, err := schema.ToInt64(resumeID)
	if err != nil {
		return fmt.Errorf("RemoveResumeSkill: resume_id: %w", err)
	}
	sid, err := schema.ToInt64(skillID)
	if err != nil {
		return fmt.Errorf("RemoveResumeSkill: skill_id: %w", err)
	}

	links, err := s.ResumeSkills.FindBy("resume_id", rid)
	if err != nil {
		return err
	}
	for _, link := range links {
		if valuesEqual(link["skill_id"], sid) {
			return s.ResumeSkills.DeleteByPK(link["id"])
		}
	}
	return fmt.Errorf("RemoveResumeSkill: pair not found")
}