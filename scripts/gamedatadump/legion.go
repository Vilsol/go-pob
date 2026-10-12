package main

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob-data/loader"
	"github.com/Vilsol/go-pob-data/poe"
	rawdata "github.com/Vilsol/go-pob-data/raw"

	"github.com/Vilsol/go-pob/data/raw"
)

//pob:src/Export/Scripts/legionPassives.lua:6-56
var legionDatErrors = map[string]struct{ matchName, matchID, replaceID string }{
	"templar_notable_minimum_frenzy_charge":    {matchName: "Powerful Faith", replaceID: "templar_notable_minimum_power_charge"},
	"templar_notable_minimum_power_charge":     {matchName: "Frenzied Faith", replaceID: "templar_notable_minimum_frenzy_charge"},
	"karui_notable_add_physical_taken_as_fire": {matchID: "karui_notable_add_physical_taken_as_fire", replaceID: "karui_notable_add_rage_on_melee_hit"},
	"karui_notable_add_faster_burn":            {matchID: "karui_notable_add_faster_burn", replaceID: "karui_notable_add_faster_ignite"},
	"maraketh_notable_add_ailment_avoid":       {matchID: "maraketh_notable_add_ailment_avoid", replaceID: "maraketh_notable_add_stun_avoid"},
	"maraketh_notable_add_flask_effect":        {matchID: "maraketh_notable_add_flask_effect", replaceID: "maraketh_notable_add_alchemists_genius"},
}

//pob:src/Export/Scripts/legionPassives.lua:58-67
func fixLegionDatErrors(id, name string) string {
	e, ok := legionDatErrors[id]
	if !ok || (e.matchName != "" && name != e.matchName) || (e.matchID != "" && id != e.matchID) {
		return id
	}
	return e.replaceID
}

type legionRoll struct {
	Key   string  `json:"key"`
	Fmt   string  `json:"fmt"`
	Index int     `json:"index"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
}

type legionRow struct {
	ID    string       `json:"id"`
	Dn    string       `json:"dn"`
	Sd    []string     `json:"sd"`
	Rolls []legionRoll `json:"rolls"`
}

type legionStat struct {
	r         *edRange
	index     int
	statOrder float64
}

var (
	legionKeystoneRe  = regexp.MustCompile(`^[A-Za-z]+_keystone_`)
	legionLeadWordRe  = regexp.MustCompile(`^[A-Za-z0-9]* `)
	legionLowerWordRe = regexp.MustCompile(`[a-z][A-Za-z0-9]*`)
)

//pob:src/Export/Scripts/legionPassives.lua:109-139
func parseLegionStats(describer *exportDescriber, id, dn string, statKeys []int, ranges [][2]int) (legionRow, error) {
	stats := map[string]*legionStat{}
	for idx, key := range statKeys {
		if key < 0 || key >= len(poe.Stats) || idx >= len(ranges) {
			return legionRow{}, fmt.Errorf("%s: stat %d (key %d) has no range", id, idx+1, key)
		}
		statID := poe.Stats[key].ID
		st := &legionStat{r: &edRange{min: float64(ranges[idx][0]), max: float64(ranges[idx][1])}, index: idx + 1, statOrder: math.Inf(1)}
		_, orders, err := describer.describeStats(map[string][2]float64{statID: {st.r.min, st.r.max}})
		if err != nil {
			return legionRow{}, fmt.Errorf("%s: %w", id, err)
		}
		if len(orders) > 0 {
			st.statOrder = orders[0]
		}
		stats[statID] = st
	}
	ranged := map[string]*edRange{}
	for statID, st := range stats {
		ranged[statID] = st.r
	}
	sd, _, err := describer.describeRanges(ranged)
	if err != nil {
		return legionRow{}, fmt.Errorf("%s: %w", id, err)
	}
	sorted := make([]string, 0, len(stats))
	for statID := range stats {
		sorted = append(sorted, statID)
	}
	sort.Strings(sorted)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := stats[sorted[i]], stats[sorted[j]]
		return a.statOrder < b.statOrder || a.statOrder == b.statOrder && a.index < b.index
	})

	if id == "vaal_keystone_2_v2" {
		// Immortal Ambition needs to be manually added
		sd = []string{
			"Energy Shield starts at zero",
			"Cannot Recharge or Regenerate Energy Shield",
			"Lose 5% of Energy Shield per second",
			"Life Leech effects are not removed when Unreserved Life is Filled",
			"Life Leech effects Recover Energy Shield instead while on Full Life",
		}
	}
	row := legionRow{ID: id, Dn: dn, Sd: append([]string{}, sd...), Rolls: []legionRoll{}}
	for i := range row.Sd {
		if i >= len(sorted) {
			break
		}
		st := stats[sorted[i]]
		row.Rolls = append(row.Rolls, legionRoll{Key: sorted[i], Fmt: st.r.fmt, Index: st.index, Min: st.r.min, Max: st.r.max})
	}
	return row, nil
}

//pob:block:src/Export/Scripts/legionPassives.lua:141-247
func dumpLegionPassives() (any, error) {
	ctx, cache := gameDataContext(), assetCache()
	var skills []*rawdata.AlternatePassiveSkill
	var additions []*rawdata.AlternatePassiveAddition
	for _, init := range []func() error{
		func() error { return poe.InitializeStats(ctx, raw.LatestVersion, cache) },
		func() error {
			return loader.InitHelper(ctx, raw.LatestVersion, "AlternatePassiveSkills", &skills, nil, cache)
		},
		func() error {
			return loader.InitHelper(ctx, raw.LatestVersion, "AlternatePassiveAdditions", &additions, nil, cache)
		},
	} {
		if err := init(); err != nil {
			return nil, err
		}
	}
	describer, err := loadStatFile("passive_skill_stat_descriptions")
	if err != nil {
		return nil, err
	}

	keystones, named, nodes, adds := map[string]legionRow{}, map[string]legionRow{}, map[string]legionRow{}, map[string]legionRow{}
	for _, s := range skills {
		id := fixLegionDatErrors(s.ID, s.Name)
		row, err := parseLegionStats(describer, id, s.Name, s.StatsKeys,
			[][2]int{{s.Stat1Min, s.Stat1Max}, {s.Stat2Min, s.Stat2Max}, {s.Stat3Min, s.Stat3Max}, {s.Stat4Min, s.Stat4Max}})
		if err != nil {
			return nil, err
		}
		nodes[id] = row
		if legionKeystoneRe.MatchString(id) {
			keystones[id] = row
		} else if id == "templar_devotion_node" || id == "eternal_small_blank" {
			named[id] = row
		}
	}
	for _, a := range additions {
		id := fixLegionDatErrors(a.ID, "")
		dn := strings.ReplaceAll(id, "_", " ")
		dn = legionLeadWordRe.ReplaceAllString(dn, "")
		dn = legionLeadWordRe.ReplaceAllString(dn, "")
		dn = legionLowerWordRe.ReplaceAllStringFunc(dn, func(w string) string { return strings.ToUpper(w[:1]) + w[1:] })
		row, err := parseLegionStats(describer, id, dn, a.StatsKeys, [][2]int{{a.Stat1Min, a.Stat1Max}, {a.Stat2Min, a.Stat2Max}})
		if err != nil {
			return nil, err
		}
		adds[id] = row
	}
	if len(keystones) == 0 || len(adds) == 0 || len(named) != 2 {
		return nil, fmt.Errorf("%d keystones, %d additions, %d of 2 named nodes", len(keystones), len(adds), len(named))
	}
	return map[string]any{
		"pobVersion": "go-pob-data " + raw.LatestVersion,
		"scanned":    len(skills),
		"keystones":  keystones,
		"named":      named,
		"nodes":      nodes,
		"additions":  adds,
	}, nil
}
