package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type checkpoints struct {
	path string
	At   map[string]int64 `json:"at"`
	Done map[string]bool  `json:"complete"`
}

func arcticCheckpointKey(sub, endpoint string, start, end time.Time) string {
	return fmt.Sprintf("reddit/%s/%s/%d-%d", sub, endpoint, start.Unix(), end.Unix())
}

func loadCheckpoints(work string) (*checkpoints, error) {
	ck := &checkpoints{
		path: filepath.Join(work, "crawl-progress.json"),
		At:   map[string]int64{},
		Done: map[string]bool{},
	}
	b, err := os.ReadFile(ck.path)
	if os.IsNotExist(err) {
		return ck, nil
	}
	if err != nil {
		return nil, fmt.Errorf("loadCheckpoints: %w", err)
	}
	if err := json.Unmarshal(b, ck); err != nil {
		return nil, fmt.Errorf("%s: %w", ck.path, err)
	}
	if ck.At == nil {
		ck.At = map[string]int64{}
	}
	if ck.Done == nil {
		ck.Done = map[string]bool{}
	}
	return ck, nil
}

func (ck *checkpoints) save() error {
	if ck == nil {
		return nil
	}
	b, err := json.MarshalIndent(ck, "", "  ")
	if err != nil {
		return fmt.Errorf("save: %w", err)
	}
	if err := os.WriteFile(ck.path, append(b, '\n'), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", ck.path, err)
	}
	return nil
}

func (ck *checkpoints) resume(key string, start time.Time) int64 {
	if ck == nil {
		return start.Unix()
	}
	if at, ok := ck.At[key]; ok && at > start.Unix() {
		return at
	}
	return start.Unix()
}

func (ck *checkpoints) mark(key string, cursor int64) {
	if ck != nil {
		ck.At[key] = cursor
	}
}

func (ck *checkpoints) done(key string) {
	if ck != nil {
		ck.Done[key] = true
	}
}

func (ck *checkpoints) complete(key string) bool { return ck != nil && ck.Done[key] }
