package pob

import (
	"github.com/Vilsol/go-pob/data"
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
	PantheonMinorGod       string           `xml:"pantheonMinorGod,attr"` // TODO Enum
	PantheonMajorGod       string           `xml:"pantheonMajorGod,attr"` // TODO Enum
	Bandit                 string           `xml:"bandit,attr"`           // TODO Enum
	ViewMode               BuildViewMode    `xml:"viewMode,attr"`
	ClassName              string           `xml:"className,attr"`       // TODO Enum
	AscendClassName        string           `xml:"ascendClassName,attr"` // TODO Enum
	Level                  int              `xml:"level,attr"`
	MainSocketGroup        int              `xml:"mainSocketGroup,attr"`
	TargetVersion          data.GameVersion `xml:"targetVersion,attr"`
	PassiveNodes           []int64
	PassiveNodesStartPaths map[int64][]int64

	PlayerStats []PlayerStat `xml:"PlayerStat" crystalline:"not_nil"`
}

type PlayerStat struct {
	Value float64 `xml:"value,attr"`
	Stat  string  `xml:"stat,attr"`
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
	ActiveItemSet      int   `xml:"activeItemSet,attr"`
	UseSecondWeaponSet *bool `xml:"useSecondWeaponSet,attr,omitempty"`

	ItemSets []ItemSet `xml:"ItemSet" crystalline:"not_nil"`
}

type Skills struct {
	SortGemsByDPSField            string  `xml:"sortGemsByDPSField,attr"`  // TODO Enum
	ShowSupportGemTypes           string  `xml:"showSupportGemTypes,attr"` // TODO Enum
	DefaultGemLevel               *string `xml:"defaultGemLevel,attr,omitempty"`
	MatchGemLevelToCharacterLevel bool    `xml:"matchGemLevelToCharacterLevel,attr"`
	ShowAltQualityGems            bool    `xml:"showAltQualityGems,attr"`
	DefaultGemQuality             *int    `xml:"defaultGemQuality,attr,omitempty"`
	ActiveSkillSet                int     `xml:"activeSkillSet,attr"`
	SortGemsByDPS                 bool    `xml:"sortGemsByDPS,attr"`

	SkillSets []SkillSet `xml:"SkillSet" crystalline:"not_nil"`
}

type TreeView struct {
	ZoomLevel           int     `xml:"zoomLevel,attr"`
	ZoomX               float64 `xml:"zoomX,attr"`
	ZoomY               float64 `xml:"zoomY,attr"`
	SearchStr           string  `xml:"searchStr,attr"`
	ShowHeatMap         *bool   `xml:"showHeatMap,attr,omitempty"`
	ShowStatDifferences bool    `xml:"showStatDifferences,attr"`
}

type Config struct {
	Inputs       []Input `xml:"Input" crystalline:"not_nil"`
	Placeholders []Input `xml:"Placeholder" crystalline:"not_nil"`
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
	ID                 string `xml:"id,attr"`
	UseSecondWeaponSet *bool  `xml:"useSecondWeaponSet,attr,omitempty"`

	Slots []Slot `xml:"Slot"`
}

type Slot struct {
	ItemID int    `xml:"itemId,attr"`
	Name   string `xml:"name,attr"`
}

type SkillSet struct {
	ID int `xml:"id,attr"`

	Skills []Skill `xml:"Skill" crystalline:"not_nil"`
}

type Skill struct {
	MainActiveSkillCalcs int    `xml:"mainActiveSkillCalcs,attr"`
	MainActiveSkill      int    `xml:"mainActiveSkill,attr"`
	Label                string `xml:"label,attr"`
	Enabled              bool   `xml:"enabled,attr"`
	IncludeInFullDPS     *bool  `xml:"includeInFullDPS,attr,omitempty"`

	Gems []Gem `xml:"Gem" crystalline:"not_nil"`

	Slot                  string // TODO Slot
	SlotEnabled           bool
	Source                interface{} // TODO Source
	DisplayLabel          string
	DisplaySkillList      interface{}
	DisplaySkillListCalcs interface{}
}

type Gem struct {
	Quality            int    `xml:"quality,attr"`
	SkillPart          int    `xml:"skillPart,attr"`
	EnableGlobal2      bool   `xml:"enableGlobal2,attr"`
	SkillPartCalcs     int    `xml:"skillPartCalcs,attr"`
	QualityID          string `xml:"qualityId,attr"`
	GemID              string `xml:"gemId,attr"`
	Enabled            bool   `xml:"enabled,attr"`
	Count              int    `xml:"count,attr"`
	EnableGlobal1      bool   `xml:"enableGlobal1,attr"`
	NameSpec           string `xml:"nameSpec,attr"`
	Level              int    `xml:"level,attr"`
	SkillID            string `xml:"skillId,attr"`
	SkillMinionItemSet int    `xml:"skillMinionItemSet,attr"`
	SkillMinion        string `xml:"skillMinion,attr"`

	// TODO
	//DisplayEffect interface{}
	//SupportEffect interface{}
}

type Spec struct {
	ClassID        int              `xml:"classID,attr"`       // TODO Enum
	AscendClassID  int              `xml:"ascendClassID,attr"` // TODO Enum
	TreeVersion    data.TreeVersion `xml:"treeVersion,attr"`   // TODO Enum
	NodesAttr      string           `xml:"nodes,attr"`
	MasteryEffects string           `xml:"masteryEffects,attr"`
	URL            string           `xml:"URL"`
}
