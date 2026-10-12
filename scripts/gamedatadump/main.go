package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/Vilsol/go-pob-data/loader"

	"github.com/Vilsol/go-pob/config"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

var dumpers = map[string]func() (any, error){
	"stat-descriptions":   dumpStatDescriptions,
	"pantheons":           dumpPantheons,
	"cluster-jewels":      dumpClusterJewels,
	"legion-passives":     dumpLegionPassives,
	"tattoo-passives":     dumpTattooPassives,
	"mod-scalability":     dumpModScalability,
	"skill-weapon-types":  dumpSkillWeaponTypes,
	"item-mod-stat-order": dumpItemModStatOrder,
	"item-craft-mods":     dumpItemCraftMods,
	"flavour-text":        dumpFlavourText,
	"gems":                dumpGems,
	"minion-data":         dumpMinionData,
	"core-constants-data": dumpCoreConstantsData,
	"skills-data":         dumpSkillsData,
}

func main() {
	if len(os.Args) != 3 || dumpers[os.Args[1]] == nil {
		names := make([]string, 0, len(dumpers))
		for name := range dumpers {
			names = append(names, name)
		}
		sort.Strings(names)
		fmt.Fprintf(os.Stderr, "usage: gamedatadump <%v> <out.json>\n", names)
		os.Exit(2)
	}
	config.InitLogging(false)
	out, err := dumpers[os.Args[1]]()
	if err != nil {
		fmt.Fprintf(os.Stderr, "REFUSING: %s: %v\n", os.Args[1], err)
		os.Exit(3)
	}
	b, err := json.MarshalIndent(out, "", "\t")
	if err != nil {
		fmt.Fprintf(os.Stderr, "REFUSING: %s does not encode: %v\n", os.Args[1], err)
		os.Exit(3)
	}
	if err := os.WriteFile(os.Args[2], append(b, '\n'), 0o644); err != nil { //nolint:gosec // the caller's output path
		fmt.Fprintf(os.Stderr, "REFUSING: %v\n", err)
		os.Exit(3)
	}
}

func gameDataContext() context.Context {
	return context.Background()
}

func assetCache() loader.AssetCache {
	return raw.AssetLoaderWrapper{Storage: storage.Get()}
}
