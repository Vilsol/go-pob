//go:build !js

package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/utils"
)

const CacheDirEnv = "GO_POB_CACHE_DIR"

const cacheExpiry = time.Hour * 24 * 30

var cacheDir string

func cachePath(key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(cacheDir, hex.EncodeToString(sum[:]))
}

type desktopStorage struct {
}

func Get() Storage {
	return desktopStorage{}
}

func (d desktopStorage) GetCache(key string) ([]byte, error) {
	slog.Log(
		context.Background(),
		utils.LevelTrace,
		"loading from cache",
		slog.String("key", key),
	)

	b, err := os.ReadFile(cachePath(key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get key from cache: %s: %w", key, err)
	}

	return b, nil
}

func (d desktopStorage) SetCache(key string, value []byte) error {
	slog.Log(
		context.Background(),
		utils.LevelTrace,
		"storing in cache",
		slog.String("key", key),
		slog.Int("len", len(value)),
	)

	tmp, err := os.CreateTemp(cacheDir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create cache temporary file: %w", err)
	}

	if _, err := tmp.Write(value); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("failed to write to cache: %w", err)
	}

	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("failed to close cache temporary file: %w", err)
	}

	if err := os.Rename(tmp.Name(), cachePath(key)); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("failed to set key on cache: %s: %w", key, err)
	}

	return nil
}

func (d desktopStorage) ExistsInCache(key string) bool {
	stat, err := os.Stat(cachePath(key))
	if err != nil {
		return false
	}

	return time.Since(stat.ModTime()) < cacheExpiry
}

func (d desktopStorage) List(dir string) ([]DirEntry, error) {
	baseDir := filepath.Join(d.buildBaseDir(), dir)
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to list builds in dir: %s: %w", dir, err)
	}

	out := make([]DirEntry, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		typ := DirEntryTypeFile
		if e.Type().IsDir() {
			typ = DirEntryTypeDir
		}

		class := ""
		level := 0
		lastEdit := ""
		if e.Type().IsRegular() {
			class = "file"
			level = 0

			fullPath := filepath.Join(baseDir, e.Name())

			stat, err := os.Stat(fullPath)
			if err != nil {
				return nil, fmt.Errorf("failed to stat file: %w", err)
			}

			lastEdit = stat.ModTime().Format(time.RFC3339)

			file, err := os.ReadFile(fullPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read file: %w", err)
			}

			if len(file) > 0 {
				parsedBuild, err := builds.ParseBuildStr(string(file))
				if err != nil {
					return nil, fmt.Errorf("failed to parse build: %w", err)
				}

				class = string(parsedBuild.Build.ClassName)
				level = parsedBuild.Build.Level
			}
		}

		out = append(out, DirEntry{
			Name:     e.Name(),
			Type:     typ,
			Class:    class,
			LastEdit: lastEdit,
			Level:    level,
		})
	}

	return out, nil
}

func (d desktopStorage) GetBuild(key string) (string, error) {
	slog.Log(
		context.Background(),
		utils.LevelTrace,
		"loading build",
		slog.String("key", key),
	)

	file, err := os.ReadFile(filepath.Join(d.buildBaseDir(), key))
	if err != nil {
		return "", fmt.Errorf("failed to read build: %w", err)
	}

	return string(file), nil
}

func (d desktopStorage) SetBuild(key string, value string) error {
	slog.Log(
		context.Background(),
		utils.LevelTrace,
		"storing build",
		slog.String("key", key),
		slog.Int("len", len(value)),
	)

	if err := os.WriteFile(filepath.Join(d.buildBaseDir(), key), []byte(value), 0600); err != nil {
		return fmt.Errorf("failed to write build: %w", err)
	}

	return nil
}

func (d desktopStorage) ExistsInBuild(key string) bool {
	_, err := os.Stat(filepath.Join(d.buildBaseDir(), key))
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		panic(fmt.Errorf("failed checking build: %w", err))
	}
	return true
}

func (d desktopStorage) NewFolder(dir string) error {
	slog.Log(
		context.Background(),
		utils.LevelTrace,
		"creating folder",
		slog.String("dir", dir),
	)

	if err := os.MkdirAll(filepath.Join(d.buildBaseDir(), dir), 0755); err != nil {
		return fmt.Errorf("failed creating directory: %w", err)
	}

	return nil
}

func (d desktopStorage) Delete(dir string) error {
	slog.Log(
		context.Background(),
		utils.LevelTrace,
		"deleting",
		slog.String("dir", dir),
	)

	if err := os.RemoveAll(filepath.Join(d.buildBaseDir(), dir)); err != nil {
		return fmt.Errorf("failed removing path: %w", err)
	}

	return nil
}

func (d desktopStorage) GetSetting(key string) ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(d.settingsDir(), key))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read setting %s: %w", key, err)
	}
	return b, nil
}

func (d desktopStorage) SetSetting(key string, value []byte) error {
	dir := d.settingsDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create the settings directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create settings temporary file: %w", err)
	}
	if _, err := tmp.Write(value); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("failed to write setting %s: %w", key, err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("failed to close settings temporary file: %w", err)
	}
	if err := os.Rename(tmp.Name(), filepath.Join(dir, key)); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("failed to set setting %s: %w", key, err)
	}
	return nil
}

func (d desktopStorage) settingsDir() string {
	return filepath.Join(filepath.Dir(d.buildBaseDir()), "settings")
}

func (d desktopStorage) buildBaseDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		panic(fmt.Errorf("failed getting user config: %w", err))
	}

	return filepath.Join(dir, "go-pob", "builds")
}

func init() {
	InitializeStorage(nil, nil, nil, nil, nil, nil)
}

func InitializeStorage(
	list func(bucket string, dir string) []DirEntry,
	get func(bucket string, key string) []byte,
	set func(bucket string, key string, value []byte),
	exists func(bucket string, key string) bool,
	createFolder func(bucket string, dir string),
	del func(bucket string, path string),
) {
	baseCacheDir := os.Getenv(CacheDirEnv)
	if baseCacheDir == "" {
		dir, err := os.UserCacheDir()
		if err != nil {
			panic(fmt.Errorf("failed getting user cache dir: %w", err))
		}

		baseCacheDir = filepath.Join(dir, "go-pob", "bundle-cache")
	}

	//nolint:gosec // G703: baseCacheDir is the user's own cache dir (GO_POB_CACHE_DIR or os.UserCacheDir).
	if err := os.MkdirAll(baseCacheDir, 0777); err != nil {
		if !os.IsExist(err) {
			panic(fmt.Errorf("failed creating cache dir: %w", err))
		}
	}

	cacheDir = baseCacheDir

	configDir, err := os.UserConfigDir()
	if err != nil {
		panic(fmt.Errorf("failed getting user config directory: %w", err))
	}

	baseBuildDir := filepath.Join(configDir, "go-pob", "builds")
	if err := os.MkdirAll(baseBuildDir, 0777); err != nil {
		if !os.IsExist(err) {
			panic(fmt.Errorf("failed creating user build directory: %w", err))
		}
	}
}
