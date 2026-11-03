package web

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"file_db/internal/engine"
)

func (s *Server) allEngines() []struct {
	Name string
	Eng  *engine.Engine
} {
	return []struct {
		Name string
		Eng  *engine.Engine
	}{
		{"seekers", s.svc.Seekers},
		{"employers", s.svc.Employers},
		{"moderators", s.svc.Moderators},
		{"resumes", s.svc.Resumes},
		{"vacancies", s.svc.Vacancies},
		{"skills", s.svc.Skills},
		{"resume_skills", s.svc.ResumeSkills},
		{"vacancy_skills", s.svc.VacancySkills},
		{"applications", s.svc.Applications},
		{"messages", s.svc.Messages},
		{"reviews", s.svc.Reviews},
	}
}

func (s *Server) handleClearAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	for _, e := range s.allEngines() {
		if err := e.Eng.Reset(); err != nil {
			writeErr(w, fmt.Errorf("%s: %w", e.Name, err))
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleDeleteAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	for _, e := range s.allEngines() {
		_ = e.Eng.Close()
	}

	entries, err := os.ReadDir("data")
	if err != nil && !os.IsNotExist(err) {
		writeErr(w, err)
		return
	}
	for _, e := range entries {
		ext := filepath.Ext(e.Name())
		if ext == ".db" || ext == ".idx" || ext == ".hash" {
			if err := os.Remove(filepath.Join("data", e.Name())); err != nil {
				writeErr(w, fmt.Errorf("remove %s: %w", e.Name(), err))
				return
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "Все БД удалены. Перезапустите приложение - создадутся пустые.",
	})
}
