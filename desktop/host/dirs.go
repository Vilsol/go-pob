package host

import (
	"fmt"
	"os"
	"path/filepath"
)

type Dirs struct {
	Config    string
	Cache     string
	Settings  string
	AppBuilds string
	Builds    func() string
}

func DefaultDirs() (Dirs, error) {
	config, err := os.UserConfigDir()
	if err != nil {
		return Dirs{}, fmt.Errorf("config folder: %w", err)
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return Dirs{}, fmt.Errorf("cache folder: %w", err)
	}
	appBuilds := filepath.Join(config, "go-pob", "builds")
	return Dirs{
		Config:    filepath.Join(config, "go-pob"),
		Cache:     filepath.Join(cache, "go-pob", "webview"),
		Settings:  filepath.Join(config, "go-pob", "settings"),
		AppBuilds: appBuilds,
		Builds:    func() string { return appBuilds },
	}, nil
}

func (d Dirs) Root(bucket string) (string, bool) {
	var dir string
	switch bucket {
	case "builds":
		dir = d.Builds()
	case "cache":
		dir = d.Cache
	case "settings":
		dir = d.Settings
	default:
		return "", false
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", false
	}
	return dir, true
}
