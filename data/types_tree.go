package data

type Tree struct {
	Tree            string                `json:"tree"`
	Classes         []Class               `json:"classes"`
	Groups          map[string]Group      `json:"groups"`
	Nodes           map[string]Node       `json:"nodes"`
	ExtraImages     map[string]ExtraImage `json:"extraImages"`
	JewelSlots      []int64               `json:"jewelSlots"`
	MinX            int64                 `json:"min_x"`
	MinY            int64                 `json:"min_y"`
	MaxX            int64                 `json:"max_x"`
	MaxY            int64                 `json:"max_y"`
	Constants       Constants             `json:"constants"`
	Sprites         Sprites               `json:"sprites"`
	ImageZoomLevels []float64             `json:"imageZoomLevels"`
	Points          Points                `json:"points"`
}

type Class struct {
	Name         ClassName    `json:"name"`
	BaseStr      int64        `json:"base_str"`
	BaseDex      int64        `json:"base_dex"`
	BaseInt      int64        `json:"base_int"`
	Ascendancies []Ascendancy `json:"ascendancies"`
}

type Ascendancy struct {
	ID                AscendancyID     `json:"id"`
	Name              AscendancyName   `json:"name"`
	FlavourText       *string          `json:"flavourText,omitempty"`
	FlavourTextColour *string          `json:"flavourTextColour,omitempty"`
	FlavourTextRect   *FlavourTextRect `json:"flavourTextRect,omitempty"`
}

type FlavourTextRect struct {
	X      int64 `json:"x"`
	Y      int64 `json:"y"`
	Width  int64 `json:"width"`
	Height int64 `json:"height"`
}

type Constants struct {
	Classes              Classes             `json:"classes"`
	CharacterAttributes  CharacterAttributes `json:"characterAttributes"`
	PSSCentreInnerRadius int64               `json:"PSSCentreInnerRadius"`
	SkillsPerOrbit       []int64             `json:"skillsPerOrbit"`
	OrbitRadii           []int64             `json:"orbitRadii"`
}

type CharacterAttributes struct {
	Strength     int64 `json:"Strength"`
	Dexterity    int64 `json:"Dexterity"`
	Intelligence int64 `json:"Intelligence"`
}

type Classes struct {
	StrDexIntClass int64 `json:"StrDexIntClass"`
	StrClass       int64 `json:"StrClass"`
	DexClass       int64 `json:"DexClass"`
	IntClass       int64 `json:"IntClass"`
	StrDexClass    int64 `json:"StrDexClass"`
	StrIntClass    int64 `json:"StrIntClass"`
	DexIntClass    int64 `json:"DexIntClass"`
}

type ExtraImage struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Image string  `json:"image"`
}

type Group struct {
	X       float64  `json:"x"`
	Y       float64  `json:"y"`
	Orbits  []int64  `json:"orbits"`
	Nodes   []string `json:"nodes"`
	IsProxy *bool    `json:"isProxy,omitempty"`
}

type Node struct {
	Skill                  *int64          `json:"skill,omitempty"`
	Name                   *string         `json:"name,omitempty"`
	Icon                   *string         `json:"icon,omitempty"`
	IsNotable              *bool           `json:"isNotable,omitempty"`
	Recipe                 []OilType       `json:"recipe,omitempty"`
	Stats                  []string        `json:"stats,omitempty"`
	Group                  *int64          `json:"group,omitempty"`
	Orbit                  *int64          `json:"orbit,omitempty"`
	OrbitIndex             *int64          `json:"orbitIndex,omitempty"`
	Out                    []string        `json:"out,omitempty"`
	In                     []string        `json:"in,omitempty"`
	ReminderText           []string        `json:"reminderText,omitempty"`
	IsMastery              *bool           `json:"isMastery,omitempty"`
	InactiveIcon           *string         `json:"inactiveIcon,omitempty"`
	ActiveIcon             *string         `json:"activeIcon,omitempty"`
	ActiveEffectImage      *string         `json:"activeEffectImage,omitempty"`
	MasteryEffects         []MasteryEffect `json:"masteryEffects,omitempty"`
	GrantedStrength        *int64          `json:"grantedStrength,omitempty"`
	AscendancyName         *string         `json:"ascendancyName,omitempty"`
	GrantedDexterity       *int64          `json:"grantedDexterity,omitempty"`
	IsAscendancyStart      *bool           `json:"isAscendancyStart,omitempty"`
	IsMultipleChoice       *bool           `json:"isMultipleChoice,omitempty"`
	GrantedIntelligence    *int64          `json:"grantedIntelligence,omitempty"`
	IsJewelSocket          *bool           `json:"isJewelSocket,omitempty"`
	ExpansionJewel         *ExpansionJewel `json:"expansionJewel,omitempty"`
	GrantedPassivePoints   *int64          `json:"grantedPassivePoints,omitempty"`
	IsKeystone             *bool           `json:"isKeystone,omitempty"`
	FlavourText            []string        `json:"flavourText,omitempty"`
	IsProxy                *bool           `json:"isProxy,omitempty"`
	IsMultipleChoiceOption *bool           `json:"isMultipleChoiceOption,omitempty"`
	IsBlighted             *bool           `json:"isBlighted,omitempty"`
	ClassStartIndex        *int64          `json:"classStartIndex,omitempty"`
}

type ExpansionJewel struct {
	Size   int64   `json:"size"`
	Index  int64   `json:"index"`
	Proxy  string  `json:"proxy"`
	Parent *string `json:"parent,omitempty"`
}

type MasteryEffect struct {
	Effect       int64    `json:"effect"`
	Stats        []string `json:"stats"`
	ReminderText []string `json:"reminderText,omitempty"`
}

type Points struct {
	TotalPoints      int64 `json:"totalPoints"`
	AscendancyPoints int64 `json:"ascendancyPoints"`
}

type Sprites struct {
	Background            map[string]Sprite `json:"background"`
	NormalActive          map[string]Sprite `json:"normalActive"`
	NotableActive         map[string]Sprite `json:"notableActive"`
	KeystoneActive        map[string]Sprite `json:"keystoneActive"`
	NormalInactive        map[string]Sprite `json:"normalInactive"`
	NotableInactive       map[string]Sprite `json:"notableInactive"`
	KeystoneInactive      map[string]Sprite `json:"keystoneInactive"`
	Mastery               map[string]Sprite `json:"mastery"`
	MasteryConnected      map[string]Sprite `json:"masteryConnected"`
	MasteryActiveSelected map[string]Sprite `json:"masteryActiveSelected"`
	MasteryInactive       map[string]Sprite `json:"masteryInactive"`
	MasteryActiveEffect   map[string]Sprite `json:"masteryActiveEffect"`
	AscendancyBackground  map[string]Sprite `json:"ascendancyBackground"`
	Ascendancy            map[string]Sprite `json:"ascendancy"`
	StartNode             map[string]Sprite `json:"startNode"`
	GroupBackground       map[string]Sprite `json:"groupBackground"`
	Frame                 map[string]Sprite `json:"frame"`
	Jewel                 map[string]Sprite `json:"jewel"`
	Line                  map[string]Sprite `json:"line"`
	JewelRadius           map[string]Sprite `json:"jewelRadius"`
}

type Coord struct {
	X int64 `json:"x"`
	Y int64 `json:"y"`
	W int64 `json:"w"`
	H int64 `json:"h"`
}

type Sprite struct {
	Filename string           `json:"filename"`
	W        int64            `json:"w"`
	H        int64            `json:"h"`
	Coords   map[string]Coord `json:"coords"`
}

type AscendancyID int

const (
	Ascendant AscendancyID = iota
	Assassin
	Berserker
	Champion
	Chieftain
	Deadeye
	Elementalist
	Gladiator
	Guardian
	Hierophant
	Inquisitor
	Juggernaut
	Necromancer
	Occultist
	Pathfinder
	Raider
	Saboteur
	Slayer
	Trickster
)

type AscendancyName string

var AscendancyNameByID = map[AscendancyID]AscendancyName{
	Ascendant:    "Ascendant",
	Assassin:     "Assassin",
	Berserker:    "Berserker",
	Champion:     "Champion",
	Chieftain:    "Chieftain",
	Deadeye:      "Deadeye",
	Elementalist: "Elementalist",
	Gladiator:    "Gladiator",
	Guardian:     "Guardian",
	Hierophant:   "Hierophant",
	Inquisitor:   "Inquisitor",
	Juggernaut:   "Juggernaut",
	Necromancer:  "Necromancer",
	Occultist:    "Occultist",
	Pathfinder:   "Pathfinder",
	Raider:       "Raider",
	Saboteur:     "Saboteur",
	Slayer:       "Slayer",
	Trickster:    "Trickster",
}

type ClassID int

const (
	Scion ClassID = iota
	Marauder
	Ranger
	Witch
	Duelist
	Templar
	Shadow
)

var ClassAscendancies = map[ClassID][]AscendancyID{
	Scion:    {Ascendant},
	Marauder: {Juggernaut, Berserker, Chieftain},
	Ranger:   {Deadeye, Pathfinder, Raider},
	Witch:    {Elementalist, Necromancer, Occultist},
	Duelist:  {Slayer, Gladiator, Champion},
	Templar:  {Hierophant, Inquisitor, Guardian},
	Shadow:   {Assassin, Trickster, Saboteur},
}

type ClassName string

var ClassNameByID = map[ClassID]ClassName{
	Scion:    "Scion",
	Marauder: "Marauder",
	Ranger:   "Ranger",
	Witch:    "Witch",
	Duelist:  "Duelist",
	Templar:  "Templar",
	Shadow:   "Shadow",
}

type OilType string
