package pob

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strconv"
	"strings"
)

func (b *PathOfBuilding) WithMainSocketGroup(mainSocketGroup int) *PathOfBuilding {
	out := *b
	out.Build.MainSocketGroup = mainSocketGroup
	return &out
}

func (b *PathOfBuilding) WithMainSkill(socketGroup int, mainActiveSkill int) *PathOfBuilding {
	out := *b
	out.Build.MainSocketGroup = socketGroup

	if len(b.Skills.SkillSets) == 0 {
		return &out
	}
	idx := b.ActiveSkillSetIndex()

	sets := make([]SkillSet, len(b.Skills.SkillSets))
	copy(sets, b.Skills.SkillSets)

	skills := make([]Skill, len(sets[idx].Skills))
	copy(skills, sets[idx].Skills)
	if socketGroup >= 1 && socketGroup <= len(skills) {
		skills[socketGroup-1].MainActiveSkill = mainActiveSkill
		skills[socketGroup-1].MainActiveSkillCalcs = mainActiveSkill
	}
	sets[idx].Skills = skills
	out.Skills.SkillSets = sets

	return &out
}

func (b *PathOfBuilding) SetMainSocketGroup(mainSocketGroup int) {
	b.Build.MainSocketGroup = mainSocketGroup
}

//pob:src/Modules/Build.lua:486-491
func (b *PathOfBuilding) SetMainActiveSkill(skillSet int, socketGroup int, mainActiveSkill int) {
	b.Skills.SkillSets[skillSet].Skills[socketGroup].MainActiveSkill = mainActiveSkill
}

//pob:src/Modules/Build.lua:492-498
func (b *PathOfBuilding) SetGemSkillPart(skillSet int, socketGroup int, gem int, skillPart int) {
	b.Skills.SkillSets[skillSet].Skills[socketGroup].Gems[gem].SkillPart = skillPart
}

//pob:src/Modules/Build.lua:504-510
func (b *PathOfBuilding) SetGemSkillStageCount(skillSet int, socketGroup int, gem int, stageCount int) {
	b.Skills.SkillSets[skillSet].Skills[socketGroup].Gems[gem].SkillStageCount = stageCount
}

//pob:src/Modules/Build.lua:516-522
func (b *PathOfBuilding) SetGemSkillMineCount(skillSet int, socketGroup int, gem int, mineCount int) {
	var count *int
	if mineCount >= 0 {
		count = &mineCount
	}
	b.Skills.SkillSets[skillSet].Skills[socketGroup].Gems[gem].SkillMineCount = count
}

//pob:src/Modules/Build.lua:523-533
func (b *PathOfBuilding) SetGemSkillMinion(skillSet int, socketGroup int, gem int, minionID string) {
	b.Skills.SkillSets[skillSet].Skills[socketGroup].Gems[gem].SkillMinion = minionID
}

//pob:src/Modules/Build.lua:523-533
func (b *PathOfBuilding) SetGemSkillMinionItemSet(skillSet int, socketGroup int, gem int, itemSetID int) {
	b.Skills.SkillSets[skillSet].Skills[socketGroup].Gems[gem].SkillMinionItemSet = itemSetID
}

//pob:src/Modules/Build.lua:555-561
func (b *PathOfBuilding) SetGemSkillMinionSkill(skillSet int, socketGroup int, gem int, minionSkill int) {
	b.Skills.SkillSets[skillSet].Skills[socketGroup].Gems[gem].SkillMinionSkill = minionSkill
}

func (b *PathOfBuilding) SetSkillGroupName(skillSet int, socketGroup int, label string) {
	b.Skills.SkillSets[skillSet].Skills[socketGroup].Label = label
}

func (b *PathOfBuilding) SetNotes(notes string) {
	b.Notes = notes
}

const (
	RichNotesFormat  = "tiptap-json"
	RichNotesVersion = 1
)

const NotesEdgeSpace = " \t\n\v\f\r"

func NotesHash(notes string) string {
	sum := sha256.Sum256([]byte(notes))
	return hex.EncodeToString(sum[:])
}

func (b *PathOfBuilding) SetRichNotes(plain string, doc string) {
	plain = strings.Trim(plain, NotesEdgeSpace)
	b.Notes = plain
	b.RichNotes = &RichNotes{
		Format:    RichNotesFormat,
		V:         RichNotesVersion,
		PlainHash: NotesHash(plain),
		Doc:       doc,
	}
}

func (b *PathOfBuilding) RichNotesStatus() string {
	r := b.RichNotes
	if r == nil || r.Format != RichNotesFormat || r.V != RichNotesVersion || r.Doc == "" {
		return "none"
	}
	if r.PlainHash != NotesHash(b.Notes) {
		return "mismatch"
	}
	return "match"
}

func (b *PathOfBuilding) RichNotesDoc() string {
	if b.RichNotes == nil {
		return ""
	}
	return b.RichNotes.Doc
}

func (b *PathOfBuilding) SetSocketGroupGems(skillSet int, socketGroup int, gems []Gem) {
	b.Skills.SkillSets[skillSet].Skills[socketGroup].Gems = gems
}

func (b *PathOfBuilding) SetSortGemsByDPS(enabled bool) {
	b.Skills.SortGemsByDPS = enabled
}

func (b *PathOfBuilding) SetSortGemsByDPSField(field string) {
	b.Skills.SortGemsByDPSField = field
}

func (b *PathOfBuilding) SetMatchGemLevelToCharacterLevel(enabled bool) {
	b.Skills.MatchGemLevelToCharacterLevel = enabled
}

func (b *PathOfBuilding) SetDefaultGemLevel(gemLevel int) {
	level := strconv.Itoa(gemLevel)
	b.Skills.DefaultGemLevel = &level
}

func (b *PathOfBuilding) SetDefaultGemLevelMode(mode string) {
	b.Skills.DefaultGemLevel = &mode
	b.Skills.MatchGemLevelToCharacterLevel = false
}

//pob:block:src/Classes/SkillsTab.lua:148-150
func (b *PathOfBuilding) SetShowLegacyGems(enabled bool) {
	b.Skills.ShowLegacyGems = enabled
}

func (b *PathOfBuilding) SetDefaultGemQuality(gemQuality int) {
	b.Skills.DefaultGemQuality = &gemQuality
}

func (b *PathOfBuilding) SetShowSupportGemTypes(gemTypes string) {
	b.Skills.ShowSupportGemTypes = gemTypes
}

func (b *PathOfBuilding) SetShowAltQualityGems(enabled bool) {
	b.Skills.ShowAltQualityGems = enabled
}

func (b *PathOfBuilding) SetConfigOption(value Input) {
	for i, input := range b.Config.Inputs {
		if input.Name == value.Name {
			b.Config.Inputs[i] = value
			return
		}
	}
	b.Config.Inputs = append(b.Config.Inputs, value)
}

func (b *PathOfBuilding) RemoveConfigOption(name string) {
	toRemove := -1
	for i, input := range b.Config.Inputs {
		if input.Name == name {
			toRemove = i
			break
		}
	}
	if toRemove >= 0 {
		b.Config.Inputs = append(b.Config.Inputs[:toRemove], b.Config.Inputs[toRemove+1:]...)
	}
}

func (b *PathOfBuilding) ActiveConfigSetIndex() int {
	return b.Config.activeConfigSetIndex()
}

func (c *Config) activeConfigSetIndex() int {
	want := c.ActiveConfigSet
	if want == 0 {
		want = 1
	}
	for i := range c.ConfigSets {
		if c.ConfigSets[i].ID == want {
			return i
		}
	}
	for i := range c.ConfigSets {
		if c.ConfigSets[i].ID == 1 {
			return i
		}
	}
	return 0
}

//pob:src/Classes/ConfigTab.lua:1278-1303
func (b *PathOfBuilding) SetActiveConfigSet(configSetID int) {
	c := &b.Config
	if len(c.ConfigSets) == 0 {
		c.ConfigSets = []ConfigSet{{ID: 1}}
	}

	next := -1
	for i := range c.ConfigSets {
		if c.ConfigSets[i].ID == configSetID {
			next = i
			break
		}
	}
	if next < 0 {
		next = 0
	}

	if cur := c.activeConfigSetIndex(); cur != next {
		c.ConfigSets[cur].Inputs = c.Inputs
		c.ConfigSets[cur].Placeholders = c.Placeholders
		c.ConfigSets[cur].CustomModifierBlocks = c.CustomModifierBlocks
		c.Inputs = slices.Clone(c.ConfigSets[next].Inputs)
		c.Placeholders = slices.Clone(c.ConfigSets[next].Placeholders)
		c.CustomModifierBlocks = slices.Clone(c.ConfigSets[next].CustomModifierBlocks)
	}
	c.ActiveConfigSet = c.ConfigSets[next].ID
}

func (b *PathOfBuilding) ResetConfigOptions() {
	b.Config.Inputs = slices.DeleteFunc(b.Config.Inputs, func(input Input) bool {
		return input.Name != "customMods"
	})
}

//pob:block:src/Classes/ConfigTab.lua:873-884
func (b *PathOfBuilding) GetCustomModifierBlocks() []CustomModifierBlock {
	blocks := b.Config.CustomModifierBlocks
	legacyText := b.GetStringOption("customMods")
	if legacyText != "" && (len(blocks) == 0 || (len(blocks) == 1 && blocks[0].Text == "")) {
		return []CustomModifierBlock{defaultCustomModifierBlock(legacyText)}
	} else if len(blocks) == 0 {
		return []CustomModifierBlock{defaultCustomModifierBlock("")}
	}
	return slices.Clone(blocks)
}

//pob:block:src/Classes/ConfigTab.lua:24-28, src/Classes/ConfigTab.lua:881-883
func (b *PathOfBuilding) SetCustomModifierBlocks(blocks []CustomModifierBlock) {
	if len(blocks) == 0 {
		blocks = []CustomModifierBlock{defaultCustomModifierBlock("")}
	}
	b.Config.CustomModifierBlocks = slices.Clone(blocks)
	b.RemoveConfigOption("customMods")
}

func defaultCustomModifierBlock(text string) CustomModifierBlock {
	title, enabled := "Default", "true"
	return CustomModifierBlock{Title: &title, Enabled: &enabled, Text: text}
}

func (b *PathOfBuilding) GetStringOption(name string) string {
	for _, input := range b.Config.Inputs {
		if input.Name == name {
			if input.String == nil {
				return ""
			}

			return *input.String
		}
	}
	return ""
}

func (b *PathOfBuilding) GetNumberOption(name string) float64 {
	value, _ := b.GetNumberOptionOK(name)
	return value
}

func (b *PathOfBuilding) GetNumberOptionOK(name string) (float64, bool) {
	for _, input := range b.Config.Inputs {
		if input.Name == name {
			if input.Number == nil {
				return 0, false
			}

			return *input.Number, true
		}
	}
	return 0, false
}

func (b *PathOfBuilding) GetNumberPlaceholder(name string) float64 {
	for _, input := range b.Config.Placeholders {
		if input.Name == name {
			if input.Number == nil {
				return 0
			}

			return *input.Number
		}
	}
	return 0
}

func (b *PathOfBuilding) GetBooleanOption(name string) bool {
	for _, input := range b.Config.Inputs {
		if input.Name == name {
			if input.Boolean == nil {
				return false
			}

			return *input.Boolean
		}
	}
	return false
}

func (b *PathOfBuilding) AddNewSocketGroup() {
	set := &b.Skills.SkillSets[b.ActiveSkillSetIndex()]
	set.Skills = append(set.Skills, Skill{
		Enabled: true,
	})
}

func (b *PathOfBuilding) DeleteSocketGroup(index int) {
	set := &b.Skills.SkillSets[b.ActiveSkillSetIndex()]
	set.Skills = append(set.Skills[:index], set.Skills[index+1:]...)
}

func (b *PathOfBuilding) DeleteAllSocketGroups() {
	b.Skills.SkillSets[b.ActiveSkillSetIndex()].Skills = make([]Skill, 0)
}

//pob:block:src/Classes/ListControl.lua:428-436, src/Classes/SkillListControl.lua:123-142
func (b *PathOfBuilding) MoveSocketGroup(from int, to int) {
	if len(b.Skills.SkillSets) == 0 {
		return
	}
	set := &b.Skills.SkillSets[b.ActiveSkillSetIndex()]
	if from == to || from < 0 || to < 0 || from >= len(set.Skills) || to >= len(set.Skills) {
		return
	}

	group := set.Skills[from]
	set.Skills = slices.Delete(set.Skills, from, from+1)
	set.Skills = slices.Insert(set.Skills, to, group)

	selIndex, selDragIndex := from+1, to+1
	follow := func(index int) int {
		if index == selIndex {
			return selDragIndex
		} else if index > selIndex && index <= selDragIndex {
			return index - 1
		} else if index < selIndex && index >= selDragIndex {
			return index + 1
		}
		return index
	}

	b.Build.MainSocketGroup = follow(b.Build.MainSocketGroup)

	skillNumber := 1
	at := -1
	for i, input := range b.Calcs.Inputs {
		if input.Name == "skill_number" && input.Number != nil {
			skillNumber = int(*input.Number)
			at = i
			break
		}
	}
	if moved := follow(skillNumber); moved != skillNumber {
		number := float64(moved)
		if at >= 0 {
			b.Calcs.Inputs[at].Number = &number
		} else {
			b.Calcs.Inputs = append(b.Calcs.Inputs, Input{Name: "skill_number", Number: &number})
		}
	}
}

//pob:block:src/Classes/SkillsTab.lua:1462-1464
func (b *PathOfBuilding) ActiveSkillSetIndex() int {
	for i, set := range b.Skills.SkillSets {
		if set.ID == b.Skills.ActiveSkillSet {
			return i
		}
	}
	return 0
}

//pob:src/Classes/SkillsTab.lua:1451-1475
func (b *PathOfBuilding) SetActiveSkillSet(skillSetID int) {
	if len(b.Skills.SkillSets) == 0 {
		b.Skills.SkillSets = []SkillSet{{ID: 1}}
	}

	found := false
	for _, set := range b.Skills.SkillSets {
		if set.ID == skillSetID {
			found = true
			break
		}
	}
	if !found {
		skillSetID = b.Skills.SkillSets[0].ID
	}

	b.Skills.ActiveSkillSet = skillSetID
}

//pob:block:src/Classes/PassiveSpec.lua:572,src/Classes/PassiveSpec.lua:584
func (b *PathOfBuilding) SetClass(className ClassName) {
	b.Build.ClassName = className
	if spec := b.activeSpec(); spec != nil {
		if id, ok := ClassIDByName(className); ok {
			spec.ClassID = id
			spec.AscendClassID = 0
		}
	}
}

func (b *PathOfBuilding) SetAscendancy(ascendancyName AscendancyName) {
	b.Build.AscendClassName = ascendancyName
	if spec := b.activeSpec(); spec != nil {
		spec.AscendClassID = AscendancyIndex(spec.ClassID, ascendancyName)
	}
}

func (b *PathOfBuilding) SetLevel(level int) {
	b.Build.Level = level
}

func (b *PathOfBuilding) syncActiveSpecNodes() {
	idx := b.Tree.ActiveSpec - 1
	if idx < 0 || idx >= len(b.Tree.Specs) {
		return
	}

	parts := make([]string, 0, len(b.Build.PassiveNodes))
	for _, node := range b.Build.PassiveNodes {
		parts = append(parts, strconv.FormatInt(node, 10))
	}
	b.Tree.Specs[idx].NodesAttr = strings.Join(parts, ",")
}

func (b *PathOfBuilding) AllocateNodes(nodeIds []int64) {
	b.Build.PassiveNodes = append(b.Build.PassiveNodes, nodeIds...)
	b.syncActiveSpecNodes()
}

func (b *PathOfBuilding) DeallocateNodes(nodeIds []int64) {
	activeNodesSet := make(map[int64]bool, len(b.Build.PassiveNodes))
	for _, node := range b.Build.PassiveNodes {
		activeNodesSet[node] = true
	}

	for _, nodeId := range nodeIds {
		delete(activeNodesSet, nodeId)
	}

	activeNodes := make([]int64, 0, len(activeNodesSet))
	for nodeId := range activeNodesSet {
		activeNodes = append(activeNodes, nodeId)
	}

	slices.Sort(activeNodes)

	b.Build.PassiveNodes = activeNodes
	b.syncActiveSpecNodes()
	b.dropMasterySelections(nodeIds)
}

var itemTextEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

func (b *PathOfBuilding) AddItem(text string) int {
	used := make(map[int]bool, len(b.Items.Items))
	for _, it := range b.Items.Items {
		used[it.ID] = true
	}
	id := 1
	for used[id] {
		id++
	}
	b.Items.Items = append(b.Items.Items, Item{ID: id, Text: itemTextEscaper.Replace(text)})
	return id
}

func (b *PathOfBuilding) SetItemText(id int, text string) bool {
	for i := range b.Items.Items {
		if b.Items.Items[i].ID == id {
			b.Items.Items[i] = Item{ID: id, Text: itemTextEscaper.Replace(text)}
			return true
		}
	}
	return false
}

func (b *PathOfBuilding) activeItemSet() *ItemSet {
	if len(b.Items.ItemSets) == 0 {
		b.Items.ActiveItemSet = 1
		b.Items.ItemSets = []ItemSet{{ID: "1"}}
	}
	for i := range b.Items.ItemSets {
		if id, err := strconv.Atoi(b.Items.ItemSets[i].ID); err == nil && id == b.Items.ActiveItemSet {
			return &b.Items.ItemSets[i]
		}
	}
	return &b.Items.ItemSets[0]
}

func (s *ItemSet) slot(slotName string) *Slot {
	for i := range s.Slots {
		if s.Slots[i].Name == slotName {
			return &s.Slots[i]
		}
	}
	s.Slots = append(s.Slots, Slot{Name: slotName})
	return &s.Slots[len(s.Slots)-1]
}

//pob:src/Classes/ItemsTab.lua:1584-1610
func (b *PathOfBuilding) SetActiveItemSet(itemSetID int) {
	b.activeItemSet()
	found := false
	for _, set := range b.Items.ItemSets {
		if id, err := strconv.Atoi(set.ID); err == nil && id == itemSetID {
			found = true
			break
		}
	}
	if !found {
		itemSetID, _ = strconv.Atoi(b.Items.ItemSets[0].ID)
	}
	b.Items.ActiveItemSet = itemSetID
}

func (b *PathOfBuilding) SetSlotItem(slotName string, itemID int) {
	b.activeItemSet().slot(slotName).ItemID = itemID
}

func (b *PathOfBuilding) SetFlaskActive(slotName string, active bool) {
	b.activeItemSet().slot(slotName).Active = active
}

func (b *PathOfBuilding) SetCalcsInputString(name string, value string) {
	b.setCalcsInput(Input{Name: name, String: &value})
}

func (b *PathOfBuilding) SetCalcsInputNumber(name string, value float64) {
	b.setCalcsInput(Input{Name: name, Number: &value})
}

func (b *PathOfBuilding) SetCalcsInputBool(name string, value bool) {
	b.setCalcsInput(Input{Name: name, Boolean: &value})
}

func (b *PathOfBuilding) CalcsInputBool(name string) bool {
	for _, in := range b.Calcs.Inputs {
		if in.Name == name && in.Boolean != nil {
			return *in.Boolean
		}
	}
	return false
}

func (b *PathOfBuilding) setCalcsInput(in Input) {
	for i := range b.Calcs.Inputs {
		if b.Calcs.Inputs[i].Name == in.Name {
			b.Calcs.Inputs[i] = in
			return
		}
	}
	b.Calcs.Inputs = append(b.Calcs.Inputs, in)
}

func (b *PathOfBuilding) CalcsInputString(name, def string) string {
	for _, in := range b.Calcs.Inputs {
		if in.Name == name && in.String != nil {
			return *in.String
		}
	}
	return def
}

func (b *PathOfBuilding) CalcsInputNumber(name string, def float64) float64 {
	for _, in := range b.Calcs.Inputs {
		if in.Name == name && in.Number != nil {
			return *in.Number
		}
	}
	return def
}

func (b *PathOfBuilding) SetCharacterLevelAutoMode(auto bool) {
	b.Build.CharacterLevelAutoMode = auto
}

func (b *PathOfBuilding) ApplyNewBuildDefaults(defaultCharLevel, defaultGemQuality int) {
	b.Build.Level = min(max(defaultCharLevel, 1), 100)
	b.Build.CharacterLevelAutoMode = b.Build.Level == 1
	quality := min(defaultGemQuality, 23)
	b.Skills.DefaultGemQuality = &quality
}
