package web

import (
	"file_db/internal/engine"
	"file_db/internal/schema"
)

type FieldInfo struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Index string `json:"index,omitempty"` // "pk" | "hash" | "btree" | ""
}

type entityOps struct {
	Name    string
	Fields  []FieldInfo
	FindAll func() ([]map[string]any, error)
	Add     func(map[string]any) error
	Get     func(any) (map[string]any, error)
	Update  func(any, map[string]any) error
	Delete  func(any) error
}

func typeName(t schema.FieldType) string {
	switch t {
	case schema.TypeInt64:
		return "int64"
	case schema.TypeFloat64:
		return "float64"
	case schema.TypeStr:
		return "str"
	case schema.TypeBool:
		return "bool"
	case schema.TypeDate:
		return "date"
	}
	return "unknown"
}

func fieldsOf(e *engine.Engine) []FieldInfo {
	s := e.Schema()
	out := make([]FieldInfo, len(s.Fields))
	for i, f := range s.Fields {
		idx := ""
		switch {
		case f.Flags&schema.FlagPK != 0:
			idx = "pk"
		case f.Flags&schema.FlagHash != 0:
			idx = "hash"
		case f.Flags&schema.FlagBTree != 0:
			idx = "btree"
		}
		out[i] = FieldInfo{Name: f.Name, Type: typeName(f.Type), Index: idx}
	}
	return out
}

var entityNames = []string{
	"vacancies", "resumes", "applications",
	"seekers", "employers", "skills", "reviews",
	"messages", "moderators", "resume_skills", "vacancy_skills",
}

func (s *Server) engineOf(name string) *engine.Engine {
	switch name {
	case "vacancies":
		return s.svc.Vacancies
	case "resumes":
		return s.svc.Resumes
	case "applications":
		return s.svc.Applications
	case "seekers":
		return s.svc.Seekers
	case "employers":
		return s.svc.Employers
	case "skills":
		return s.svc.Skills
	case "reviews":
		return s.svc.Reviews
	case "messages":
		return s.svc.Messages
	case "moderators":
		return s.svc.Moderators
	case "resume_skills":
		return s.svc.ResumeSkills
	case "vacancy_skills":
		return s.svc.VacancySkills
	}
	return nil
}

func (s *Server) entityOps(name string) entityOps {
	wrap := func(e *engine.Engine) entityOps {
		return entityOps{
			Name:    name,
			Fields:  fieldsOf(e),
			FindAll: e.FindAll,
			Add:     e.Add,
			Get:     e.GetByPK,
			Update:  e.Update,
			Delete:  e.DeleteByPK,
		}
	}
	switch name {
	case "vacancies":
		e := s.svc.Vacancies
		return entityOps{Name: name, Fields: fieldsOf(e), FindAll: e.FindAll,
			Add: s.svc.AddVacancy, Get: s.svc.GetVacancy, Update: s.svc.UpdateVacancy, Delete: s.svc.DeleteVacancy}
	case "resumes":
		e := s.svc.Resumes
		return entityOps{Name: name, Fields: fieldsOf(e), FindAll: e.FindAll,
			Add: s.svc.AddResume, Get: s.svc.GetResume, Update: s.svc.UpdateResume, Delete: s.svc.DeleteResume}
	case "applications":
		e := s.svc.Applications
		return entityOps{Name: name, Fields: fieldsOf(e), FindAll: e.FindAll,
			Add: s.svc.AddApplication, Get: s.svc.GetApplication, Update: e.Update, Delete: s.svc.DeleteApplication}
	case "seekers":
		e := s.svc.Seekers
		return entityOps{Name: name, Fields: fieldsOf(e), FindAll: e.FindAll,
			Add: s.svc.AddSeeker, Get: s.svc.GetSeeker, Update: e.Update, Delete: s.svc.DeleteSeeker}
	case "employers":
		e := s.svc.Employers
		return entityOps{Name: name, Fields: fieldsOf(e), FindAll: e.FindAll,
			Add: s.svc.AddEmployer, Get: s.svc.GetEmployer, Update: e.Update, Delete: s.svc.DeleteEmployer}
	case "skills":
		return wrap(s.svc.Skills)
	case "reviews":
		e := s.svc.Reviews
		return entityOps{Name: name, Fields: fieldsOf(e), FindAll: e.FindAll,
			Add: s.svc.AddReview, Get: s.svc.GetReview, Update: e.Update, Delete: s.svc.DeleteReview}
	case "moderators":
		return wrap(s.svc.Moderators)
	case "messages":
		e := s.svc.Messages
		return entityOps{Name: name, Fields: fieldsOf(e), FindAll: e.FindAll,
			Add: s.svc.AddMessage, Get: e.GetByPK, Update: e.Update, Delete: e.DeleteByPK}
	case "resume_skills":
		return wrap(s.svc.ResumeSkills)
	case "vacancy_skills":
		return wrap(s.svc.VacancySkills)
	}
	return entityOps{}
}