//go:build !windows

package host

import "path/filepath"

func UserDocuments(home string) string { return filepath.Join(home, "Documents") }
