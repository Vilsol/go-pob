package host

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

type PoBBuilds struct {
	Dir      string `json:"dir"`
	Settings string `json:"settings"`
}

var prefixGlobs = []string{
	".wine",
	".steam/steam/steamapps/compatdata/*/pfx",
	".local/share/Steam/steamapps/compatdata/*/pfx",
	".var/app/com.valvesoftware.Steam/.local/share/Steam/steamapps/compatdata/*/pfx",
	"Games/*",
	".local/share/bottles/bottles/*",
	".var/app/com.usebottles.bottles/data/bottles/bottles/*",
}

var windowsDrive = regexp.MustCompile(`^([A-Za-z]):[\\/]`)

func FindPoBBuilds(home, winePrefix, documents string) []PoBBuilds {
	var prefixes []string
	if winePrefix != "" {
		prefixes = append(prefixes, winePrefix)
	}
	for _, g := range prefixGlobs {
		m, _ := filepath.Glob(filepath.Join(home, g))
		prefixes = append(prefixes, m...)
	}

	type userPath struct{ dir, prefix string }
	var users []userPath
	for _, p := range prefixes {
		for _, docs := range []string{"Documents", "My Documents"} {
			m, _ := filepath.Glob(filepath.Join(p, "drive_c", "users", "*", docs, "Path of Building"))
			for _, u := range m {
				users = append(users, userPath{u, p})
			}
		}
	}
	users = append(users, userPath{filepath.Join(documents, "Path of Building"), ""})

	seen := map[string]bool{}
	var out []PoBBuilds
	for _, u := range users {
		if st, err := os.Stat(u.dir); err != nil || !st.IsDir() {
			continue
		}
		settings := filepath.Join(u.dir, "Settings.xml")
		dir := filepath.Join(u.dir, "Builds")
		if custom := settingsBuildPath(settings); custom != "" {
			if mapped := wineToUnix(custom, u.prefix); mapped != "" {
				dir = mapped
			}
		}
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			continue
		}
		resolved, err := filepath.EvalSymlinks(dir)
		if err != nil || seen[resolved] {
			continue
		}
		seen[resolved] = true
		if _, err := os.Stat(settings); err != nil {
			settings = ""
		}
		out = append(out, PoBBuilds{Dir: dir, Settings: settings})
	}
	slices.SortStableFunc(out, func(a, b PoBBuilds) int { return modTime(b.Dir).Compare(modTime(a.Dir)) })
	return out
}

func modTime(path string) time.Time {
	st, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return st.ModTime()
}

func settingsBuildPath(path string) string {
	b, err := os.ReadFile(path) //nolint:gosec // PoB's own Settings.xml
	if err != nil {
		return ""
	}
	var s struct {
		Misc struct {
			BuildPath string `xml:"buildPath,attr"`
		} `xml:"Misc"`
	}
	if xml.Unmarshal(b, &s) != nil {
		return ""
	}
	return s.Misc.BuildPath
}

func wineToUnix(path, prefix string) string {
	m := windowsDrive.FindStringSubmatch(path)
	if m == nil || prefix == "" {
		if filepath.IsAbs(path) {
			return filepath.Clean(path)
		}
		return ""
	}
	rest := filepath.FromSlash(strings.ReplaceAll(path[3:], `\`, "/"))
	drive := strings.ToLower(m[1])
	if root, err := filepath.EvalSymlinks(filepath.Join(prefix, "dosdevices", drive+":")); err == nil {
		return filepath.Join(root, rest)
	}
	if drive == "c" {
		return filepath.Join(prefix, "drive_c", rest)
	}
	return ""
}
