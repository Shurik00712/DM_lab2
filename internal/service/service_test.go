package service_test

import (
	"path/filepath"
	"testing"

	"file_db/internal/engine"
	"file_db/internal/schema"
	"file_db/internal/service"
	"file_db/internal/storage"
)

func openEng(t *testing.T, dir, name string, s *schema.Schema) *engine.Engine {
	t.Helper()
	path := filepath.Join(dir, name+".db")
	db, err := storage.Create(path, s)
	if err != nil { t.Fatalf("create %s: %v", name, err) }
	eng, err := engine.New(db, s, name, path)
	if err != nil {
		db.Close()
		t.Fatalf("engine %s: %v", name, err)
	}
	t.Cleanup(func() { _ = eng.Close() })    
	return eng
}

func setupService(t *testing.T) *service.Service {
	t.Helper()
	dir := t.TempDir()

	mustSchema := func(s *schema.Schema, err error) *schema.Schema {
		t.Helper()
		if err != nil { t.Fatalf("schema: %v", err) }
		return s
	}

	engines := &service.Engines{
		Seekers:       openEng(t, dir, "seeker",       mustSchema(schema.SeekerSchema())),
		Employers:     openEng(t, dir, "employer",     mustSchema(schema.EmployerSchema())),
		Moderators:    openEng(t, dir, "moderator",    mustSchema(schema.ModeratorSchema())),
		Resumes:       openEng(t, dir, "resume",       mustSchema(schema.ResumeSchema())),
		Vacancies:     openEng(t, dir, "vacancy",      mustSchema(schema.VacancySchema())),
		Skills:        openEng(t, dir, "skill",        mustSchema(schema.SkillSchema())),
		ResumeSkills:  openEng(t, dir, "resume_skill", mustSchema(schema.ResumeSkillSchema())),
		VacancySkills: openEng(t, dir, "vacancy_skill",mustSchema(schema.VacancySkillSchema())),
		Applications:  openEng(t, dir, "application",  mustSchema(schema.ApplicationSchema())),
		Messages:      openEng(t, dir, "message",      mustSchema(schema.MessageSchema())),
		Reviews:       openEng(t, dir, "review",       mustSchema(schema.ReviewSchema())),
	}
	svc, err := service.New(engines)
	if err != nil { t.Fatalf("service: %v", err) }
	return svc
}

func TestAddResumeFKFails(t *testing.T) {
	s := setupService(t)
	err := s.AddResume(map[string]any{"seeker_id": int64(999), "position": "Go dev"})
	if err == nil { t.Fatal("expected FK error") }

	n, _ := s.Resumes.Count()
	if n != 0 { t.Errorf("resume was added despite FK failure: count=%d", n) }
}

func TestAddResumeFKOk(t *testing.T) {
	s := setupService(t)
	_ = s.AddSeeker(map[string]any{"email": "a@x.com", "full_name": "Alice"})

	if err := s.AddResume(map[string]any{"seeker_id": int64(1), "position": "Go dev"}); err != nil {
		t.Fatalf("add resume: %v", err)
	}
}

func TestAddResumeSkillDuplicate(t *testing.T) {
	s := setupService(t)
	_ = s.AddSeeker(map[string]any{"email": "a@x.com", "full_name": "A"})
	_ = s.AddResume(map[string]any{"seeker_id": int64(1), "position": "Go"})
	_ = s.AddSkill("Python")

	if err := s.AddResumeSkill(int64(1), int64(1)); err != nil {
		t.Fatalf("add 1: %v", err)
	}
	if err := s.AddResumeSkill(int64(1), int64(1)); err == nil {
		t.Fatal("expected error on duplicate M:N pair")
	}
}

func TestGetResumeSkills(t *testing.T) {
	s := setupService(t)
	_ = s.AddSeeker(map[string]any{"email": "a@x.com", "full_name": "A"})
	_ = s.AddResume(map[string]any{"seeker_id": int64(1), "position": "Go"})
	_ = s.AddSkill("Go")
	_ = s.AddSkill("SQL")
	_ = s.AddResumeSkill(int64(1), int64(1))
	_ = s.AddResumeSkill(int64(1), int64(2))

	skills, err := s.GetResumeSkills(int64(1))
	if err != nil { t.Fatalf("get skills: %v", err) }
	if len(skills) != 2 { t.Errorf("got %d skills, want 2", len(skills)) }
}

func TestApplicationResumeMustBelongToSeeker(t *testing.T) {
	s := setupService(t)
	_ = s.AddSeeker(map[string]any{"email": "a@x.com", "full_name": "A"})
	_ = s.AddSeeker(map[string]any{"email": "b@x.com", "full_name": "B"})
	_ = s.AddEmployer(map[string]any{"email": "e@x.com", "company_name": "E"})
	_ = s.AddVacancy(map[string]any{"employer_id": int64(1), "title": "Go", "status": "active"})
	_ = s.AddResume(map[string]any{"seeker_id": int64(1), "position": "Go"})

	err := s.AddApplication(map[string]any{
		"seeker_id": int64(2), "resume_id": int64(1), "vacancy_id": int64(1),
	})
	if err == nil { t.Fatal("expected error: resume does not belong to seeker") }
}

func TestDeleteSeekerCascade(t *testing.T) {
	s := setupService(t)
	_ = s.AddSeeker(map[string]any{"email": "a@x.com", "full_name": "A"})
	_ = s.AddEmployer(map[string]any{"email": "e@x.com", "company_name": "E"})
	_ = s.AddVacancy(map[string]any{"employer_id": int64(1), "title": "Go", "status": "active"})
	_ = s.AddResume(map[string]any{"seeker_id": int64(1), "position": "Go"})
	_ = s.AddResumeSkill(int64(1), int64(1)) 
	_ = s.AddApplication(map[string]any{
		"seeker_id": int64(1), "resume_id": int64(1), "vacancy_id": int64(1),
	})

	if err := s.DeleteSeeker(int64(1)); err != nil { t.Fatalf("delete: %v", err) }

	for name, eng := range map[string]*engine.Engine{
		"seekers":       s.Seekers,
		"resumes":       s.Resumes,
		"resume_skills": s.ResumeSkills,
		"applications":  s.Applications,
	} {
		n, _ := eng.Count()
		if n != 0 { t.Errorf("%s: %d rows remain after cascade", name, n) }
	}
}

func TestAddEmployerEmptyInn(t *testing.T) {
	s := setupService(t)
	if err := s.AddEmployer(map[string]any{"email": "a@x.com", "company_name": "A"}); err != nil {
		t.Fatalf("add 1: %v", err)
	}
	if err := s.AddEmployer(map[string]any{"email": "b@x.com", "company_name": "B"}); err != nil {
		t.Fatalf("add 2: %v", err)
	}
}

func TestApproveVacancy(t *testing.T) {
	s := setupService(t)
	_ = s.AddModerator(map[string]any{"email": "m@x.com", "full_name": "M"})
	_ = s.AddEmployer(map[string]any{"email": "e@x.com", "company_name": "E"})
	_ = s.AddVacancy(map[string]any{"employer_id": int64(1), "title": "Go", "status": "pending"})

	if err := s.ApproveVacancy(int64(1), int64(1)); err != nil {
		t.Fatalf("approve: %v", err)
	}
	v, _ := s.Vacancies.GetByPK(int64(1))
	if v["status"] != "active" { t.Errorf("status: got %v, want active", v["status"]) }
	if v["moderator_id"] != int64(1) { t.Errorf("moderator_id: got %v", v["moderator_id"]) }
}