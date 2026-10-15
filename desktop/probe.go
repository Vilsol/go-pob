package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

func probe(native *http.ServeMux, quit func()) string {
	path := os.Getenv("GOPOB_DESKTOP_PROBE")
	if path == "" {
		return ""
	}
	js, err := os.ReadFile(path) //nolint:gosec // the test runner names the probe
	if err != nil {
		log.Fatal(err)
	}
	native.HandleFunc("POST /probe", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		log.Printf("probe: %s", b)
		w.WriteHeader(http.StatusNoContent)
	})
	native.HandleFunc("POST /probe/quit", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		go quit()
	})
	return string(js)
}

func probeDialogs(native dialogs) dialogs {
	s := &scriptedDialogs{saveTo: os.Getenv("GOPOB_DESKTOP_SAVE_TO"), folder: os.Getenv("GOPOB_DESKTOP_FOLDER")}
	if open := os.Getenv("GOPOB_DESKTOP_OPEN"); open != "" {
		s.open = filepath.SplitList(open)
	}
	if os.Getenv("GOPOB_DESKTOP_PROBE") == "" || (s.saveTo == "" && s.folder == "" && s.open == nil) {
		return native
	}
	return s
}

type scriptedDialogs struct {
	saveTo, folder string

	mu   sync.Mutex
	open []string
}

func (s *scriptedDialogs) ChooseFolder(string) (string, error) { return s.folder, nil }

func (s *scriptedDialogs) SaveFile(suggested string) (string, error) {
	if s.saveTo == "" {
		return "", nil
	}
	return filepath.Join(s.saveTo, suggested), nil
}

func (s *scriptedDialogs) OpenBuild() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.open) == 0 {
		return "", nil
	}
	path := s.open[0]
	s.open = append(s.open[1:], path)
	return path, nil
}
