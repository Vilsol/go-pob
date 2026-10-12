package builds

import (
	"strings"
	"testing"
)

// TestSpecWithoutURLOmitsTheElement: PassiveSpec.lua:99-104 refuses a <URL> element with no text.
func TestSpecWithoutURLOmitsTheElement(t *testing.T) {
	out, err := SerializeBuild(EmptyBuild())
	if err != nil {
		t.Fatalf("SerializeBuild: %v", err)
	}
	if strings.Contains(string(out), "<URL") {
		t.Errorf("a spec with no URL saves a <URL> element, which PoB refuses:\n%s", out)
	}
}

// TestSpecURLRoundTrips: The control: a spec that has a URL keeps it, text and all, through a save and a load.
func TestSpecURLRoundTrips(t *testing.T) {
	const url = "https://www.pathofexile.com/passive-skill-tree/AAAABgMCJBBY"
	in := `<?xml version="1.0" encoding="UTF-8"?><PathOfBuilding>` +
		`<Build level="90" className="Witch" ascendClassName="Occultist"/>` +
		`<Tree activeSpec="1"><Spec treeVersion="3_29" nodes="" classId="3" ascendClassId="1"><URL>` + url + `</URL></Spec></Tree>` +
		`</PathOfBuilding>`
	build, err := ParseBuildStr(in)
	if err != nil {
		t.Fatalf("ParseBuild: %v", err)
	}
	out, err := SerializeBuild(build)
	if err != nil {
		t.Fatalf("SerializeBuild: %v", err)
	}
	if !strings.Contains(string(out), "<URL>"+url+"</URL>") {
		t.Errorf("the saved spec lost its URL:\n%s", out)
	}
	back, err := ParseBuild(out)
	if err != nil {
		t.Fatalf("ParseBuild (saved): %v", err)
	}
	if got := back.Tree.Specs[0].URL; got != url {
		t.Errorf("URL read back as %q, want %q", got, url)
	}
}
