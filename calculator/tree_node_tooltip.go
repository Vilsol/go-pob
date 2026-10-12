package calculator

import (
	"slices"
	"strconv"

	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/utils"
)

//pob:block:src/Classes/PassiveTreeView.lua:1607-1627
func NodeSkillTooltips(build *pob.PathOfBuilding, nodeID int64, ctx ItemTooltipContext) [][]TooltipLine {
	node, ok := specTree().Nodes[strconv.FormatInt(nodeID, 10)]
	if !ok {
		return nil
	}
	if build != nil {
		ctx.CharacterLevel = build.Build.Level
	}
	var mods []mod.Mod
	for _, line := range node.Stats {
		if entry := ParseMod(line, false); entry != nil {
			mods = append(mods, entry.ModList...)
		}
	}
	list := asModList(mods)
	type granted struct {
		skillID string
		level   int
	}
	var skills []granted
	for _, v := range utils.CastSlice[mod.ExtraSkill](list.List(nil, "ExtraSkill")) {
		skills = append(skills, granted{v.SkillID, v.Level})
	}
	for _, v := range utils.CastSlice[mod.ExtraSupport](list.List(nil, "ExtraSupport")) {
		skills = append(skills, granted{v.SkillID, v.Level})
	}
	var out [][]TooltipLine
	for _, g := range skills {
		level := g.level
		if level == 0 {
			level = 1
		}
		t := ItemTooltip{}
		if gemID, ok := gemForSkill()[g.skillID]; ok {
			if AddGemTooltip(&t, ctx, pob.Gem{GemID: gemID, SkillID: g.skillID, Level: level, Enabled: true}, nil, false) {
				out = append(out, t.Lines)
			}
		} else if addSkillOnlyTooltip(&t, ctx, g.skillID, level) {
			out = append(out, t.Lines)
		}
	}
	return out
}

func GoldRespecPrice(build *pob.PathOfBuilding) int {
	prices := raw2.GoldRespecPrices()
	if level := build.Build.Level; level >= 1 && level <= len(prices) {
		return prices[level-1]
	}
	return 0
}

type TreeCompareSpec struct {
	Nodes     []int64
	Masteries []TreeCompareMastery
}

type TreeCompareMastery struct {
	Node   int64
	Effect int64
	Stats  []string
}

//pob:block:src/Classes/TreeTab.lua:106-128
func TreeSpecCompare(build *pob.PathOfBuilding, specID int) TreeCompareSpec {
	if specID < 1 || specID > len(build.Tree.Specs) {
		return TreeCompareSpec{}
	}
	spec := build.Tree.Specs[specID-1]
	out := TreeCompareSpec{Nodes: pob.SpecNodes(spec)}
	tree := specTree()
	for _, m := range pob.SpecMasterySelections(spec) {
		c := TreeCompareMastery{Node: m.Node, Effect: m.Effect}
		if n, ok := tree.Nodes[strconv.FormatInt(m.Node, 10)]; ok {
			for _, e := range n.MasteryEffects {
				if e.Effect == m.Effect {
					c.Stats = e.Stats
				}
			}
		}
		out.Masteries = append(out.Masteries, c)
	}
	return out
}

//pob:block:src/Classes/GemTooltip.lua:239-258
func addSkillOnlyTooltip(t *ItemTooltip, ctx ItemTooltipContext, skillID string, level int) bool {
	primary := newGemTooltipEffect(skillID)
	if primary == nil {
		return false
	}
	gi := &gemTooltipInstance{gem: pob.Gem{SkillID: skillID, Level: level, Enabled: true}, row: raw2.PoBGem{GrantedEffectID: skillID}}
	gi.display = GemDisplay{Level: level}
	t.addLine(colorGem + primary.text.Name)
	t.addSeparator()
	if primary.text.Legacy {
		t.addLine(colorWarning + "Legacy Gem")
		t.addLine(colorWarning + "Gem only exists in Standard League")
		t.addSeparator()
	}
	addCommonGemInfo(t, ctx, gi, primary, true, false, nil)
	if len(primary.text.FlavourText) > 0 && ctx.ShowFlavourText {
		t.addSeparator()
		for _, line := range primary.text.FlavourText {
			t.addLine(colorUnique + line)
		}
	}
	return true
}

//pob:block:src/Classes/PassiveSpec.lua:1034-1048
func (m *MiscCalculator) SplitPersonalityPath() []int64 {
	env := m.base
	specs := env.Build.Tree.Specs
	idx := env.Build.Tree.ActiveSpec - 1
	if idx < 0 || idx >= len(specs) {
		return nil
	}
	sockets, byID := overrideSockets(env, specs[idx].Sockets, itemTextByID(env))
	tree := specTree()
	on := map[int64]bool{}
	for _, socket := range sockets {
		text, ok := byID[socket.ItemID]
		if !ok {
			continue
		}
		if _, ok := jewelIncEffectFromClassStart(peekItemCached(env.Cache, text)); !ok {
			continue
		}
		for id := range shortestPathToClassStart(env, tree, strconv.Itoa(socket.NodeID)) {
			if n, err := strconv.ParseInt(id, 10, 64); err == nil {
				on[n] = true
			}
		}
	}
	out := make([]int64, 0, len(on))
	for id := range on {
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

//pob:block:src/Classes/PassiveTreeView.lua:1494-1497,src/Classes/PassiveSpec.lua:1598-1606
func (m *MiscCalculator) SocketDistanceToClassStart(nodeID int) int {
	if m.base == nil || m.base.Spec == nil {
		return 0
	}
	return m.base.Spec.DistanceToClassStart[strconv.Itoa(nodeID)]
}
