//go:build !js

package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MarvinJWendt/testza"
)

// TestDesktopListKeepsEveryBuild: A PoE2 save, or a file that is not a build, lists without a summary instead of failing the whole folder.
func TestDesktopListKeepsEveryBuild(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := Get()
	poe2, err := os.ReadFile(filepath.Join("..", "builds", "testdata", "poe2_minimal.xml"))
	testza.AssertNoError(t, err)
	dir, err := os.UserConfigDir()
	testza.AssertNoError(t, err)
	builds := filepath.Join(dir, "go-pob", "builds")
	testza.AssertNoError(t, os.MkdirAll(builds, 0o777))
	for name, body := range map[string]string{
		"poe1.xml":  `<PathOfBuilding><Build level="12" className="Witch"/><Tree activeSpec="1"><Spec/></Tree></PathOfBuilding>`,
		"poe2.xml":  string(poe2),
		"notes.xml": "not a build",
	} {
		testza.AssertNoError(t, os.WriteFile(filepath.Join(builds, name), []byte(body), 0o600))
	}

	entries, err := s.List("")
	testza.AssertNoError(t, err)
	byName := map[string]DirEntry{}
	for _, e := range entries {
		byName[e.Name] = e
	}
	testza.AssertLen(t, byName, 3)
	testza.AssertEqual(t, "Witch", byName["poe1.xml"].Class)
	testza.AssertEqual(t, 12, byName["poe1.xml"].Level)
	testza.AssertEqual(t, "poe2", byName["poe2.xml"].Game)
}
