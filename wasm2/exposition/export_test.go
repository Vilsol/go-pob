package exposition

import (
	"compress/zlib"
	"encoding/base64"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
)

var base64url = regexp.MustCompile(`^[A-Za-z0-9_-]+={0,2}$`)

// ImportTab.lua:136 (PoB-PoE2) writes common.base64.encode(Deflate(xml)) with + and / swapped for - and _; its import
// (ImportTab.lua:263) reverses that, so a code opens as the XML it was made from.
func TestExportCodeOpensAsTheSavedBuild(t *testing.T) {
	src, err := os.ReadFile("../../calculator2/testdata/item_fixture.xml")
	if err != nil {
		t.Fatal(err)
	}
	calc, err := ParseBuild(string(src))
	if err != nil {
		t.Fatal(err)
	}
	player := []SavedStat{{Stat: "Life", Value: 683}, {Stat: "Speed", Value: 1.0 / 0.7}}
	code, err := calc.ExportCode(player, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !base64url.MatchString(code) {
		t.Fatalf("code is not base64url: %q", code)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.NewReplacer("-", "+", "_", "/").Replace(code))
	if err != nil {
		t.Fatal(err)
	}
	r, err := zlib.NewReader(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	xml, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := calc.SerializeWithStats(player, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(xml) != saved {
		t.Fatalf("the code decodes to\n%s\nwant the saved build\n%s", xml, saved)
	}
	reopened, err := ParseBuild(string(xml))
	if err != nil {
		t.Fatal(err)
	}
	again, err := reopened.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if again != saved {
		t.Errorf("the imported code saves as\n%s\nwant\n%s", again, saved)
	}
	if !strings.Contains(again, `<PlayerStat stat="Life" value="683"/>`) {
		t.Errorf("the code lost the saved stats:\n%s", again)
	}
}
