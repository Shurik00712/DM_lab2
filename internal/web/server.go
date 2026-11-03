package web

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	"file_db/internal/service"
)

//go:embed static/*
var staticFS embed.FS

type Server struct {
	svc *service.Service
}

func New(svc *service.Service) *Server { return &Server{svc: svc} }

func (s *Server) Run(addr string) error {
	mux := http.NewServeMux()

	static, _ := fs.Sub(staticFS, "static")
	mux.Handle("/", http.FileServer(http.FS(static)))

	for _, name := range entityNames {
		ops := s.entityOps(name)
		mux.HandleFunc("/api/"+name, crudListCreate(ops))
		mux.HandleFunc("/api/"+name+"/search", s.handleSearch(ops))
		mux.HandleFunc("/api/"+name+"/range", s.handleRange(ops))
		mux.HandleFunc("/api/"+name+"/by-field", s.handleDeleteByField(ops))
		mux.HandleFunc("/api/"+name+"/", crudByID(ops))
	}

	mux.HandleFunc("/api/search/vacancies", s.handleSearchVacancies)
	mux.HandleFunc("/api/moderation/vacancies/", s.handleModeration)
	mux.HandleFunc("/api/moderation/reviews/", s.handleModerationReview)
	mux.HandleFunc("/api/resumes-skills/", s.handleResumeSkills)
	mux.HandleFunc("/api/vacancies-skills/", s.handleVacancySkills)
	mux.HandleFunc("/api/backup", s.handleBackup)
	mux.HandleFunc("/api/restore", s.handleRestore)
	mux.HandleFunc("/api/export/xlsx", s.handleExportXLSX)
	mux.HandleFunc("/api/counts", s.handleCounts)
	mux.HandleFunc("/api/db/clear", s.handleClearAll)
	mux.HandleFunc("/api/db/delete", s.handleDeleteAll)
	mux.HandleFunc("/api/db/save", s.handleSaveAll)

	fmt.Println("→ http://" + addr)
	srv := &http.Server{Addr: addr, Handler: mux}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()
	go openBrowser("http://" + addr)
	return srv.ListenAndServe()
}

func (s *Server) handleSaveAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	for _, e := range s.allEngines() {
		if err := e.Eng.Sync(); err != nil {
			writeErr(w, fmt.Errorf("%s: %w", e.Name, err))
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

func logErr(step string, err error) {
	if err != nil {
		log.Printf("%s: %v", step, err)
	}
}