package builds

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCharacterLevelAutoModeRoundTrips: Build.lua:964 / 1009: characterLevelAutoMode is read from and written back to <Build>.
func TestCharacterLevelAutoModeRoundTrips(t *testing.T) {
	xml, err := os.ReadFile(filepath.Join("..", "testdata", "corpus-3.29", "0042-pathfinder-withering-step.xml"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	build, err := ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	if !build.Build.CharacterLevelAutoMode {
		t.Fatal("the fixture saves characterLevelAutoMode=\"true\" and it was not read")
	}
	build.SetCharacterLevelAutoMode(false)
	out, err := SerializeBuild(build)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `characterLevelAutoMode="false"`) {
		t.Fatal("the mode was not written back")
	}
}
