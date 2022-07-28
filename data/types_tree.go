package data

type Tree struct {
	Classes         []Class               `json:"classes"`
	Groups          map[string]Group      `json:"groups"`
	Nodes           map[string]Node       `json:"nodes"`
	ExtraImages     map[string]ExtraImage `json:"extraImages"`
	MinX            int64                 `json:"min_x"`
	MinY            int64                 `json:"min_y"`
	MaxX            int64                 `json:"max_x"`
	MaxY            int64                 `json:"max_y"`
	Assets          Assets                `json:"assets"`
	Constants       Constants             `json:"constants"`
	SkillSprites    SkillSprites          `json:"skillSprites"`
	ImageZoomLevels []float64             `json:"imageZoomLevels"`
	JewelSlots      []int64               `json:"jewelSlots,omitempty"`
	Tree            *string               `json:"tree,omitempty"`
	Points          *Points               `json:"points,omitempty"`
}

type Assets struct {
	PSSkillFrame                            ScaledAsset  `json:"PSSkillFrame"`
	PSSkillFrameHighlighted                 ScaledAsset  `json:"PSSkillFrameHighlighted"`
	PSSkillFrameActive                      ScaledAsset  `json:"PSSkillFrameActive"`
	KeystoneFrameUnallocated                ScaledAsset  `json:"KeystoneFrameUnallocated"`
	KeystoneFrameCanAllocate                ScaledAsset  `json:"KeystoneFrameCanAllocate"`
	KeystoneFrameAllocated                  ScaledAsset  `json:"KeystoneFrameAllocated"`
	PSGroupBackground1                      ScaledAsset  `json:"PSGroupBackground1"`
	PSGroupBackground2                      ScaledAsset  `json:"PSGroupBackground2"`
	PSGroupBackground3                      ScaledAsset  `json:"PSGroupBackground3"`
	GroupBackgroundSmallAlt                 ScaledAsset  `json:"GroupBackgroundSmallAlt"`
	GroupBackgroundMediumAlt                ScaledAsset  `json:"GroupBackgroundMediumAlt"`
	GroupBackgroundLargeHalfAlt             ScaledAsset  `json:"GroupBackgroundLargeHalfAlt"`
	Orbit1Normal                            ScaledAsset  `json:"Orbit1Normal"`
	Orbit1Intermediate                      ScaledAsset  `json:"Orbit1Intermediate"`
	Orbit1Active                            ScaledAsset  `json:"Orbit1Active"`
	Orbit2Normal                            ScaledAsset  `json:"Orbit2Normal"`
	Orbit2Intermediate                      ScaledAsset  `json:"Orbit2Intermediate"`
	Orbit2Active                            ScaledAsset  `json:"Orbit2Active"`
	Orbit3Normal                            ScaledAsset  `json:"Orbit3Normal"`
	Orbit3Intermediate                      ScaledAsset  `json:"Orbit3Intermediate"`
	Orbit3Active                            ScaledAsset  `json:"Orbit3Active"`
	Orbit4Normal                            ScaledAsset  `json:"Orbit4Normal"`
	Orbit4Intermediate                      ScaledAsset  `json:"Orbit4Intermediate"`
	Orbit4Active                            ScaledAsset  `json:"Orbit4Active"`
	LineConnectorNormal                     ScaledAsset  `json:"LineConnectorNormal"`
	LineConnectorIntermediate               ScaledAsset  `json:"LineConnectorIntermediate"`
	LineConnectorActive                     ScaledAsset  `json:"LineConnectorActive"`
	PSLineDeco                              ScaledAsset  `json:"PSLineDeco"`
	PSLineDecoHighlighted                   ScaledAsset  `json:"PSLineDecoHighlighted"`
	PSStartNodeBackgroundInactive           ScaledAsset  `json:"PSStartNodeBackgroundInactive"`
	Centerduelist                           ScaledAsset  `json:"centerduelist"`
	Centermarauder                          ScaledAsset  `json:"centermarauder"`
	Centerranger                            ScaledAsset  `json:"centerranger"`
	Centershadow                            ScaledAsset  `json:"centershadow"`
	Centertemplar                           ScaledAsset  `json:"centertemplar"`
	Centerwitch                             ScaledAsset  `json:"centerwitch"`
	Centerscion                             ScaledAsset  `json:"centerscion"`
	PSPointsFrame                           BasicAsset   `json:"PSPointsFrame"`
	NotableFrameUnallocated                 ScaledAsset  `json:"NotableFrameUnallocated"`
	NotableFrameCanAllocate                 ScaledAsset  `json:"NotableFrameCanAllocate"`
	NotableFrameAllocated                   ScaledAsset  `json:"NotableFrameAllocated"`
	BlightedNotableFrameUnallocated         ScaledAsset  `json:"BlightedNotableFrameUnallocated"`
	BlightedNotableFrameCanAllocate         ScaledAsset  `json:"BlightedNotableFrameCanAllocate"`
	BlightedNotableFrameAllocated           ScaledAsset  `json:"BlightedNotableFrameAllocated"`
	JewelFrameUnallocated                   ScaledAsset  `json:"JewelFrameUnallocated"`
	JewelFrameCanAllocate                   ScaledAsset  `json:"JewelFrameCanAllocate"`
	JewelFrameAllocated                     ScaledAsset  `json:"JewelFrameAllocated"`
	JewelSocketActiveBlue                   ScaledAsset  `json:"JewelSocketActiveBlue"`
	JewelSocketActiveGreen                  ScaledAsset  `json:"JewelSocketActiveGreen"`
	JewelSocketActiveRed                    ScaledAsset  `json:"JewelSocketActiveRed"`
	JewelSocketActivePrismatic              ScaledAsset  `json:"JewelSocketActivePrismatic"`
	JewelSocketActiveAbyss                  ScaledAsset  `json:"JewelSocketActiveAbyss"`
	JewelCircle1                            BasicAsset   `json:"JewelCircle1"`
	JewelCircle1Inverse                     BasicAsset   `json:"JewelCircle1Inverse"`
	VaalJewelCircle1                        BasicAsset   `json:"VaalJewelCircle1"`
	VaalJewelCircle2                        BasicAsset   `json:"VaalJewelCircle2"`
	KaruiJewelCircle1                       BasicAsset   `json:"KaruiJewelCircle1"`
	KaruiJewelCircle2                       BasicAsset   `json:"KaruiJewelCircle2"`
	MarakethJewelCircle1                    BasicAsset   `json:"MarakethJewelCircle1"`
	MarakethJewelCircle2                    BasicAsset   `json:"MarakethJewelCircle2"`
	TemplarJewelCircle1                     BasicAsset   `json:"TemplarJewelCircle1"`
	TemplarJewelCircle2                     BasicAsset   `json:"TemplarJewelCircle2"`
	EternalEmpireJewelCircle1               BasicAsset   `json:"EternalEmpireJewelCircle1"`
	EternalEmpireJewelCircle2               BasicAsset   `json:"EternalEmpireJewelCircle2"`
	JewelSocketAltNormal                    ScaledAsset  `json:"JewelSocketAltNormal"`
	JewelSocketAltCanAllocate               ScaledAsset  `json:"JewelSocketAltCanAllocate"`
	JewelSocketAltActive                    ScaledAsset  `json:"JewelSocketAltActive"`
	JewelSocketActiveBlueAlt                ScaledAsset  `json:"JewelSocketActiveBlueAlt"`
	JewelSocketActiveGreenAlt               ScaledAsset  `json:"JewelSocketActiveGreenAlt"`
	JewelSocketActiveRedAlt                 ScaledAsset  `json:"JewelSocketActiveRedAlt"`
	JewelSocketActivePrismaticAlt           ScaledAsset  `json:"JewelSocketActivePrismaticAlt"`
	JewelSocketActiveAbyssAlt               ScaledAsset  `json:"JewelSocketActiveAbyssAlt"`
	JewelSocketClusterAltNormal1Small       ScaledAsset  `json:"JewelSocketClusterAltNormal1Small"`
	JewelSocketClusterAltCanAllocate1Small  ScaledAsset  `json:"JewelSocketClusterAltCanAllocate1Small"`
	JewelSocketClusterAltNormal1Medium      ScaledAsset  `json:"JewelSocketClusterAltNormal1Medium"`
	JewelSocketClusterAltCanAllocate1Medium ScaledAsset  `json:"JewelSocketClusterAltCanAllocate1Medium"`
	JewelSocketClusterAltNormal1Large       ScaledAsset  `json:"JewelSocketClusterAltNormal1Large"`
	JewelSocketClusterAltCanAllocate1Large  ScaledAsset  `json:"JewelSocketClusterAltCanAllocate1Large"`
	AscendancyButton                        ScaledAsset  `json:"AscendancyButton"`
	AscendancyButtonHighlight               ScaledAsset  `json:"AscendancyButtonHighlight"`
	AscendancyButtonPressed                 ScaledAsset  `json:"AscendancyButtonPressed"`
	AscendancyFrameLargeAllocated           ScaledAsset  `json:"AscendancyFrameLargeAllocated"`
	AscendancyFrameLargeCanAllocate         ScaledAsset  `json:"AscendancyFrameLargeCanAllocate"`
	AscendancyFrameLargeNormal              ScaledAsset  `json:"AscendancyFrameLargeNormal"`
	AscendancyFrameSmallAllocated           ScaledAsset  `json:"AscendancyFrameSmallAllocated"`
	AscendancyFrameSmallCanAllocate         ScaledAsset  `json:"AscendancyFrameSmallCanAllocate"`
	AscendancyFrameSmallNormal              ScaledAsset  `json:"AscendancyFrameSmallNormal"`
	AscendancyMiddle                        ScaledAsset  `json:"AscendancyMiddle"`
	ClassesAscendant                        ScaledAsset  `json:"ClassesAscendant"`
	ClassesJuggernaut                       ScaledAsset  `json:"ClassesJuggernaut"`
	ClassesBerserker                        ScaledAsset  `json:"ClassesBerserker"`
	ClassesChieftain                        ScaledAsset  `json:"ClassesChieftain"`
	ClassesRaider                           ScaledAsset  `json:"ClassesRaider"`
	ClassesDeadeye                          ScaledAsset  `json:"ClassesDeadeye"`
	ClassesPathfinder                       ScaledAsset  `json:"ClassesPathfinder"`
	ClassesOccultist                        ScaledAsset  `json:"ClassesOccultist"`
	ClassesElementalist                     ScaledAsset  `json:"ClassesElementalist"`
	ClassesNecromancer                      ScaledAsset  `json:"ClassesNecromancer"`
	ClassesSlayer                           ScaledAsset  `json:"ClassesSlayer"`
	ClassesGladiator                        ScaledAsset  `json:"ClassesGladiator"`
	ClassesChampion                         ScaledAsset  `json:"ClassesChampion"`
	ClassesInquisitor                       ScaledAsset  `json:"ClassesInquisitor"`
	ClassesHierophant                       ScaledAsset  `json:"ClassesHierophant"`
	ClassesGuardian                         ScaledAsset  `json:"ClassesGuardian"`
	ClassesAssassin                         ScaledAsset  `json:"ClassesAssassin"`
	ClassesTrickster                        ScaledAsset  `json:"ClassesTrickster"`
	ClassesSaboteur                         ScaledAsset  `json:"ClassesSaboteur"`
	Background1                             *ScaledAsset `json:"Background1,omitempty"`
	BackgroundDex                           ScaledAsset  `json:"BackgroundDex"`
	BackgroundDexInt                        ScaledAsset  `json:"BackgroundDexInt"`
	BackgroundInt                           ScaledAsset  `json:"BackgroundInt"`
	BackgroundStr                           ScaledAsset  `json:"BackgroundStr"`
	BackgroundStrDex                        ScaledAsset  `json:"BackgroundStrDex"`
	BackgroundStrInt                        ScaledAsset  `json:"BackgroundStrInt"`
	ImgPSFadeCorner                         BasicAsset   `json:"imgPSFadeCorner"`
	ImgPSFadeSide                           BasicAsset   `json:"imgPSFadeSide"`
	ClearOil                                *BasicAsset  `json:"ClearOil,omitempty"`
	SepiaOil                                *BasicAsset  `json:"SepiaOil,omitempty"`
	AmberOil                                *BasicAsset  `json:"AmberOil,omitempty"`
	VerdantOil                              *BasicAsset  `json:"VerdantOil,omitempty"`
	TealOil                                 *BasicAsset  `json:"TealOil,omitempty"`
	AzureOil                                *BasicAsset  `json:"AzureOil,omitempty"`
	VioletOil                               *BasicAsset  `json:"VioletOil,omitempty"`
	CrimsonOil                              *BasicAsset  `json:"CrimsonOil,omitempty"`
	BlackOil                                *BasicAsset  `json:"BlackOil,omitempty"`
	OpalescentOil                           *BasicAsset  `json:"OpalescentOil,omitempty"`
	SilverOil                               *BasicAsset  `json:"SilverOil,omitempty"`
	GoldenOil                               *BasicAsset  `json:"GoldenOil,omitempty"`
	JewelSocketActiveLegion                 *ScaledAsset `json:"JewelSocketActiveLegion,omitempty"`
	JewelSocketActiveLegionAlt              *ScaledAsset `json:"JewelSocketActiveLegionAlt,omitempty"`
	JewelSocketActiveAltRed                 *ScaledAsset `json:"JewelSocketActiveAltRed,omitempty"`
	JewelSocketActiveAltBlue                *ScaledAsset `json:"JewelSocketActiveAltBlue,omitempty"`
	JewelSocketActiveAltPurple              *ScaledAsset `json:"JewelSocketActiveAltPurple,omitempty"`
	Orbit5Normal                            *ScaledAsset `json:"Orbit5Normal,omitempty"`
	Orbit5Intermediate                      *ScaledAsset `json:"Orbit5Intermediate,omitempty"`
	Orbit5Active                            *ScaledAsset `json:"Orbit5Active,omitempty"`
	Orbit6Normal                            *ScaledAsset `json:"Orbit6Normal,omitempty"`
	Orbit6Intermediate                      *ScaledAsset `json:"Orbit6Intermediate,omitempty"`
	Orbit6Active                            *ScaledAsset `json:"Orbit6Active,omitempty"`
	Background2                             *ScaledAsset `json:"Background2,omitempty"`
	PassiveMasteryConnectedButton           *ScaledAsset `json:"PassiveMasteryConnectedButton,omitempty"`
}

type BasicAsset struct {
	The1 string `json:"1"`
}

type ScaledAsset struct {
	The01246 string `json:"0.1246"`
	The02109 string `json:"0.2109"`
	The02972 string `json:"0.2972"`
	The03835 string `json:"0.3835"`
}

type Class struct {
	Name         ClassName    `json:"name"`
	BaseStr      int64        `json:"base_str"`
	BaseDex      int64        `json:"base_dex"`
	BaseInt      int64        `json:"base_int"`
	Ascendancies []Ascendancy `json:"ascendancies"`
}

type Ascendancy struct {
	ID                AscendancyName     `json:"id"`
	Name              AscendancyName     `json:"name"`
	FlavourText       *string            `json:"flavourText,omitempty"`
	FlavourTextColour *FlavourTextColour `json:"flavourTextColour,omitempty"`
	FlavourTextRect   *FlavourTextRect   `json:"flavourTextRect,omitempty"`
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
	Image Image   `json:"image"`
}

type Group struct {
	X       float64  `json:"x"`
	Y       float64  `json:"y"`
	Orbits  []int64  `json:"orbits"`
	Nodes   []string `json:"nodes"`
	IsProxy *bool    `json:"isProxy,omitempty"`
}

type Node struct {
	Group                  *int64          `json:"group,omitempty"`
	Orbit                  *int64          `json:"orbit,omitempty"`
	OrbitIndex             *int64          `json:"orbitIndex,omitempty"`
	Out                    []string        `json:"out,omitempty"`
	In                     []string        `json:"in,omitempty"`
	Skill                  *int64          `json:"skill,omitempty"`
	Name                   *string         `json:"name,omitempty"`
	Icon                   *string         `json:"icon,omitempty"`
	Stats                  []string        `json:"stats,omitempty"`
	ReminderText           []string        `json:"reminderText,omitempty"`
	IsNotable              *bool           `json:"isNotable,omitempty"`
	Recipe                 []OilType       `json:"recipe,omitempty"`
	GrantedDexterity       *int64          `json:"grantedDexterity,omitempty"`
	GrantedIntelligence    *int64          `json:"grantedIntelligence,omitempty"`
	IsMastery              *bool           `json:"isMastery,omitempty"`
	IsKeystone             *bool           `json:"isKeystone,omitempty"`
	FlavourText            []string        `json:"flavourText,omitempty"`
	AscendancyName         *AscendancyName `json:"ascendancyName,omitempty"`
	IsAscendancyStart      *bool           `json:"isAscendancyStart,omitempty"`
	GrantedStrength        *int64          `json:"grantedStrength,omitempty"`
	ClassStartIndex        *int64          `json:"classStartIndex,omitempty"`
	IsJewelSocket          *bool           `json:"isJewelSocket,omitempty"`
	ExpansionJewel         *ExpansionJewel `json:"expansionJewel,omitempty"`
	IsBlighted             *bool           `json:"isBlighted,omitempty"`
	InactiveIcon           *string         `json:"inactiveIcon,omitempty"`
	ActiveIcon             *string         `json:"activeIcon,omitempty"`
	ActiveEffectImage      *string         `json:"activeEffectImage,omitempty"`
	MasteryEffects         []MasteryEffect `json:"masteryEffects,omitempty"`
	IsMultipleChoiceOption *bool           `json:"isMultipleChoiceOption,omitempty"`
	GrantedPassivePoints   *int64          `json:"grantedPassivePoints,omitempty"`
	IsMultipleChoice       *bool           `json:"isMultipleChoice,omitempty"`
	IsProxy                *bool           `json:"isProxy,omitempty"`
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

type SkillSprites struct {
	NormalActive          []Active  `json:"normalActive"`
	NotableActive         []Active  `json:"notableActive"`
	KeystoneActive        []Active  `json:"keystoneActive"`
	NormalInactive        []Active  `json:"normalInactive"`
	NotableInactive       []Active  `json:"notableInactive"`
	KeystoneInactive      []Active  `json:"keystoneInactive"`
	Mastery               []Mastery `json:"mastery,omitempty"`
	MasteryConnected      []Mastery `json:"masteryConnected,omitempty"`
	MasteryActiveSelected []Mastery `json:"masteryActiveSelected,omitempty"`
	MasteryInactive       []Mastery `json:"masteryInactive,omitempty"`
	MasteryActiveEffect   []Mastery `json:"masteryActiveEffect,omitempty"`
}

type Active struct {
	Filename string           `json:"filename"`
	Coords   map[string]Coord `json:"coords"`
}

type Coord struct {
	X int64 `json:"x"`
	Y int64 `json:"y"`
	W int64 `json:"w"`
	H int64 `json:"h"`
}

type Mastery struct {
	Filename string           `json:"filename"`
	Coords   map[string]Coord `json:"coords"`
}

type FlavourTextColour string

type AscendancyName string

const (
	Ascendant    AscendancyName = "Ascendant"
	Assassin     AscendancyName = "Assassin"
	Berserker    AscendancyName = "Berserker"
	Champion     AscendancyName = "Champion"
	Chieftain    AscendancyName = "Chieftain"
	Deadeye      AscendancyName = "Deadeye"
	Elementalist AscendancyName = "Elementalist"
	Gladiator    AscendancyName = "Gladiator"
	Guardian     AscendancyName = "Guardian"
	Hierophant   AscendancyName = "Hierophant"
	Inquisitor   AscendancyName = "Inquisitor"
	Juggernaut   AscendancyName = "Juggernaut"
	Necromancer  AscendancyName = "Necromancer"
	Occultist    AscendancyName = "Occultist"
	Pathfinder   AscendancyName = "Pathfinder"
	Raider       AscendancyName = "Raider"
	Saboteur     AscendancyName = "Saboteur"
	Slayer       AscendancyName = "Slayer"
	Trickster    AscendancyName = "Trickster"
)

type ClassName string

const (
	Duelist  ClassName = "Duelist"
	Marauder ClassName = "Marauder"
	Ranger   ClassName = "Ranger"
	Scion    ClassName = "Scion"
	Shadow   ClassName = "Shadow"
	Templar  ClassName = "Templar"
	Witch    ClassName = "Witch"
)

type Image string

const (
	Art2DArtBaseClassIllustrationsDexIntPNG Image = "Art/2DArt/BaseClassIllustrations/DexInt.png"
	Art2DArtBaseClassIllustrationsDexPNG    Image = "Art/2DArt/BaseClassIllustrations/Dex.png"
	Art2DArtBaseClassIllustrationsIntPNG    Image = "Art/2DArt/BaseClassIllustrations/Int.png"
	Art2DArtBaseClassIllustrationsStrDexPNG Image = "Art/2DArt/BaseClassIllustrations/StrDex.png"
	Art2DArtBaseClassIllustrationsStrIntPNG Image = "Art/2DArt/BaseClassIllustrations/StrInt.png"
	Art2DArtBaseClassIllustrationsStrPNG    Image = "Art/2DArt/BaseClassIllustrations/Str.png"
)

type OilType string

const (
	AmberOil      OilType = "AmberOil"
	AzureOil      OilType = "AzureOil"
	BlackOil      OilType = "BlackOil"
	ClearOil      OilType = "ClearOil"
	CrimsonOil    OilType = "CrimsonOil"
	GoldenOil     OilType = "GoldenOil"
	IndigoOil     OilType = "IndigoOil"
	OpalescentOil OilType = "OpalescentOil"
	SepiaOil      OilType = "SepiaOil"
	SilverOil     OilType = "SilverOil"
	TealOil       OilType = "TealOil"
	VerdantOil    OilType = "VerdantOil"
	VioletOil     OilType = "VioletOil"
)

var ClassAscendancies = map[ClassName][]AscendancyName{
	Duelist:  {Slayer, Gladiator, Champion},
	Marauder: {Juggernaut, Berserker, Chieftain},
	Ranger:   {Deadeye, Pathfinder, Raider},
	Scion:    {Ascendant},
	Shadow:   {Assassin, Trickster, Saboteur},
	Templar:  {Hierophant, Inquisitor, Guardian},
	Witch:    {Elementalist, Necromancer, Occultist},
}

var ClassIDs = map[ClassName]int{
	Duelist:  4,
	Marauder: 1,
	Ranger:   2,
	Scion:    0,
	Shadow:   6,
	Templar:  5,
	Witch:    3,
}
