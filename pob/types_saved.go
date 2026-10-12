package pob

import (
	"encoding/xml"
	"fmt"
)

type PathOfBuilding struct {
	Build    Build    `xml:"Build"`
	Tree     Tree     `xml:"Tree"`
	Calcs    Calcs    `xml:"Calcs"`
	Notes    string   `xml:"Notes"`
	Items    Items    `xml:"Items"`
	Skills   Skills   `xml:"Skills"`
	TreeView TreeView `xml:"TreeView"`
	Config   Config   `xml:"Config"`
	Party    Party    `xml:"Party"`
	Import   *Import  `xml:"Import,omitempty"`

	RichNotes *RichNotes `xml:"RichNotes,omitempty"`
}

type RichNotes struct {
	Format    string `xml:"format,attr"`
	V         int    `xml:"v,attr"`
	PlainHash string `xml:"plainHash,attr"`
	Doc       string `xml:",chardata"`
}

type BuildViewMode string

const (
	ViewModeTree   = BuildViewMode("TREE")
	ViewModeItems  = BuildViewMode("ITEMS")
	ViewModeImport = BuildViewMode("IMPORT")
	ViewModeNotes  = BuildViewMode("NOTES")
	ViewModeConfig = BuildViewMode("CONFIG")
	ViewModeSkills = BuildViewMode("SKILLS")
	ViewModeCalc   = BuildViewMode("CALC")
)

type Build struct {
	PantheonMinorGod       string         `xml:"pantheonMinorGod,attr"`
	PantheonMajorGod       string         `xml:"pantheonMajorGod,attr"`
	Bandit                 string         `xml:"bandit,attr"`
	ViewMode               BuildViewMode  `xml:"viewMode,attr"`
	ClassName              ClassName      `xml:"className,attr"`
	AscendClassName        AscendancyName `xml:"ascendClassName,attr"`
	Level                  int            `xml:"level,attr"`
	CharacterLevelAutoMode bool           `xml:"characterLevelAutoMode,attr"`
	MainSocketGroup        int            `xml:"mainSocketGroup,attr"`
	TargetVersion          GameVersion    `xml:"targetVersion,attr"`

	PassiveNodes []int64 `xml:"-"`

	PlayerStats   []PlayerStat   `xml:"PlayerStat" crystalline:"not_nil"`
	FullDPSSkills []FullDPSSkill `xml:"FullDPSSkill"`
	MinionStats   []PlayerStat   `xml:"MinionStat"`

	Spectres []Spectre `xml:"Spectre" crystalline:"not_nil"`

	TimelessData *TimelessData `xml:"TimelessData"`
}

type Spectre struct {
	ID string `xml:"id,attr"`
}

type PlayerStat struct {
	Value float64 `xml:"value,attr"`
	Stat  string  `xml:"stat,attr"`
}

type FullDPSSkill struct {
	Stat      string `xml:"stat,attr"`
	Value     string `xml:"value,attr"`
	SkillPart string `xml:"skillPart,attr"`
	Source    string `xml:"source,attr"`
}

type Tree struct {
	ActiveSpec int `xml:"activeSpec,attr"`

	Specs []Spec `xml:"Spec" crystalline:"not_nil"`
}

type Calcs struct {
	Inputs   []Input   `xml:"Input" crystalline:"not_nil"`
	Sections []Section `xml:"Section" crystalline:"not_nil"`
}

type Items struct {
	ActiveItemSet       int   `xml:"activeItemSet,attr"`
	UseSecondWeaponSet  *bool `xml:"useSecondWeaponSet,attr,omitempty"`
	ShowStatDifferences *bool `xml:"showStatDifferences,attr,omitempty"`

	ItemSets []ItemSet `xml:"ItemSet" crystalline:"not_nil"`
	Items    []Item    `xml:"Item"`
}

type Item struct {
	ID          int  `xml:"id,attr"`
	Variant     *int `xml:"variant,attr,omitempty"`
	VariantAlt  *int `xml:"variantAlt,attr,omitempty"`
	VariantAlt2 *int `xml:"variantAlt2,attr,omitempty"`
	VariantAlt3 *int `xml:"variantAlt3,attr,omitempty"`
	VariantAlt4 *int `xml:"variantAlt4,attr,omitempty"`
	VariantAlt5 *int `xml:"variantAlt5,attr,omitempty"`

	Text string `xml:",innerxml"`
}

type Skills struct {
	SortGemsByDPSField            string  `xml:"sortGemsByDPSField,attr"`
	ShowSupportGemTypes           string  `xml:"showSupportGemTypes,attr"`
	DefaultGemLevel               *string `xml:"defaultGemLevel,attr,omitempty"`
	MatchGemLevelToCharacterLevel bool    `xml:"matchGemLevelToCharacterLevel,attr,omitempty"`
	ShowAltQualityGems            bool    `xml:"showAltQualityGems,attr,omitempty"`
	DefaultGemQuality             *int    `xml:"defaultGemQuality,attr,omitempty"`
	ActiveSkillSet                int     `xml:"activeSkillSet,attr"`
	SortGemsByDPS                 bool    `xml:"sortGemsByDPS,attr"`
	ShowLegacyGems                bool    `xml:"showLegacyGems,attr"`

	SkillSets []SkillSet `xml:"SkillSet" crystalline:"not_nil"`
}

func (s *Skills) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type plainSkills Skills
	var raw struct {
		plainSkills
		LooseSkills []Skill `xml:"Skill"`
	}

	if err := d.DecodeElement(&raw, &start); err != nil {
		return fmt.Errorf("failed to parse skills: %w", err)
	}

	*s = Skills(raw.plainSkills)

	if len(raw.LooseSkills) == 0 {
		return nil
	}

	target := -1
	for i := range s.SkillSets {
		if s.SkillSets[i].ID == 1 {
			target = i
			break
		}
	}

	if target < 0 {
		s.SkillSets = append([]SkillSet{{ID: 1, Skills: raw.LooseSkills}}, s.SkillSets...)
	} else {
		s.SkillSets[target].Skills = append(s.SkillSets[target].Skills, raw.LooseSkills...)
	}

	if s.ActiveSkillSet < 1 {
		s.ActiveSkillSet = 1
	}

	return nil
}

type TreeView struct {
	ZoomLevel           int     `xml:"zoomLevel,attr"`
	ZoomX               float64 `xml:"zoomX,attr"`
	ZoomY               float64 `xml:"zoomY,attr"`
	SearchStr           string  `xml:"searchStr,attr"`
	ShowHeatMap         *bool   `xml:"showHeatMap,attr,omitempty"`
	ShowStatDifferences bool    `xml:"showStatDifferences,attr"`
}

type Party struct {
	Destination      string        `xml:"destination,attr,omitempty"`
	Append           string        `xml:"append,attr,omitempty"`
	ShowAdvanceTools string        `xml:"ShowAdvanceTools,attr,omitempty"`
	ImportedBuffs    []PartyBuffs  `xml:"ImportedBuffs" crystalline:"not_nil"`
	ExportedBuffs    []PartyBuffs  `xml:"ExportedBuffs" crystalline:"not_nil"`
	Members          []PartyMember `xml:"Member"`

	applied map[string]string
}

type Import struct {
	ExportParty bool `xml:"exportParty,attr"`
}

type PartyMember struct {
	Name        string       `xml:"name,attr"`
	ClassName   string       `xml:"className,attr"`
	Ascendancy  string       `xml:"ascendClassName,attr"`
	Level       int          `xml:"level,attr"`
	Destination string       `xml:"destination,attr"`
	Active      bool         `xml:"active,attr"`
	Buffs       []PartyBuffs `xml:"ExportedBuffs"`
}

type PartyBuffs struct {
	Name string `xml:"name,attr"`
	Text string `xml:",chardata"`
}

type Config struct {
	Inputs       []Input `xml:"Input" crystalline:"not_nil"`
	Placeholders []Input `xml:"Placeholder" crystalline:"not_nil"`

	ActiveConfigSet int         `xml:"activeConfigSet,attr"`
	ConfigSets      []ConfigSet `xml:"ConfigSet" crystalline:"not_nil"`

	CustomModifierBlocks []CustomModifierBlock `xml:"CustomModifierBlock" crystalline:"not_nil"`
}

type ConfigSet struct {
	ID           int     `xml:"id,attr"`
	Title        *string `xml:"title,attr,omitempty"`
	Inputs       []Input `xml:"Input" crystalline:"not_nil"`
	Placeholders []Input `xml:"Placeholder" crystalline:"not_nil"`

	CustomModifierBlocks []CustomModifierBlock `xml:"CustomModifierBlock" crystalline:"not_nil"`
}

//pob:block:src/Classes/ConfigTab.lua:910-940
func (c Config) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	type plainConfig Config
	out := plainConfig(c)

	if len(out.ConfigSets) > 0 {
		sets := make([]ConfigSet, len(out.ConfigSets))
		copy(sets, out.ConfigSets)

		active := -1
		want := out.ActiveConfigSet
		if want == 0 {
			want = 1
		}
		for i := range sets {
			if sets[i].ID == want {
				active = i
				break
			}
		}
		if active < 0 {
			for i := range sets {
				if sets[i].ID == 1 {
					active = i
					break
				}
			}
		}
		if active < 0 {
			active = 0
		}

		sets[active].Inputs = out.Inputs
		sets[active].Placeholders = out.Placeholders
		sets[active].CustomModifierBlocks = out.CustomModifierBlocks
		out.ConfigSets = sets
		out.Inputs = nil
		out.Placeholders = nil
		out.CustomModifierBlocks = nil
	}

	start.Name = xml.Name{Local: "Config"}
	if err := e.EncodeElement(out, start); err != nil {
		return fmt.Errorf("failed to encode Config: %w", err)
	}
	return nil
}

type Input struct {
	Name    string   `xml:"name,attr"`
	Boolean *bool    `xml:"boolean,attr"`
	Number  *float64 `xml:"number,attr"`
	String  *string  `xml:"string,attr"`
}

type Section struct {
	Collapsed  bool   `xml:"collapsed,attr"`
	ID         string `xml:"id,attr"`
	Subsection string `xml:"subsection,attr"`
}

type ItemSet struct {
	ID                 string  `xml:"id,attr"`
	Title              *string `xml:"title,attr,omitempty"`
	UseSecondWeaponSet *bool   `xml:"useSecondWeaponSet,attr,omitempty"`

	Slots []Slot `xml:"Slot"`
}

type Slot struct {
	ItemID int    `xml:"itemId,attr"`
	Name   string `xml:"name,attr"`
	Active bool   `xml:"active,attr,omitempty"`
}

type SkillSet struct {
	ID    int     `xml:"id,attr"`
	Title *string `xml:"title,attr,omitempty"`

	Skills []Skill `xml:"Skill" crystalline:"not_nil"`
}

type Skill struct {
	MainActiveSkillCalcs int    `xml:"mainActiveSkillCalcs,attr,omitempty"`
	MainActiveSkill      int    `xml:"mainActiveSkill,attr,omitempty"`
	Label                string `xml:"label,attr"`
	Enabled              bool   `xml:"enabled,attr"`
	IncludeInFullDPS     *bool  `xml:"includeInFullDPS,attr,omitempty"`
	GroupCount           *int   `xml:"groupCount,attr,omitempty"`

	Gems []Gem `xml:"Gem" crystalline:"not_nil"`

	Slot          string  `xml:"slot,attr,omitempty"`
	ImbuedSupport *string `xml:"imbuedSupport,attr,omitempty"`
	SourceAttr    *string `xml:"source,attr,omitempty"`

	Source string `xml:"-"`

	SlotEnabled           bool   `xml:"-"`
	DisplayLabel          string `xml:"-"`
	displaySkillList      interface{}
	displaySkillListCalcs interface{}
}

func (s *Skill) SetDisplaySkillList(list interface{}) {
	s.displaySkillList = list
}

func (s *Skill) SetDisplaySkillListCalcs(list interface{}) {
	s.displaySkillListCalcs = list
}

func DisplaySkillListOf(s *Skill) interface{} {
	return s.displaySkillList
}

func DisplaySkillListCalcsOf(s *Skill) interface{} {
	return s.displaySkillListCalcs
}

type Gem struct {
	Quality              int     `xml:"quality,attr"`
	SkillPart            int     `xml:"skillPart,attr,omitempty"`
	EnableGlobal2        bool    `xml:"enableGlobal2,attr"`
	SkillPartCalcs       int     `xml:"skillPartCalcs,attr,omitempty"`
	QualityID            string  `xml:"qualityId,attr,omitempty"`
	GemID                string  `xml:"gemId,attr,omitempty"`
	Enabled              bool    `xml:"enabled,attr"`
	Count                *int    `xml:"count,attr,omitempty"`
	EnableGlobal1        bool    `xml:"enableGlobal1,attr"`
	NameSpec             string  `xml:"nameSpec,attr"`
	VariantID            *string `xml:"variantId,attr,omitempty"`
	Level                int     `xml:"level,attr"`
	SkillID              string  `xml:"skillId,attr,omitempty"`
	SkillMinionItemSet   int     `xml:"skillMinionItemSet,attr,omitempty"`
	SkillMinion          string  `xml:"skillMinion,attr,omitempty"`
	SkillMinionSkill     int     `xml:"skillMinionSkill,attr,omitempty"`
	SkillStageCount      int     `xml:"skillStageCount,attr,omitempty"`
	SkillStageCountCalcs int     `xml:"skillStageCountCalcs,attr,omitempty"`

	SkillMineCount      *int `xml:"skillMineCount,attr"`
	SkillMineCountCalcs *int `xml:"skillMineCountCalcs,attr"`

	SkillMinionCalcs        *string `xml:"skillMinionCalcs,attr,omitempty"`
	SkillMinionSkillCalcs   *int    `xml:"skillMinionSkillCalcs,attr,omitempty"`
	SkillMinionItemSetCalcs *int    `xml:"skillMinionItemSetCalcs,attr,omitempty"`

	explodeSource string
}

func (g *Gem) SetExplodeSource(source string) {
	g.explodeSource = source
}

func (g *Gem) ExplodeSource() string {
	return g.explodeSource
}

type Spec struct {
	ClassID                  ClassID      `xml:"classId,attr"`
	AscendClassID            AscendancyID `xml:"ascendClassId,attr"`
	SecondaryAscendClassID   *int         `xml:"secondaryAscendClassId,attr,omitempty"`
	ClusterHashFormatVersion *int         `xml:"clusterHashFormatVersion,attr,omitempty"`
	Title                    *string      `xml:"title,attr,omitempty"`
	TreeVersion              TreeVersion  `xml:"treeVersion,attr"`
	NodesAttr                string       `xml:"nodes,attr"`
	MasteryEffects           string       `xml:"masteryEffects,attr"`
	URL                      string       `xml:"URL,omitempty"`
	Sockets                  []Socket     `xml:"Sockets>Socket" crystalline:"not_nil"`
	Overrides                []Override   `xml:"Overrides>Override" crystalline:"not_nil"`
}

type Socket struct {
	ItemID int `xml:"itemId,attr"`
	NodeID int `xml:"nodeId,attr"`
}

type Override struct {
	NodeID            int     `xml:"nodeId,attr"`
	DN                string  `xml:"dn,attr"`
	Icon              *string `xml:"icon,attr,omitempty"`
	ActiveEffectImage *string `xml:"activeEffectImage,attr,omitempty"`
}

type CustomModifierBlock struct {
	Title   *string `xml:"title,attr"`
	Enabled *string `xml:"enabled,attr"`

	Text string `xml:",chardata"`
}

func (b CustomModifierBlock) BlockTitle() string {
	if b.Title == nil {
		return "Default"
	}
	return *b.Title
}

func (b CustomModifierBlock) IsEnabled() bool {
	return b.Enabled == nil || *b.Enabled == "true"
}
