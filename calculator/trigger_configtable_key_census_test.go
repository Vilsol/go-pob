package calculator

import (
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestTriggerConfigTableKeyCensus is the configTable key census AT THE DISPATCH RUNG.
func TestTriggerConfigTableKeyCensus(t *testing.T) {
	if os.Getenv("KEYCENSUS") != "true" || os.Getenv("CORPUS") != "true" {
		t.Skip("set KEYCENSUS=true CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)
	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		t.Fatalf("%v", err)
	}

	all := map[string]int{}
	withRate := map[string]int{}
	served := map[string]int{}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		raw, err := os.ReadFile(corpusDir + "/" + name)
		if err != nil {
			continue
		}
		build, err := builds.ParseBuild(raw)
		if err != nil {
			continue
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		if env == nil || env.Player == nil || env.Player.MainSkill == nil {
			continue
		}
		ms := env.Player.MainSkill
		if ms.ActiveEffect == nil || ms.ActiveEffect.GrantedEffect == nil {
			continue
		}
		var keys []string
		keys = append(keys, ms.ActiveEffect.GrantedEffect.ID())
		keys = append(keys, strings.ToLower(ms.ActiveEffect.GrantedEffect.Name()))
		if ms.TriggeredBy != nil && ms.TriggeredBy.GrantedEffect != nil {
			tn := strings.ToLower(ms.TriggeredBy.GrantedEffect.Name())
			keys = append(keys, tn, strings.TrimPrefix(tn, "awakened "))
		}
		if u := getUniqueItemTriggerName(ms); u != "" {
			keys = append(keys, strings.ToLower(u))
		}
		rec, hasRate := oracle[name]
		hasRate = hasRate && rec.Output["TriggerRateCap"] != 0
		seen := map[string]bool{}
		for _, k := range keys {
			if k == "" || seen[k] {
				continue
			}
			seen[k] = true
			all[k]++
			if hasRate {
				withRate[k]++
				if triggerConfigFor(env, k, ms) != nil {
					served[k]++
				}
			}
		}
	}

	type row struct {
		k       string
		a, r, s int
	}
	rows := make([]row, 0, len(withRate))
	for k, v := range withRate {
		rows = append(rows, row{k, all[k], v, served[k]})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].r > rows[j].r })
	var b strings.Builder
	b.WriteString("\nkey                                        anyRung  w/rate  servedHere\n")
	for _, r := range rows {
		if r.r == 0 {
			continue
		}
		b.WriteString(strings.Repeat(" ", 0))
		b.WriteString(zzPadRight(r.k, 42) + zzItoa(r.a, 7) + zzItoa(r.r, 8) + zzItoa(r.s, 12) + "\n")
	}
	t.Log(b.String())
	if len(rows) == 0 {
		t.Fatal("REFUSING: no key resolved at any dispatch rung across the whole corpus")
	}
}

func zzPadRight(s string, n int) string {
	for len(s) < n {
		s += " "
	}
	return s
}

func zzItoa(v, w int) string {
	s := ""
	if v == 0 {
		s = "0"
	}
	for v > 0 {
		s = string(rune('0'+v%10)) + s
		v /= 10
	}
	for len(s) < w {
		s = " " + s
	}
	return s
}
