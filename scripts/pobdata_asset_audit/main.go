package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"sort"
	"sync"

	"github.com/Vilsol/go-pob-data/fields"
	"github.com/Vilsol/go-pob-data/loader"
	"github.com/andybalholm/brotli"
)

var assets = []string{
	"ActiveSkillType", "ActiveSkills", "AlternatePassiveAdditions", "AlternatePassiveSkills",
	"ArmourTypes", "BaseItemTypes", "ComponentAttributeRequirements", "ComponentCharges",
	"CostTypes", "CraftingBenchOptions", "DefaultMonsterStats", "Essences", "Flasks",
	"GrantedEffectQualityStats", "GrantedEffectStatSets", "GrantedEffectStatSetsPerLevel",
	"GrantedEffects", "GrantedEffectsPerLevel", "ItemClasses", "ItemExperiencePerLevel", "Mods",
	"MonsterMapBossDifficulty", "MonsterMapDifficulty", "MonsterVarieties", "PantheonPanelLayout",
	"PassiveTreeExpansionJewels", "PassiveTreeExpansionSkills", "PassiveTreeExpansionSpecialSkills",
	"ShieldTypes", "SkillGems", "SkillTotemVariations", "SkillTotems", "Stats", "Tags",
	"WeaponTypes",
}

type row struct {
	name, status string
	corrupt      bool
}

func main() {
	versions := os.Args[1:]
	if len(versions) == 0 {
		fmt.Fprintln(os.Stderr, "usage: go run . <version>...   e.g. 3.18 3.20 3.21")
		fmt.Fprintln(os.Stderr, "REFUSING: auditing nothing would print a clean report.")
		os.Exit(2)
	}

	exit := 0
	for _, v := range versions {
		rows := make([]row, len(assets))
		var wg sync.WaitGroup
		sem := make(chan struct{}, 8)
		for i, name := range assets {
			wg.Add(1)
			go func(i int, name string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				rows[i] = check(v, name)
			}(i, name)
		}
		wg.Wait()

		var corrupt, absent int
		for _, r := range rows {
			if r.corrupt {
				corrupt++
			}
			if r.status == "absent (404)" {
				absent++
			}
		}
		fmt.Printf("=== %s: %d of %d CORRUPT, %d absent ===\n", v, corrupt, len(assets), absent)
		sort.Slice(rows, func(a, b int) bool { return rows[a].name < rows[b].name })
		for _, r := range rows {
			if r.corrupt {
				fmt.Printf("  CORRUPT  %-32s %s\n", r.name, r.status)
			}
		}
		for _, r := range rows {
			if r.status == "absent (404)" {
				fmt.Printf("  ABSENT   %-32s %s\n", r.name, r.status)
			}
		}
		if corrupt > 0 {
			fmt.Printf("  -> this version CANNOT be loaded: poe.InitializeAll panics on the first one\n")
			exit = 1
		}
		fmt.Println()
	}
	os.Exit(exit)
}

func check(version, name string) row {
	t, ok := fields.RowType("raw/" + name)
	if !ok {
		return row{name, "go-pob-data has no row type for it", true}
	}
	ctx := loader.WithVariants(context.Background(), fields.MinVariant)
	url := loader.VariantURL(ctx, version, "raw/"+name, reflect.New(t).Interface())
	resp, err := http.Get(url)
	if err != nil {
		return row{name, "FETCH ERR: " + err.Error(), true}
	}
	raw, _ := io.ReadAll(resp.Body)
	code := resp.StatusCode
	resp.Body.Close()

	if code == 404 {
		return row{name, "absent (404)", false}
	}
	if code != 200 {
		return row{name, fmt.Sprintf("HTTP %d", code), true}
	}

	out, derr := io.ReadAll(brotli.NewReader(bytes.NewReader(raw)))
	if derr != nil {
		return row{name, fmt.Sprintf("%v (raw %d, decoded %d before failing)", derr, len(raw), len(out)), true}
	}
	return row{name, fmt.Sprintf("ok (raw %d -> %d)", len(raw), len(out)), false}
}
