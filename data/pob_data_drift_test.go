package data

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data/raw"
)

// TestPoBDataTableDrift is the only gate that checks the CONTENT of a hand-transcribed table.
func TestPoBDataTableDrift(t *testing.T) {
	dump := loadPoBTableDump(t)

	var findings []string
	report := func(table string, detail string) {
		findings = append(findings, fmt.Sprintf("%s: %s", table, detail))
	}

	graded := map[string]bool{}
	grade := func(name string) (pobTable, bool) {
		tbl, ok := dump.Tables[name]
		if ok {
			graded[name] = true
		}
		return tbl, ok
	}

	for name, ours := range map[string][]float64{
		"monsterAllyLifeTable":            MonsterAllyLifeTable,
		"monsterLifeTable":                MonsterLifeTable,
		"monsterEvasionTable":             MonsterEvasionTable,
		"monsterAccuracyTable":            MonsterAccuracyTable,
		"monsterDamageTable":              MonsterDamageTable,
		"monsterArmourTable":              MonsterArmourTable,
		"monsterPhysConversionMultiTable": MonsterPhysConversionMultiTable,
	} {
		tbl, ok := grade(name)
		if !ok {
			report(name, "ABSENT FROM THE DUMP — the dumper declares it, so this should be unreachable")
			continue
		}
		if len(ours) == 0 || ours[0] != 0 {
			report(name, fmt.Sprintf("our slice does not start with the 0 sentinel this comparison assumes "+
				"(first element %v) — the offset handling below is invalid, fix it rather than the table", first(ours)))
			continue
		}
		pobValues := tbl.numbers(t, name)
		got := ours[1:]
		if len(got) != len(pobValues) {
			report(name, fmt.Sprintf("length %d (after dropping the sentinel) vs PoB %d", len(got), len(pobValues)))
			continue
		}
		var diffs []string
		for i := range got {
			if math.Abs(got[i]-pobValues[i]) > 1e-6 && len(diffs) < 5 {
				diffs = append(diffs, fmt.Sprintf("[%d] ours=%g pob=%g", i+1, got[i], pobValues[i]))
			}
		}
		if n := countDiffs(got, pobValues); n > 0 {
			report(name, fmt.Sprintf("%d of %d entries differ; first: %v", n, len(got), diffs))
		}
	}

	if tbl, ok := grade("nonDamagingAilment"); ok {
		for ailment, ours := range NonDamagingAilments {
			pobRow, ok := tbl.Rows[string(ailment)]
			if !ok {
				report("nonDamagingAilment", fmt.Sprintf("%s: we have a row PoB does not", ailment))
				continue
			}
			for _, f := range []struct {
				name string
				ours float64
				set  bool
			}{
				{"min", ours.Min, true},
				{"max", ours.Max, true},
				{"precision", ours.Precision, true},
				{"default", deref(ours.Default), ours.Default != nil},
				{"duration", deref(ours.Duration), ours.Duration != nil},
			} {
				pobVal, pobSet := pobRow[f.name]
				pobNum, isNum := pobVal.(float64)
				switch {
				case f.set && (!pobSet || !isNum):
					report("nonDamagingAilment", fmt.Sprintf("%s.%s: ours=%g, PoB has none (nil)", ailment, f.name, f.ours))
				case !f.set && pobSet && isNum:
					report("nonDamagingAilment", fmt.Sprintf("%s.%s: ours=nil, PoB=%g", ailment, f.name, pobNum))
				case f.set && pobSet && isNum && math.Abs(f.ours-pobNum) > 1e-9:
					report("nonDamagingAilment", fmt.Sprintf("%s.%s: ours=%g PoB=%g", ailment, f.name, f.ours, pobNum))
				}
			}
		}
		for name := range tbl.Rows {
			if _, ok := NonDamagingAilments[Ailment(name)]; !ok {
				report("nonDamagingAilment", fmt.Sprintf("%s: PoB has a row we do not", name))
			}
		}
	}

	if tbl, ok := grade("keystones"); ok {
		extra, absent := diffSets(toSet(Keystones), toSet(tbl.strings(t, "keystones")))
		if len(extra) > 0 || len(absent) > 0 {
			report("keystones", fmt.Sprintf("%d ours-only %v, %d PoB-only %v",
				len(extra), capped(extra), len(absent), capped(absent)))
		}
	}

	if tbl, ok := grade("modScalability"); ok {
		ours := make(map[string]string, len(raw.ModScalability))
		for line, entries := range raw.ModScalability {
			parts := make([]string, 0, len(entries))
			for _, e := range entries {
				one := "F"
				if e.IsScalable {
					one = "T"
				}
				if e.Formats != nil {
					one += ":" + strings.Join(e.Formats, ",")
				}
				parts = append(parts, one)
			}
			ours[line] = strings.Join(parts, ";")
		}
		theirs := tbl.keyvals(t, "modScalability")
		var oursOnly, pobOnly, differ []string
		for k, v := range ours {
			if w, ok := theirs[k]; !ok {
				oursOnly = append(oursOnly, k)
			} else if v != w {
				differ = append(differ, fmt.Sprintf("%q ours=%q pob=%q", k, v, w))
			}
		}
		for k := range theirs {
			if _, ok := ours[k]; !ok {
				pobOnly = append(pobOnly, k)
			}
		}
		sort.Strings(oursOnly)
		sort.Strings(pobOnly)
		sort.Strings(differ)
		if len(oursOnly) > 0 || len(pobOnly) > 0 || len(differ) > 0 {
			report("modScalability", fmt.Sprintf(
				"%d ours-only %v, %d PoB-only %v, %d rows differ %v",
				len(oursOnly), capped(oursOnly), len(pobOnly), capped(pobOnly),
				len(differ), capped(differ)))
		}
	}

	if tbl, ok := grade("itemModStatOrder"); ok {
		ranked := func(rows map[string][2]float64) map[string]string {
			all := make([]float64, 0, 2*len(rows))
			for _, r := range rows {
				all = append(all, r[0], r[1])
			}
			sort.Float64s(all)
			rank := map[float64]int{}
			for _, v := range all {
				if _, ok := rank[v]; !ok && v != 0 {
					rank[v] = len(rank) + 1
				}
			}
			half := func(v float64) string {
				if v == 0 {
					return ""
				}
				return strconv.Itoa(rank[v])
			}
			out := make(map[string]string, len(rows))
			for k, r := range rows {
				out[k] = half(r[0]) + "|" + half(r[1])
			}
			return out
		}
		oursRows := make(map[string][2]float64, len(raw.ItemModStatOrder))
		for k, e := range raw.ItemModStatOrder {
			oursRows[k] = [2]float64{e.Exact, e.Normalised}
		}
		theirsRows := map[string][2]float64{}
		for k, v := range tbl.keyvals(t, "itemModStatOrder") {
			exact, normalised, _ := strings.Cut(v, "|")
			var r [2]float64
			for i, s := range []string{exact, normalised} {
				if s == "" {
					continue
				}
				f, err := strconv.ParseFloat(s, 64)
				if err != nil {
					t.Fatalf("itemModStatOrder %q: %v", k, err)
				}
				r[i] = f
			}
			theirsRows[k] = r
		}
		ours, theirs := ranked(oursRows), ranked(theirsRows)
		var oursOnly, pobOnly, differ []string
		for k, v := range ours {
			if w, ok := theirs[k]; !ok {
				oursOnly = append(oursOnly, k)
			} else if v != w {
				differ = append(differ, fmt.Sprintf("%q ours=%q pob=%q", k, v, w))
			}
		}
		for k := range theirs {
			if _, ok := ours[k]; !ok {
				pobOnly = append(pobOnly, k)
			}
		}
		sort.Strings(oursOnly)
		sort.Strings(pobOnly)
		sort.Strings(differ)
		if len(oursOnly) > 0 || len(pobOnly) > 0 || len(differ) > 0 {
			report("itemModStatOrder", fmt.Sprintf(
				"%d ours-only %v, %d PoB-only %v, %d rows differ %v",
				len(oursOnly), capped(oursOnly), len(pobOnly), capped(pobOnly),
				len(differ), capped(differ)))
		}
	} else {
		report("itemModStatOrder", "ABSENT FROM THE DUMP — the dumper declares it, so the "+
			"reference predates it; regenerate with `task pob-data-tables`")
	}

	if tbl, ok := grade("misc"); ok {
		compareMisc(t, tbl, report)
	}

	if tbl, ok := grade("mapLevelLifeMult"); ok {
		ours := map[string]float64{}
		for level, v := range MapLevelLifeMult {
			ours[strconv.Itoa(level)] = v
		}
		diffLevelScalars(t, "mapLevelLifeMult", ours, tbl, report)
	} else {
		report("mapLevelLifeMult", "ABSENT FROM THE DUMP — the dumper declares it, so the "+
			"reference predates it; regenerate with `task pob-data-tables`")
	}

	if tbl, ok := grade("weaponTypeInfo"); ok {
		ours := map[string]map[string]any{}
		for class, info := range WeaponTypes {
			row := map[string]any{
				"oneHand": info.OneHand,
				"melee":   info.Melee,
				"flag":    info.Flag,
			}
			if info.Label != "" {
				row["label"] = info.Label
			}
			ours[string(class)] = row
		}
		diffRows("weaponTypeInfo", ours, tbl.Rows, report)
	}

	if tbl, ok := grade("clusterJewels.jewels"); ok {
		ours := map[string]map[string]any{}
		for name, jewel := range ClusterJewels.Jewels {
			row := map[string]any{
				"size":          jewel.Size,
				"sizeIndex":     float64(jewel.SizeIndex),
				"minNodes":      float64(jewel.MinNodes),
				"maxNodes":      float64(jewel.MaxNodes),
				"totalIndicies": float64(jewel.TotalIndicies),
			}
			for field, list := range map[string][]int{
				"smallIndicies":   jewel.SmallIndicies,
				"notableIndicies": jewel.NotableIndicies,
				"socketIndicies":  jewel.SocketIndicies,
			} {
				for i, v := range list {
					row[fmt.Sprintf("%s[%d]", field, i+1)] = float64(v)
				}
			}
			ours[name] = row
		}
		diffRows("clusterJewels.jewels", ours, tbl.Rows, report)
	}

	if tbl, ok := grade("clusterJewels.skills"); ok {
		ours := map[string]map[string]any{}
		for jewelName, jewel := range ClusterJewels.Jewels {
			for skillKey, skill := range jewel.Skills {
				row := map[string]any{"name": skill.Name, "icon": skill.Icon, "tag": skill.Tag}
				if skill.MasteryIcon != "" {
					row["masteryIcon"] = skill.MasteryIcon
				}
				for i, s := range skill.Stats {
					row[fmt.Sprintf("stats[%d]", i+1)] = s
				}
				for i, s := range skill.Enchant {
					row[fmt.Sprintf("enchant[%d]", i+1)] = s
				}
				ours[jewelName+"|"+skillKey] = row
			}
		}
		diffRows("clusterJewels.skills", ours, tbl.Rows, report)
	}

	if tbl, ok := grade("clusterJewels.keystones"); ok {
		extra, absent := diffSets(toSet(ClusterJewels.Keystones), toSet(tbl.strings(t, "clusterJewels.keystones")))
		if len(extra) > 0 || len(absent) > 0 {
			report("clusterJewels.keystones", fmt.Sprintf("%d ours-only %v, %d PoB-only %v",
				len(extra), capped(extra), len(absent), capped(absent)))
		}
	}

	if tbl, ok := grade("clusterJewels.notableSortOrder"); ok {
		ours := map[string]map[string]any{}
		for name, order := range ClusterJewels.NotableSortOrder {
			ours[name] = map[string]any{"order": float64(order)}
		}
		theirs := map[string]map[string]any{}
		for name, v := range tbl.scalars(t, "clusterJewels.notableSortOrder") {
			theirs[name] = map[string]any{"order": v}
		}
		diffRows("clusterJewels.notableSortOrder", ours, theirs, report)
	}

	for _, spec := range []struct {
		name string
		ours map[string]struct{}
	}{
		{"spectres", keysOfAny(raw.Spectres)},
		{"minions", keysOfAny(raw.Minions)},
		{"skillStatMap", keysOfAny(raw.SkillStatMap)},
		{"pantheons", keysOfAny(Pantheons)},
	} {
		tbl, ok := grade(spec.name)
		if !ok {
			continue
		}
		extra, absent := diffSets(spec.ours, toSet(tbl.Keys))
		if len(extra) > 0 || len(absent) > 0 {
			report(spec.name, fmt.Sprintf("%d ours-only, %d PoB-only (of %d PoB rows); PoB-only sample %v",
				len(extra), len(absent), len(tbl.Keys), capped(absent)))
		}
	}

	sort.Strings(findings)

	fmt.Printf("\n=== PoB DATA TABLE DRIFT (PoB %s, tree %s) ===\n", dump.PoBVersion, dump.TreeVersion)
	fmt.Printf("%d tables dumped, %d graded, %d ungraded\n",
		len(dump.Tables), len(graded), len(dump.Tables)-len(graded))
	if len(findings) == 0 {
		fmt.Printf("no drift across the %d GRADED tables\n", len(graded))
	}
	for _, f := range findings {
		fmt.Printf("  %s\n", f)
	}

	for name := range dump.Tables {
		reason, declared := uncomparedTables[name]
		switch {
		case graded[name] && declared:
			t.Errorf("STALE UNCOMPARED DECLARATION — %q IS compared now, but uncomparedTables still\n"+
				"    excuses it (%q). Delete the entry: an excuse for a table that is already graded\n"+
				"    reads as a known blind spot and hides the next real one.", name, reason)
		case graded[name]:
		case declared:
			t.Logf("NOT COMPARED: %s\n    reason: %s", name, reason)
		default:
			t.Errorf("DUMPED BUT UNGRADED — %q is in the dump and nothing in this test reads it.\n"+
				"    A dumped-but-ungraded table is worse than an absent one: it inflates the banner's\n"+
				"    table count, so the gate reports confidence over data it never looked at. Either\n"+
				"    compare it, or add it to uncomparedTables with a reason.", name)
		}
	}
	for name := range uncomparedTables {
		if _, ok := dump.Tables[name]; !ok {
			t.Errorf("STALE UNCOMPARED DECLARATION — %q is declared uncompared but is not in the dump.\n"+
				"    The dumper stopped emitting it (or it was renamed); an excuse pointing at nothing\n"+
				"    is how this list rots into fiction.", name)
		}
	}

	fired := map[string]bool{}
	for _, f := range findings {
		if key, reason, ok := declarationFor(f); ok {
			fired[key] = true
			t.Logf("DECLARED: %s\n    reason: %s", f, reason)
			continue
		}
		t.Errorf("UNDECLARED DATA DRIFT — %s\n"+
			"    A hand-transcribed table no longer matches the pinned PoB. Either transcribe the\n"+
			"    current value, or add an entry to declaredDataDrift saying why the divergence is\n"+
			"    correct. Do NOT silence it by regenerating the dump — the dump is the reference.", f)
	}

	var closed []string
	for table := range declaredDataDrift {
		if !fired[table] {
			closed = append(closed, table)
		}
	}
	sort.Strings(closed)
	for _, table := range closed {
		t.Logf("A DATA DRIFT CLOSED — %q no longer diverges from the pinned PoB. "+
			"Delete its entry from declaredDataDrift in this commit; a stale allowlist entry "+
			"will mask the NEXT divergence in that same table.", table)
	}
}

var uncomparedTables = map[string]string{}

var declaredDataDrift = map[string]string{
	"modScalability: 3 ours-only [#% increased Toughness of Monsters in Area #% reduced Toughness of Monsters in Area Monsters in Area have increased Toughness and Reward], 0 PoB-only [], 0 rows differ []": "game drift: go-pob-data v0.5.0 derives the table from 3.29.3.3's stat_descriptions, which describes these map stats; the pin's export predates them",
}

func miscOurs() map[string]float64 {
	return map[string]float64{
		"ServerTickTime":                  ServerTickTime,
		"ServerTickRate":                  ServerTickRate,
		"TemporalChainsEffectCap":         TemporalChainsEffectCap,
		"DamageReductionCap":              DamageReductionCap,
		"ResistFloor":                     ResistFloor,
		"BlockChanceCap":                  BlockChanceCap,
		"EnemyPhysicalDamageReductionCap": EnemyPhysicalDamageReductionCap,
		"FortifyBaseDuration":             FortifyBaseDuration,
		"ManaRegenBase":                   ManaRegenBase,
		"MatchingSocketQualityBonus":      MatchingSocketQualityBonus,
		"PlayerMovementSpeed":             PlayerMovementSpeed,
		"maxHitSmoothingPasses":           MaxHitSmoothingPasses,
		"maxStatIncrease":                 MaxStatIncrease,
		"PvpElemental1":                   PvpElemental1,
		"PvpElemental2":                   PvpElemental2,
		"PvpNonElemental1":                PvpNonElemental1,
		"PvpNonElemental2":                PvpNonElemental2,
		"MaxResistCap":                    MaxResistCap,
		"EvadeChanceCap":                  EvadeChanceCap,
		"DodgeChanceCap":                  DodgeChanceCap,
		"SuppressionChanceCap":            SuppressionChanceCap,
		"SuppressionEffect":               SuppressionEffect,
		"AvoidChanceCap":                  AvoidChanceCap,
		"EnergyShieldRechargeBase":        EnergyShieldRechargeBase,
		"EnergyShieldRechargeDelay":       EnergyShieldRechargeDelay,
		"WardRechargeDelay":               WardRechargeDelay,
		"Transfiguration":                 Transfiguration,
		"EnemyMaxResist":                  EnemyMaxResist,
		"LeechRateBase":                   LeechRateBase,
		"BleedPercentBase":                BleedPercentBase,
		"BleedDurationBase":               BleedDurationBase,
		"PoisonPercentBase":               PoisonPercentBase,
		"PoisonDurationBase":              PoisonDurationBase,
		"IgnitePercentBase":               IgnitePercentBase,
		"IgniteDurationBase":              IgniteDurationBase,
		"IgniteMinDuration":               IgniteMinDuration,
		"ImpaleStoredDamageBase":          ImpaleStoredDamageBase,
		"BuffExpirationSlowCap":           BuffExpirationSlowCap,
		"TrapTriggerRadiusBase":           TrapTriggerRadiusBase,
		"MineDetonationRadiusBase":        MineDetonationRadiusBase,
		"MineAuraRadiusBase":              MineAuraRadiusBase,
		"MaxEnemyLevel":                   MaxEnemyLevel,
		"LowPoolThreshold":                LowPoolThreshold,
		"AccuracyPerDexBase":              AccuracyPerDexBase,
		"BrandAttachmentRangeBase":        BrandAttachmentRangeBase,
		"ProjectileDistanceCap":           ProjectileDistanceCap,
		"stdBossDPSMult":                  StdBossDPSMult,
		"pinnacleBossDPSMult":             PinnacleBossDPSMult,
		"pinnacleBossPen":                 PinnacleBossPen,
		"uberBossDPSMult":                 UberBossDPSMult,
		"uberBossPen":                     UberBossPen,
		"ehpCalcSpeedUp":                  EhpCalcSpeedUp,
		"StunBaseMult":                    StunBaseMult,
		"StunBaseDuration":                StunBaseDuration,
		"StunNotMeleeDamageMult":          StunNotMeleeDamageMult,
		"MinStunChanceNeeded":             MinStunChanceNeeded,
		"ehpCalcMaxDamage":                EhpCalcMaxDamage,
		"ehpCalcMaxIterationsToCalc":      EhpCalcMaxIterationsToCalc,
		"DotDpsCap":                       DotDpsCap,
	}
}

var miscUnported = map[string]string{
	"experiencePenaltyMultiplier":       "unported: character XP curve; go-pob computes no experience at all.",
	"maxExperiencePenaltyFreeAreaLevel": "unported: character XP curve; go-pob computes no experience at all.",
}

var miscRemovedUpstream = map[string]string{
	"IgniteMinDuration": "REMOVED in PoB 3.29 (0 occurrences in src/). CalcOffence.lua:4904 " +
		"computes IgniteDuration with no minimum clamp, and calc_offence.go no longer applies " +
		"one either -- the clamp was dropped on 2026-08-15 with the weighted-average ignite " +
		"chain, which is the re-port this entry was waiting for. Removing it measured EXACTLY " +
		"ZERO on the gate: the clamp never bound on a graded build, so it was a latent " +
		"divergence rather than an active one. The constant is kept frozen because nothing " +
		"reads it now and a bare deletion would look like drift rather than a decision.",
}

const ulpClassLimit = 4

func compareMisc(t *testing.T, tbl pobTable, report func(string, string)) {
	t.Helper()

	if len(tbl.Exact) == 0 {
		t.Fatalf("REFUSING: the misc table carries no %q field. Regenerate the dump with "+
			"`task pob-data-tables` — Values is written at 14 significant digits and cannot "+
			"answer an exact comparison.", "exact")
	}

	ours := miscOurs()
	for key, ourVal := range ours {
		if name, cut := strings.CutPrefix(key, "__oursOnly:"); cut {
			if _, ok := tbl.Exact[name]; !ok {
				report("misc", fmt.Sprintf("%s: ours=%g, PoB has no such key", name, ourVal))
			}
			continue
		}
		raw, ok := tbl.Exact[key]
		if !ok {
			if reason, declared := miscRemovedUpstream[key]; declared {
				t.Logf("misc.%s REMOVED UPSTREAM (ours=%g, frozen): %s", key, ourVal, reason)
				continue
			}
			report("misc", fmt.Sprintf("%s: ours=%g, PoB has no such key", key, ourVal))
			continue
		}
		pobVal, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			t.Fatalf("REFUSING: misc.%s is not a parseable float (%q): %v", key, raw, err)
		}
		switch ulps := ulpsApart(ourVal, pobVal); {
		case ulps == 0:
		case ulps <= ulpClassLimit:
			report("misc.ULP", fmt.Sprintf("%s: ours=%.17g PoB=%.17g (%d ULP) — evaluation order, "+
				"not a transcription error; see ulpClassLimit", key, ourVal, pobVal, ulps))
		default:
			report("misc", fmt.Sprintf("%s: ours=%g PoB=%g", key, ourVal, pobVal))
		}
	}
	for key := range tbl.Exact {
		if _, ok := ours[key]; ok {
			continue
		}
		if reason, ok := miscUnported[key]; ok {
			t.Logf("misc.%s NOT PORTED: %s", key, reason)
			continue
		}
		report("misc", fmt.Sprintf("%s: PoB has a key we neither port nor declare (PoB=%s)", key, tbl.Exact[key]))
	}
	for key := range miscUnported {
		if _, ok := tbl.Exact[key]; !ok {
			report("misc", fmt.Sprintf("%s: declared unported but PoB no longer has the key — stale declaration", key))
		}
	}
}

func diffLevelScalars(t *testing.T, name string, ours map[string]float64, tbl pobTable, report func(string, string)) {
	t.Helper()

	if len(tbl.Exact) == 0 {
		t.Fatalf("REFUSING: the %s table carries no %q field. Regenerate the dump with "+
			"`task pob-data-tables` — Values is written at 14 significant digits and cannot "+
			"answer an exact comparison.", name, "exact")
	}
	for key, raw := range tbl.Exact {
		pobVal, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			t.Fatalf("REFUSING: %s.%s is not a parseable float (%q): %v", name, key, raw, err)
		}
		ourVal, ok := ours[key]
		if !ok {
			report(name, fmt.Sprintf("%s: PoB=%g and we have no such key — every reader is "+
				"`table[level] or 1`, so this reads as no scaling rather than as an error",
				key, pobVal))
			continue
		}
		switch ulps := ulpsApart(ourVal, pobVal); {
		case ulps == 0:
		case ulps <= ulpClassLimit:
			report(name+".ULP", fmt.Sprintf("%s: ours=%.17g PoB=%.17g (%d ULP)", key, ourVal, pobVal, ulps))
		default:
			report(name, fmt.Sprintf("%s: ours=%g PoB=%g", key, ourVal, pobVal))
		}
	}
	for key, ourVal := range ours {
		if _, ok := tbl.Exact[key]; !ok {
			report(name, fmt.Sprintf("%s: ours=%g, PoB has no such key", key, ourVal))
		}
	}
}

func ulpsApart(a, b float64) uint64 {
	if a == b {
		return 0
	}
	if math.IsNaN(a) || math.IsNaN(b) || math.IsInf(a, 0) || math.IsInf(b, 0) || (a < 0) != (b < 0) {
		return math.MaxUint64
	}
	ai, bi := math.Float64bits(math.Abs(a)), math.Float64bits(math.Abs(b))
	if ai > bi {
		return ai - bi
	}
	return bi - ai
}

func diffRows(table string, ours, theirs map[string]map[string]any, report func(string, string)) {
	oursOnly, theirsOnly := diffSets(keysOfAny(ours), keysOfAny(theirs))
	if len(oursOnly) > 0 || len(theirsOnly) > 0 {
		report(table, fmt.Sprintf("%d ours-only rows %v, %d PoB-only rows %v",
			len(oursOnly), capped(oursOnly), len(theirsOnly), capped(theirsOnly)))
	}

	var diffs []string
	names := make([]string, 0, len(ours))
	for name := range ours {
		if _, ok := theirs[name]; ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	total := 0
	for _, name := range names {
		ourRow, pobRow := ours[name], theirs[name]
		fields := map[string]struct{}{}
		for f := range ourRow {
			fields[f] = struct{}{}
		}
		for f := range pobRow {
			if f != "_dropped" {
				fields[f] = struct{}{}
			}
		}
		keys := make([]string, 0, len(fields))
		for f := range fields {
			keys = append(keys, f)
		}
		sort.Strings(keys)
		for _, f := range keys {
			ourVal, ourSet := ourRow[f]
			pobVal, pobSet := pobRow[f]
			if ourSet && pobSet && sameScalar(ourVal, pobVal) {
				continue
			}
			total++
			if len(diffs) < 5 {
				diffs = append(diffs, fmt.Sprintf("%s.%s ours=%v pob=%v", name, f, orAbsent(ourVal, ourSet), orAbsent(pobVal, pobSet)))
			}
		}
	}
	if total > 0 {
		report(table, fmt.Sprintf("%d field differences across %d shared rows; first: %v", total, len(names), diffs))
	}
}

func sameScalar(a, b any) bool {
	af, aok := a.(float64)
	bf, bok := b.(float64)
	if aok && bok {
		return math.Abs(af-bf) <= 1e-9
	}
	return a == b
}

func orAbsent(v any, set bool) any {
	if !set {
		return "<absent>"
	}
	return v
}

type pobTableDump struct {
	PoBVersion  string              `json:"pobVersion"`
	TreeVersion string              `json:"treeVersion"`
	PoBPin      pobPinRecord        `json:"pobPin"`
	Tables      map[string]pobTable `json:"tables"`
}

type pobTable struct {
	Kind   string                    `json:"kind"`
	Values json.RawMessage           `json:"values"`
	Keys   []string                  `json:"keys"`
	Rows   map[string]map[string]any `json:"rows"`

	Exact map[string]string `json:"exact"`
}

func (p pobTable) numbers(t *testing.T, name string) []float64 {
	t.Helper()
	var out []float64
	if err := json.Unmarshal(p.Values, &out); err != nil {
		t.Fatalf("REFUSING: table %q has kind %q but its values are not numeric: %v", name, p.Kind, err)
	}
	return out
}

func (p pobTable) keyvals(t *testing.T, name string) map[string]string {
	t.Helper()
	var out map[string]string
	if err := json.Unmarshal(p.Values, &out); err != nil {
		t.Fatalf("REFUSING: table %q has kind %q but its values are not a string map: %v", name, p.Kind, err)
	}
	if len(out) == 0 {
		t.Fatalf("REFUSING: table %q decoded to zero rows, so every comparison against it would "+
			"report our whole table as ours-only rather than as agreeing", name)
	}
	return out
}

func (p pobTable) scalars(t *testing.T, name string) map[string]float64 {
	t.Helper()
	var out map[string]float64
	if err := json.Unmarshal(p.Values, &out); err != nil {
		t.Fatalf("REFUSING: table %q has kind %q but its values are not a scalar map: %v", name, p.Kind, err)
	}
	return out
}

func (p pobTable) strings(t *testing.T, name string) []string {
	t.Helper()
	var out []string
	if err := json.Unmarshal(p.Values, &out); err != nil {
		t.Fatalf("REFUSING: table %q has kind %q but its values are not strings: %v", name, p.Kind, err)
	}
	return out
}

const referenceDir = "../testdata/pob-data-tables"

const recordedPinPath = "../scripts/pob_pin.json"

type pobPinRecord struct {
	ContentSHA256 string `json:"content_sha256"`
	FileCount     int    `json:"file_count"`

	Describe         string `json:"describe"`
	DescribeAdvisory string `json:"describe_advisory"`
}

func (p pobPinRecord) label() string {
	for _, s := range []string{p.Describe, p.DescribeAdvisory} {
		if s != "" {
			return s
		}
	}
	return "unknown"
}

func loadPoBTableDump(t *testing.T) *pobTableDump {
	t.Helper()

	want := loadRecordedPin(t)
	dump, path, err := resolvePoBTableReference(referenceDir, want)
	if err != nil {
		t.Fatalf("%v", err)
	}

	if len(dump.Tables) == 0 {
		t.Fatalf("REFUSING: %s declares 0 tables. An empty reference passes every comparison "+
			"trivially, which reads exactly like no drift.", path)
	}
	t.Logf("reference %s — PoB %s, tree %s, pin %s (content_sha256 %s, %d files)",
		filepath.Base(path), dump.PoBVersion, dump.TreeVersion,
		dump.PoBPin.label(), shortHash(dump.PoBPin.ContentSHA256), dump.PoBPin.FileCount)
	return dump
}

func resolvePoBTableReference(dir string, want pobPinRecord) (*pobTableDump, string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, "", fmt.Errorf("REFUSING: cannot read %s: %w", dir, err)
	}

	var (
		path    string
		dump    pobTableDump
		matched []string
	)
	available := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		candidatePath := filepath.Join(dir, entry.Name())
		blob, err := os.ReadFile(candidatePath)
		if err != nil {
			return nil, "", fmt.Errorf("REFUSING: cannot read %s: %w", candidatePath, err)
		}
		var candidate pobTableDump
		if err := json.Unmarshal(blob, &candidate); err != nil {
			return nil, "", fmt.Errorf("REFUSING: %s is not valid JSON: %w", candidatePath, err)
		}
		available = append(available, fmt.Sprintf("%s (PoB %s, tree %s, pin %s %s)",
			entry.Name(), candidate.PoBVersion, candidate.TreeVersion,
			candidate.PoBPin.label(), shortHash(candidate.PoBPin.ContentSHA256)))
		if candidate.PoBPin.ContentSHA256 != "" && candidate.PoBPin.ContentSHA256 == want.ContentSHA256 {
			matched = append(matched, entry.Name())
			path, dump = candidatePath, candidate
		}
	}
	sort.Strings(available)

	switch {
	case len(matched) == 0:
		return nil, "", fmt.Errorf("REFUSING: no reference in %s was dumped from the pinned PoB.\n"+
			"    %s pins %s (content_sha256 %s).\n"+
			"    Found: %v\n"+
			"    Regenerate with `task pob-data-tables`.\n"+
			"    ⚠️ DO NOT relax this to pick the closest or the newest file. Grading against the\n"+
			"    PREVIOUS pin's tables reports \"no drift\" on exactly the version bump this gate\n"+
			"    exists to catch, which is the worst failure available to this instrument.",
			dir, recordedPinPath, want.label(), want.ContentSHA256, available)
	case len(matched) > 1:
		return nil, "", fmt.Errorf("REFUSING: %d references in %s claim pin %s: %v.\n"+
			"    Which is authoritative is unanswerable, and picking either would make this gate's\n"+
			"    verdict depend on directory order.", len(matched), dir, want.ContentSHA256, matched)
	}
	return &dump, path, nil
}

func loadRecordedPin(t *testing.T) pobPinRecord {
	t.Helper()

	blob, err := os.ReadFile(recordedPinPath)
	if err != nil {
		t.Fatalf("REFUSING: cannot read the PoB pin record %s: %v. Without it there is no way to "+
			"tell which reference belongs to the current checkout.", recordedPinPath, err)
	}
	var pin pobPinRecord
	if err := json.Unmarshal(blob, &pin); err != nil {
		t.Fatalf("REFUSING: %s is not valid JSON: %v", recordedPinPath, err)
	}
	if pin.ContentSHA256 == "" {
		t.Fatalf("REFUSING: %s records no \"content_sha256\". An empty pin matches nothing and "+
			"would be reported as a missing reference, which is a different and misleading "+
			"diagnosis.", recordedPinPath)
	}
	return pin
}

func shortHash(h string) string {
	if len(h) <= 12 {
		return h
	}
	return h[:12] + "…"
}

func keysOfAny[V any](m map[string]V) map[string]struct{} {
	out := make(map[string]struct{}, len(m))
	for k := range m {
		out[k] = struct{}{}
	}
	return out
}

func toSet(list []string) map[string]struct{} {
	out := make(map[string]struct{}, len(list))
	for _, s := range list {
		out[s] = struct{}{}
	}
	return out
}

func diffSets(ours, theirs map[string]struct{}) ([]string, []string) {
	var oursOnly, theirsOnly []string
	for k := range ours {
		if _, ok := theirs[k]; !ok {
			oursOnly = append(oursOnly, k)
		}
	}
	for k := range theirs {
		if _, ok := ours[k]; !ok {
			theirsOnly = append(theirsOnly, k)
		}
	}
	sort.Strings(oursOnly)
	sort.Strings(theirsOnly)
	return oursOnly, theirsOnly
}

func countDiffs(a, b []float64) int {
	n := 0
	for i := range a {
		if math.Abs(a[i]-b[i]) > 1e-6 {
			n++
		}
	}
	return n
}

func capped(s []string) []string {
	if len(s) <= 3 {
		return s
	}
	return s[:3]
}

func deref(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

func first(s []float64) any {
	if len(s) == 0 {
		return "<empty>"
	}
	return s[0]
}

func declarationFor(finding string) (string, string, bool) {
	bestKey, bestReason := "", ""
	for key, reason := range declaredDataDrift {
		if !strings.HasPrefix(finding, key) {
			continue
		}
		if rest := finding[len(key):]; rest != "" && rest[0] != ':' && rest[0] != ' ' {
			continue
		}
		if len(key) > len(bestKey) {
			bestKey, bestReason = key, reason
		}
	}
	return bestKey, bestReason, bestKey != ""
}
