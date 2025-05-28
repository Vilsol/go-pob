//go:build !js

package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/utils"

	"gopkg.in/djherbis/fscache.v0"
)

var cache *fscache.FSCache

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

	r, _, err := cache.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get key from cache: %s: %w", key, err)
	}

	if r == nil {
		return nil, nil
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read from cache: %w", err)
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

	_ = cache.Remove(key)

	_, w, err := cache.Get(key)
	if err != nil {
		return fmt.Errorf("failed to set key on cache: %s: %w", key, err)
	}

	if w == nil {
		return fmt.Errorf("could not write to cache")
	}

	defer w.Close()

	if _, err := w.Write(value); err != nil {
		return fmt.Errorf("failed to write to cache: %w", err)
	}

	return nil
}

func (d desktopStorage) ExistsInCache(key string) bool {
	return cache.Exists(key)
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
	dir, err := os.UserCacheDir()
	if err != nil {
		panic(fmt.Errorf("failed getting user cache dir: %w", err))
	}

	baseCacheDir := filepath.Join(dir, "go-pob", "bundle-cache")
	if err := os.MkdirAll(baseCacheDir, 0777); err != nil {
		if !os.IsExist(err) {
			panic(fmt.Errorf("failed creating cache dir: %w", err))
		}
	}

	cache, err = fscache.New(baseCacheDir, 0755, time.Hour*24*30) // 30 day cache
	if err != nil {
		panic(fmt.Errorf("failed creating cache: %w", err))
	}

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
