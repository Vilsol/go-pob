/* eslint-disable */
export declare namespace builds {
  function ParseBuild(rawXML?: Uint8Array): [pob.PathOfBuilding | undefined, Error];
  function ParseBuildStr(rawXML: string): [pob.PathOfBuilding | undefined, Error];
}
export declare namespace cache {
  interface ComputationCache {
    Calculate: (arg1: string) => raw.StatMap | undefined;
    Data?: Record<string, raw.StatMap | undefined>;
    Get(arg1: string): raw.StatMap | undefined;
  }
  function InitializeDiskCache(
    arg1: (arg1: string) => Promise<Uint8Array | undefined>,
    arg2: (arg1: string, arg2?: Uint8Array) => Promise<void>,
    arg3: (arg1: string) => Promise<boolean>
  ): Promise<void>;
}
export declare namespace calculator {
  interface ActiveSkill {
    SkillFlags?: Record<string, boolean>;
    SkillModList?: calculator.ModList;
    SkillCfg?: calculator.ListCfg;
    SkillTypes?: Record<string, boolean>;
    SkillData?: Record<string, unknown | undefined>;
    ActiveEffect?: calculator.GemEffect;
    Weapon1Cfg?: calculator.ListCfg;
    Weapon2Cfg?: calculator.ListCfg;
    SupportList?: Array<calculator.GemEffect | undefined>;
    Actor?: calculator.Actor;
    SocketGroup?: unknown;
    SummonSkill?: calculator.ActiveSkill;
    ConversionTable?: Record<string, calculator.ConversionTable>;
    Minion?: unknown;
    Weapon1Flags: number;
    Weapon2Flags: number;
    EffectList?: Array<calculator.GemEffect | undefined>;
    DisableReason: string;
    BaseSkillModList?: calculator.ModList;
    SlotName: string;
    MinionSkillTypes?: Record<string, boolean>;
  }
  interface Actor {
    ModDB?: calculator.ModDB;
    Level: number;
    Enemy?: calculator.Actor;
    ItemList?: Record<string, unknown | undefined>;
    ActiveSkillList?: Array<calculator.ActiveSkill | undefined>;
    Output?: Record<string, number>;
    OutputTable?: Record<string, Record<string, number> | undefined>;
    MainSkill?: calculator.ActiveSkill;
    Breakdown?: unknown;
    WeaponData1?: Record<string, unknown | undefined>;
    WeaponData2?: Record<string, unknown | undefined>;
    StrDmgBonus: number;
  }
  interface Calculator {
    PoB?: pob.PathOfBuilding;
    BuildOutput(mode: string): calculator.Environment | undefined;
  }
  interface ConversionTable {
    Targets?: Record<string, number>;
    Mult: number;
  }
  interface Environment {
    Build?: pob.PathOfBuilding;
    Mode: string;
    Spec?: calculator.PassiveSpec;
    ModDB?: calculator.ModDB;
    EnemyModDB?: calculator.ModDB;
    ItemModDB?: calculator.ModDB;
    Minion?: calculator.ModDB;
    EnemyLevel: number;
    Player?: calculator.Actor;
    Enemy?: calculator.Actor;
    RequirementsTableItems?: Record<string, unknown | undefined>;
    RequirementsTableGems?: Array<calculator.RequirementsTableGems | undefined>;
    RadiusJewelList?: Record<string, unknown | undefined>;
    ExtraRadiusNodeList?: Record<string, unknown | undefined>;
    GrantedSkills?: Record<string, unknown | undefined>;
    GrantedSkillsNodes?: Record<string, unknown | undefined>;
    GrantedSkillsItems?: Record<string, unknown | undefined>;
    Flasks?: Record<string, unknown | undefined>;
    GrantedPassives?: Record<string, unknown | undefined>;
    AuxSkillList?: Record<string, unknown | undefined>;
    ModeBuffs: boolean;
    ModeCombat: boolean;
    ModeEffective: boolean;
    KeystonesAdded?: Record<string, unknown | undefined>;
    MainSocketGroup: number;
  }
  interface GemEffect {
    GrantedEffect?: calculator.GrantedEffect;
    Level: number;
    Quality: number;
    QualityID: string;
    SrcInstance?: pob.Gem;
    GemData?: raw.SkillGem;
    GrantedEffectLevel?: raw.CalculatedLevel;
    Superseded: boolean;
    IsSupporting?: Record<pob.Gem | undefined, boolean>;
    Values?: Record<string, number>;
  }
  interface GrantedEffect {
    Raw?: raw.GrantedEffect;
    Parts?: Array<unknown | undefined>;
    SkillTypes?: Record<string, boolean>;
    BaseFlags?: Record<string, boolean>;
    BaseMultiplier(): number;
    CastTime(): number;
    DamageEffectiveness(): number;
    WeaponTypes(): Array<string> | undefined;
  }
  interface ListCfg {
    Flags?: number;
    KeywordFlags?: number;
    Source?: string;
    SkillStats?: Record<string, number>;
    SkillCond?: Record<string, boolean>;
    SlotName: string;
  }
  interface ModDB {
    ModStore?: calculator.ModStore;
    Mods?: Record<string, Array<unknown | undefined> | undefined>;
    AddDB(db?: calculator.ModDB): void;
    AddMod(newMod?: unknown): void;
    Clone(): unknown | undefined;
    EvalMod(arg1?: unknown, arg2?: calculator.ListCfg): unknown | undefined;
    Flag(cfg?: calculator.ListCfg, names?: Array<string>): boolean;
    GetCondition(arg1: string, arg2?: calculator.ListCfg, arg3: boolean): [boolean, boolean];
    GetMultiplier(arg1: string, arg2?: calculator.ListCfg, arg3: boolean): number;
    GetStat(arg1: string, arg2?: calculator.ListCfg): number;
    List(cfg?: calculator.ListCfg, names?: Array<string>): Array<unknown | undefined> | undefined;
    More(cfg?: calculator.ListCfg, names?: Array<string>): number;
    Override(cfg?: calculator.ListCfg, names?: Array<string>): unknown | undefined;
    Sum(modType: string, cfg?: calculator.ListCfg, names?: Array<string>): number;
  }
  interface ModList {
    ModStore?: calculator.ModStore;
    mods?: Array<unknown | undefined>;
    AddDB(db?: calculator.ModList): void;
    AddMod(newMod?: unknown): void;
    Clone(): unknown | undefined;
    EvalMod(arg1?: unknown, arg2?: calculator.ListCfg): unknown | undefined;
    Flag(cfg?: calculator.ListCfg, names?: Array<string>): boolean;
    GetCondition(arg1: string, arg2?: calculator.ListCfg, arg3: boolean): [boolean, boolean];
    GetMultiplier(arg1: string, arg2?: calculator.ListCfg, arg3: boolean): number;
    GetStat(arg1: string, arg2?: calculator.ListCfg): number;
    List(cfg?: calculator.ListCfg, names?: Array<string>): Array<unknown | undefined> | undefined;
    More(cfg?: calculator.ListCfg, names?: Array<string>): number;
    Override(cfg?: calculator.ListCfg, names?: Array<string>): unknown | undefined;
    Sum(modType: string, cfg?: calculator.ListCfg, names?: Array<string>): number;
  }
  interface ModStore {
    Parent?: unknown;
    Actor?: calculator.Actor;
    Multipliers?: Record<string, number>;
    Conditions?: Record<string, boolean>;
    Clone(): calculator.ModStore | undefined;
    EvalMod(m?: unknown, cfg?: calculator.ListCfg): unknown | undefined;
    GetCondition(variable: string, cfg?: calculator.ListCfg, noMod: boolean): [boolean, boolean];
    GetMultiplier(variable: string, cfg?: calculator.ListCfg, noMod: boolean): number;
    GetStat(stat: string, cfg?: calculator.ListCfg): number;
  }
  interface PassiveSpec {
    Build?: pob.PathOfBuilding;
    TreeVersion: string;
    Nodes?: Record<string, unknown | undefined>;
    AllocNodes?: Record<string, unknown | undefined>;
    AllocSubgraphNodes?: Record<string, unknown | undefined>;
    AllocExtendedNodes?: Record<string, unknown | undefined>;
    Jewels?: Record<string, unknown | undefined>;
    SubGraphs?: Record<string, unknown | undefined>;
    MasterySelections?: Record<string, unknown | undefined>;
    ClassName: string;
    AscendancyName: string;
    AllocatedNotableCount: number;
    AllocatedMasteryCount: number;
    Class(): data.Class;
    SelectAscendancyClass(ascendancyName: string): void;
    SelectClass(className: string): void;
    Tree(): data.Tree | undefined;
  }
  interface RequirementsTableGems {
    Source: string;
    SourceGem: pob.Gem;
    Str: number;
    Dex: number;
    Int: number;
  }
  function NewCalculator(build: pob.PathOfBuilding): calculator.Calculator | undefined;
}
export declare namespace config {
  function InitLogging(withTime: boolean): void;
}
export declare namespace data {
  interface Active {
    Filename: string;
    Coords?: Record<string, data.Coord>;
  }
  interface Ascendancy {
    ID: string;
    Name: string;
    FlavourText?: string;
    FlavourTextColour?: string;
    FlavourTextRect?: data.FlavourTextRect;
  }
  interface Assets {
    PSSkillFrame: data.ScaledAsset;
    PSSkillFrameHighlighted: data.ScaledAsset;
    PSSkillFrameActive: data.ScaledAsset;
    KeystoneFrameUnallocated: data.ScaledAsset;
    KeystoneFrameCanAllocate: data.ScaledAsset;
    KeystoneFrameAllocated: data.ScaledAsset;
    PSGroupBackground1: data.ScaledAsset;
    PSGroupBackground2: data.ScaledAsset;
    PSGroupBackground3: data.ScaledAsset;
    GroupBackgroundSmallAlt: data.ScaledAsset;
    GroupBackgroundMediumAlt: data.ScaledAsset;
    GroupBackgroundLargeHalfAlt: data.ScaledAsset;
    Orbit1Normal: data.ScaledAsset;
    Orbit1Intermediate: data.ScaledAsset;
    Orbit1Active: data.ScaledAsset;
    Orbit2Normal: data.ScaledAsset;
    Orbit2Intermediate: data.ScaledAsset;
    Orbit2Active: data.ScaledAsset;
    Orbit3Normal: data.ScaledAsset;
    Orbit3Intermediate: data.ScaledAsset;
    Orbit3Active: data.ScaledAsset;
    Orbit4Normal: data.ScaledAsset;
    Orbit4Intermediate: data.ScaledAsset;
    Orbit4Active: data.ScaledAsset;
    LineConnectorNormal: data.ScaledAsset;
    LineConnectorIntermediate: data.ScaledAsset;
    LineConnectorActive: data.ScaledAsset;
    PSLineDeco: data.ScaledAsset;
    PSLineDecoHighlighted: data.ScaledAsset;
    PSStartNodeBackgroundInactive: data.ScaledAsset;
    Centerduelist: data.ScaledAsset;
    Centermarauder: data.ScaledAsset;
    Centerranger: data.ScaledAsset;
    Centershadow: data.ScaledAsset;
    Centertemplar: data.ScaledAsset;
    Centerwitch: data.ScaledAsset;
    Centerscion: data.ScaledAsset;
    PSPointsFrame: data.BasicAsset;
    NotableFrameUnallocated: data.ScaledAsset;
    NotableFrameCanAllocate: data.ScaledAsset;
    NotableFrameAllocated: data.ScaledAsset;
    BlightedNotableFrameUnallocated: data.ScaledAsset;
    BlightedNotableFrameCanAllocate: data.ScaledAsset;
    BlightedNotableFrameAllocated: data.ScaledAsset;
    JewelFrameUnallocated: data.ScaledAsset;
    JewelFrameCanAllocate: data.ScaledAsset;
    JewelFrameAllocated: data.ScaledAsset;
    JewelSocketActiveBlue: data.ScaledAsset;
    JewelSocketActiveGreen: data.ScaledAsset;
    JewelSocketActiveRed: data.ScaledAsset;
    JewelSocketActivePrismatic: data.ScaledAsset;
    JewelSocketActiveAbyss: data.ScaledAsset;
    JewelCircle1: data.BasicAsset;
    JewelCircle1Inverse: data.BasicAsset;
    VaalJewelCircle1: data.BasicAsset;
    VaalJewelCircle2: data.BasicAsset;
    KaruiJewelCircle1: data.BasicAsset;
    KaruiJewelCircle2: data.BasicAsset;
    MarakethJewelCircle1: data.BasicAsset;
    MarakethJewelCircle2: data.BasicAsset;
    TemplarJewelCircle1: data.BasicAsset;
    TemplarJewelCircle2: data.BasicAsset;
    EternalEmpireJewelCircle1: data.BasicAsset;
    EternalEmpireJewelCircle2: data.BasicAsset;
    JewelSocketAltNormal: data.ScaledAsset;
    JewelSocketAltCanAllocate: data.ScaledAsset;
    JewelSocketAltActive: data.ScaledAsset;
    JewelSocketActiveBlueAlt: data.ScaledAsset;
    JewelSocketActiveGreenAlt: data.ScaledAsset;
    JewelSocketActiveRedAlt: data.ScaledAsset;
    JewelSocketActivePrismaticAlt: data.ScaledAsset;
    JewelSocketActiveAbyssAlt: data.ScaledAsset;
    JewelSocketClusterAltNormal1Small: data.ScaledAsset;
    JewelSocketClusterAltCanAllocate1Small: data.ScaledAsset;
    JewelSocketClusterAltNormal1Medium: data.ScaledAsset;
    JewelSocketClusterAltCanAllocate1Medium: data.ScaledAsset;
    JewelSocketClusterAltNormal1Large: data.ScaledAsset;
    JewelSocketClusterAltCanAllocate1Large: data.ScaledAsset;
    AscendancyButton: data.ScaledAsset;
    AscendancyButtonHighlight: data.ScaledAsset;
    AscendancyButtonPressed: data.ScaledAsset;
    AscendancyFrameLargeAllocated: data.ScaledAsset;
    AscendancyFrameLargeCanAllocate: data.ScaledAsset;
    AscendancyFrameLargeNormal: data.ScaledAsset;
    AscendancyFrameSmallAllocated: data.ScaledAsset;
    AscendancyFrameSmallCanAllocate: data.ScaledAsset;
    AscendancyFrameSmallNormal: data.ScaledAsset;
    AscendancyMiddle: data.ScaledAsset;
    ClassesAscendant: data.ScaledAsset;
    ClassesJuggernaut: data.ScaledAsset;
    ClassesBerserker: data.ScaledAsset;
    ClassesChieftain: data.ScaledAsset;
    ClassesRaider: data.ScaledAsset;
    ClassesDeadeye: data.ScaledAsset;
    ClassesPathfinder: data.ScaledAsset;
    ClassesOccultist: data.ScaledAsset;
    ClassesElementalist: data.ScaledAsset;
    ClassesNecromancer: data.ScaledAsset;
    ClassesSlayer: data.ScaledAsset;
    ClassesGladiator: data.ScaledAsset;
    ClassesChampion: data.ScaledAsset;
    ClassesInquisitor: data.ScaledAsset;
    ClassesHierophant: data.ScaledAsset;
    ClassesGuardian: data.ScaledAsset;
    ClassesAssassin: data.ScaledAsset;
    ClassesTrickster: data.ScaledAsset;
    ClassesSaboteur: data.ScaledAsset;
    Background1?: data.ScaledAsset;
    BackgroundDex: data.ScaledAsset;
    BackgroundDexInt: data.ScaledAsset;
    BackgroundInt: data.ScaledAsset;
    BackgroundStr: data.ScaledAsset;
    BackgroundStrDex: data.ScaledAsset;
    BackgroundStrInt: data.ScaledAsset;
    ImgPSFadeCorner: data.BasicAsset;
    ImgPSFadeSide: data.BasicAsset;
    ClearOil?: data.BasicAsset;
    SepiaOil?: data.BasicAsset;
    AmberOil?: data.BasicAsset;
    VerdantOil?: data.BasicAsset;
    TealOil?: data.BasicAsset;
    AzureOil?: data.BasicAsset;
    VioletOil?: data.BasicAsset;
    CrimsonOil?: data.BasicAsset;
    BlackOil?: data.BasicAsset;
    OpalescentOil?: data.BasicAsset;
    SilverOil?: data.BasicAsset;
    GoldenOil?: data.BasicAsset;
    JewelSocketActiveLegion?: data.ScaledAsset;
    JewelSocketActiveLegionAlt?: data.ScaledAsset;
    JewelSocketActiveAltRed?: data.ScaledAsset;
    JewelSocketActiveAltBlue?: data.ScaledAsset;
    JewelSocketActiveAltPurple?: data.ScaledAsset;
    Orbit5Normal?: data.ScaledAsset;
    Orbit5Intermediate?: data.ScaledAsset;
    Orbit5Active?: data.ScaledAsset;
    Orbit6Normal?: data.ScaledAsset;
    Orbit6Intermediate?: data.ScaledAsset;
    Orbit6Active?: data.ScaledAsset;
    Background2?: data.ScaledAsset;
    PassiveMasteryConnectedButton?: data.ScaledAsset;
  }
  interface BasicAsset {
    The1: string;
  }
  interface CharacterAttributes {
    Strength: number;
    Dexterity: number;
    Intelligence: number;
  }
  interface Class {
    Name: string;
    BaseStr: number;
    BaseDex: number;
    BaseInt: number;
    Ascendancies?: Array<data.Ascendancy>;
  }
  interface Classes {
    StrDexIntClass: number;
    StrClass: number;
    DexClass: number;
    IntClass: number;
    StrDexClass: number;
    StrIntClass: number;
    DexIntClass: number;
  }
  interface Constants {
    Classes: data.Classes;
    CharacterAttributes: data.CharacterAttributes;
    PSSCentreInnerRadius: number;
    SkillsPerOrbit?: Array<number>;
    OrbitRadii?: Array<number>;
  }
  interface Coord {
    X: number;
    Y: number;
    W: number;
    H: number;
  }
  interface ExpansionJewel {
    Size: number;
    Index: number;
    Proxy: string;
    Parent?: string;
  }
  interface ExtraImage {
    X: number;
    Y: number;
    Image: string;
  }
  interface FlavourTextRect {
    X: number;
    Y: number;
    Width: number;
    Height: number;
  }
  interface Group {
    X: number;
    Y: number;
    Orbits?: Array<number>;
    Nodes?: Array<string>;
    IsProxy?: boolean;
  }
  interface Mastery {
    Filename: string;
    Coords?: Record<string, data.Coord>;
  }
  interface MasteryEffect {
    Effect: number;
    Stats?: Array<string>;
    ReminderText?: Array<string>;
  }
  interface Node {
    Group?: number;
    Orbit?: number;
    OrbitIndex?: number;
    Out?: Array<string>;
    In?: Array<string>;
    Skill?: number;
    Name?: string;
    Icon?: string;
    Stats?: Array<string>;
    ReminderText?: Array<string>;
    IsNotable?: boolean;
    Recipe?: Array<string>;
    GrantedDexterity?: number;
    GrantedIntelligence?: number;
    IsMastery?: boolean;
    IsKeystone?: boolean;
    FlavourText?: Array<string>;
    AscendancyName?: string;
    IsAscendancyStart?: boolean;
    GrantedStrength?: number;
    ClassStartIndex?: number;
    IsJewelSocket?: boolean;
    ExpansionJewel?: data.ExpansionJewel;
    IsBlighted?: boolean;
    InactiveIcon?: string;
    ActiveIcon?: string;
    ActiveEffectImage?: string;
    MasteryEffects?: Array<data.MasteryEffect>;
    IsMultipleChoiceOption?: boolean;
    GrantedPassivePoints?: number;
    IsMultipleChoice?: boolean;
    IsProxy?: boolean;
  }
  interface Points {
    TotalPoints: number;
    AscendancyPoints: number;
  }
  interface ScaledAsset {
    The01246: string;
    The02109: string;
    The02972: string;
    The03835: string;
  }
  interface SkillSprites {
    NormalActive?: Array<data.Active>;
    NotableActive?: Array<data.Active>;
    KeystoneActive?: Array<data.Active>;
    NormalInactive?: Array<data.Active>;
    NotableInactive?: Array<data.Active>;
    KeystoneInactive?: Array<data.Active>;
    Mastery?: Array<data.Mastery>;
    MasteryConnected?: Array<data.Mastery>;
    MasteryActiveSelected?: Array<data.Mastery>;
    MasteryInactive?: Array<data.Mastery>;
    MasteryActiveEffect?: Array<data.Mastery>;
  }
  interface Tree {
    Classes?: Array<data.Class>;
    Groups?: Record<string, data.Group>;
    Nodes?: Record<string, data.Node>;
    ExtraImages?: Record<string, data.ExtraImage>;
    MinX: number;
    MinY: number;
    MaxX: number;
    MaxY: number;
    Assets: data.Assets;
    Constants: data.Constants;
    SkillSprites: data.SkillSprites;
    ImageZoomLevels?: Array<number>;
    JewelSlots?: Array<number>;
    Tree?: string;
    Points?: data.Points;
  }
}
export declare namespace pob {
  interface Build {
    PantheonMinorGod: string;
    PantheonMajorGod: string;
    Bandit: string;
    ViewMode: string;
    ClassName: string;
    AscendClassName: string;
    Level: number;
    MainSocketGroup: number;
    TargetVersion: string;
    PlayerStats?: Array<pob.PlayerStat>;
  }
  interface Calcs {
    Inputs?: Array<pob.Input>;
    Sections?: Array<pob.Section>;
  }
  interface Config {
    Inputs?: Array<pob.Input>;
    Placeholders?: Array<pob.Input>;
  }
  interface Gem {
    Quality: number;
    SkillPart: number;
    EnableGlobal2: boolean;
    SkillPartCalcs: number;
    QualityID: string;
    GemID: string;
    Enabled: boolean;
    Count: number;
    EnableGlobal1: boolean;
    NameSpec: string;
    Level: number;
    SkillID: string;
    SkillMinionItemSet: number;
    SkillMinion: string;
    DisplayEffect?: unknown;
    SupportEffect?: unknown;
  }
  interface Input {
    Name: string;
    Boolean?: boolean;
    Number?: number;
    String?: string;
  }
  interface ItemSet {
    ID: string;
    UseSecondWeaponSet?: boolean;
    Slots?: Array<pob.Slot>;
  }
  interface Items {
    ActiveItemSet: number;
    UseSecondWeaponSet?: boolean;
    ItemSets?: Array<pob.ItemSet>;
  }
  interface PathOfBuilding {
    Build: pob.Build;
    Tree: pob.Tree;
    Calcs: pob.Calcs;
    Notes: string;
    Items: pob.Items;
    Skills: pob.Skills;
    TreeView: pob.TreeView;
    Config: pob.Config;
    WithMainSocketGroup(mainSocketGroup: number): pob.PathOfBuilding | undefined;
  }
  interface PlayerStat {
    Value: number;
    Stat: string;
  }
  interface Section {
    Collapsed: boolean;
    ID: string;
    Subsection: string;
  }
  interface Skill {
    MainActiveSkillCalcs: number;
    MainActiveSkill: number;
    Label: string;
    Enabled: boolean;
    IncludeInFullDPS?: boolean;
    Gems?: Array<pob.Gem>;
    Slot: string;
    SlotEnabled: boolean;
    Source?: unknown;
    DisplayLabel: string;
    DisplaySkillList?: unknown;
    DisplaySkillListCalcs?: unknown;
  }
  interface SkillSet {
    ID: number;
    Skills?: Array<pob.Skill>;
  }
  interface Skills {
    SortGemsByDPSField: string;
    ShowSupportGemTypes: string;
    DefaultGemLevel?: number;
    MatchGemLevelToCharacterLevel: boolean;
    ShowAltQualityGems: boolean;
    DefaultGemQuality?: number;
    ActiveSkillSet: number;
    SortGemsByDPS: boolean;
    SkillSets?: Array<pob.SkillSet>;
  }
  interface Slot {
    ItemID: number;
    Name: string;
  }
  interface Spec {
    ClassID: number;
    AscendClassID: number;
    TreeVersion: string;
    Nodes: string;
    MasteryEffects: string;
    URL: string;
  }
  interface Tree {
    ActiveSpec: number;
    Specs?: Array<pob.Spec>;
  }
  interface TreeView {
    ZoomLevel: number;
    ZoomX: number;
    ZoomY: number;
    SearchStr: string;
    ShowHeatMap?: boolean;
    ShowStatDifferences: boolean;
  }
  function CompressEncode(xml: string): [string, Error];
  function DecodeDecompress(code: string): [string, Error];
}
export declare namespace raw {
  interface ActiveSkill {
    AIFile: string;
    ActiveSkillTargetTypes?: Array<number>;
    ActiveSkillTypes?: Array<number>;
    AlternateSkillTargetingBehavioursKey?: number;
    Description: string;
    DisplayedName: string;
    IconDDSFile: string;
    ID: string;
    InputStatKeys?: Array<number>;
    IsManuallyCasted: boolean;
    MinionActiveSkillTypes?: Array<number>;
    OutputStatKeys?: Array<number>;
    SkillTotemID: number;
    WeaponRestrictionItemClassesKeys?: Array<number>;
    WebsiteDescription: string;
    WebsiteImage: string;
    Key: number;
    GetActiveSkillTypes(): Array<raw.ActiveSkillType | undefined> | undefined;
    GetWeaponRestrictions(): Array<raw.ItemClass | undefined> | undefined;
  }
  interface ActiveSkillType {
    FlagStat?: number;
    ID: string;
    Key: number;
  }
  interface CalculatedLevel {
    Level: number;
    Values?: Array<number>;
    Cost?: Record<string, number>;
    StatInterpolation?: Array<number>;
    LevelRequirement: number;
    ManaReservationFlat?: number;
    ManaReservationPercent?: number;
    LifeReservationFlat?: number;
    LifeReservationPercent?: number;
    ManaMultiplier?: number;
    DamageEffectiveness?: number;
    CritChance?: number;
    BaseMultiplier?: number;
    AttackSpeedMultiplier?: number;
    AttackTime?: number;
    Cooldown?: number;
    SoulCost?: number;
    SkillUseStorage?: number;
    SoulPreventionDuration?: number;
  }
  interface CostType {
    FormatText: string;
    ID: string;
    StatsKey: number;
    Key: number;
  }
  interface GrantedEffect {
    ID: string;
    IsSupport: boolean;
    SupportTypes?: Array<number>;
    SupportGemLetter: string;
    Attribute: number;
    AddTypes?: Array<number>;
    ExcludeTypes?: Array<number>;
    SupportsGemsOnly: boolean;
    CannotBeSupported: boolean;
    CastTime: number;
    ActiveSkill?: number;
    IgnoreMinionTypes: boolean;
    AddMinionTypes?: Array<number>;
    Animation?: number;
    WeaponRestrictions?: Array<number>;
    PlusVersionOf?: number;
    GrantedEffectStatSets: number;
    Key: number;
    calculatedStats?: Array<string>;
    calculatedLevels?: Record<number, raw.CalculatedLevel | undefined>;
    calculatedConstantStats?: Record<string, number>;
    calculatedStatMap?: cache.ComputationCache;
    Calculate(): void;
    GetActiveSkill(): raw.ActiveSkill | undefined;
    GetCalculatedConstantStats(): Record<string, number> | undefined;
    GetCalculatedLevels(): Record<number, raw.CalculatedLevel | undefined> | undefined;
    GetCalculatedStatMap(): cache.ComputationCache | undefined;
    GetCalculatedStats(): Array<string> | undefined;
    GetEffectQualityStats(): Record<number, raw.GrantedEffectQualityStat | undefined> | undefined;
    GetEffectStatSetsPerLevel(): Record<number, raw.GrantedEffectStatSetsPerLevel | undefined> | undefined;
    GetEffectsPerLevel(): Record<number, raw.GrantedEffectsPerLevel | undefined> | undefined;
    GetExcludeTypes(): Array<raw.ActiveSkillType | undefined> | undefined;
    GetGrantedEffectStatSet(): raw.GrantedEffectStatSet | undefined;
    GetSupportTypes(): Array<raw.ActiveSkillType | undefined> | undefined;
    HasGlobalEffect(): boolean;
    Levels(): Record<number, raw.GrantedEffectsPerLevel | undefined> | undefined;
  }
  interface GrantedEffectQualityStat {
    GrantedEffectsKey: number;
    SetID: number;
    StatsKeys?: Array<number>;
    StatsValuesPermille?: Array<number>;
    Weight: number;
    Key: number;
    GetStats(): Array<raw.Stat | undefined> | undefined;
  }
  interface GrantedEffectStatSet {
    Key: number;
    ID: string;
    ImplicitStats?: Array<number>;
    ConstantStats?: Array<number>;
    ConstantStatsValues?: Array<number>;
    BaseEffectiveness: number;
    IncrementalEffectiveness: number;
    GetConstantStats(): Array<raw.Stat | undefined> | undefined;
    GetImplicitStats(): Array<raw.Stat | undefined> | undefined;
  }
  interface GrantedEffectStatSetsPerLevel {
    AdditionalBooleanStats?: Array<number>;
    AdditionalStats?: Array<number>;
    AdditionalStatsValues?: Array<number>;
    AttackCritChance: number;
    BaseMultiplier: number;
    BaseResolvedValues?: Array<number>;
    DamageEffectiveness: number;
    FloatStats?: Array<number>;
    FloatStatsValues?: Array<number>;
    GemLevel: number;
    GrantedEffects?: Array<number>;
    InterpolationBases?: Array<number>;
    PlayerLevelReq: number;
    OffhandCritChance: number;
    StatInterpolations?: Array<number>;
    StatSet: number;
    Key: number;
    GetAdditionalBooleanStats(): Array<raw.Stat | undefined> | undefined;
    GetAdditionalStats(): Array<raw.Stat | undefined> | undefined;
    GetFloatStats(): Array<raw.Stat | undefined> | undefined;
  }
  interface GrantedEffectsPerLevel {
    AttackSpeedMultiplier: number;
    AttackTime: number;
    Cooldown: number;
    CooldownBypassType: number;
    CooldownGroup: number;
    CostAmounts?: Array<number>;
    CostMultiplier: number;
    CostTypes?: Array<number>;
    GrantedEffect: number;
    Level: number;
    LifeReservationFlat: number;
    LifeReservationPercent: number;
    ManaReservationFlat: number;
    ManaReservationPercent: number;
    PlayerLevelReq: number;
    SoulGainPreventionDuration: number;
    StoredUses: number;
    VaalSouls: number;
    VaalStoredUses: number;
    Key: number;
    GetCostTypes(): Array<raw.CostType | undefined> | undefined;
  }
  interface ItemClass {
    AllocateToMapOwner: boolean;
    AlwaysAllocate: boolean;
    AlwaysShow: boolean;
    CanBeCorrupted: boolean;
    CanBeDoubleCorrupted: boolean;
    CanHaveAspects: boolean;
    CanHaveIncubators: boolean;
    CanHaveInfluence: boolean;
    CanHaveVeiledMods: boolean;
    CanScourge: boolean;
    CanTransferSkin: boolean;
    Flags?: Array<number>;
    ID: string;
    ItemClassCategory?: number;
    ItemStance?: number;
    Name: string;
    RemovedIfLeavesArea: boolean;
    Key: number;
  }
  interface SkillGem {
    BaseItemType: number;
    GrantedEffect: number;
    Str: number;
    Dex: number;
    Int: number;
    Tags?: Array<number>;
    VaalGem?: number;
    IsVaalGem: boolean;
    Description: string;
    HungryLoopMod?: number;
    SecondaryGrantedEffect?: number;
    GlobalGemLevelStat?: number;
    SecondarySupportName: string;
    AwakenedVariant?: number;
    RegularVariant?: number;
    Key: number;
    DefaultLevel(): number;
    GetGrantedEffect(): raw.GrantedEffect | undefined;
    GetGrantedEffects(): Array<raw.GrantedEffect | undefined> | undefined;
    GetSecondaryGrantedEffect(): raw.GrantedEffect | undefined;
    GetTags(): Record<string, raw.Tag | undefined> | undefined;
  }
  interface Stat {
    BelongsStatsKey?: Array<string>;
    Category?: number;
    ContextFlags?: Array<number>;
    Hash32: number;
    ID: string;
    IsLocal: boolean;
    IsScalable: boolean;
    IsVirtual: boolean;
    IsWeaponLocal: boolean;
    MainHandAliasStatsKey?: number;
    OffHandAliasStatsKey?: number;
    Semantics: number;
    Text: string;
    Key: number;
  }
  interface StatMap {
    Mods?: Array<unknown | undefined>;
    Value?: number;
    Mult?: number;
    Div?: number;
    Base?: number;
    Clone(): raw.StatMap | undefined;
  }
  interface Tag {
    DisplayString: string;
    ID: string;
    Name: string;
    Key: number;
  }
  function InitializeAll(version: string, updateFunc: (arg1: string) => Promise<void>): Promise<Error>;
}
export const initializeCrystalline: () => void;
