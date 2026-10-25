// Package calculator2 is the Path of Building (PoE2) calculation engine. The PoE1 engine never
// imports it.
package calculator2

import (
	"fmt"
	"strconv"

	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob2"
)

// Options carries what BuildOutput needs beside the build file.
type Options struct {
	// Tree is the build's passive tree, loaded by tree2.Load for the spec's treeVersion.
	Tree *tree2.PassiveTree
	// ModSource parses a mod line; nil is WithParser(ParseMod), PoB's modLib.parseMod.
	ModSource ModSource
	// Items is build.itemsTab, from NewItemsTab; nil means the build has no items.
	Items *ItemsTab
	// SpecItems is the passive spec's view of build.itemsTab.items (jewel radius and jewelData) that
	// allocation reads; nil is Items.SpecItem, or no item when Items is nil too.
	SpecItems func(itemID int) *tree2.Item
}

// WithParser is modLib.parseMod: PoB-PoE2's shipped ModCache first, and the parser for a line the
// cache does not hold.
//
//pob2:src/Modules/ModParser.lua:7435-7454
func WithParser(parse ModSource) ModSource {
	return func(line string) ([]mod.Mod, string) {
		if shippedCacheLookup(line) == nil {
			return parse(line)
		}
		return ModCache(line)
	}
}

// BuildOutput is calcs.buildOutput's first half for the "MAIN" mode: initEnv then perform. The returned
// env's Player.Output holds the computed stats.
//
//pob2:src/Modules/Calcs.lua:518-523
func BuildOutput(build *pob2.PathOfBuilding2, opts Options) (*Environment, error) {
	if opts.Tree == nil {
		return nil, fmt.Errorf("no passive tree")
	}
	env := &Environment{
		Build:        build,
		Mode:         "MAIN",
		ConfigInput:  configInput(build),
		modSource:    opts.ModSource,
		items:        opts.Items,
		nodeModLists: map[int]*moddb.ModList{},
	}
	if env.modSource == nil {
		env.modSource = WithParser(ParseMod)
	}
	specItems := opts.SpecItems
	if specItems == nil && opts.Items != nil {
		specItems = opts.Items.SpecItem
	}
	spec, err := loadSpec(build, opts.Tree, env.modSource, specItems)
	if err != nil {
		return nil, err
	}
	env.Spec = spec
	if opts.Items != nil {
		opts.Items.SetActiveSpec(spec)
	}
	env.ClassID = spec.CurClassID
	if spec.Tree.Classes[env.ClassID] == nil {
		return nil, fmt.Errorf("class %d is not in the tree", env.ClassID)
	}
	initEnv(env)
	perform(env)
	return env, nil
}

// loadSpec is TreeTab:Load for the active spec.
//
//pob2:src/Classes/PassiveSpec.lua:117-379
func loadSpec(build *pob2.PathOfBuilding2, tree *tree2.PassiveTree, source ModSource, specItems func(int) *tree2.Item) (*tree2.PassiveSpec, error) {
	specs := build.Tree.Specs
	if len(specs) == 0 {
		return nil, fmt.Errorf("the build has no passive spec")
	}
	idx := build.Tree.ActiveSpec
	if idx < 1 || idx > len(specs) {
		idx = 1
	}
	s := specs[idx-1]
	in := tree2.SpecInput{
		Nodes:                s.Nodes,
		MasteryEffects:       s.MasteryEffects,
		AscendancyInternalID: s.AscendancyInternalID,
		WeaponSets:           map[int]string{},
	}
	if s.Title != nil {
		in.Title = *s.Title
	}
	if s.TreeVersion != nil {
		in.TreeVersion = *s.TreeVersion
	}
	itoa := func(v *int) *string {
		if v == nil {
			return nil
		}
		out := strconv.Itoa(*v)
		return &out
	}
	in.ClassID = itoa(s.ClassID)
	in.AscendClassID = itoa(s.AscendClassID)
	in.ClassInternalID = itoa(s.ClassInternalID)
	if !s.SecondaryAscendClassID.IsNil {
		in.SecondaryAscendClassID = itoa(&s.SecondaryAscendClassID.V)
	}
	if s.URL != nil {
		in.URL = &s.URL.Text
	}
	if s.WeaponSet1 != nil {
		in.WeaponSets[1] = s.WeaponSet1.Nodes
	}
	if s.WeaponSet2 != nil {
		in.WeaponSets[2] = s.WeaponSet2.Nodes
	}
	if s.Sockets != nil {
		for _, sock := range s.Sockets.Sockets {
			in.Sockets = append(in.Sockets, tree2.SocketInput{NodeID: strconv.Itoa(sock.NodeID), ItemID: strconv.Itoa(sock.ItemID)})
		}
	}
	if s.Overrides != nil {
		for _, o := range s.Overrides.AttributeOverrides {
			in.AttributeOverrides = append(in.AttributeOverrides, tree2.AttributeOverrideInput{StrNodes: o.StrNodes, DexNodes: o.DexNodes, IntNodes: o.IntNodes})
		}
	}
	spec := tree2.NewPassiveSpec(tree)
	spec.Items = specItems
	spec.AllocateFromNodeRadius = func(node *tree2.SpecNode) []tree2.NodeRadius {
		return allocateFromNodeRadius(processStats(source, node.ID, node.Type, node.Dn, node.Sd))
	}
	if err := spec.Load(in); err != nil {
		return nil, err
	}
	return spec, nil
}

// configInput is ConfigTab:Load's input table for the active config set, as setInputAndPlaceholder
// stores it; the per-option defaults CreateConfigSet seeds are applied where each option is read.
//
//pob2:src/Classes/ConfigTab.lua:878-978
func configInput(build *pob2.PathOfBuilding2) map[string]ConfigValue {
	inputs := build.Config.Inputs
	for _, set := range build.Config.ConfigSets {
		if set.ID == activeConfigSetID(build) {
			inputs = set.Inputs
		}
	}
	out := make(map[string]ConfigValue, len(inputs))
	for _, in := range inputs {
		var v ConfigValue
		switch {
		case in.Number != nil:
			v.Number, v.IsNumber = *in.Number, true
		case in.String != nil:
			v.String, v.IsString = *in.String, true
		case in.Boolean != nil:
			v.Boolean, v.IsBoolean = *in.Boolean, true
		}
		out[in.Name] = v
	}
	return out
}

func activeConfigSetID(build *pob2.PathOfBuilding2) int {
	if build.Config.ActiveConfigSet == 0 {
		return 1
	}
	return build.Config.ActiveConfigSet
}

// customModifierBlocks is the active config set's customModsList.
func customModifierBlocks(build *pob2.PathOfBuilding2) []pob2.CustomModifierBlock {
	blocks := build.Config.CustomModifierBlocks
	for _, set := range build.Config.ConfigSets {
		if set.ID == activeConfigSetID(build) {
			blocks = set.CustomModifierBlocks
		}
	}
	return blocks
}

// activeSkillSet is skillsTab.socketGroupList: the active skill set's groups, removed ones left out.
//
//pob2:block:src/Classes/SkillsTab.lua:393-398
func activeSkillSet(env *Environment) []pob2.Skill {
	skills := env.Build.Skills
	var list []pob2.Skill
	if len(skills.SkillSets) == 0 {
		list = skills.Skills
	} else {
		list = skills.SkillSets[0].Skills
		for _, set := range skills.SkillSets {
			if set.ID == skills.ActiveSkillSet {
				list = set.Skills
			}
		}
	}
	out := make([]pob2.Skill, 0, len(list))
	for _, s := range list {
		if s.Removed != nil && *s.Removed {
			continue
		}
		out = append(out, s)
	}
	return out
}

// useSecondWeaponSet is itemsTab.activeItemSet.useSecondWeaponSet and 2 or 1.
//
//pob2:block:src/Classes/SkillsTab.lua:1400
func useSecondWeaponSet(env *Environment) int {
	items := env.Build.Items
	for _, set := range items.ItemSets {
		if set.ID == items.ActiveItemSet {
			if !set.UseSecondWeaponSet.IsNil && set.UseSecondWeaponSet.V {
				return 2
			}
			return 1
		}
	}
	if !items.UseSecondWeaponSet.IsNil && items.UseSecondWeaponSet.V {
		return 2
	}
	return 1
}

//pob2:block:src/Classes/PassiveSpec.lua:1499-1503
func allocateFromNodeRadius(modList *moddb.ModList) []tree2.NodeRadius {
	var out []tree2.NodeRadius
	for _, v := range modList.List(nil, "AllocateFromNodeRadius") {
		t, ok := v.(LuaTable)
		if !ok {
			continue
		}
		r := tree2.NodeRadius{}
		r.From, _ = t["from"].(string)
		idx, _ := t["radiusIndex"].(float64)
		r.RadiusIndex = int(idx)
		to, _ := t["to"].([]any)
		for _, x := range to {
			if s, ok := x.(string); ok {
				r.To = append(r.To, s)
			}
		}
		out = append(out, r)
	}
	return out
}
