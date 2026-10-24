package builds

import (
	"os"
	"testing"
)

func TestDetectGame(t *testing.T) {
	poe2, err := os.ReadFile("testdata/poe2_minimal.xml")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		raw  string
		want Game
	}{
		{"PoE2 root", string(poe2), GamePoE2},
		{"PoE1 build", `<?xml version="1.0" encoding="UTF-8"?><PathOfBuilding><Build level="1" targetVersion="3_0"/></PathOfBuilding>`, GamePoE1},
		{"PoE1 build with no targetVersion", `<PathOfBuilding><Build level="1"/></PathOfBuilding>`, GamePoE1},
		{"PoE2 save from before the root rename", `<PathOfBuilding><Build level="1" targetVersion="0_1"/></PathOfBuilding>`, GamePoE2Legacy},
		{"another root", `<Settings/>`, GameUnknown},
		{"not XML", `hello`, GameUnknown},
	}
	for _, c := range cases {
		if got := DetectGame([]byte(c.raw)); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

// A PoE1 build keeps its parse path: ParseBuild still reads it exactly as before.
func TestDetectGameLeavesPoE1ParseAlone(t *testing.T) {
	raw := `<?xml version="1.0" encoding="UTF-8"?><PathOfBuilding><Build level="7" targetVersion="3_0"/><Tree activeSpec="1"><Spec nodes="1,2" treeVersion="3_25"/></Tree></PathOfBuilding>`
	if DetectGame([]byte(raw)) != GamePoE1 {
		t.Fatal("not detected as PoE1")
	}
	b, err := ParseBuildStr(raw)
	if err != nil || b.Build.Level != 7 || len(b.Build.PassiveNodes) != 2 {
		t.Fatalf("PoE1 parse changed: %v %+v", err, b)
	}
}
