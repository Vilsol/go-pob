package storage

import (
	"strings"
)

type DirEntryType string

const (
	DirEntryTypeFile = DirEntryType("file")
	DirEntryTypeDir  = DirEntryType("dir")
)

type DirEntry struct {
	Name     string
	Type     DirEntryType
	Class    string
	LastEdit string
	Level    int
}

type Storage interface {
	GetCache(key string) ([]byte, error)
	SetCache(key string, value []byte) error
	ExistsInCache(key string) bool

	List(dir string) ([]DirEntry, error)
	GetBuild(key string) (string, error)
	SetBuild(key string, value string) error
	ExistsInBuild(key string) bool
	NewFolder(dir string) error
	Delete(key string) error
}

func ListBuilds(dir string) ([]DirEntry, error) {
	return Get().List(strings.ReplaceAll(dir, "//", "/"))
}

func NewFolder(path string) error {
	return Get().NewFolder(strings.ReplaceAll(path, "//", "/"))
}

func GetBuild(path string) (string, error) {
	return Get().GetBuild(strings.ReplaceAll(path, "//", "/"))
}

func SetBuild(path string, value string) error {
	return Get().SetBuild(strings.ReplaceAll(path, "//", "/"), value)
}

func DeleteBuild(path string) error {
	return Get().Delete(strings.ReplaceAll(path, "//", "/"))
}
