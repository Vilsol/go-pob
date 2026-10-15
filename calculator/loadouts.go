package calculator

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

type LoadoutList struct {
	Loadouts []string
	Selected int
}

var (
	loadoutLinkRe  = regexp.MustCompile(`\{([0-9A-Za-z,]+)\}`)
	loadoutMatchRe = regexp.MustCompile(`\{([0-9A-Za-z]+)\}`)
)

type loadoutLink struct {
	setID   int
	setName string
	linkID  string
}

type loadoutSet struct {
	id    int
	title string
}

func orDefault(title *string) string {
	if title == nil {
		return "Default"
	}
	return *title
}

func loadoutItemSets(build *pob.PathOfBuilding) []loadoutSet {
	out := make([]loadoutSet, 0, len(build.Items.ItemSets))
	for _, s := range build.Items.ItemSets {
		id, _ := strconv.Atoi(s.ID)
		out = append(out, loadoutSet{id: id, title: orDefault(s.Title)})
	}
	return out
}

func loadoutSkillSets(build *pob.PathOfBuilding) []loadoutSet {
	out := make([]loadoutSet, 0, len(build.Skills.SkillSets))
	for _, s := range build.Skills.SkillSets {
		out = append(out, loadoutSet{id: s.ID, title: orDefault(s.Title)})
	}
	return out
}

func loadoutConfigSets(build *pob.PathOfBuilding) []loadoutSet {
	out := make([]loadoutSet, 0, len(build.Config.ConfigSets))
	for _, s := range build.Config.ConfigSets {
		out = append(out, loadoutSet{id: s.ID, title: orDefault(s.Title)})
	}
	return out
}

func splitLink(title string) ([]string, string, bool) {
	m := loadoutLinkRe.FindStringSubmatch(title)
	if m == nil {
		return nil, "", false
	}
	name := strings.TrimSpace(strings.Replace(title, "{"+m[1]+"}", "", 1))
	if name == "" {
		name = "Default"
	}
	return strings.Split(m[1], ","), name, true
}

//pob:block:src/Modules/Build.lua:816-840
func identifyLinks(sets []loadoutSet) (map[string]bool, map[string]loadoutLink) {
	titles := map[string]bool{}
	links := map[string]loadoutLink{}
	for _, s := range sets {
		if ids, name, ok := splitLink(s.title); ok {
			for _, id := range ids {
				if id != "" {
					links[id] = loadoutLink{setID: s.id, setName: name}
				}
			}
		} else {
			titles[s.title] = true
		}
	}
	return titles, links
}

type loadoutState struct {
	treeList                       []string
	treeLinks                      []loadoutLink
	treeLinkByID                   map[string]loadoutLink
	items, skills, configs         map[string]bool
	itemLinks, skillLinks, cfgLink map[string]loadoutLink
	oneItem, oneSkill, oneConfig   bool
}

func loadoutsOf(build *pob.PathOfBuilding) loadoutState {
	st := loadoutState{treeLinkByID: map[string]loadoutLink{}}
	for id, spec := range build.Tree.Specs {
		title := orDefault(spec.Title)
		if ids, name, ok := splitLink(title); ok {
			for _, linkID := range ids {
				if linkID == "" {
					continue
				}
				l := loadoutLink{setID: id + 1, setName: name, linkID: linkID}
				st.treeLinkByID[linkID] = l
				st.treeLinks = append(st.treeLinks, l)
			}
			continue
		}
		label := title
		if spec.TreeVersion != data.LatestTreeVersion {
			label = "[" + treeVersionDisplay(spec.TreeVersion) + "] " + title
		}
		st.treeList = append(st.treeList, label)
	}
	items, skills, configs := loadoutItemSets(build), loadoutSkillSets(build), loadoutConfigSets(build)
	st.oneItem, st.oneSkill, st.oneConfig = len(items) == 1, len(skills) == 1, len(configs) == 1
	st.items, st.itemLinks = identifyLinks(items)
	st.skills, st.skillLinks = identifyLinks(skills)
	st.configs, st.cfgLink = identifyLinks(configs)
	return st
}

func activeTitle(sets []loadoutSet, id int) string {
	for _, s := range sets {
		if s.id == id {
			return s.title
		}
	}
	return "Default"
}

//pob:src/Modules/Build.lua:771-896
func Loadouts(build *pob.PathOfBuilding) LoadoutList {
	st := loadoutsOf(build)
	out := LoadoutList{Loadouts: []string{}, Selected: -1}
	for _, tree := range st.treeList {
		if (st.oneItem || st.items[tree]) && (st.oneSkill || st.skills[tree]) && (st.oneConfig || st.configs[tree]) {
			out.Loadouts = append(out.Loadouts, tree)
		}
	}
	for _, t := range st.treeLinks {
		_, item := st.itemLinks[t.linkID]
		_, skill := st.skillLinks[t.linkID]
		_, cfg := st.cfgLink[t.linkID]
		if (st.oneItem || item) && (st.oneSkill || skill) && (st.oneConfig || cfg) {
			out.Loadouts = append(out.Loadouts, t.setName+" {"+t.linkID+"}")
		}
	}

	i := build.ActiveSpecIndex()
	if i < 0 {
		return out
	}
	treeName := orDefault(build.Tree.Specs[i].Title)
	for k, l := range out.Loadouts {
		if l != treeName {
			continue
		}
		linkMatch := treeName
		if m := loadoutMatchRe.FindStringSubmatch(treeName); m != nil {
			linkMatch = m[1]
		}
		skillMatch := st.oneSkill || strings.Contains(activeTitle(loadoutSkillSets(build), build.Skills.ActiveSkillSet), linkMatch)
		itemMatch := st.oneItem || strings.Contains(activeTitle(loadoutItemSets(build), build.Items.ActiveItemSet), linkMatch)
		configMatch := st.oneConfig || strings.Contains(activeTitle(loadoutConfigSets(build), build.Config.ActiveConfigSet), linkMatch)
		if skillMatch && itemMatch && configMatch {
			out.Selected = k
		}
		break
	}
	return out
}

//pob:block:src/Modules/Build.lua:359-373,src/Modules/Build.lua:391-398
func findLoadoutSet(sets []loadoutSet, links map[string]loadoutLink, value string) (int, bool) {
	if len(sets) == 1 {
		return sets[0].id, true
	}
	for _, s := range sets {
		if s.title == value {
			return s.id, true
		}
		if m := loadoutMatchRe.FindStringSubmatch(value); m != nil {
			l, ok := links[m[1]]
			return l.setID, ok
		}
	}
	return 0, false
}

//pob:block:src/Modules/Build.lua:375-418
func SelectLoadout(build *pob.PathOfBuilding, value string) bool {
	st := loadoutsOf(build)
	specID, found := 0, false
	for k, row := range TreeSpecs(build) {
		if row.Label == value {
			specID, found = k+1, true
			break
		}
		if m := loadoutMatchRe.FindStringSubmatch(value); m != nil {
			l, ok := st.treeLinkByID[m[1]]
			specID, found = l.setID, ok
			break
		}
	}
	itemID, okItem := findLoadoutSet(loadoutItemSets(build), st.itemLinks, value)
	skillID, okSkill := findLoadoutSet(loadoutSkillSets(build), st.skillLinks, value)
	configID, okConfig := findLoadoutSet(loadoutConfigSets(build), st.cfgLink, value)
	if !found || !okItem || !okSkill || !okConfig {
		return false
	}
	if specID != build.ActiveSpecIndex()+1 {
		build.SetActiveSpec(specID)
	}
	if itemID != build.Items.ActiveItemSet {
		build.SetActiveItemSet(itemID)
	}
	if skillID != build.Skills.ActiveSkillSet {
		build.SetActiveSkillSet(skillID)
	}
	if configID != build.Config.ActiveConfigSet {
		build.SetActiveConfigSet(configID)
	}
	return true
}

func freeSetID(ids []int) int {
	used := map[int]bool{}
	for _, id := range ids {
		used[id] = true
	}
	id := len(ids) + 1
	for used[id] {
		id++
	}
	return id
}

//pob:block:src/Modules/Build.lua:326-347
func NewLoadout(build *pob.PathOfBuilding, name string) {
	if len(build.Items.ItemSets) == 0 {
		build.Items.ItemSets = []pob.ItemSet{{ID: "1"}}
	}
	if len(build.Skills.SkillSets) == 0 {
		build.Skills.SkillSets = []pob.SkillSet{{ID: 1, Skills: []pob.Skill{}}}
	}
	if len(build.Config.ConfigSets) == 0 {
		build.Config.ConfigSets = []pob.ConfigSet{{ID: 1, Inputs: []pob.Input{}, Placeholders: []pob.Input{}, CustomModifierBlocks: []pob.CustomModifierBlock{}}}
	}
	NewTreeSpec(build, name)

	ids := make([]int, 0, len(build.Items.ItemSets))
	for _, s := range loadoutItemSets(build) {
		ids = append(ids, s.id)
	}
	itemTitle := name
	build.Items.ItemSets = append(build.Items.ItemSets, pob.ItemSet{ID: strconv.Itoa(freeSetID(ids)), Title: &itemTitle})

	ids = ids[:0]
	for _, s := range build.Skills.SkillSets {
		ids = append(ids, s.ID)
	}
	skillTitle := name
	build.Skills.SkillSets = append(build.Skills.SkillSets, pob.SkillSet{ID: freeSetID(ids), Title: &skillTitle, Skills: []pob.Skill{}})

	ids = ids[:0]
	for _, s := range build.Config.ConfigSets {
		ids = append(ids, s.ID)
	}
	configTitle := name
	build.Config.ConfigSets = append(build.Config.ConfigSets, pob.ConfigSet{ID: freeSetID(ids), Title: &configTitle,
		Inputs: []pob.Input{}, Placeholders: []pob.Input{}, CustomModifierBlocks: []pob.CustomModifierBlock{}})
}
