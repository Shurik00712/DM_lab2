package web

import (
	"os"
	"io"
	"net/http"
	"strconv"
	"strings"

	"file_db/internal/backup"
	"file_db/internal/export"
	"file_db/internal/service"
)

// GET /api/search/vacancies?city=&salary_from=&employment_type=&status=&is_remote=true|false
func (s *Server) handleSearchVacancies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { http.Error(w, "", http.StatusMethodNotAllowed); return }

	q := r.URL.Query()
	f := service.VacancyFilter{
		City:           q.Get("city"),
		EmploymentType: q.Get("employment_type"),
		Status:         q.Get("status"),
	}
	if sf := q.Get("salary_from"); sf != "" {
		if n, err := strconv.ParseInt(sf, 10, 64); err == nil { f.SalaryFrom = n }
	}
	if ir := q.Get("is_remote"); ir != "" {
		b := ir == "true"
		f.IsRemote = &b
	}

	rows, err := s.svc.SearchVacancies(f)
	if err != nil { writeErr(w, err); return }
	writeJSON(w, http.StatusOK, map[string]any{"rows": rows})
}

// POST /api/moderation/vacancies/{id}/approve
// POST /api/moderation/vacancies/{id}/reject
func (s *Server) handleModeration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { http.Error(w, "", http.StatusMethodNotAllowed); return }

	modID := int64(1)

	path := strings.TrimPrefix(r.URL.Path, "/api/moderation/vacancies/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		http.Error(w, "bad path", http.StatusBadRequest); return
	}
	id, action := parts[0], parts[1]

	var err error
	switch action {
	case "approve": err = s.svc.ApproveVacancy(modID, id)
	case "reject":  err = s.svc.RejectVacancy(modID, id)
	default:        http.Error(w, "unknown action", http.StatusBadRequest); return
	}
	if err != nil { writeErr(w, err); return }
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// POST /api/moderation/reviews/{id}/approve
// POST /api/moderation/reviews/{id}/reject
func (s *Server) handleModerationReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { http.Error(w, "", http.StatusMethodNotAllowed); return }
	modID := int64(1)

	path := strings.TrimPrefix(r.URL.Path, "/api/moderation/reviews/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 { http.Error(w, "bad path", http.StatusBadRequest); return }
	id, action := parts[0], parts[1]

	var err error
	switch action {
	case "approve": err = s.svc.ApproveReview(modID, id)
	case "reject":  err = s.svc.RejectReview(modID, id)
	default:        http.Error(w, "unknown action", http.StatusBadRequest); return
	}
	if err != nil { writeErr(w, err); return }
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// GET    /api/resumes-skills/{resumeID}       
// POST   /api/resumes-skills/{resumeID}/{skillID}
// DELETE /api/resumes-skills/{resumeID}/{skillID} 
func (s *Server) handleResumeSkills(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/resumes-skills/")
	parts := strings.Split(path, "/")

	switch r.Method {
	case http.MethodGet:
		if len(parts) != 1 { http.Error(w, "bad path", http.StatusBadRequest); return }
		skills, err := s.svc.GetResumeSkills(parts[0])
		if err != nil { writeErr(w, err); return }
		writeJSON(w, http.StatusOK, map[string]any{"rows": skills})

	case http.MethodPost, http.MethodDelete:
		if len(parts) != 2 { http.Error(w, "bad path", http.StatusBadRequest); return }
		var err error
		if r.Method == http.MethodPost {
			err = s.svc.AddResumeSkill(parts[0], parts[1])
		} else {
			err = s.svc.RemoveResumeSkill(parts[0], parts[1])
		}
		if err != nil { writeErr(w, err); return }
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})

	default:
		http.Error(w, "", http.StatusMethodNotAllowed)
	}
}

// GET    /api/vacancies-skills/{vacancyID}
// POST   /api/vacancies-skills/{vacancyID}/{skillID}
// DELETE /api/vacancies-skills/{vacancyID}/{skillID}
func (s *Server) handleVacancySkills(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/vacancies-skills/")
	parts := strings.Split(path, "/")

	switch r.Method {
	case http.MethodGet:
		if len(parts) != 1 { http.Error(w, "bad path", http.StatusBadRequest); return }
		skills, err := s.svc.GetVacancySkills(parts[0])
		if err != nil { writeErr(w, err); return }
		writeJSON(w, http.StatusOK, map[string]any{"rows": skills})

	case http.MethodPost, http.MethodDelete:
		if len(parts) != 2 { http.Error(w, "bad path", http.StatusBadRequest); return }
		var err error
		if r.Method == http.MethodPost {
			err = s.svc.AddVacancySkill(parts[0], parts[1])
		} else {
			err = s.svc.RemoveVacancySkill(parts[0], parts[1])
		}
		if err != nil { writeErr(w, err); return }
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})

	default:
		http.Error(w, "", http.StatusMethodNotAllowed)
	}
}

// GET /api/backup
func (s *Server) handleBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { http.Error(w, "", http.StatusMethodNotAllowed); return }

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="backup.zip"`)
	if err := backup.Create(w, "data"); err != nil {
		logErr("backup", err)
	}
}

// POST /api/restore 
func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { http.Error(w, "", http.StatusMethodNotAllowed); return }

	tmp, err := io.ReadAll(r.Body)
	if err != nil { writeErr(w, err); return }

	tmpPath := "data-restore.tmp"
	if err := writeFile(tmpPath, tmp); err != nil { writeErr(w, err); return }

	if err := backup.Restore(tmpPath, "data"); err != nil { writeErr(w, err); return }
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "перезапустите приложение"})
}

// GET /api/export/xlsx?entity=vacancies
func (s *Server) handleExportXLSX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { http.Error(w, "", http.StatusMethodNotAllowed); return }
	entity := r.URL.Query().Get("entity")
	if entity == "" { entity = "vacancies" }

	ops := s.entityOps(entity)
	if ops.FindAll == nil { http.Error(w, "unknown entity", http.StatusBadRequest); return }

	rows, err := ops.FindAll()
	if err != nil { writeErr(w, err); return }

	var eng = s.svc.Vacancies
	switch entity {
	case "resumes": eng = s.svc.Resumes
	case "applications": eng = s.svc.Applications
	case "seekers": eng = s.svc.Seekers
	case "employers": eng = s.svc.Employers
	case "skills": eng = s.svc.Skills
	case "reviews": eng = s.svc.Reviews
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="`+entity+`.xlsx"`)
	if err := export.XLSX(w, entity, rows, eng.Schema()); err != nil {
		logErr("xlsx", err)
	}
}

// GET /api/counts
func (s *Server) handleCounts(w http.ResponseWriter, r *http.Request) {
	out := map[string]int{}
	for name, eng := range map[string]interface{ Count() (int, error) }{
		"vacancies":    s.svc.Vacancies,
		"resumes":      s.svc.Resumes,
		"applications": s.svc.Applications,
		"seekers":      s.svc.Seekers,
		"employers":    s.svc.Employers,
		"skills":       s.svc.Skills,
		"reviews":      s.svc.Reviews,
	} {
		n, _ := eng.Count()
		out[name] = n
	}
	writeJSON(w, http.StatusOK, out)
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}