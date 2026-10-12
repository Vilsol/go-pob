package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestSourcelessModCensus is a throwaway diagnostic.
func TestSourcelessModCensus(t *testing.T) {
	if os.Getenv("PROBE") != "true" {
		t.Skip("set PROBE=true")
	}
	requireGameData(t, "3.29")

	for _, build := range []string{"0001-necromancer-unknown.xml", "0002-pathfinder-unknown.xml"} {
		env := corpusEnvForLoadedData(t, build)

		found := 0
		for _, m := range env.ModDB.Tabulate(mod.TypeBase, nil, "EnergyShield") {
			src := string(m.Mod.GetSource())
			if src == "" {
				src = "<none>"
			}
			t.Logf("%s  BASE  src=%-44s val=%v", build, src, m.Value)
			found++
		}
		t.Logf("%s: %d EnergyShield BASE mods", build, found)

		for _, v := range env.ModDB.List(nil, "ExtraAura") {
			ea, ok := v.(mod.ExtraAura)
			if !ok {
				continue
			}
			val := 0.0
			if f := ea.Mod.Value(); f != nil {
				val = f.Float()
			}
			t.Logf("%s  EXTRAAURA name=%-24s val=%-10v onlyAllies=%v src=%q",
				build, ea.Mod.Name(), val, ea.OnlyAllies, string(ea.Mod.GetSource()))
		}

		sourceless := map[string]int{}
		total, none := 0, 0
		for name, mods := range env.ModDB.Mods {
			for _, m := range mods {
				total++
				if string(m.GetSource()) == "" {
					none++
					sourceless[name]++
				}
			}
		}
		t.Logf("%s: %d of %d mods carry NO source, across %d distinct names", build, none, total, len(sourceless))
		shown := 0
		for name, n := range sourceless {
			t.Logf("    sourceless: %-40s x%d", name, n)
			if shown++; shown > 24 {
				t.Logf("    ... %d more names", len(sourceless)-shown)
				break
			}
		}

		if n := len(env.ModDB.Tabulate(mod.TypeBase, nil, "ZZZNotAModName")); n != 0 {
			t.Fatalf("REFUSING: control returned %d mods for a nonsense name; this query cannot "+
				"distinguish present from absent", n)
		}
		if found == 0 {
			t.Fatalf("REFUSING: no EnergyShield BASE mods at all on %s, but the corpus gate "+
				"reports a nonzero EnergyShield there — the query is wrong, not the tree", build)
		}
	}
}
