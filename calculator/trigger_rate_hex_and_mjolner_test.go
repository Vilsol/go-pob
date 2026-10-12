package calculator

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

type trigCarrier struct {
	name    string
	support string
}

// TestVixensAndMjolnerPublishTriggerRates pins CalcTriggers.lua's ["vixen's entrapment"] (:1040-1045) and ["mjolner"] (:1109-1116) entries.
func TestVixensAndMjolnerPublishTriggerRates(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	carriers := []trigCarrier{
		{"11763-ladder-Rossixcx_3616.ifispeakLOtwo", "vixen"},
		{"12301-ladder-TerroristVampire_1238.tradeishumiliation", "vixen"},
		{"12874-ladder-_VizQuel_3382.LASTxddd", "vixen"},
		{"13690-ladder-heloł_3022.wiktortrzeciblastowy", "vixen"},
		{"14191-ladder-maxxiskills_5536.APE_ASS", "vixen"},
		{"15164-ladder-xpoppa_3527.Poppa_erDiablo", "vixen"},
		{"9743-ladder-Dredgeforsix_2105.twomartinilunch", "vixen"},
		{"0745-juggernaut-discharge-of-misery", "mjolner"},
		{"1075-hierophant-ball-lightning-of-orbiting", "mjolner"},
		{"1297-elementalist-discharge-of-misery", "mjolner"},
		{"1511-juggernaut-discharge-of-misery-2", "mjolner"},
		{"1512-juggernaut-discharge-of-misery-3", "mjolner"},
	}

	const tol = 0.005

	perKey := map[string][2]int{}
	var diverged []string
	for _, c := range carriers {
		rec, ok := oracle[c.name+".xml"]
		if !ok {
			t.Fatalf("REFUSING: %s is not in the oracle; the fixture population has moved", c.name)
		}
		if rec.Output["TriggerRateCap"] == 0 {
			t.Fatalf("REFUSING: %s publishes no TriggerRateCap upstream; this build no longer "+
				"exercises the entry", c.name)
		}
		raw, err := os.ReadFile(filepath.Join(corpusDir, c.name+".xml"))
		if err != nil {
			t.Fatalf("REFUSING: %s: %v", c.name, err)
		}
		build, err := builds.ParseBuild(raw)
		if err != nil {
			t.Fatalf("REFUSING: %s: %v", c.name, err)
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		if env == nil || env.Player == nil {
			t.Fatalf("REFUSING: %s produced no player environment", c.name)
		}

		agrees := true
		for _, key := range []string{"TriggerRateCap", "EffectiveSourceRate", "SkillTriggerRate"} {
			exp, has := rec.Output[key]
			if !has {
				continue
			}
			got := env.Player.Output[key]
			if math.IsNaN(got) || math.IsInf(got, 0) {
				t.Fatalf("%s: %s is %v -- a NaN/Inf compares equal to nothing and cannot be graded",
					c.name, key, got)
			}
			if math.Abs(got-exp) > tol*math.Abs(exp) {
				agrees = false
				if len(diverged) < 12 {
					diverged = append(diverged, fmt.Sprintf("%s/%s pob=%g ours=%g", c.name, key, exp, got))
				}
			}
		}
		v := perKey[c.support]
		v[1]++
		if agrees {
			v[0]++
		}
		perKey[c.support] = v
	}

	t.Logf("vixen's entrapment: %d of %d agree; mjolner: %d of %d agree",
		perKey["vixen"][0], perKey["vixen"][1], perKey["mjolner"][0], perKey["mjolner"][1])

	if perKey["vixen"][0] < perKey["vixen"][1] {
		t.Errorf("vixen's entrapment: %d of %d carriers agree; diverged: %v",
			perKey["vixen"][0], perKey["vixen"][1], diverged)
	}
	if perKey["mjolner"][0] < perKey["mjolner"][1] {
		t.Errorf("mjolner: %d of %d carriers agree; diverged: %v",
			perKey["mjolner"][0], perKey["mjolner"][1], diverged)
	}
}
