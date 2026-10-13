package exposition

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

//crystalline:exports
func ImportPassiveTreeExports(r bind.Registry) {
	r.Func(ImportPassiveTreeAndJewels, bind.AsPromise())
	r.Func(ImportAccountPassiveTree, bind.AsPromise())
	r.Func(ImportAccountItems, bind.AsPromise())
}

type characterPassives struct {
	Hashes         []int64          `json:"hashes"`
	HashesEx       []int64          `json:"hashes_ex"`
	MasteryEffects map[string]int64 `json:"mastery_effects"`
	JewelData      map[string]struct {
		Subgraph *struct {
			Groups map[string]struct {
				Proxy string   `json:"proxy"`
				Nodes []string `json:"nodes"`
			} `json:"groups"`
			Nodes map[string]struct {
				Group      string `json:"group"`
				OrbitIndex int64  `json:"orbitIndex"`
				IsMastery  bool   `json:"isMastery"`
			} `json:"nodes"`
		} `json:"subgraph"`
	} `json:"jewel_data"`
	SkillOverrides map[string]struct {
		Name string `json:"name"`
	} `json:"skill_overrides"`
	AlternateAscendancy json.RawMessage  `json:"alternate_ascendancy"`
	BanditChoice        *string          `json:"bandit_choice"`
	PantheonMajor       *string          `json:"pantheon_major"`
	PantheonMinor       *string          `json:"pantheon_minor"`
	Items               []importItemData `json:"items"`
}

//pob:block:src/Modules/Build.lua:711-747
var importActs = []struct{ level, questPoints int }{
	{1, 0}, {12, 2}, {22, 4}, {32, 6}, {40, 7}, {44, 9}, {50, 12}, {54, 15}, {60, 18}, {64, 20}, {67, 23},
}

//pob:block:src/Modules/Build.lua:880-891,src/Modules/Build.lua:910
func estimatedAct(pointsUsed int, extra int) int {
	act, level := 0, 1
	for {
		act++
		actExtra := 0
		if act > 2 {
			actExtra = extra
		}
		level = min(max(pointsUsed+1-importActs[act-1].questPoints-actExtra, importActs[act-1].level), 100)
		if act == 11 || level <= importActs[act].level {
			break
		}
	}
	if level < 90 && act <= 10 {
		return act
	}
	return 0
}

//pob:block:src/Classes/ImportTab.lua:1168-1181
func isAscendancyInTree(tree *data.Tree, className string) bool {
	for _, class := range tree.Classes {
		if string(class.Name) == className {
			return true
		}
		for _, ascend := range class.Ascendancies {
			if string(ascend.Name) == className {
				return true
			}
		}
	}
	return false
}

//pob:block:src/Classes/PassiveSpec.lua:252-258
func importClassIDs(tree *data.Tree, className string) (pob.ClassID, pob.AscendancyID, bool) {
	for classID, class := range tree.Classes {
		if string(class.Name) == className {
			return pob.ClassID(classID), 0, true
		}
	}
	for classID, class := range tree.Classes {
		for i, ascend := range class.Ascendancies {
			if string(ascend.Name) == className || string(ascend.ID) == className {
				return pob.ClassID(classID), pob.AscendancyID(i + 1), true
			}
		}
	}
	return 0, 0, false
}

//pob:block:src/Classes/ImportTab.lua:1183-1195,src/Classes/PassiveTree.lua:185-200
func importSecondaryAscendancy(tree *data.Tree, raw json.RawMessage) int {
	if len(raw) == 0 || string(raw) == "null" {
		return 0
	}
	var name string
	if json.Unmarshal(raw, &name) == nil {
		for i, alt := range tree.AlternateAscendancies {
			if string(alt.ID) == name && !legacyAlternateAscendancy[string(alt.ID)] {
				return i + 1
			}
		}
		return 0
	}
	var id int
	_ = json.Unmarshal(raw, &id)
	return id
}

//pob:block:src/Classes/PassiveTree.lua:111-116
var legacyAlternateAscendancy = map[string]bool{"Warden": true, "Warlock": true, "Primalist": true}

//pob:block:src/Classes/ImportTab.lua:1228-1234,src/Modules/ConfigOptions.lua:136-150
func setSelByVal(build *pob.PathOfBuilding, name string, value string) {
	valid := value == "None"
	switch name {
	case "bandit":
		valid = valid || value == "Oak" || value == "Kraityn" || value == "Alira"
	case "pantheonMajorGod", "pantheonMinorGod":
		god, ok := data.Pantheons[value]
		valid = valid || ok && god.IsMajorGod == (name == "pantheonMajorGod")
	}
	if valid {
		build.SetConfigOption(pob.Input{Name: name, String: &value})
	}
}

//pob:block:src/Classes/PassiveSpec.lua:1747-1760,src/Classes/PassiveSpec.lua:1774-1786,src/Classes/PassiveSpec.lua:1944-1982,src/Classes/PassiveSpec.lua:2255-2260
func importedSubgraphNodes(build *pob.PathOfBuilding, tree *data.Tree, charPassives characterPassives) []int64 {
	type importedNode struct {
		id         string
		group      string
		orbitIndex int64
		isMastery  bool
	}
	importedGroups := map[string][]string{}
	proxyGroups := map[int64]string{}
	var importedNodes []importedNode
	for _, value := range charPassives.JewelData {
		if value.Subgraph == nil {
			continue
		}
		for groupID, groupData := range value.Subgraph.Groups {
			importedGroups[groupID] = groupData.Nodes
			if proxy, ok := tree.Nodes[groupData.Proxy]; ok && proxy.Group != nil {
				proxyGroups[*proxy.Group] = groupID
			}
		}
		for nodeID, nodeValue := range value.Subgraph.Nodes {
			importedNodes = append(importedNodes, importedNode{nodeID, nodeValue.Group, nodeValue.OrbitIndex, nodeValue.IsMastery})
		}
	}
	slices.SortFunc(importedNodes, func(a, b importedNode) int { return strings.Compare(a.id, b.id) })

	addToAllocatedSubgraphNodes := func(node calculator.TreeClusterNode) bool {
		proxyGroup, ok := proxyGroups[node.Group]
		if !ok {
			return false
		}
		for _, imported := range importedNodes {
			if proxyGroup == imported.group && node.OrbitIndex == imported.orbitIndex && !imported.isMastery {
				id, err := strconv.ParseInt(imported.id, 10, 64)
				return err == nil && slices.Contains(importedGroups[proxyGroup], imported.id) && slices.Contains(charPassives.HashesEx, id)
			}
		}
		return false
	}
	var out []int64
	for _, node := range calculator.TreeClusterNodes(build) {
		if addToAllocatedSubgraphNodes(node) {
			out = append(out, node.ID)
		}
	}
	return out
}

//pob:src/Classes/ImportTab.lua:1130-1252
func importPassiveTreeAndJewels(build *pob.PathOfBuilding, charData *characterData, charPassives characterPassives, deleteJewels bool) string {
	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	extra := int(calculator.NewCalculator(*build).BuildOutput(calculator.OutputModeMain).Player.Output["ExtraPoints"])
	if build.ActiveSpecIndex() < 0 {
		build.AddSpec(pob.Spec{Sockets: []pob.Socket{}, Overrides: []pob.Override{}})
		build.SetActiveSpec(1)
	}
	spec := &build.Tree.Specs[build.ActiveSpecIndex()]

	masteries := map[int64]int64{}
	for key, value := range charPassives.MasteryEffects {
		if id, err := strconv.ParseInt(key, 10, 64); err == nil {
			masteries[id] = value
		}
	}
	overrides := []pob.Override{}
	for nodeID, override := range charPassives.SkillOverrides {
		id, err := strconv.Atoi(nodeID)
		target, ok := data.TattooTargets[override.Name]
		if err != nil || !ok {
			continue
		}
		overrides = append(overrides, pob.Override{NodeID: id, DN: override.Name, Icon: &target.Icon, ActiveEffectImage: &target.ActiveEffectImage})
	}
	slices.SortFunc(overrides, func(a, b pob.Override) int { return a.NodeID - b.NodeID })

	if deleteJewels {
		for _, socket := range slices.Clone(spec.Sockets) {
			if socket.ItemID != 0 {
				build.DeleteItem(socket.ItemID)
			}
		}
	}
	t := &importTab{build: build}
	for _, itemData := range charData.Jewels {
		t.importItem(itemData, "", false, 0)
	}

	alternateAscendancyID := importSecondaryAscendancy(tree, charPassives.AlternateAscendancy)
	ruthlessSuffix := ""
	if strings.Contains(charData.League, "Ruthless") {
		ruthlessSuffix = "_ruthless"
	}
	phreciaSuffix := "_alternate"
	if isAscendancyInTree(tree, charData.Class) {
		phreciaSuffix = ""
	}

	//pob:block:src/Classes/PassiveSpec.lua:245-301
	two := 2
	classID, ascendClassID, _ := importClassIDs(tree, charData.Class)
	spec = &build.Tree.Specs[build.ActiveSpecIndex()]
	if treeVersion := pob.TreeVersion(string(data.LatestTreeVersion) + ruthlessSuffix + phreciaSuffix); treeVersion != spec.TreeVersion {
		spec.TreeVersion = treeVersion
		spec.Sockets = []pob.Socket{}
	}
	spec.ClusterHashFormatVersion = &two
	spec.ClassID = classID
	spec.AscendClassID = ascendClassID
	spec.SecondaryAscendClassID = &alternateAscendancyID
	spec.Overrides = overrides
	var selections []pob.MasterySelection
	for id, effect := range masteries {
		// ignore ggg codes from profile import
		if effect < 65536 {
			selections = append(selections, pob.MasterySelection{Node: id, Effect: effect})
		}
	}
	slices.SortFunc(selections, func(a, b pob.MasterySelection) int { return int(a.Node - b.Node) })
	spec.MasteryEffects = pob.FormatMasterySelections(selections)
	nodes := make([]string, 0, len(charPassives.Hashes))
	for _, id := range charPassives.Hashes {
		if node, ok := tree.Nodes[strconv.FormatInt(id, 10)]; ok && node.IsMastery != nil && *node.IsMastery {
			if !slices.ContainsFunc(selections, func(s pob.MasterySelection) bool { return s.Node == id }) {
				continue
			}
		}
		nodes = append(nodes, strconv.FormatInt(id, 10))
	}
	for _, id := range importedSubgraphNodes(build, tree, charPassives) {
		if key := strconv.FormatInt(id, 10); !slices.Contains(nodes, key) {
			nodes = append(nodes, key)
		}
	}
	spec.NodesAttr = strings.Join(nodes, ",")
	build.SetActiveSpec(build.ActiveSpecIndex() + 1)

	level := charData.Level
	if level == 0 {
		level = 100
	}
	build.SetLevel(level)
	build.Build.CharacterLevelAutoMode = false

	resistancePenaltyIndex := 3
	if act := estimatedAct(calculator.TreeSpecsPoints(build)[build.ActiveSpecIndex()].Used, extra); act != 0 {
		switch {
		case act < 5:
			resistancePenaltyIndex = 1
		case act > 5 && act < 11:
			resistancePenaltyIndex = 2
		}
	}
	penalty := []float64{0, -30, -60}[resistancePenaltyIndex-1]
	build.SetConfigOption(pob.Input{Name: "resistancePenalty", Number: &penalty})

	bandit := "None"
	if charPassives.BanditChoice != nil && *charPassives.BanditChoice != "Eramir" {
		bandit = *charPassives.BanditChoice
	}
	setSelByVal(build, "bandit", bandit)
	majorGod := "None"
	if charPassives.PantheonMajor != nil {
		majorGod = *charPassives.PantheonMajor
	}
	setSelByVal(build, "pantheonMajorGod", majorGod)
	minorGod := "None"
	if charPassives.PantheonMinor != nil {
		minorGod = *charPassives.PantheonMinor
	}
	setSelByVal(build, "pantheonMinorGod", minorGod)
	return "Passive tree and jewels successfully imported."
}

func ImportPassiveTreeAndJewels(build *pob.PathOfBuilding, characterJSON string, deleteJewels bool) (string, error) {
	if build == nil {
		return "", fmt.Errorf("no build")
	}
	charData, err := decodeCharacterJSON(characterJSON)
	if err != nil {
		return "", err
	}
	var passives characterPassives
	if err := json.Unmarshal(charData.Passives, &passives); err != nil {
		return "", fmt.Errorf("Invalid JSON format (decode error)")
	}
	return importPassiveTreeAndJewels(build, charData, passives, deleteJewels), nil
}

//pob:block:src/Classes/ImportTab.lua:829-847
func ImportAccountPassiveTree(build *pob.PathOfBuilding, characterJSON string, passivesJSON string, deleteJewels bool) (string, error) {
	if build == nil {
		return "", fmt.Errorf("no build")
	}
	if strings.TrimSpace(passivesJSON) == "false" {
		return "", fmt.Errorf("Failed to retrieve character data, try again.")
	}
	var charData characterData
	var passives characterPassives
	if json.Unmarshal([]byte(characterJSON), &charData) != nil || json.Unmarshal([]byte(passivesJSON), &passives) != nil {
		return "", fmt.Errorf("Error importing character data, try again (invalid JSON)")
	}
	keep := func(answer *string, name string) *string {
		if answer != nil {
			return answer
		}
		value := build.GetStringOption(name)
		if value == "" {
			value = "None"
		}
		return &value
	}
	passives.BanditChoice = keep(passives.BanditChoice, "bandit")
	passives.PantheonMajor = keep(passives.PantheonMajor, "pantheonMajorGod")
	passives.PantheonMinor = keep(passives.PantheonMinor, "pantheonMinorGod")
	charData.Jewels = passives.Items
	return importPassiveTreeAndJewels(build, &charData, passives, deleteJewels), nil
}

//pob:block:src/Classes/ImportTab.lua:867-883
func ImportAccountItems(build *pob.PathOfBuilding, characterJSON string, itemsJSON string, clearItems bool, clearSkills bool, ignoreWeaponSwap bool) (string, error) {
	if build == nil {
		return "", fmt.Errorf("no build")
	}
	if strings.TrimSpace(itemsJSON) == "false" {
		return "", fmt.Errorf("Failed to retrieve character data, try again.")
	}
	var charData characterData
	var answer struct {
		Items    []importItemData `json:"items"`
		Guardian []importItemData `json:"guardian"`
	}
	if json.Unmarshal([]byte(characterJSON), &charData) != nil || json.Unmarshal([]byte(itemsJSON), &answer) != nil {
		return "", fmt.Errorf("Error importing character data, try again (invalid JSON)")
	}
	charData.Equipment = answer.Items
	charData.Guardian = answer.Guardian
	return importItemsAndSkills(build, &charData, clearItems, clearSkills, ignoreWeaponSwap), nil
}
