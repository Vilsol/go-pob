package host

import (
	"path/filepath"

	"golang.org/x/sys/windows"
)

func UserDocuments(home string) string {
	if p, err := windows.KnownFolderPath(windows.FOLDERID_Documents, windows.KF_FLAG_DEFAULT); err == nil && p != "" {
		return p
	}
	return filepath.Join(home, "Documents")
}
