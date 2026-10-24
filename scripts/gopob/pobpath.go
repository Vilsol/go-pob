package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// pobPinDir mirrors the Taskfile's POB_DEFAULT: POB_DIR, else the PathOfBuilding checkout beside the repository.
func pobPinDir() string { return pobPinResolve("POB_DIR", "PathOfBuilding") }

// pob2PinDir mirrors POB2_DEFAULT for the PoE2 pin.
func pob2PinDir() string { return pobPinResolve("POB2_DIR", "PathOfBuilding-PoE2") }

func pobPinResolve(env, name string) string {
	if v := os.Getenv(env); v != "" {
		return v
	}
	out, err := exec.CommandContext(context.Background(), "git", "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		return filepath.Join("..", name)
	}
	common := strings.TrimSpace(string(out))
	if r, err := filepath.EvalSymlinks(common); err == nil {
		common = r
	}
	return filepath.Join(common, "..", "..", name)
}
