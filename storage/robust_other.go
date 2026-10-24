//go:build !js && !windows

package storage

import "os"

func readFileRobust(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:wrapcheck // os.ReadFile's error, passed on
}

func renameRobust(from, to string) error {
	return os.Rename(from, to) //nolint:wrapcheck // os.Rename's error, passed on
}
