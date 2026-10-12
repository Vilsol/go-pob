package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data"
)

// TestEnemyEffMultInputCensus splits the enemy-side inputs to EffMult into their two classes over a BUILD LIST.
func TestEnemyEffMultInputCensus(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	listFile := os.Getenv("BUILDS")
	if listFile == "" {
		t.Skip("set BUILDS=<file of build names> (the EffMult-first-diverging set)")
	}
	requireGameData(t, "3.29")
	dir := diagDumpsDir(t)

	raw, err := os.ReadFile(listFile)
	if err != nil {
		t.Fatalf("REFUSING: cannot read BUILDS=%s: %v", listFile, err)
	}
	var names []string
	for _, ln := range strings.Split(string(raw), "\n") {
		if ln = strings.TrimSpace(ln); ln != "" {
			names = append(names, strings.TrimSuffix(ln, ".xml"))
		}
	}
	if len(names) == 0 {
		t.Fatalf("REFUSING: %s lists no builds — an empty population must say so, not report 0 "+
			"divergences as though it had looked", listFile)
	}

	takenNames := map[string]bool{
		"DamageTaken": true, "DamageTakenOverTime": true, "ElementalDamageTaken": true,
		"ProjectileDamageTaken": true, "ProjectileAttackDamageTaken": true,
		"TrapMineDamageTaken": true,
	}
	resistNames := map[string]bool{"ElementalResist": true}
	for _, dt := range []data.DamageType{
		data.DamageTypePhysical, data.DamageTypeLightning, data.DamageTypeCold,
		data.DamageTypeFire, data.DamageTypeChaos,
	} {
		takenNames[string(dt)+"DamageTaken"] = true
		takenNames[string(dt)+"DamageTakenOverTime"] = true
		resistNames[string(dt)+"Resist"] = true
		resistNames[string(dt)+"ResistMax"] = true
	}
	classOf := func(n string) string {
		switch {
		case takenNames[n]:
			return "taken"
		case resistNames[n]:
			return "resist"
		}
		return ""
	}

	corpus := corpusDirForLoadedData(t)
	type counter struct {
		onlyPoB, onlyOurs         int
		buildsOnlyPoB, buildsOurs map[string]bool
	}
	newCounters := func() map[string]*counter {
		return map[string]*counter{
			"taken":  {buildsOnlyPoB: map[string]bool{}, buildsOurs: map[string]bool{}},
			"resist": {buildsOnlyPoB: map[string]bool{}, buildsOurs: map[string]bool{}},
		}
	}
	per, perBlind := newCounters(), newCounters()
	srcOnlyPoB, srcOnlyOurs := map[string]int{}, map[string]int{}
	gateUngated, gateDeadCond, gateDeadUnverifiable, gateUnresolved := 0, 0, 0, 0
	deadBySrc, ungatedBySrc := map[string]int{}, map[string]int{}
	condVarSeen := map[string]bool{}
	type pendingDead struct {
		vars []string
		src  string
	}
	var pending []pendingDead
	conditionVars := func(fm string) ([]string, []string, bool) {
		var need, negated []string
		i := strings.LastIndex(fm, "|")
		if i < 0 {
			return nil, nil, false
		}
		legible := true
		for _, tag := range strings.Split(fm[i+1:], ",") {
			if tag == "" || tag == "-" {
				continue
			}
			parts := strings.Split(tag, "/")
			kind := strings.TrimPrefix(parts[0], "type=")
			if kind != "Condition" {
				continue
			}
			var v string
			neg := false
			for _, kv := range parts[1:] {
				switch {
				case strings.HasPrefix(kv, "var="):
					v = strings.TrimPrefix(kv, "var=")
				case kv == "neg=true":
					neg = true
				case strings.HasPrefix(kv, "varList="):
					legible = false
				}
			}
			if v == "" {
				legible = false
				continue
			}
			if neg {
				negated = append(negated, v)
			} else {
				need = append(need, v)
			}
		}
		return need, negated, legible
	}
	pairObs := map[string]map[string]int{}
	ourToPoB := map[string]map[string]bool{}
	type buildSets struct {
		name           string
		pobSet, ourSet map[string]int
	}
	var stored []buildSets

	pobSkillSourced, ourSkillSourced := 0, 0
	checked, unhealthy, unusable := 0, 0, 0

	for _, name := range names {
		dump, healthy := diagLoadDump(dir, name)
		if dump == nil {
			unusable++
			continue
		}
		if !healthy {
			unhealthy++
			continue
		}
		xml, err := os.ReadFile(filepath.Join(corpus, name+".xml"))
		if err != nil {
			unusable++
			continue
		}
		build, err := builds.ParseBuild(xml)
		if err != nil {
			unusable++
			continue
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, corpusCacheForTest(t))
		if env == nil || env.Player == nil {
			unusable++
			continue
		}
		pobMods, ourDB, _ := diagActorModDB(dump, env, "enemy")
		if ourDB == nil || pobMods == nil {
			unusable++
			continue
		}
		checked++

		key := func(src, typ string, val float64) string {
			return src + "|" + typ + "|" + strconv.FormatFloat(val, 'g', 10, 64)
		}
		pobSet, ourSet := map[string]int{}, map[string]int{}
		pobBlind, ourBlind := map[string]int{}, map[string]int{}
		pobFmt := map[string]string{}
		pobFmtSrc := map[string]string{}
		pobBlindSrcs, ourBlindSrcs := map[string][]string{}, map[string][]string{}
		enemyConds := map[string]bool{}
		for modName := range pobMods {
			if strings.HasPrefix(modName, "Condition:") {
				v := strings.TrimPrefix(modName, "Condition:")
				enemyConds[v] = true
				condVarSeen[v] = true
			}
		}
		for modName, mods := range pobMods {
			for _, m := range mods {
				s := normalizeSource(m.Source)
				if strings.HasPrefix(s, "Skill:") {
					pobSkillSourced++
				}
				if classOf(modName) == "" {
					continue
				}
				pk := modName + "||" + key(s, m.Type, pobModValue(m.Type, m.Value))
				pobSet[pk]++
				bk := modName + "||" + key("", m.Type, pobModValue(m.Type, m.Value))
				pobBlind[bk]++
				pobFmt[bk] = m.Fmt
				pobFmtSrc[bk] = s
				pobBlindSrcs[bk] = append(pobBlindSrcs[bk], s)
			}
		}
		for modName, mods := range ourDB.Mods {
			for _, m := range mods {
				s := normalizeSource(string(m.GetSource()))
				if strings.HasPrefix(s, "Skill:") {
					ourSkillSourced++
				}
				if classOf(modName) == "" {
					continue
				}
				ourSet[modName+"||"+key(s, string(m.Type()), ourModValue(m))]++
				obk := modName + "||" + key("", string(m.Type()), ourModValue(m))
				ourBlind[obk]++
				ourBlindSrcs[obk] = append(ourBlindSrcs[obk], s)
			}
		}

		for bk, pl := range pobBlindSrcs {
			ol := ourBlindSrcs[bk]
			if len(pl) != 1 || len(ol) != 1 || pobBlind[bk] != ourBlind[bk] || pl[0] == ol[0] {
				continue
			}
			if pairObs[pl[0]] == nil {
				pairObs[pl[0]] = map[string]int{}
			}
			pairObs[pl[0]][ol[0]]++
			if ourToPoB[ol[0]] == nil {
				ourToPoB[ol[0]] = map[string]bool{}
			}
			ourToPoB[ol[0]][pl[0]] = true
		}
		stored = append(stored, buildSets{name: name, pobSet: pobSet, ourSet: ourSet})

		tally := func(a, b map[string]int, cs map[string]*counter, srcs map[string]int, ours bool) {
			for k, n := range a {
				extra := n - b[k]
				if extra <= 0 {
					continue
				}
				modName := k[:strings.Index(k, "||")]
				c := cs[classOf(modName)]
				if ours {
					c.onlyOurs += extra
					c.buildsOurs[name] = true
				} else {
					c.onlyPoB += extra
					c.buildsOnlyPoB[name] = true
				}
				if srcs != nil {
					srcs[strings.SplitN(k[strings.Index(k, "||")+2:], "|", 2)[0]] += extra
				}
			}
		}
		for k, n := range pobBlind {
			if n-ourBlind[k] <= 0 {
				continue
			}
			need, neg, legible := conditionVars(pobFmt[k])
			dead := false
			for _, v := range need {
				if !enemyConds[v] {
					dead = true
				}
			}
			for _, v := range neg {
				if enemyConds[v] {
					dead = true
				}
			}
			src := pobFmtSrc[k]
			switch {
			case dead:
				pending = append(pending, pendingDead{vars: append(append([]string{}, need...), neg...), src: src})
			case !legible:
				gateUnresolved++
			case len(need) == 0 && len(neg) == 0:
				gateUngated++
				ungatedBySrc[src+"  "+k[:strings.Index(k, "||")]]++
			default:
				gateUnresolved++
			}
		}
		tally(pobSet, ourSet, per, srcOnlyPoB, false)
		tally(ourSet, pobSet, per, srcOnlyOurs, true)
		tally(pobBlind, ourBlind, perBlind, nil, false)
		tally(ourBlind, pobBlind, perBlind, nil, true)
	}

	for _, pd := range pending {
		verifiable := false
		for _, v := range pd.vars {
			if condVarSeen[v] {
				verifiable = true
			}
		}
		if verifiable {
			gateDeadCond++
			deadBySrc[pd.src]++
		} else {
			gateDeadUnverifiable++
			deadBySrc["(unverifiable) "+pd.src]++
		}
	}

	rename := map[string]string{}
	type ambig struct {
		pob  string
		ours []string
		obs  int
	}
	var ambiguous []ambig
	for pl, m := range pairObs {
		ours := make([]string, 0, len(m))
		total := 0
		for o, n := range m {
			ours = append(ours, o)
			total += n
		}
		sort.Strings(ours)
		if len(ours) != 1 || len(ourToPoB[ours[0]]) != 1 {
			ambiguous = append(ambiguous, ambig{pob: pl, ours: ours, obs: total})
			continue
		}
		rename[ours[0]] = pl
	}
	sort.Slice(ambiguous, func(i, j int) bool { return ambiguous[i].obs > ambiguous[j].obs })

	perDedup := newCounters()
	srcDedupPoB, srcDedupOurs := map[string]int{}, map[string]int{}
	relabel := func(k string) string {
		i := strings.Index(k, "||")
		rest := k[i+2:]
		j := strings.Index(rest, "|")
		src := rest[:j]
		if to, ok := rename[src]; ok {
			src = to
		}
		return k[:i+2] + src + rest[j:]
	}
	for _, bs := range stored {
		pd, od := map[string]int{}, map[string]int{}
		for k, n := range bs.pobSet {
			pd[relabel(k)] += n
		}
		for k, n := range bs.ourSet {
			od[relabel(k)] += n
		}
		for k, n := range pd {
			if extra := n - od[k]; extra > 0 {
				c := perDedup[classOf(k[:strings.Index(k, "||")])]
				c.onlyPoB += extra
				c.buildsOnlyPoB[bs.name] = true
				srcDedupPoB[strings.SplitN(k[strings.Index(k, "||")+2:], "|", 2)[0]] += extra
			}
		}
		for k, n := range od {
			if extra := n - pd[k]; extra > 0 {
				c := perDedup[classOf(k[:strings.Index(k, "||")])]
				c.onlyOurs += extra
				c.buildsOurs[bs.name] = true
				srcDedupOurs[strings.SplitN(k[strings.Index(k, "||")+2:], "|", 2)[0]] += extra
			}
		}
	}

	if checked == 0 {
		t.Fatalf("REFUSING: 0 of %d listed builds were comparable (%d unhealthy dumps, %d "+
			"unusable) — empty must mean 'I looked and found nothing', never 'I did not look'",
			len(names), unhealthy, unusable)
	}

	fmt.Printf("\n=== ENEMY EffMult INPUT CENSUS over %s ===\n", filepath.Base(listFile))
	fmt.Printf("listed %d · compared %d · unhealthy dump %d · unusable %d\n",
		len(names), checked, unhealthy, unusable)

	top := func(label string, m map[string]int) {
		type kv struct {
			s string
			n int
		}
		rows := make([]kv, 0, len(m))
		for k, v := range m {
			rows = append(rows, kv{k, v})
		}
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].n != rows[j].n {
				return rows[i].n > rows[j].n
			}
			return rows[i].s < rows[j].s
		})
		fmt.Printf("  %s\n", label)
		if len(rows) == 0 {
			fmt.Printf("    (none)\n")
		}
		for i, r := range rows {
			if i >= 12 {
				fmt.Printf("    (%d further sources)\n", len(rows)-i)
				break
			}
			fmt.Printf("    %-52s %d\n", r.s, r.n)
		}
	}

	fmt.Printf("\n--- SOURCE-LABEL DEDUPE (learned, not asserted) ---\n")
	fmt.Printf("  merged %d label pair(s) proven 1:1 across the population:\n", len(rename))
	{
		type kv struct{ o, p string }
		rows := make([]kv, 0, len(rename))
		for o, pl := range rename {
			rows = append(rows, kv{o, pl})
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].o < rows[j].o })
		for i, r := range rows {
			if i >= 12 {
				fmt.Printf("    (%d more)\n", len(rows)-i)
				break
			}
			fmt.Printf("    ours %-40s == pob %s\n", r.o, r.p)
		}
		if len(rows) == 0 {
			fmt.Printf("    (none)\n")
		}
	}
	fmt.Printf("  ⚠️ NOT MERGED — mapping is not 1:1, so these stay separate: %d\n", len(ambiguous))
	for i, a := range ambiguous {
		if i >= 8 {
			fmt.Printf("    (%d more)\n", len(ambiguous)-i)
			break
		}
		fmt.Printf("    pob %-34s paired with %d of ours %v (%d obs)\n", a.pob, len(a.ours), a.ours, a.obs)
	}

	fmt.Printf("\n--- ONLY-POB (PoB's enemy carries it, we do not) — SOUND ---\n")
	for _, c := range []string{"taken", "resist"} {
		fmt.Printf("  %-7s DEDUPED %4d over %d builds · source-blind %4d over %d · raw by-source %4d over %d\n",
			c, perDedup[c].onlyPoB, len(perDedup[c].buildsOnlyPoB),
			perBlind[c].onlyPoB, len(perBlind[c].buildsOnlyPoB),
			per[c].onlyPoB, len(per[c].buildsOnlyPoB))
	}
	top("by source, DEDUPED (the ranking):", srcDedupPoB)

	fmt.Printf("\n--- ONLY-POB, one rung further: can it fire at all? (SOURCE-BLIND rows) ---\n")
	fmt.Printf("  ungated (no Condition tag)                     %5d\n", gateUngated)
	fmt.Printf("  DEAD (gated on a condition PoB never creates)   %5d\n", gateDeadCond)
	fmt.Printf("  DEAD? (same, but the var is never seen as a mod anywhere here — could be\n"+
		"        a conditions-table entry the dump does not carry)     %5d\n", gateDeadUnverifiable)
	fmt.Printf("  unresolved (condition exists; truth needs an evaluator) %5d\n", gateUnresolved)
	top("ungated by source + mod name (the live candidates):", ungatedBySrc)
	top("dead-gated by source:", deadBySrc)
	fmt.Printf("  ⚠️ `unresolved` is NOT `active` — PoB's conditions are themselves conditional\n" +
		"     FLAG mods, so the dump can prove a condition is never CREATED and cannot prove one\n" +
		"     is TRUE. Only the DEAD row is a sound negative.\n")

	fmt.Printf("\n--- dump-route discriminator ---\n")
	fmt.Printf("  Skill:-sourced enemy mods — PoB %d · ours %d\n", pobSkillSourced, ourSkillSourced)

	if pobSkillSourced == 0 && ourSkillSourced > 0 {
		fmt.Printf("\n--- ONLY-OURS — WITHHELD ---\n")
		fmt.Printf("  REFUSING to report the ONLY-OURS column: across %d compared builds PoB's\n"+
			"  dumped enemy carries ZERO Skill:-sourced mods while ours carries %d. PoB adds enemy\n"+
			"  mods by two routes and only one of them attributes to a skill, so \"PoB does not\n"+
			"  apply it\" and \"the dump was taken before CalcPerform.lua:3298 merged the debuffs\"\n"+
			"  are the same observation here. A count would launder that into a corpus-wide number.\n"+
			"  WHAT RESOLVES IT: one build that certainly curses the enemy. If PoB's enemy carries\n"+
			"  the curse's mods, the dump sees the debuff route and this column becomes readable;\n"+
			"  if it does not, the dump is blind to the route and the column can never be read from\n"+
			"  these dumps at all.\n", checked, ourSkillSourced)
		return
	}

	fmt.Printf("\n--- ONLY-OURS (we carry it, PoB's enemy does not) — readable, see discriminator ---\n")
	for _, c := range []string{"taken", "resist"} {
		fmt.Printf("  %-7s DEDUPED %4d over %d builds · source-blind %4d over %d · raw by-source %4d over %d\n",
			c, perDedup[c].onlyOurs, len(perDedup[c].buildsOurs),
			perBlind[c].onlyOurs, len(perBlind[c].buildsOurs),
			per[c].onlyOurs, len(per[c].buildsOurs))
	}
	top("by source, DEDUPED (the ranking):", srcDedupOurs)
	fmt.Printf("  ⚠️ `Other:<none>` IS OUR ATTRIBUTION PLACEHOLDER, NOT A SOURCE. A mod whose origin\n" +
		"     this tree never recorded lands there, so it cannot be matched against PoB's labelled\n" +
		"     one and shows up as ours-only. It is the head of this ranking and it is an instrument\n" +
		"     limit — the 24 pairs the dedupe REFUSED are all of that shape. Read it as a ceiling on\n" +
		"     what the ours-side ranking can resolve, never as mods we invent.\n")
}
