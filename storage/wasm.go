//go:build js

package storage

import (
	"context"
	"encoding/base64"
	"log/slog"

	"github.com/Vilsol/go-pob/utils"
)

var store *wasmStorage

type wasmStorage struct {
	list         func(bucket string, dir string) []DirEntry
	get          func(bucket string, key string) []byte
	set          func(bucket string, key string, value []byte)
	exists       func(bucket string, key string) bool
	createFolder func(bucket string, dir string)
	delete       func(bucket string, path string)
}

func Get() Storage {
	if store == nil {
		return &wasmStorage{
			list: func(bucket string, dir string) []DirEntry {
				return nil
			},
			get: func(bucket string, key string) []byte {
				return nil
			},
			set: func(bucket string, key string, value []byte) {
			},
			exists: func(bucket string, key string) bool {
				return false
			},
			createFolder: func(bucket string, dir string) {
			},
			delete: func(bucket string, path string) {
			},
		}
	}

	return store
}

func (d *wasmStorage) GetCache(key string) ([]byte, error) {
	hash := base64.RawURLEncoding.EncodeToString([]byte(key))

	slog.Log(
		context.Background(),
		utils.LevelTrace,
		"loading from cache",
		slog.String("key", key),
		slog.String("hash", hash),
	)

	return d.get("cache", hash), nil
}

func (d *wasmStorage) SetCache(key string, value []byte) error {
	hash := base64.RawURLEncoding.EncodeToString([]byte(key))

	slog.Log(
		context.Background(),
		utils.LevelTrace,
		"storing in cache",
		slog.String("key", key),
		slog.String("hash", hash),
		slog.Int("len", len(value)),
	)

	d.set("cache", hash, value)

	return nil
}

func (d *wasmStorage) ExistsInCache(key string) bool {
	hash := base64.RawURLEncoding.EncodeToString([]byte(key))
	return d.exists("cache", hash)
}

func (d *wasmStorage) List(dir string) ([]DirEntry, error) {
	return d.list("builds", dir), nil
}

func (d *wasmStorage) GetBuild(key string) (string, error) {
	slog.Log(
		context.Background(),
		utils.LevelTrace,
		"loading build",
		slog.String("key", key),
	)
	return string(d.get("builds", key)), nil
}

func (d *wasmStorage) SetBuild(key string, value string) error {
	slog.Log(
		context.Background(),
		utils.LevelTrace,
		"storing build",
		slog.String("key", key),
		slog.Int("len", len(value)),
	)
	d.set("builds", key, []byte(value))
	return nil
}

func (d *wasmStorage) NewFolder(dir string) error {
	slog.Log(
		context.Background(),
		utils.LevelTrace,
		"creating folder",
		slog.String("dir", dir),
	)
	d.createFolder("builds", dir)
	return nil
}

func (d *wasmStorage) Delete(dir string) error {
	slog.Log(
		context.Background(),
		utils.LevelTrace,
		"deleting",
		slog.String("dir", dir),
	)
	d.delete("builds", dir)
	return nil
}

func (d *wasmStorage) ExistsInBuild(key string) bool {
	return d.exists("builds", key)
}

func InitializeStorage(
	list func(bucket string, dir string) []DirEntry,
	get func(bucket string, key string) []byte,
	set func(bucket string, key string, value []byte),
	exists func(bucket string, key string) bool,
	createFolder func(bucket string, dir string),
	del func(bucket string, path string),
) {
	store = &wasmStorage{
		list:         list,
		get:          get,
		set:          set,
		exists:       exists,
		createFolder: createFolder,
		delete:       del,
	}
}
