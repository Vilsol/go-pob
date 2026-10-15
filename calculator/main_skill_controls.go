package calculator

import (
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data"
	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
)

type MinionOption struct {
	Label     string
	MinionID  string
	ItemSetID int
	Tooltip   []string
}

type MainSkillControls struct {
	SkillSet    int
	SocketGroup int
	GemIndex    int

	Skills          []string
	SkillEnabled    bool
	MainActiveSkill int

	PartShown bool
	Parts     []string
	SkillPart int

	StageShown bool
	StageCount string

	MineShown bool
	MineCount string

	MinionShown        bool
	MinionEnabled      bool
	MinionLibraryShown bool
	Minions            []MinionOption
	Minion             int
	MinionUses         []string

	MinionSkillShown   bool
	MinionSkillEnabled bool
	MinionSkills       []string
	MinionSkill        int

	SkillDisabled      bool
	DisableReason      string
	InfoMessage        string
	InfoMessage2       string
	MinionInfoMessage  string
	MinionInfoMessage2 string
}

func displayGemIndexes(group *pob.Skill, list []*ActiveSkill) []int {
	out := make([]int, len(list))
	next := 0
	for i, activeSkill := range list {
		out[i] = -1
		if activeSkill.ActiveEffect == nil || activeSkill.ActiveEffect.SrcInstance == nil {
			continue
		}
		src := *activeSkill.ActiveEffect.SrcInstance
		for j := next; j < len(group.Gems); j++ {
			if group.Gems[j] == src {
				out[i] = j
				next = j
				break
			}
		}
	}
	return out
}

func explodeSourceName(source *ExplodeSource) string {
	if source.ModSource != "" {
		return strings.TrimPrefix(source.ModSource, "Item:-1:")
	}
	node := resolveTreeNode(data.TreeVersions[data.LatestTreeVersion].Tree(), source.NodeID)
	if node.Name != nil {
		return *node.Name
	}
	return source.NodeID
}

func (env *Environment) MainSkillControls() *MainSkillControls {
	return env.skillSelectControls(false)
}

func (env *Environment) CalcsSkillControls() *MainSkillControls {
	return env.skillSelectControls(true)
}

//pob:src/Modules/Build.lua:1618-1716
func (env *Environment) skillSelectControls(calcs bool) *MainSkillControls {
	if env == nil || env.Build == nil {
		return nil
	}
	skillSet := activeSkillSetIndex(env.Build.Skills)
	if skillSet < 0 || len(env.Build.Skills.SkillSets[skillSet].Skills) == 0 {
		return nil
	}
	groups := env.Build.Skills.SkillSets[skillSet].Skills
	if env.MainSocketGroup < 0 || env.MainSocketGroup >= len(groups) {
		return nil
	}
	mainSocketGroup := &groups[env.MainSocketGroup]
	displaySkillList, _ := pob.DisplaySkillListOf(mainSocketGroup).([]*ActiveSkill)
	mainActiveSkill := mainSocketGroup.MainActiveSkill
	if calcs {
		displaySkillList, _ = pob.DisplaySkillListCalcsOf(mainSocketGroup).([]*ActiveSkill)
		mainActiveSkill = mainSocketGroup.MainActiveSkillCalcs
	}

	if mainActiveSkill == 0 {
		mainActiveSkill = 1
	}
	controls := &MainSkillControls{
		SkillSet:        skillSet,
		SocketGroup:     env.MainSocketGroup,
		GemIndex:        -1,
		Skills:          make([]string, 0, len(displaySkillList)),
		SkillEnabled:    len(displaySkillList) > 1,
		MainActiveSkill: mainActiveSkill,
	}
	//pob:block:src/Modules/Build.lua:2112-2115
	if !calcs && env.Player != nil && env.Player.MainSkill != nil && env.Player.MainSkill.SkillFlags[SkillFlagDisable] {
		controls.SkillDisabled = true
		controls.DisableReason = env.Player.MainSkill.DisableReason
	}
	//pob:block:src/Modules/Build.lua:2081-2107
	if !calcs && env.Player != nil && env.Player.MainSkill != nil {
		controls.InfoMessage, controls.InfoMessage2 = env.Player.MainSkill.infoMessage, env.Player.MainSkill.infoMessage2
	}
	if !calcs && env.Minion != nil && env.Minion.Actor != nil && env.Minion.MainSkill != nil {
		controls.MinionInfoMessage, controls.MinionInfoMessage2 = env.Minion.MainSkill.infoMessage, env.Minion.MainSkill.infoMessage2
	}
	for _, activeSkill := range displaySkillList {
		if source := activeSkill.ActiveEffect.ExplodeSource; source != nil {
			controls.Skills = append(controls.Skills, "From "+explodeSourceName(source))
			continue
		}
		controls.Skills = append(controls.Skills, activeSkill.ActiveEffect.GrantedEffect.Name())
	}

	if len(displaySkillList) == 0 || mainActiveSkill < 1 || mainActiveSkill > len(displaySkillList) {
		return controls
	}
	activeSkill := displaySkillList[mainActiveSkill-1]
	activeEffect := activeSkill.ActiveEffect
	if activeEffect == nil {
		return controls
	}
	controls.GemIndex = displayGemIndexes(mainSocketGroup, displaySkillList)[mainActiveSkill-1]
	src := activeEffect.SrcInstance
	if src == nil {
		src = &pob.Gem{}
	}
	if calcs {
		src = calcsSuffixed(src)
	}

	stagesMax := activeSkill.SkillData.StagesMax
	parts := activeEffect.GrantedEffect.Parts
	if len(parts) > 1 {
		controls.PartShown = true
		for _, part := range parts {
			controls.Parts = append(controls.Parts, part.Name)
		}
		controls.SkillPart = src.SkillPart
		if controls.SkillPart == 0 {
			controls.SkillPart = 1
		}
		if controls.SkillPart <= len(parts) && parts[controls.SkillPart-1].Flags["stages"] {
			controls.StageShown = true
			controls.StageCount = stageBuf(src.SkillStageCount, stagesMax)
		}
	}
	if activeSkill.SkillFlags[SkillFlagMine] {
		controls.MineShown = true
		if src.SkillMineCount != nil {
			controls.MineCount = strconv.Itoa(*src.SkillMineCount)
		}
	}
	if activeSkill.SkillFlags[SkillFlagMultiStage] && len(parts) <= 1 {
		controls.StageShown = true
		controls.StageCount = stageBuf(src.SkillStageCount, stagesMax)
	}

	grantedMinionList, hasGrantedMinionList := raw2.SkillMinionLists[activeEffect.GrantedEffect.ID()]
	if !activeSkill.SkillFlags[SkillFlagDisable] && (hasGrantedMinionList || len(activeSkill.MinionList) > 0) {
		if raw2.SkillMinionHasItemSet[activeEffect.GrantedEffect.ID()] {
			for _, slot := range orderedItemSlots {
				if raw2.SkillMinionUses[activeEffect.GrantedEffect.ID()][slot.name] {
					controls.MinionUses = append(controls.MinionUses, slot.name)
				}
			}
			selected := src.SkillMinionItemSet
			if selected == 0 {
				selected = 1
			}
			for _, itemSet := range env.Build.Items.ItemSets {
				id, _ := strconv.Atoi(itemSet.ID)
				label := "Default Item Set"
				if itemSet.Title != nil {
					label = *itemSet.Title
				}
				//pob:block:src/Modules/Build.lua:551-558
				tooltip := append(ItemSetTooltip(env.Build, itemSet), "", colorTip+"Tip: You can drag items from the Items tab onto this dropdown to equip them onto the minion.")
				controls.Minions = append(controls.Minions, MinionOption{Label: label, ItemSetID: id, Tooltip: tooltip})
				if id == selected {
					controls.Minion = len(controls.Minions)
				}
			}
		} else {
			controls.MinionLibraryShown = hasGrantedMinionList && len(grantedMinionList) == 0
			for _, minionID := range activeSkill.MinionList {
				label := minionID
				if minion := raw2.Minions[minionID]; minion != nil {
					label = minion.Name
				}
				controls.Minions = append(controls.Minions, MinionOption{Label: label, MinionID: minionID})
				if minionID == src.SkillMinion {
					controls.Minion = len(controls.Minions)
				}
			}
		}
		if controls.Minion == 0 {
			controls.Minion = 1
		}
		controls.MinionEnabled = len(controls.Minions) > 1
		controls.MinionShown = true
		if activeSkill.Minion != nil {
			for _, minionSkill := range activeSkill.Minion.ActiveSkillList {
				controls.MinionSkills = append(controls.MinionSkills, minionSkill.ActiveEffect.GrantedEffect.Name())
			}
			controls.MinionSkill = src.SkillMinionSkill
			if controls.MinionSkill == 0 {
				controls.MinionSkill = 1
			}
			controls.MinionSkillShown = true
			controls.MinionSkillEnabled = len(controls.MinionSkills) > 1
		} else {
			controls.Minions = append(controls.Minions, MinionOption{Label: "<No spectres in build>"})
		}
	}

	return controls
}

func stageBuf(skillStageCount int, stagesMax float64) string {
	switch {
	case skillStageCount != 0:
		return strconv.Itoa(skillStageCount)
	case stagesMax != 0:
		return strconv.FormatFloat(stagesMax, 'f', -1, 64)
	}
	return "1"
}

//pob:src/Classes/PassiveSpec.lua:846-866
func (env *Environment) CountAllocNodes() (int, int, int, int) {
	used, ascUsed, secondaryAscUsed, sockets := 0, 0, 0, 0
	if env == nil || env.Build == nil {
		return used, ascUsed, secondaryAscUsed, sockets
	}
	secondaryAscendNameMap := make(map[string]bool)
	for _, ascendClass := range data.TreeVersions[data.LatestTreeVersion].Tree().AlternateAscendancies {
		secondaryAscendNameMap[string(ascendClass.ID)] = true
	}
	seen := make(map[int64]bool, len(env.Build.Build.PassiveNodes))
	for _, id := range env.Build.Build.PassiveNodes {
		if seen[id] {
			continue
		}
		seen[id] = true
		node, ok := env.AllocatedNodes[strconv.FormatInt(id, 10)]
		if !ok || (node.Name == nil && node.Skill == nil) {
			continue
		}
		//pob:block:src/Classes/PassiveSpec.lua:294,src/Classes/PassiveSpec.lua:1425-1452
		if node.IsMastery != nil && *node.IsMastery && !masteryEffectSelected(env.Spec, id, node) {
			continue
		}
		if node.ClassStartIndex != nil || (node.IsAscendancyStart != nil && *node.IsAscendancyStart) {
			continue
		}
		if node.AscendancyName != nil && *node.AscendancyName != "" {
			if node.IsMultipleChoiceOption == nil || !*node.IsMultipleChoiceOption {
				ascUsed++
				if secondaryAscendNameMap[*node.AscendancyName] {
					secondaryAscUsed++
				}
			}
		} else {
			used++
		}
		if node.IsJewelSocket != nil && *node.IsJewelSocket {
			sockets++
		}
	}
	return used, ascUsed, secondaryAscUsed, sockets
}

func masteryEffectSelected(spec *PassiveSpec, id int64, node data.Node) bool {
	effectID, ok := spec.MasterySelections[id]
	if !ok {
		return false
	}
	for _, effect := range node.MasteryEffects {
		if effect.Effect == effectID {
			return true
		}
	}
	return false
}

func calcsSuffixed(src *pob.Gem) *pob.Gem {
	g := *src
	g.SkillPart = src.SkillPartCalcs
	g.SkillStageCount = src.SkillStageCountCalcs
	g.SkillMineCount = src.SkillMineCountCalcs
	g.SkillMinion = ""
	if src.SkillMinionCalcs != nil {
		g.SkillMinion = *src.SkillMinionCalcs
	}
	g.SkillMinionItemSet = 0
	if src.SkillMinionItemSetCalcs != nil {
		g.SkillMinionItemSet = *src.SkillMinionItemSetCalcs
	}
	g.SkillMinionSkill = 0
	if src.SkillMinionSkillCalcs != nil {
		g.SkillMinionSkill = *src.SkillMinionSkillCalcs
	}
	return &g
}
