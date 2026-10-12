package exposition

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

var treacheryOverride = regexp.MustCompile(`(?s)<Override [^>]*nodeId="39836" dn="Runegraft of Treachery">.*?</Override>`)

// TestRemoveTattooMatchesTheSavedTree: "Reset Node" on 0007's Runegraft of Treachery (node 39836) calculates as 0007 saved without it.
func TestRemoveTattooMatchesTheSavedTree(t *testing.T) {
	initItemData(t)
	const file = "0007-elementalist-storm-burst-of-repulsion.xml"
	build := corpusBuild(t, file)
	xml, err := os.ReadFile(filepath.Join("..", "..", "testdata", "corpus-3.29", file))
	if err != nil {
		t.Fatal(err)
	}
	edited := treacheryOverride.ReplaceAll(xml, nil)
	if len(edited) == len(xml) {
		t.Fatal("REFUSING: the Runegraft of Treachery override was not found")
	}
	saved, err := builds.ParseBuild(edited)
	if err != nil {
		t.Fatal(err)
	}
	if got := build.TattooOn(39836); got != "Runegraft of Treachery" {
		t.Fatalf("TattooOn(39836) = %q", got)
	}
	count := build.TattooCount()
	if !build.RemoveTattoo(39836) || build.TattooOn(39836) != "" {
		t.Fatal("RemoveTattoo did not reset 39836")
	}
	if build.RemoveTattoo(39836) {
		t.Error("resetting a node with no tattoo reported a change")
	}
	if build.TattooCount() != count {
		t.Errorf("removing a runegraft changed the tattoo count %d -> %d", count, build.TattooCount())
	}
	got, want := GetMiscCalculator(build).Base().Output, GetMiscCalculator(saved).Base().Output
	base := GetMiscCalculator(corpusBuild(t, file)).Base().Output
	differs := false
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %v after the reset, %v saved without the runegraft", k, got[k], v)
		}
		differs = differs || base[k] != v
	}
	if !differs {
		t.Error("vacuity: the runegraft changes nothing")
	}
}
