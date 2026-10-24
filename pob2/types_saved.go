package pob2

// The build file as PoB-PoE2 writes it. Each type names the attributes and children its tab's Load reads and Save
// writes; a pointer is an attribute or child that Save can leave out, Nil one that Save can write as "nil".

//pob2:src/Modules/Build.lua:2691-2770
type PathOfBuilding2 struct {
	Build    Build    `pob:"Build"`
	Calcs    Calcs    `pob:"Calcs"`
	Config   Config   `pob:"Config"`
	Import   Import   `pob:"Import"`
	Items    Items    `pob:"Items"`
	Notes    Notes    `pob:"Notes"`
	Party    Party    `pob:"Party"`
	Skills   Skills   `pob:"Skills"`
	Tree     Tree     `pob:"Tree"`
	TreeView TreeView `pob:"TreeView"`
	Unparsed
}

//pob2:src/Modules/Build.lua:1138-1270
type Build struct {
	TargetVersion          string `pob:"targetVersion,attr"`
	ViewMode               string `pob:"viewMode,attr"`
	Level                  int    `pob:"level,attr"`
	ClassName              string `pob:"className,attr"`
	AscendClassName        string `pob:"ascendClassName,attr"`
	MainSocketGroup        int    `pob:"mainSocketGroup,attr"`
	MainSkillIndex         *int   `pob:"mainSkillIndex,attr"`
	CharacterLevelAutoMode bool   `pob:"characterLevelAutoMode,attr"`

	Spectres        []MinionID     `pob:"Spectre"`
	BeastCompanions []MinionID     `pob:"BeastCompanion"`
	PlayerStats     []PlayerStat   `pob:"PlayerStat"`
	FullDPSSkills   []FullDPSSkill `pob:"FullDPSSkill"`
	MinionStats     []PlayerStat   `pob:"MinionStat"`
	Buffs           *Buffs         `pob:"Buffs"`
	TimelessData    *TimelessData  `pob:"TimelessData"`
	Unparsed
}

type MinionID struct {
	ID string `pob:"id,attr"`
	Unparsed
}

type PlayerStat struct {
	Stat  string  `pob:"stat,attr"`
	Value float64 `pob:"value,attr"`
	Unparsed
}

type FullDPSSkill struct {
	Stat      string  `pob:"stat,attr"`
	Value     float64 `pob:"value,attr"`
	SkillPart string  `pob:"skillPart,attr"`
	Source    string  `pob:"source,attr"`
	Unparsed
}

type Buffs struct {
	BuffList   *string `pob:"buffList,attr"`
	CombatList *string `pob:"combatList,attr"`
	CurseList  *string `pob:"curseList,attr"`
	Unparsed
}

type TimelessData struct {
	JewelTypeID           *int     `pob:"jewelTypeId,attr"`
	ConquerorTypeID       *int     `pob:"conquerorTypeId,attr"`
	DevotionVariant1      Nil[int] `pob:"devotionVariant1,attr"`
	DevotionVariant2      Nil[int] `pob:"devotionVariant2,attr"`
	JewelSocketID         *int     `pob:"jewelSocketId,attr"`
	FallbackWeightModeIdx *int     `pob:"fallbackWeightModeIdx,attr"`
	SocketFilter          *bool    `pob:"socketFilter,attr"`
	SocketFilterDistance  *int     `pob:"socketFilterDistance,attr"`
	SearchList            *string  `pob:"searchList,attr"`
	SearchListFallback    *string  `pob:"searchListFallback,attr"`
	Unparsed
}

//pob2:src/Classes/CalcsTab.lua:200-260
type Calcs struct {
	Inputs   []Input   `pob:"Input"`
	Sections []Section `pob:"Section"`
	Unparsed
}

type Input struct {
	Name    string   `pob:"name,attr"`
	Number  *float64 `pob:"number,attr"`
	String  *string  `pob:"string,attr"`
	Boolean *bool    `pob:"boolean,attr"`
	Unparsed
}

type Section struct {
	ID         string `pob:"id,attr"`
	Subsection string `pob:"subsection,attr"`
	Collapsed  bool   `pob:"collapsed,attr"`
	Unparsed
}

// Inputs, Placeholders and CustomModifierBlocks directly under Config are the pre-ConfigSet layout Load still reads.
//
//pob2:src/Classes/ConfigTab.lua:878-1045
type Config struct {
	ActiveConfigSet int `pob:"activeConfigSet,attr"`

	ConfigSets           []ConfigSet           `pob:"ConfigSet"`
	Inputs               []Input               `pob:"Input"`
	Placeholders         []Input               `pob:"Placeholder"`
	CustomModifierBlocks []CustomModifierBlock `pob:"CustomModifierBlock"`
	Unparsed
}

type ConfigSet struct {
	ID    int     `pob:"id,attr"`
	Title *string `pob:"title,attr"`

	Inputs               []Input               `pob:"Input"`
	Placeholders         []Input               `pob:"Placeholder"`
	CustomModifierBlocks []CustomModifierBlock `pob:"CustomModifierBlock"`
	Unparsed
}

type CustomModifierBlock struct {
	Title   *string `pob:"title,attr"`
	Enabled *bool   `pob:"enabled,attr"`
	Text    string  `pob:",text,always"`
	Unparsed
}

//pob2:src/Classes/ImportTab.lua:556-592
type Import struct {
	LastRealm            *string `pob:"lastRealm,attr"`
	LastLeague           *string `pob:"lastLeague,attr"`
	LastAccountHash      *string `pob:"lastAccountHash,attr"`
	LastCharacterHash    *string `pob:"lastCharacterHash,attr"`
	ExportParty          bool    `pob:"exportParty,attr"`
	UseGeneratedItemText bool    `pob:"useGeneratedItemText,attr"`
	ImportLink           *string `pob:"importLink,attr"`
	Unparsed
}

// Slots directly under Items are the pre-ItemSet layout Load still reads.
//
//pob2:src/Classes/ItemsTab.lua:1220-1439
type Items struct {
	ActiveItemSet       int       `pob:"activeItemSet,attr"`
	UseSecondWeaponSet  Nil[bool] `pob:"useSecondWeaponSet,attr"`
	ShowStatDifferences *bool     `pob:"showStatDifferences,attr"`

	Items              []Item              `pob:"Item"`
	Slots              []Slot              `pob:"Slot"`
	ItemSets           []ItemSet           `pob:"ItemSet"`
	TradeSearchWeights *TradeSearchWeights `pob:"TradeSearchWeights"`
	Unparsed
}

type Item struct {
	ID          int  `pob:"id,attr"`
	Variant     *int `pob:"variant,attr"`
	VariantAlt  *int `pob:"variantAlt,attr"`
	VariantAlt2 *int `pob:"variantAlt2,attr"`
	VariantAlt3 *int `pob:"variantAlt3,attr"`
	VariantAlt4 *int `pob:"variantAlt4,attr"`
	VariantAlt5 *int `pob:"variantAlt5,attr"`

	Raw       string     `pob:",text,always"`
	ModRanges []ModRange `pob:"ModRange"`
	Unparsed
}

type ModRange struct {
	ID    int     `pob:"id,attr"`
	Range float64 `pob:"range,attr"`
	Unparsed
}

type ItemSet struct {
	ID                 int       `pob:"id,attr"`
	Title              *string   `pob:"title,attr"`
	UseSecondWeaponSet Nil[bool] `pob:"useSecondWeaponSet,attr"`

	Slots        []Slot        `pob:"Slot"`
	SocketIDURLs []SocketIDURL `pob:"SocketIdURL"`
	RuneSlots    []RuneSlot    `pob:"RuneSlot"`
	Unparsed
}

type Slot struct {
	Name      string  `pob:"name,attr"`
	ItemID    int     `pob:"itemId,attr"`
	ItemPbURL *string `pob:"itemPbURL,attr"`
	Active    *bool   `pob:"active,attr"`
	Note      *string `pob:"note,attr"`
	Unparsed
}

type SocketIDURL struct {
	Name      string  `pob:"name,attr"`
	NodeID    int     `pob:"nodeId,attr"`
	ItemPbURL *string `pob:"itemPbURL,attr"`
	Unparsed
}

type RuneSlot struct {
	SlotName string `pob:"slotName,attr"`
	RuneName string `pob:"runeName,attr"`
	Unparsed
}

type TradeSearchWeights struct {
	Stats []TradeSearchStat `pob:"Stat"`
	Unparsed
}

type TradeSearchStat struct {
	Label      *string `pob:"label,attr"`
	Stat       *string `pob:"stat,attr"`
	WeightMult *string `pob:"weightMult,attr"`
	Unparsed
}

//pob2:src/Classes/NotesTab.lua:75-88
type Notes struct {
	Text string `pob:",text,always"`
	Unparsed
}

//pob2:src/Classes/PartyTab.lua:516-668
type Party struct {
	Destination      *string `pob:"destination,attr"`
	Append           bool    `pob:"append,attr"`
	ShowAdvanceTools bool    `pob:"ShowAdvanceTools,attr"`

	ImportedBuffs []PartyBuffs `pob:"ImportedBuffs"`
	ExportedBuffs []PartyBuffs `pob:"ExportedBuffs"`
	Unparsed
}

type PartyBuffs struct {
	Name *string `pob:"name,attr"`
	Text string  `pob:",text"`
	Unparsed
}

// Skills directly under Skills are the pre-SkillSet layout Load still reads.
//
//pob2:src/Classes/SkillsTab.lua:401-566
type Skills struct {
	ActiveSkillSet                int      `pob:"activeSkillSet,attr"`
	DefaultGemLevel               *string  `pob:"defaultGemLevel,attr"`
	DefaultGemQuality             Nil[int] `pob:"defaultGemQuality,attr"`
	SortGemsByDPS                 bool     `pob:"sortGemsByDPS,attr"`
	ShowSupportGemTypes           *string  `pob:"showSupportGemTypes,attr"`
	SortGemsByDPSField            *string  `pob:"sortGemsByDPSField,attr"`
	ShowLegacyGems                bool     `pob:"showLegacyGems,attr"`
	MatchGemLevelToCharacterLevel *bool    `pob:"matchGemLevelToCharacterLevel,attr"`

	SkillSets []SkillSet `pob:"SkillSet"`
	Skills    []Skill    `pob:"Skill"`
	Unparsed
}

type SkillSet struct {
	ID    int     `pob:"id,attr"`
	Title *string `pob:"title,attr"`

	Skills []Skill `pob:"Skill"`
	Unparsed
}

// A socket group; Active is the legacy spelling of enabled that LoadSkill still reads.
//
//pob2:src/Classes/SkillsTab.lua:288-399
type Skill struct {
	Enabled              bool      `pob:"enabled,attr"`
	Active               *bool     `pob:"active,attr"`
	IncludeInFullDPS     Nil[bool] `pob:"includeInFullDPS,attr"`
	GroupCount           *int      `pob:"groupCount,attr"`
	Label                *string   `pob:"label,attr"`
	Slot                 *string   `pob:"slot,attr"`
	Source               *string   `pob:"source,attr"`
	Set1                 bool      `pob:"set1,attr"`
	Set2                 bool      `pob:"set2,attr"`
	MainActiveSkill      Nil[int]  `pob:"mainActiveSkill,attr"`
	MainActiveSkillCalcs Nil[int]  `pob:"mainActiveSkillCalcs,attr"`
	SkillPart            *int      `pob:"skillPart,attr"`
	Removed              *bool     `pob:"removed,attr"`
	RemovedSource        *string   `pob:"removedSource,attr"`
	RemovedSlot          *string   `pob:"removedSlot,attr"`
	RemovedSkillID       *string   `pob:"removedSkillId,attr"`

	Gems []Gem `pob:"Gem"`
	Unparsed
}

type Gem struct {
	NameSpec                string    `pob:"nameSpec,attr"`
	SkillID                 *string   `pob:"skillId,attr"`
	GemID                   *string   `pob:"gemId,attr"`
	VariantID               *string   `pob:"variantId,attr"`
	Level                   Nil[int]  `pob:"level,attr"`
	Quality                 Nil[int]  `pob:"quality,attr"`
	Enabled                 bool      `pob:"enabled,attr"`
	EnableGlobal1           Nil[bool] `pob:"enableGlobal1,attr"`
	EnableGlobal2           Nil[bool] `pob:"enableGlobal2,attr"`
	Count                   Nil[int]  `pob:"count,attr"`
	StatSetIndex            *Nil[int] `pob:"statSetIndex,attr"`
	StatSetIndexCalcs       *Nil[int] `pob:"statSetIndexCalcs,attr"`
	SkillPart               *int      `pob:"skillPart,attr"`
	SkillPartCalcs          *int      `pob:"skillPartCalcs,attr"`
	SkillStageCount         *int      `pob:"skillStageCount,attr"`
	SkillStageCountCalcs    *int      `pob:"skillStageCountCalcs,attr"`
	SkillMineCount          *int      `pob:"skillMineCount,attr"`
	SkillMineCountCalcs     *int      `pob:"skillMineCountCalcs,attr"`
	SkillMinion             *string   `pob:"skillMinion,attr"`
	SkillMinionCalcs        *string   `pob:"skillMinionCalcs,attr"`
	SkillMinionItemSet      *int      `pob:"skillMinionItemSet,attr"`
	SkillMinionItemSetCalcs *int      `pob:"skillMinionItemSetCalcs,attr"`
	SkillMinionSkill        *int      `pob:"skillMinionSkill,attr"`
	SkillMinionSkillCalcs   *int      `pob:"skillMinionSkillCalcs,attr"`
	Corrupted               Nil[bool] `pob:"corrupted,attr"`
	CorruptLevel            Nil[int]  `pob:"corruptLevel,attr"`
	Note                    *string   `pob:"note,attr"`

	StatSetIndexes              []StatSetIndex           `pob:"StatSetIndex"`
	StatSetCalcsIndexes         []StatSetIndex           `pob:"StatSetCalcsIndex"`
	MinionSkillIndexLookups     []MinionSkillIndexLookup `pob:"MinionSkillIndexLookup"`
	MinionSkillIndexLookupCalcs []MinionSkillIndexLookup `pob:"MinionSkillIndexLookupCalcs"`
	Unparsed
}

type StatSetIndex struct {
	GrantedEffect string   `pob:"grantedEffect,attr"`
	Index         Nil[int] `pob:"index,attr"`
	Unparsed
}

type MinionSkillIndexLookup struct {
	GrantedEffect string                `pob:"grantedEffect,attr"`
	Maps          []MinionSkillIndexMap `pob:"MinionSkillIndexMap"`
	Unparsed
}

type MinionSkillIndexMap struct {
	SkillIndex   Nil[int] `pob:"skillIndex,attr"`
	StatSetIndex Nil[int] `pob:"statSetIndex,attr"`
	Unparsed
}

//pob2:src/Classes/TreeTab.lua:493-539
type Tree struct {
	ActiveSpec int    `pob:"activeSpec,attr"`
	Specs      []Spec `pob:"Spec"`
	Unparsed
}

// Nodes, the WeaponSet lists and the AttributeOverride lists are comma-joined node ids; NodeIDs reads them as Load does.
//
//pob2:src/Classes/PassiveSpec.lua:117-379
type Spec struct {
	Title                  *string  `pob:"title,attr"`
	TreeVersion            *string  `pob:"treeVersion,attr"`
	ClassID                *int     `pob:"classId,attr"`
	AscendClassID          *int     `pob:"ascendClassId,attr"`
	ClassInternalID        *int     `pob:"classInternalId,attr"`
	AscendancyInternalID   *string  `pob:"ascendancyInternalId,attr"`
	SecondaryAscendClassID Nil[int] `pob:"secondaryAscendClassId,attr"`
	Nodes                  *string  `pob:"nodes,attr"`
	MasteryEffects         *string  `pob:"masteryEffects,attr"`

	URL        *URL       `pob:"URL"`
	WeaponSet1 *WeaponSet `pob:"WeaponSet1"`
	WeaponSet2 *WeaponSet `pob:"WeaponSet2"`
	Sockets    *Sockets   `pob:"Sockets"`
	Overrides  *Overrides `pob:"Overrides"`
	NodeNotes  *NodeNotes `pob:"Notes"`
	Unparsed
}

type URL struct {
	Text string `pob:",text,always"`
	Unparsed
}

type WeaponSet struct {
	Nodes string `pob:"nodes,attr"`
	Unparsed
}

type Sockets struct {
	Sockets []Socket `pob:"Socket"`
	Unparsed
}

type Socket struct {
	NodeID int `pob:"nodeId,attr"`
	ItemID int `pob:"itemId,attr"`
	Unparsed
}

type Overrides struct {
	AttributeOverrides []AttributeOverride `pob:"AttributeOverride"`
	Unparsed
}

type AttributeOverride struct {
	StrNodes string `pob:"strNodes,attr"`
	DexNodes string `pob:"dexNodes,attr"`
	IntNodes string `pob:"intNodes,attr"`
	Unparsed
}

type NodeNotes struct {
	Notes []NodeNote `pob:"Note"`
	Unparsed
}

type NodeNote struct {
	NodeID int     `pob:"nodeId,attr"`
	Text   string  `pob:",text"`
	Attr   *string `pob:"text,attr"`
	Unparsed
}

//pob2:src/Classes/PassiveTreeView.lua:60-87
type TreeView struct {
	ZoomLevel           float64 `pob:"zoomLevel,attr"`
	ZoomX               float64 `pob:"zoomX,attr"`
	ZoomY               float64 `pob:"zoomY,attr"`
	SearchStr           *string `pob:"searchStr,attr"`
	ShowStatDifferences bool    `pob:"showStatDifferences,attr"`
	Unparsed
}
