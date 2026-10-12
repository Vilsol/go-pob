package main

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob-data/loader"
	"github.com/Vilsol/go-pob-data/poe"
	rawdata "github.com/Vilsol/go-pob-data/raw"

	"github.com/Vilsol/go-pob/data/raw"
)

type tattooNode struct {
	ID         string   `json:"id"`
	Dn         string   `json:"dn"`
	Sd         []string `json:"sd"`
	IsTattoo   bool     `json:"isTattoo"`
	IsKeystone bool     `json:"isKeystone"`

	OverrideType      string   `json:"overrideType"`
	TargetType        string   `json:"targetType"`
	TargetValue       string   `json:"targetValue"`
	MinimumConnected  int      `json:"MinimumConnected"`
	MaximumConnected  int      `json:"MaximumConnected"`
	ReminderText      []string `json:"reminderText,omitempty"`
	Legacy            *bool    `json:"legacy,omitempty"`
	Icon              string   `json:"icon"`
	ActiveEffectImage string   `json:"activeEffectImage"`
}

func tattooSortSd(sd []string, descOrders map[string]float64) {
	sort.SliceStable(sd, func(i, j int) bool { return descOrders[sd[i]] < descOrders[sd[j]] })
}

//pob:src/Export/Scripts/tattooPassives.lua:50-74
func tattooPassiveStats(describer *exportDescriber, node *rawdata.PassiveSkill) ([]string, error) {
	values := []int64{node.Stat1, node.Stat2, node.Stat3, node.Stat4, node.Stat5}
	var sd []string
	descOrders := map[string]float64{}
	for idx, key := range node.Stats {
		if idx >= len(values) {
			return nil, fmt.Errorf("%s: stat %d has no value", node.ID, idx+1)
		}
		v := float64(values[idx])
		lines, orders, err := describer.describeStats(map[string][2]float64{poe.Stats[key].ID: {v, v}})
		if err != nil {
			return nil, fmt.Errorf("%s: %w", node.ID, err)
		}
		for i, line := range lines {
			sd = append(sd, line)
			descOrders[line] = orders[i]
		}
	}
	tattooSortSd(sd, descOrders)
	return sd, nil
}

//pob:src/Export/Scripts/tattooPassives.lua:76-98
func tattooStats(describer *exportDescriber, o *poe.PassiveSkillOverride) ([]string, error) {
	stats := map[string][2]float64{}
	for idx, key := range o.Stats {
		if idx >= len(o.StatValues) {
			return nil, fmt.Errorf("%s: stat %d has no value", o.ID, idx+1)
		}
		v := float64(o.StatValues[idx])
		stats[poe.Stats[key].ID] = [2]float64{v, v}
	}
	lines, orders, err := describer.describeStats(stats)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.ID, err)
	}
	descOrders := map[string]float64{}
	for i, line := range lines {
		descOrders[line] = orders[i]
	}
	sd := append([]string{}, lines...)
	tattooSortSd(sd, descOrders)
	return sd, nil
}

//pob:block:src/Export/Scripts/tattooPassives.lua:100-193
func dumpTattooPassives() (any, error) {
	ctx, cache := gameDataContext(), assetCache()
	var passives []*rawdata.PassiveSkill
	for _, init := range []func() error{
		func() error { return poe.InitializeStats(ctx, raw.LatestVersion, cache) },
		func() error { return poe.InitializeClientStrings(ctx, raw.LatestVersion, cache) },
		func() error { return poe.InitializePassiveSkillOverrideTypes(ctx, raw.LatestVersion, cache) },
		func() error { return poe.InitializePassiveOverrideLimits(ctx, raw.LatestVersion, cache) },
		func() error { return poe.InitializePassiveSkillOverrides(ctx, raw.LatestVersion, cache) },
		func() error { return poe.InitializePassiveSkillTattooTargetSets(ctx, raw.LatestVersion, cache) },
		func() error { return poe.InitializePassiveSkillTattoos(ctx, raw.LatestVersion, cache) },
		func() error { return poe.InitializeBaseItemTypes(ctx, raw.LatestVersion, cache) },
		func() error { return poe.InitializeCurrencyExchanges(ctx, raw.LatestVersion, cache) },
		func() error { return loader.InitHelper(ctx, raw.LatestVersion, "PassiveSkills", &passives, nil, cache) },
	} {
		if err := init(); err != nil {
			return nil, err
		}
	}
	describer, err := loadStatFile("passive_skill_stat_descriptions")
	if err != nil {
		return nil, err
	}
	clientString := func(id string) (string, error) {
		for _, s := range poe.ClientStrings {
			if s.ID == id {
				return s.Text, nil
			}
		}
		return "", fmt.Errorf("no ClientStrings row %s", id)
	}

	//pob:block:src/Export/Scripts/tattooPassives.lua:105-113
	tattooRows := map[string]*poe.PassiveSkillTattoo{}
	for _, t := range poe.PassiveSkillTattoos {
		tattooRows[poe.PassiveSkillOverrides[t.Override].ID] = t
	}

	nodes := map[string]tattooNode{}
	for _, o := range poe.PassiveSkillOverrides {
		node := tattooNode{ID: o.ID, IsTattoo: true, OverrideType: poe.PassiveSkillOverrideTypes[o.Type].ID}
		tattooRow := tattooRows[o.ID]
		if tattooRow == nil {
			tattooRow = tattooRows["DisplayRandomKeystone"]
		}
		if tattooRow == nil {
			return nil, fmt.Errorf("%s has no PassiveSkillTattoos row and there is no DisplayRandomKeystone row", o.ID)
		}
		target := poe.PassiveSkillTattooTargetSets[tattooRow.Set]
		node.TargetType, node.TargetValue = target.Name, target.Qualifier
		node.MinimumConnected = o.RequiresAdjacent
		if o.RequiresAdjacent > 0 {
			text, err := clientString("PassiveSkillTattooAdjacentRequirementLower")
			if err != nil {
				return nil, err
			}
			node.ReminderText = []string{strings.ReplaceAll(text, "{}", strconv.Itoa(o.RequiresAdjacent))}
		}
		node.MaximumConnected = 100
		if o.MaxAdjacent > 0 {
			text, err := clientString("PassiveSkillTattooAdjacentRequirementUpper")
			if err != nil {
				return nil, err
			}
			node.ReminderText = []string{strings.ReplaceAll(text, "{}", strconv.Itoa(o.MaxAdjacent))}
			node.MaximumConnected = o.MaxAdjacent
		}
		node.ActiveEffectImage = o.PassiveBG + ".png"
		icon := o.NodeIcon
		var limitText string
		if o.Limit != nil {
			text, err := clientString("PassiveSkillTattooLimitReminder")
			if err != nil {
				return nil, err
			}
			limitText = strings.ReplaceAll(text, "{0}", poe.PassiveOverrideLimits[*o.Limit].Description)
		}
		name, rowID := o.Name, o.ID
		switch poe.PassiveSkillOverrideTypes[o.Type].ID {
		case "KeystoneTattoo":
			node.IsKeystone = true
			if o.AllocatedPassiveSkill == nil {
				return nil, fmt.Errorf("%s is a keystone tattoo without a passive", o.ID)
			}
			passive := passives[*o.AllocatedPassiveSkill]
			name, rowID, icon = passive.Name, passive.ID, passive.Icon
			if node.Sd, err = tattooPassiveStats(describer, passive); err != nil {
				return nil, err
			}
		default:
			if node.Sd, err = tattooStats(describer, o); err != nil {
				return nil, err
			}
		}
		node.Dn = name
		if name != "" && !node.IsKeystone {
			if legacy, ok := tattooLegacy(name); ok {
				node.Legacy = &legacy
			}
		}
		node.Icon = strings.TrimSuffix(icon, ".dds")
		if node.Icon != icon {
			node.Icon += ".png"
		}
		if limitText != "" {
			node.Sd = append(node.Sd, limitText)
		}
		if node.Sd == nil {
			node.Sd = []string{}
		}
		if rowID != "DisplayRandomKeystone" && !strings.Contains(name, "DNT") && !strings.Contains(name, "of the Test") {
			nodes[name] = node
		}
	}
	scanned := len(nodes)
	delete(nodes, "")
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no PassiveSkillOverrides row")
	}
	return map[string]any{"pobVersion": "go-pob-data " + raw.LatestVersion, "scanned": scanned, "nodes": nodes}, nil
}

//pob:block:src/Export/Scripts/tattooPassives.lua:179-184
func tattooLegacy(name string) (bool, bool) {
	base := slices.IndexFunc(poe.BaseItemTypes, func(b *poe.BaseItemType) bool { return b.Name == name })
	if base < 0 {
		return false, false
	}
	for _, c := range poe.CurrencyExchanges {
		if c.Item == base {
			return !c.EnabledInChallengeLeague, true
		}
	}
	return false, false
}
