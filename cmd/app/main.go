package main

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"file_db/internal/engine"
	"file_db/internal/schema"
	"file_db/internal/service"
	"file_db/internal/storage"
	"file_db/internal/web"
)

const dataDir = "data"

func main() {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatalf("mkdir: %v", err)
	}

	mustS := func(s *schema.Schema, err error) *schema.Schema {
		if err != nil { log.Fatalf("schema: %v", err) }
		return s
	}

	engines := &service.Engines{
		Seekers:       openEngine("seeker", mustS(schema.SeekerSchema())),
		Employers:     openEngine("employer", mustS(schema.EmployerSchema())),
		Moderators:    openEngine("moderator", mustS(schema.ModeratorSchema())),
		Resumes:       openEngine("resume", mustS(schema.ResumeSchema())),
		Vacancies:     openEngine("vacancy", mustS(schema.VacancySchema())),
		Skills:        openEngine("skill", mustS(schema.SkillSchema())),
		ResumeSkills:  openEngine("resume_skill", mustS(schema.ResumeSkillSchema())),
		VacancySkills: openEngine("vacancy_skill", mustS(schema.VacancySkillSchema())),
		Applications:  openEngine("application", mustS(schema.ApplicationSchema())),
		Messages:      openEngine("message", mustS(schema.MessageSchema())),
		Reviews:       openEngine("review", mustS(schema.ReviewSchema())),
	}

	svc, err := service.New(engines)
	if err != nil { log.Fatalf("service: %v", err) }

	defer closeAll(engines)

	if err := web.New(svc).Run("localhost:8080"); err != nil {
		log.Fatal(err)
	}
}

func openEngine(name string, s *schema.Schema) *engine.Engine {
	path := filepath.Join(dataDir, name+".db")
	db, err := storage.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		db, err = storage.Create(path, s)
	}
	if err != nil { log.Fatalf("open %s: %v", name, err) }
	eng, err := engine.New(db, s, name, path)
	if err != nil { log.Fatalf("engine %s: %v", name, err) }
	return eng
}

func closeAll(e *service.Engines) {
	for _, eng := range []*engine.Engine{
		e.Seekers, e.Employers, e.Moderators,
		e.Resumes, e.Vacancies, e.Skills,
		e.ResumeSkills, e.VacancySkills,
		e.Applications, e.Messages, e.Reviews,
	} {
		if eng != nil { _ = eng.Close() }
	}
}