package calculator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
)

// TestSpectreAllyLifePool covers CalcPerform.lua's spectre arm.
func TestSpectreAllyLifePool(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true required")
	}
	const build = "0518-reliquarian-dark-bargain"

	raw, err := os.ReadFile(filepath.Join(dumpsDirForLoadedData(t), build+".json"))
	if err != nil {
		t.Fatalf("REFUSING: no dump for %s: %v", build, err)
	}
	var d struct {
		Output map[string]any `json:"output"`
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatalf("REFUSING: cannot parse the dump: %v", err)
	}
	want, _ := d.Output["TotalSpectreLife"].(float64)
	if want == 0 {
		t.Fatalf("REFUSING: PoB reports no TotalSpectreLife for %s. This was the ONLY build in the "+
			"corpus carrying takenFromSpectresBeforeYou; if that is no longer true the spectre arm "+
			"is unmeasured here and this test is grading nothing.", build)
	}
	limit, _ := d.Output["ActiveSpectreLimit"].(float64)
	if limit < 2 {
		t.Fatalf("REFUSING: ActiveSpectreLimit is %g on %s. The top-up at :3680-3687 only fires when "+
			"the limit EXCEEDS the number of Raise Spectre groups, so a limit below 2 gives this "+
			"fixture no work and it would pass without exercising the branch.", limit, build)
	}

	xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), build+".xml"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := builds.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	stripped := 0
	strip := func(in []pob.Input) []pob.Input {
		out := in[:0]
		for _, i := range in {
			if i.Name == "TotalSpectreLife" {
				stripped++
				continue
			}
			out = append(out, i)
		}
		return out
	}
	parsed.Config.Inputs = strip(parsed.Config.Inputs)
	for k := range parsed.Config.ConfigSets {
		parsed.Config.ConfigSets[k].Inputs = strip(parsed.Config.ConfigSets[k].Inputs)
	}
	if stripped == 0 {
		t.Fatalf("REFUSING: %s no longer saves a TotalSpectreLife override; grade the arm against "+
			"PoB's %g directly again", build, want)
	}
	env := (&Calculator{PoB: parsed}).BuildOutput(OutputModeMain)

	got := env.Player.ModDB.Sum(mod.TypeBase, nil, "TotalSpectreLife")
	if got == 0 {
		t.Fatalf("TotalSpectreLife = 0, PoB %g — the spectre arm did not fire at all", want)
	}

	perSpectre := float64(0)
	for _, as := range env.Player.ActiveSkillList {
		if as.SkillFlags[SkillFlagSpectre] && as.Minion != nil && as.Minion.Output != nil {
			perSpectre = as.Minion.Output["Life"]
			break
		}
	}
	if perSpectre <= 0 {
		t.Fatalf("REFUSING: no Raise Spectre minion with Life on %s, so the top-up has nothing to fill with", build)
	}
	if got != perSpectre*limit {
		t.Errorf("TotalSpectreLife %g is not %g x the first spectre's Life %g — the :3680-3687 top-up "+
			"did not fill to the limit", got, limit, perSpectre)
	}
}
