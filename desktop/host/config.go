package host

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	BuildsDir   string `json:"buildsDir,omitempty"`
	FlatpakHint bool   `json:"flatpakHint,omitempty"`
}

func LoadConfig(path string) Config {
	var c Config
	if b, err := os.ReadFile(path); err == nil { //nolint:gosec // the host's own config file
		_ = json.Unmarshal(b, &c)
	}
	return c
}

func SaveConfig(path string, c Config) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	return nil
}

type BuildsChoice struct {
	Dir    string      `json:"dir"`
	Source string      `json:"source"`
	PoB    []PoBBuilds `json:"pob"`
}

func ChooseBuilds(c Config, pob []PoBBuilds, appDir string) BuildsChoice {
	switch {
	case c.BuildsDir != "":
		return BuildsChoice{Dir: c.BuildsDir, Source: "setting", PoB: pob}
	case len(pob) > 0:
		return BuildsChoice{Dir: pob[0].Dir, Source: "pob", PoB: pob}
	default:
		return BuildsChoice{Dir: appDir, Source: "app", PoB: pob}
	}
}

const flatpakHint = "Builds folder\nBuilds are kept in go-pob's own folder. To use Path of Building's Builds folder, choose it with File › Builds folder…"

func FlatpakHint(c Config, choice BuildsChoice, flatpak bool) string {
	if !flatpak || c.FlatpakHint || choice.Source != "app" {
		return ""
	}
	return flatpakHint
}
