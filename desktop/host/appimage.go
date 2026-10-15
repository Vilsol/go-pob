package host

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func IntegrateAppImage(ctx context.Context, appImage, dataHome string, entry, icon []byte) (string, error) {
	apps := filepath.Join(dataHome, "applications")
	icons := filepath.Join(dataHome, "icons", "hicolor", "512x512", "apps")
	for _, dir := range []string{apps, icons} {
		if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // the user's own XDG data folder
			return "", fmt.Errorf("applications menu: %w", err)
		}
	}
	if err := os.WriteFile(filepath.Join(icons, "go-pob.png"), icon, 0o644); err != nil { //nolint:gosec // a menu icon is world-readable
		return "", fmt.Errorf("applications menu icon: %w", err)
	}
	if err := os.WriteFile(filepath.Join(apps, "go-pob.desktop"), AppImageEntry(entry, appImage), 0o644); err != nil { //nolint:gosec // a desktop entry is world-readable
		return "", fmt.Errorf("applications menu entry: %w", err)
	}
	var notes []string
	for _, cmd := range [][]string{
		{"update-desktop-database", apps},
		{"xdg-mime", "default", "go-pob.desktop", "x-scheme-handler/pob"},
	} {
		if _, err := exec.LookPath(cmd[0]); err != nil {
			notes = append(notes, cmd[0]+" not found")
			continue
		}
		if out, err := exec.CommandContext(ctx, cmd[0], cmd[1:]...).CombinedOutput(); err != nil { //nolint:gosec // fixed commands
			notes = append(notes, fmt.Sprintf("%s: %v %s", cmd[0], err, bytes.TrimSpace(out)))
		}
	}
	return strings.Join(notes, "\n"), nil
}

func AppImageEntry(entry []byte, appImage string) []byte {
	var out bytes.Buffer
	sc := bufio.NewScanner(bytes.NewReader(entry))
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "Exec="):
			line = "Exec=" + execQuote(appImage) + " %u"
		case strings.HasPrefix(line, "TryExec="):
			continue
		}
		out.WriteString(line + "\n")
		if line == "[Desktop Entry]" {
			out.WriteString("TryExec=" + appImage + "\n")
		}
	}
	return out.Bytes()
}

func execQuote(arg string) string {
	arg = strings.ReplaceAll(arg, "%", "%%")
	if !strings.ContainsAny(arg, " \t\n\"'\\><~|&;$*?#()`") {
		return arg
	}
	r := strings.NewReplacer(`\`, `\\\\`, `"`, `\\"`, "`", "\\\\`", `$`, `\\$`)
	return `"` + r.Replace(arg) + `"`
}

func AppImagePath() string {
	p := os.Getenv("APPIMAGE")
	if p == "" {
		return ""
	}
	if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) { //nolint:gosec // the AppImage runtime names its own file
		return ""
	}
	return p
}
