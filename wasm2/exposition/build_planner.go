package exposition

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/calculator2"
	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/pob2"
)

// BuildPlannerSet is a row of the Import/Export tab's Tree, Skill and Item dropdowns.
type BuildPlannerSet struct {
	Label string
	Key   int
}

// BuildPlannerSets is the three dropdowns' rows in order.
type BuildPlannerSets struct {
	Trees     []BuildPlannerSet
	SkillSets []BuildPlannerSet
	ItemSets  []BuildPlannerSet
}

// BuildPlannerMeta is what the Import/Export tab's fields give BuildTable.
type BuildPlannerMeta struct {
	Name                 string
	Author               string
	Description          string
	UseGeneratedItemText bool
}

// BuildPlannerFile is a .build file: its name and its JSON.
type BuildPlannerFile struct {
	FileName string
	JSON     string
}

type buildPlannerPassive struct {
	ID             string `json:"id"`
	WeaponSet      int    `json:"weapon_set"`
	AdditionalText string `json:"additional_text,omitempty"`
}

type buildPlannerSlot struct {
	InventoryID    string `json:"inventory_id"`
	SlotX          int    `json:"slot_x"`
	AdditionalText string `json:"additional_text"`
}

type buildPlannerTable struct {
	Name           string                          `json:"name"`
	Ascendancy     string                          `json:"ascendancy,omitempty"`
	Author         string                          `json:"author"`
	Description    string                          `json:"description"`
	Passives       []buildPlannerPassive           `json:"passives"`
	Skills         []calculator2.BuildPlannerSkill `json:"skills"`
	InventorySlots []buildPlannerSlot              `json:"inventory_slots"`
}

// buildFileInventorySlotMap is Data/InventorySlots.lua, in its order.
//
//pob2:src/Data/InventorySlots.lua:4-26
var buildFileInventorySlotMap = []struct {
	slot  string
	id    string
	slotX int
}{
	{"Weapon 1", "Weapon1", 0},
	{"Weapon 2", "Offhand1", 0},
	{"Helmet", "Helm1", 0},
	{"Amulet", "Amulet1", 0},
	{"Ring 1", "Ring1", 0},
	{"Ring 2", "Ring2", 0},
	{"Gloves", "Gloves1", 0},
	{"Boots", "Boots1", 0},
	{"Belt", "Belt1", 0},
	{"Flask 1", "Flask1", 0},
	{"Flask 2", "Flask1", 1},
	{"Charm 1", "Flask1", 2},
	{"Charm 2", "Flask1", 3},
	{"Charm 3", "Flask1", 4},
	{"Weapon 1 Swap", "Weapon2", 0},
	{"Weapon 2 Swap", "Offhand2", 0},
	{"Trinket", "Trinket1", 0},
	{"Ring 3", "Ring3", 0},
	{"Weapon3", "Weapon3", 0},
	{"Offhand3", "Offhand3", 0},
	{"Body Armour", "BodyArmour1", 0},
}

var (
	unsafeFilenameRe = regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f\x7f]`)
	colorCodeRe      = regexp.MustCompile(`\^x[0-9A-Fa-f]{6}`)
	braceRe          = regexp.MustCompile(`[{}]`)
)

//pob2:src/Modules/BuildExportPoE2.lua:21-25
func safeFilename(name string) string {
	if name == "" {
		name = "Unnamed"
	}
	return unsafeFilenameRe.ReplaceAllString(name, "-")
}

//pob2:src/Modules/BuildExportPoE2.lua:27-29
func treeVersionSuffix(treeVersion string) string {
	return " [" + strings.ReplaceAll(treeVersion, "_", ".") + "]"
}

// BuildPlannerFileName is BuildPath's file name; the browser saves into the folder the viewer picks.
//
//pob2:src/Modules/BuildExportPoE2.lua:38-41
func BuildPlannerFileName(buildName, treeVersion string) string {
	return safeFilename(buildName) + treeVersionSuffix(treeVersion) + ".build"
}

//pob2:src/Data/Global.lua:113-119
func colorCodeToMarkupColour(code string) string {
	code = strings.TrimPrefix(code, "^x")
	r, _ := strconv.ParseUint(code[0:2], 16, 8)
	g, _ := strconv.ParseUint(code[2:4], 16, 8)
	b, _ := strconv.ParseUint(code[4:6], 16, 8)
	return fmt.Sprintf("<rgb(%d, %d, %d)>", r, g, b)
}

// BuildPlannerSets is the Tree, Skill and Item dropdowns' rows.
//
//pob2:src/Classes/ImportTab.lua:494-531
func (c *Calculator) BuildPlannerSets() BuildPlannerSets {
	var out BuildPlannerSets
	for i := range c.build.Tree.Specs {
		out.Trees = append(out.Trees, BuildPlannerSet{Label: specTitle(&c.build.Tree.Specs[i]), Key: i + 1})
	}
	sets, _ := calculator2.SkillSetList(c.build)
	for _, s := range sets {
		out.SkillSets = append(out.SkillSets, BuildPlannerSet{Label: s.Title, Key: s.ID})
	}
	for _, s := range c.build.Items.ItemSets {
		label := "Default"
		if s.Title != nil {
			label = *s.Title
		}
		out.ItemSets = append(out.ItemSets, BuildPlannerSet{Label: label, Key: s.ID})
	}
	return out
}

//pob2:src/Modules/BuildExportPoE2.lua:62-69
func buildAscendancy(spec *tree2.PassiveSpec) string {
	if spec == nil || spec.CurClass == nil {
		return ""
	}
	if spec.CurAscendClassID < 0 || spec.CurAscendClassID >= len(spec.CurClass.Classes) {
		return ""
	}
	if asc := spec.CurClass.Classes[spec.CurAscendClassID]; asc != nil {
		return asc.InternalID
	}
	return ""
}

// buildPassives reads the spec's node notes from the saved spec, as PassiveSpec:Load does.
//
//pob2:src/Modules/BuildExportPoE2.lua:71-89
//pob2:block:src/Classes/PassiveSpec.lua:161-171
func buildPassives(spec *tree2.PassiveSpec, saved *pob2.Spec) []buildPlannerPassive {
	out := []buildPlannerPassive{}
	if spec == nil {
		return out
	}
	notes := map[int]string{}
	if saved != nil && saved.NodeNotes != nil {
		for _, n := range saved.NodeNotes.Notes {
			text := n.Text
			if text == "" && n.Attr != nil {
				text = *n.Attr
			}
			if text != "" {
				notes[n.NodeID] = text
			}
		}
	}
	ids := make([]int, 0, len(spec.AllocNodes))
	for id := range spec.AllocNodes {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, nodeID := range ids {
		node := spec.AllocNodes[nodeID]
		if node.Node == nil || node.StringID == "" || nodeID >= 65536 {
			continue
		}
		out = append(out, buildPlannerPassive{ID: node.StringID, WeaponSet: node.AllocMode, AdditionalText: notes[nodeID]})
	}
	return out
}

// plannerModLine is itemLib.formatModLine(modLine, nil, true): an unsupported line takes the colour a supported one
// would.
//
//pob2:src/Modules/ItemTools.lua:438-458
func plannerModLine(item *calculator2.Item, modLine *calculator2.ModLine) (string, bool) {
	line := item.RangedLine(modLine)
	if isZeroValueLine(line) {
		return "", false
	}
	if modLine.Disabled {
		return colorDisabled + line, true
	}
	var colorCode string
	switch {
	case modLine.Fractured:
		colorCode = colorFractured
	case modLine.Crafted:
		colorCode = colorCrafted
	case modLine.Enchant:
		colorCode = colorEnchanted
	case modLine.Mutated:
		colorCode = colorMutated
	case modLine.Custom && !modLine.Desecrated:
		colorCode = colorCustom
	default:
		colorCode = colorMagic
	}
	return colorCode + line, true
}

//pob2:src/Modules/BuildExportPoE2.lua:194-228
func buildPlannerItemText(item *calculator2.Item) string {
	parts := []string{fmt.Sprintf("%s{<b>{%s}}\n", colorCodeToMarkupColour(rarityColors[item.Rarity]), strings.ReplaceAll(item.Name, ", ", "\n"))}
	appendPlain := func(modLines []*calculator2.ModLine) {
		for _, modLine := range modLines {
			variantCount := 0
			if modLine.Line != "" {
				variantCount = item.GetModLineVariantCount(modLine)
			}
			if variantCount <= 0 {
				continue
			}
			formatted, ok := plannerModLine(item, modLine)
			if !ok {
				continue
			}
			colorCode := colorCodeRe.FindString(formatted)
			formatted = colorCodeRe.ReplaceAllString(formatted, "")
			line := fmt.Sprintf("%s{%s}", colorCodeToMarkupColour(colorCode), braceRe.ReplaceAllString(formatted, ""))
			for range variantCount {
				parts = append(parts, line)
			}
		}
	}
	appendPlain(item.EnchantModLines)
	appendPlain(item.RuneModLines)
	appendPlain(item.ImplicitModLines)
	appendPlain(item.ExplicitModLines)
	return strings.Join(parts, "\n")
}

//pob2:src/Modules/BuildExportPoE2.lua:230-252
func buildItems(itemsTab *calculator2.ItemsTab, itemSet *pob2.ItemSet, useGeneratedItemText bool) []buildPlannerSlot {
	out := []buildPlannerSlot{}
	if itemsTab == nil || itemSet == nil {
		return out
	}
	for _, mapping := range buildFileInventorySlotMap {
		var slotEntry *pob2.Slot
		for i := range itemSet.Slots {
			if itemSet.Slots[i].Name == mapping.slot {
				slotEntry = &itemSet.Slots[i]
			}
		}
		var item *calculator2.Item
		note := ""
		if slotEntry != nil {
			if slotEntry.ItemID != 0 {
				item = itemsTab.Items[slotEntry.ItemID]
			}
			if slotEntry.Note != nil {
				note = *slotEntry.Note
			}
		}
		if note == "" && (item == nil || !useGeneratedItemText) {
			continue
		}
		text := note
		if text == "" {
			text = buildPlannerItemText(item)
		}
		out = append(out, buildPlannerSlot{InventoryID: mapping.id, SlotX: mapping.slotX, AdditionalText: text})
	}
	return out
}

// BuildPlannerExport is BuildExportPoE2.Export for one tree, skill set and item set: the .build file the in-game
// build planner loads, and the name BuildPath gives it.
//
//pob2:src/Modules/BuildExportPoE2.lua:254-262
//pob2:src/Modules/BuildExportPoE2.lua:286-313
func (c *Calculator) BuildPlannerExport(meta BuildPlannerMeta, specIndex, skillSetID, itemSetID int) (BuildPlannerFile, error) {
	if c.env == nil {
		return BuildPlannerFile{}, notCalculatedError{}
	}
	var spec *tree2.PassiveSpec
	var saved *pob2.Spec
	treeVersion := ""
	if specIndex >= 1 && specIndex <= len(c.build.Tree.Specs) {
		s, err := c.specAt(specIndex)
		if err != nil {
			return BuildPlannerFile{}, err
		}
		spec, saved = s, &c.build.Tree.Specs[specIndex-1]
		treeVersion = spec.TreeVersion
	}
	var itemSet *pob2.ItemSet
	for i := range c.build.Items.ItemSets {
		if c.build.Items.ItemSets[i].ID == itemSetID {
			itemSet = &c.build.Items.ItemSets[i]
		}
	}
	itemsTab, err := c.itemsTab()
	if err != nil {
		return BuildPlannerFile{}, err
	}
	root := buildPlannerTable{
		Name:           meta.Name,
		Ascendancy:     buildAscendancy(spec),
		Author:         meta.Author,
		Description:    meta.Description,
		Passives:       buildPassives(spec, saved),
		Skills:         calculator2.BuildPlannerSkills(c.env, skillSetID),
		InventorySlots: buildItems(itemsTab, itemSet, meta.UseGeneratedItemText),
	}
	if root.Name == "" {
		root.Name = "Unnamed"
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(root); err != nil {
		return BuildPlannerFile{}, fmt.Errorf("JSON encode failed: %w", err)
	}
	return BuildPlannerFile{FileName: BuildPlannerFileName(meta.Name, treeVersion), JSON: strings.TrimSuffix(buf.String(), "\n")}, nil
}

// BuildPlannerLoadout is a GetLoadouts entry: a loadout's plain name, the name its file takes, and its sets; 0 is nil.
type BuildPlannerLoadout struct {
	Name       string
	FileName   string
	SpecIndex  int
	SkillSetID int
	ItemSetID  int
}

// BuildPlannerExportAll is what WriteAllLoadouts writes: a file per loadout, or the errors that stopped it.
type BuildPlannerExportAll struct {
	Files  []BuildPlannerFile
	Errors []string
}

var (
	loadoutLinkRe       = regexp.MustCompile(`\{([0-9A-Za-z,]+)\}`)
	loadoutMatchRe      = regexp.MustCompile(`\{([0-9A-Za-z]+)\}`)
	loadoutNameSuffixRe = regexp.MustCompile(` \{.*\}$`)
	loadoutExtRe        = regexp.MustCompile(`^(.*)(\.[^./\\]*)$`)
	loadoutVersionRe    = regexp.MustCompile(` \[\d+\.\d+\]$`)
)

type loadoutSet struct {
	id    int
	title string
}

type loadoutLink struct {
	setID   int
	setName string
	linkID  string
}

func titleOrDefault(title *string) string {
	if title == nil {
		return "Default"
	}
	return *title
}

// loadoutSets is the item, skill and config sets in their order lists; a build without config sets has the one the
// config tab makes.
func (c *Calculator) loadoutSets() ([]loadoutSet, []loadoutSet, []loadoutSet) {
	items := make([]loadoutSet, 0, len(c.build.Items.ItemSets))
	for _, s := range c.build.Items.ItemSets {
		items = append(items, loadoutSet{id: s.ID, title: titleOrDefault(s.Title)})
	}
	sets, _ := calculator2.SkillSetList(c.build)
	skills := make([]loadoutSet, 0, len(sets))
	configs := make([]loadoutSet, 0, len(c.build.Config.ConfigSets))
	for _, s := range sets {
		skills = append(skills, loadoutSet{id: s.ID, title: s.Title})
	}
	for _, s := range c.build.Config.ConfigSets {
		configs = append(configs, loadoutSet{id: s.ID, title: titleOrDefault(s.Title)})
	}
	if len(configs) == 0 {
		configs = []loadoutSet{{id: 1, title: "Default"}}
	}
	return items, skills, configs
}

// splitLoadoutLink is a title's {ids} and the title without them.
func splitLoadoutLink(title string) ([]string, string, bool) {
	m := loadoutLinkRe.FindStringSubmatch(title)
	if m == nil {
		return nil, "", false
	}
	name := strings.TrimSpace(strings.ReplaceAll(title, "{"+m[1]+"}", ""))
	if name == "" {
		name = "Default"
	}
	return strings.Split(m[1], ","), name, true
}

//pob2:block:src/Modules/Build.lua:686-709
func identifyLoadoutLinks(sets []loadoutSet) (map[string]bool, map[string]loadoutLink) {
	titles := map[string]bool{}
	links := map[string]loadoutLink{}
	for _, s := range sets {
		if ids, name, ok := splitLoadoutLink(s.title); ok {
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

// loadoutState is what SyncLoadouts leaves: the loadout dropdown's names and the special links GetLoadoutByName reads.
type loadoutState struct {
	names                        []string
	treeLinks                    map[string]loadoutLink
	itemLinks, skillLinks        map[string]loadoutLink
	items, skills, configs       []loadoutSet
	oneItem, oneSkill, oneConfig bool
}

//pob2:block:src/Modules/Build.lua:637-732
func (c *Calculator) syncLoadouts() loadoutState {
	st := loadoutState{treeLinks: map[string]loadoutLink{}}
	st.items, st.skills, st.configs = c.loadoutSets()
	st.oneItem, st.oneSkill, st.oneConfig = len(st.items) == 1, len(st.skills) == 1, len(st.configs) == 1
	var sortedTreeLinks []loadoutLink
	var treeList []string
	for i := range c.build.Tree.Specs {
		spec := &c.build.Tree.Specs[i]
		title := specTitle(spec)
		if ids, name, ok := splitLoadoutLink(title); ok {
			for _, linkID := range ids {
				if linkID == "" {
					continue
				}
				l := loadoutLink{setID: i + 1, setName: name, linkID: linkID}
				st.treeLinks[linkID] = l
				sortedTreeLinks = append(sortedTreeLinks, l)
			}
			continue
		}
		if v := specTreeVersion(spec); v != TreeVersion {
			title = "[" + treeVersionDisplay(v) + "] " + title
		}
		treeList = append(treeList, title)
	}
	itemList, itemLinks := identifyLoadoutLinks(st.items)
	skillList, skillLinks := identifyLoadoutLinks(st.skills)
	configList, configLinks := identifyLoadoutLinks(st.configs)
	st.itemLinks, st.skillLinks = itemLinks, skillLinks
	for _, tree := range treeList {
		if (st.oneItem || itemList[tree]) && (st.oneSkill || skillList[tree]) && (st.oneConfig || configList[tree]) {
			st.names = append(st.names, tree)
		}
	}
	for _, tree := range sortedTreeLinks {
		_, item := itemLinks[tree.linkID]
		_, skill := skillLinks[tree.linkID]
		_, config := configLinks[tree.linkID]
		if (st.oneItem || item) && (st.oneSkill || skill) && (st.oneConfig || config) {
			st.names = append(st.names, tree.setName+" {"+tree.linkID+"}")
		}
	}
	return st
}

//pob2:block:src/Modules/Build.lua:902-914
func findLoadoutSetID(sets []loadoutSet, value string, links map[string]loadoutLink) int {
	for _, s := range sets {
		if value == s.title {
			return s.id
		}
		if m := loadoutMatchRe.FindStringSubmatch(value); m != nil {
			return links[m[1]].setID
		}
	}
	return 0
}

// getLoadoutByName is GetLoadoutByName's spec, skill set and item set; ok is false for its nil.
//
//pob2:block:src/Modules/Build.lua:899-954
func (c *Calculator) getLoadoutByName(st loadoutState, name string) (int, int, int, bool) {
	var specID, skillSetID, itemSetID int
	for i, label := range c.TreeSpecs().Labels {
		if name == label {
			specID = i + 1
			break
		}
		if m := loadoutMatchRe.FindStringSubmatch(name); m != nil {
			specID = st.treeLinks[m[1]].setID
			break
		}
	}
	if st.oneItem {
		itemSetID = st.items[0].id
	} else {
		itemSetID = findLoadoutSetID(st.items, name, st.itemLinks)
	}
	if st.oneSkill {
		skillSetID = st.skills[0].id
	} else {
		skillSetID = findLoadoutSetID(st.skills, name, st.skillLinks)
	}
	configID := st.configs[0].id
	return specID, skillSetID, itemSetID, specID != 0 || itemSetID != 0 || skillSetID != 0 || configID != 0
}

// BuildPlannerLoadouts is every loadout the build's sets make.
//
//pob2:src/Modules/BuildExportPoE2.lua:264-282
func (c *Calculator) BuildPlannerLoadouts() []BuildPlannerLoadout {
	st := c.syncLoadouts()
	out := make([]BuildPlannerLoadout, 0, len(st.names))
	for _, displayName := range st.names {
		l := BuildPlannerLoadout{Name: loadoutNameSuffixRe.ReplaceAllString(displayName, ""), FileName: displayName}
		if specID, skillSetID, itemSetID, ok := c.getLoadoutByName(st, displayName); ok {
			l.SpecIndex, l.SkillSetID, l.ItemSetID = specID, skillSetID, itemSetID
		}
		out = append(out, l)
	}
	return out
}

// buildPlannerLoadoutPath inserts a loadout name before the extension: "My Build.build" -> "My Build - Leveling.build".
//
//pob2:src/Modules/BuildExportPoE2.lua:54-60
func buildPlannerLoadoutPath(basePath, loadoutName, treeVersion string) string {
	suffix := " - " + safeFilename(loadoutName)
	stem, ext := basePath, ""
	if m := loadoutExtRe.FindStringSubmatch(basePath); m != nil {
		stem, ext = m[1], m[2]
	}
	versionSuffix := ""
	if treeVersion != "" {
		versionSuffix = treeVersionSuffix(treeVersion)
	} else {
		versionSuffix = loadoutVersionRe.FindString(stem)
	}
	return loadoutVersionRe.ReplaceAllString(stem, "") + suffix + versionSuffix + ext
}

// BuildPlannerExportAll is Export All Loadouts: a file per loadout, named after the file the Tree dropdown's tree
// would take, or the error that stopped it. The browser saves the files; an error writing one is its own.
//
//pob2:block:src/Classes/ImportTab.lua:440-472
//pob2:src/Modules/BuildExportPoE2.lua:330-366
func (c *Calculator) BuildPlannerExportAll(meta BuildPlannerMeta, specIndex int) (BuildPlannerExportAll, error) {
	out := BuildPlannerExportAll{Files: []BuildPlannerFile{}}
	treeVersion := ""
	if specIndex >= 1 && specIndex <= len(c.build.Tree.Specs) {
		treeVersion = specTreeVersion(&c.build.Tree.Specs[specIndex-1])
	}
	basePath := BuildPlannerFileName(meta.Name, treeVersion)
	loadouts := c.BuildPlannerLoadouts()
	if len(loadouts) == 0 {
		out.Errors = []string{"This build has no passive trees to export."}
		return out, nil
	}
	paths := make([]string, len(loadouts))
	pathNames := map[string]string{}
	for i, loadout := range loadouts {
		version := ""
		if loadout.SpecIndex >= 1 && loadout.SpecIndex <= len(c.build.Tree.Specs) {
			version = specTreeVersion(&c.build.Tree.Specs[loadout.SpecIndex-1])
		}
		fileName := loadout.FileName
		if fileName == "" {
			fileName = loadout.Name
		}
		path := buildPlannerLoadoutPath(basePath, fileName, version)
		key := strings.ToLower(path)
		if other, ok := pathNames[key]; ok {
			out.Errors = []string{fmt.Sprintf("Loadouts '%s' and '%s' export to the same file: %s", other, loadout.Name, path)}
			return out, nil
		}
		pathNames[key] = loadout.Name
		paths[i] = path
	}
	for i, loadout := range loadouts {
		loadoutMeta := BuildPlannerMeta{
			Name:                 fmt.Sprintf("%s - %s", meta.Name, loadout.Name),
			Author:               meta.Author,
			Description:          meta.Description,
			UseGeneratedItemText: meta.UseGeneratedItemText,
		}
		file, err := c.BuildPlannerExport(loadoutMeta, loadout.SpecIndex, loadout.SkillSetID, loadout.ItemSetID)
		if err != nil {
			return BuildPlannerExportAll{}, err
		}
		file.FileName = paths[i]
		out.Files = append(out.Files, file)
	}
	return out, nil
}
