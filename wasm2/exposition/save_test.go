package exposition

import (
	"math"
	"strings"
	"testing"
)

// Build.lua:1196-1244 (PoB-PoE2) writes the rows afresh after Spectre and BeastCompanion and before Buffs.
func TestSerializeWithStatsReplacesTheLoadedRows(t *testing.T) {
	in := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<PathOfBuilding2>\n\t<Build level=\"1\" className=\"Ranger\">\n" +
		"\t\t<Buffs buffList=\"\"/>\n\t\t<PlayerStat stat=\"Life\" value=\"1\"/>\n\t\t<Spectre id=\"Metadata/Monsters/A\"/>\n" +
		"\t\t<MinionStat stat=\"Life\" value=\"2\"/>\n\t\t<FullDPSSkill stat=\"Old\" value=\"3\" skillPart=\"\" source=\"\"/>\n\t</Build>\n</PathOfBuilding2>\n"
	calc, err := ParseBuild(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := calc.SerializeWithStats(
		[]SavedStat{{Stat: "Life", Value: 683}, {Stat: "Speed", Value: 1.0 / 0.7}, {Stat: "Spec:LifeInc", Value: 5}, {Stat: "Cap", Value: math.Inf(1)}},
		nil,
		[]SavedStat{{Stat: "Speed", Value: 2}, {Stat: "Speed", Value: 2}},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := "\t\t<Spectre id=\"Metadata/Monsters/A\"/>\n" +
		"\t\t<PlayerStat stat=\"Life\" value=\"683\"/>\n" +
		"\t\t<PlayerStat stat=\"Speed\" value=\"1.4285714285714\"/>\n" +
		"\t\t<PlayerStat stat=\"Spec:LifeInc\" value=\"5\"/>\n" +
		"\t\t<PlayerStat stat=\"Cap\" value=\"inf\"/>\n" +
		"\t\t<MinionStat stat=\"Speed\" value=\"2\"/>\n" +
		"\t\t<MinionStat stat=\"Speed\" value=\"2\"/>\n" +
		"\t\t<Buffs buffList=\"\"/>\n"
	if !strings.Contains(out, want) || strings.Contains(out, "Old") || strings.Count(out, "<PlayerStat") != 4 {
		t.Errorf("saved Build:\n%s\nwant it to hold, in order:\n%s", out, want)
	}
	reopened, err := ParseBuild(out)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.build.Build.PlayerStats; len(got) != 4 || got[1].Value != 1.4285714285714 {
		t.Errorf("reopened PlayerStat rows %+v", got)
	}
	again, err := calc.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(again, "<PlayerStat stat=\"Life\" value=\"1\"/>") {
		t.Errorf("SerializeWithStats changed the open build:\n%s", again)
	}
}
