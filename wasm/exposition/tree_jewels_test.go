package exposition

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
)

// TestSocketTreeJewelMatchesTheSavedSocket: Emptying 0008's socket 2311 calculates as 0008 saved without that jewel; putting it back as saved.
func TestSocketTreeJewelMatchesTheSavedSocket(t *testing.T) {
	initItemData(t)
	const file = "0008-necromancer-vaal-summon-skeletons.xml"
	build := corpusBuild(t, file)
	xml, err := os.ReadFile(filepath.Join("..", "..", "testdata", "corpus-"+raw.LatestVersion, file))
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(xml), `<Socket itemId="5" nodeId="2311"/>`, ``, 1)
	if edited == string(xml) {
		t.Fatal("REFUSING: socket 2311 not found")
	}
	saved, err := builds.ParseBuild([]byte(edited))
	if err != nil {
		t.Fatal(err)
	}
	if changed, err := SocketTreeJewel(build, 2311, 0); err != nil || !changed {
		t.Fatalf("SocketTreeJewel(None) = %v, %v", changed, err)
	}
	got, want := GetMiscCalculator(build).Base().Output, GetMiscCalculator(saved).Base().Output
	base := GetMiscCalculator(corpusBuild(t, file)).Base().Output
	differs := false
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %v after emptying, %v saved without it", k, got[k], v)
		}
		differs = differs || base[k] != v
	}
	if !differs {
		t.Error("vacuity: the jewel in 2311 changes nothing")
	}
	if _, err := SocketTreeJewel(build, 2311, 5); err != nil {
		t.Fatal(err)
	}
	got = GetMiscCalculator(build).Base().Output
	for k, v := range base {
		if got[k] != v {
			t.Errorf("%s: %v after putting it back, %v as saved", k, got[k], v)
		}
	}
	if _, err := SocketTreeJewel(build, 2311, 3); err == nil {
		t.Error("a large cluster jewel was accepted in socket 2311")
	}
	found := false
	for _, j := range TreeSocketedJewels(build) {
		found = found || (j.NodeID == 2311 && j.ItemID == 5)
	}
	if !found {
		t.Errorf("TreeSocketedJewels = %+v", TreeSocketedJewels(build))
	}
}
