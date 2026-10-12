package calculator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
)

// TestSpeedSourceProbe dumps OUR side of the Speed expression.
func TestSpeedSourceProbe(t *testing.T) {
	requireOperatorRun(t, "report-only probe")
	listPath := os.Getenv("SPEED_PROBE")
	if listPath == "" {
		t.Skip("set SPEED_PROBE=<file of build names>")
	}
	requireGameData(t, "3.29")

	fh, err := os.Open(listPath)
	if err != nil {
		t.Fatalf("REFUSING: cannot read %s: %v", listPath, err)
	}
	defer fh.Close()

	var names []string
	sc := bufio.NewScanner(fh)
	for sc.Scan() {
		if n := sc.Text(); n != "" && n[0] != '#' {
			names = append(names, n)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("REFUSING: cannot scan %s: %v", listPath, err)
	}
	if len(names) == 0 {
		t.Fatalf("REFUSING: %s named 0 builds", listPath)
	}

	out := os.Stdout
	if p := os.Getenv("SPEED_PROBE_OUT"); p != "" {
		f, cerr := os.Create(p)
		if cerr != nil {
			t.Fatalf("REFUSING: cannot write SPEED_PROBE_OUT=%s: %v", p, cerr)
		}
		defer f.Close()
		out = f
	}

	corpus := corpusDirForLoadedData(t)
	emitted, skipped := 0, 0
	for _, name := range names {
		xml, err := os.ReadFile(filepath.Join(corpus, name+".xml"))
		if err != nil {
			skipped++
			continue
		}
		b, err := builds.ParseBuild(xml)
		if err != nil {
			skipped++
			continue
		}
		env := (&Calculator{PoB: b}).BuildOutput(OutputModeMain)
		if env == nil || env.Player == nil || env.Player.MainSkill == nil {
			skipped++
			continue
		}
		as := env.Player.MainSkill

		rec := map[string]any{"build": name}
		if as.ActiveEffect != nil && as.ActiveEffect.GrantedEffect != nil {
			if raw := as.ActiveEffect.GrantedEffect.Raw.GetActiveSkill(); raw != nil {
				rec["skill"] = raw.DisplayedName
			}
		}
		flags := map[string]bool{}
		for f, v := range as.SkillFlags {
			if v {
				flags[string(f)] = true
			}
		}
		rec["skillFlags"] = flags

		o := map[string]float64{}
		for _, k := range []string{"Speed", "Time", "Repeats", "ActionSpeedMod", "CastRate",
			"TriggerRateCap", "EffectiveSourceRate", "SkillTriggerRate", "TriggerTime"} {
			o[k] = env.Player.Output[k]
		}
		if mh := env.Player.OutputTable[OutTableMainHand]; mh != nil {
			o["MainHand.Speed"] = mh["Speed"]
			o["MainHand.Time"] = mh["Time"]
		}
		rec["output"] = o

		if cfg := as.SkillCfg; cfg != nil {
			rec["cfgFlags"] = fmt.Sprintf("0x%x", cfg.Flags.Get())
			rec["cfgKeywordFlags"] = fmt.Sprintf("0x%x", cfg.KeywordFlags.Get())
			rec["inc"] = as.SkillModList.Sum(mod.TypeIncrease, cfg, "Speed")
			rec["more"] = as.SkillModList.More(cfg, "Speed")
			rec["baseSpeedSum"] = as.SkillModList.Sum(mod.TypeBase, cfg, "Speed")

			srcs := map[string][]map[string]any{}
			for _, mt := range []mod.Type{mod.TypeIncrease, mod.TypeMore, mod.TypeBase, mod.TypeOverride} {
				for _, r := range as.SkillModList.Tabulate(mt, cfg, "Speed") {
					tags := make([]string, 0, len(r.Mod.Tags()))
					for _, tg := range r.Mod.Tags() {
						tags = append(tags, fmt.Sprintf("%s:%+v", tg.Type(), tg))
					}
					srcs[string(mt)] = append(srcs[string(mt)], map[string]any{
						"source": string(r.Mod.GetSource()),
						"value":  r.Value,
						"flags":  fmt.Sprintf("0x%x", r.Mod.Flags()),
						"kflags": fmt.Sprintf("0x%x", r.Mod.KeywordFlags()),
						"tags":   tags,
					})
				}
			}
			rec["speedSources"] = srcs
		}
		if as.Weapon1Cfg != nil {
			rec["w1Flags"] = fmt.Sprintf("0x%x", as.Weapon1Cfg.Flags.Get())
			rec["w1inc"] = as.SkillModList.Sum(mod.TypeIncrease, as.Weapon1Cfg, "Speed")
			rec["w1more"] = as.SkillModList.More(as.Weapon1Cfg, "Speed")
		}
		rec["weapon1Flags"] = fmt.Sprintf("0x%x", as.Weapon1Flags)
		rec["weapon2Flags"] = fmt.Sprintf("0x%x", as.Weapon2Flags)
		if env.Player.WeaponData1 != nil {
			rec["attackRate1"] = env.Player.WeaponData1.AttackRate
			rec["attackSpeedInc1"] = env.Player.WeaponData1.AttackSpeedInc
		}
		if env.Player.WeaponData2 != nil {
			rec["attackRate2"] = env.Player.WeaponData2.AttackRate
		}
		if as.SkillData != nil {
			rec["attackSpeedMultiplier"] = as.SkillData.AttackSpeedMultiplier
		}

		enc, err := json.Marshal(rec)
		if err != nil {
			t.Fatalf("REFUSING: cannot encode %s: %v", name, err)
		}
		fmt.Fprintln(out, string(enc))
		emitted++
	}
	if emitted == 0 {
		t.Fatalf("REFUSING: emitted 0 of %d builds (%d skipped) — empty must mean "+
			"'I looked and found nothing', never 'I did not look'", len(names), skipped)
	}
	fmt.Fprintf(os.Stderr, "=== SPEED SOURCE PROBE === emitted %d, skipped %d\n", emitted, skipped)
}
