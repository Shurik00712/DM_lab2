package backup

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var mu sync.Mutex

func Create(w io.Writer, dataDir string) error {
	mu.Lock()
	defer mu.Unlock()
	zw := zip.NewWriter(w)
	defer zw.Close()

	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return fmt.Errorf("read dir: %w", err)
	}
	added := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".db" && ext != ".idx" && ext != ".hash" && ext != ".btree" {
			continue
		}
		if err := addFile(zw, filepath.Join(dataDir, name), name); err != nil {
			return fmt.Errorf("add %s: %w", name, err)
		}
		added++
	}
	if added == 0 {
		return errors.New("nothing to backup: no .db/.idx/.hash/.btree files")
	}
	return nil
}

func addFile(zw *zip.Writer, path, name string) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, src)
	return err
}

func Restore(zipPath, dataDir string) error {
	mu.Lock()
	defer mu.Unlock()

	if _, err := os.Stat(dataDir); err != nil {
		return fmt.Errorf("data dir %q: %w", dataDir, err)
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer zr.Close()

	staging := dataDir + ".restore"
	if err := os.RemoveAll(staging); err != nil {
		return err
	}
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return err
	}
	success := false
	defer func() {
		if !success {
			os.RemoveAll(staging)
		}
	}()
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(f.Name))
		if ext != ".db" && ext != ".idx" && ext != ".hash" && ext != ".btree" {
			return fmt.Errorf("unexpected file: %s", f.Name)
		}
		if strings.ContainsAny(f.Name, "/\\") || strings.Contains(f.Name, "..") {
			return fmt.Errorf("invalid name: %s", f.Name)
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		dst, err := os.Create(filepath.Join(staging, f.Name))
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(dst, rc)
		dst.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}

	old := dataDir + ".old"
	os.RemoveAll(old)
	if err := os.Rename(dataDir, old); err != nil {
		return fmt.Errorf("rename data → old: %w (на Windows закройте приложение)", err)
	}
	if err := os.Rename(staging, dataDir); err != nil {
		_ = os.Rename(old, dataDir)
		return fmt.Errorf("rename staging → data: %w", err)
	}
	success = true
	os.RemoveAll(old)
	return nil
}