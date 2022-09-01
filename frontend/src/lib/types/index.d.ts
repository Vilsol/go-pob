/* eslint-disable */
export declare namespace builds {
  function ParseBuild(rawXML?: Uint8Array): [(pob.PathOfBuilding | undefined), Error];
  function ParseBuildStr(rawXML: string): [(pob.PathOfBuilding | undefined), Error];
}
export declare namespace cache {
  interface ComputationCache {
    Calculate: (arg1: string) => (raw.StatMap | undefined);
    Data?: Record<string, raw.StatMap | undefined>;
    Get(arg1: string): (raw.StatMap | undefined);
  }
  function InitializeDiskCache(arg1: (arg1: string) => Promise<(Uint8Array | undefined)>, arg2: (arg1: string, arg2?: Uint8Array) => Promise<void>, arg3: (arg1: string) => Promise<boolean>): Promise<void>;
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
    BuildOutput(mode: string): (calculator.Environment | undefined);
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
    WeaponTypes(): (Array<string> | undefined);
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
    AddList(list?: calculator.ModList): void;
    AddMod(newMod?: unknown): void;
    Clone(): (unknown | undefined);
    EvalMod(arg1?: unknown, arg2?: calculator.ListCfg): (unknown | undefined);
    Flag(cfg?: calculator.ListCfg, names?: Array<string>): boolean;
    GetCondition(arg1: string, arg2?: calculator.ListCfg, arg3: boolean): [boolean, boolean];
    GetMultiplier(arg1: string, arg2?: calculator.ListCfg, arg3: boolean): number;
    GetStat(arg1: string, arg2?: calculator.ListCfg): number;
    List(cfg?: calculator.ListCfg, names?: Array<string>): (Array<unknown | undefined> | undefined);
    More(cfg?: calculator.ListCfg, names?: Array<string>): number;
    Override(cfg?: calculator.ListCfg, names?: Array<string>): (unknown | undefined);
    Sum(modType: string, cfg?: calculator.ListCfg, names?: Array<string>): number;
  }
  interface ModList {
    ModStore?: calculator.ModStore;
    mods?: Array<unknown | undefined>;
    AddDB(db?: calculator.ModList): void;
    AddMod(newMod?: unknown): void;
    Clone(): (unknown | undefined);
    EvalMod(arg1?: unknown, arg2?: calculator.ListCfg): (unknown | undefined);
    Flag(cfg?: calculator.ListCfg, names?: Array<string>): boolean;
    GetCondition(arg1: string, arg2?: calculator.ListCfg, arg3: boolean): [boolean, boolean];
    GetMultiplier(arg1: string, arg2?: calculator.ListCfg, arg3: boolean): number;
    GetStat(arg1: string, arg2?: calculator.ListCfg): number;
    List(cfg?: calculator.ListCfg, names?: Array<string>): (Array<unknown | undefined> | undefined);
    More(cfg?: calculator.ListCfg, names?: Array<string>): number;
    Override(cfg?: calculator.ListCfg, names?: Array<string>): (unknown | undefined);
    Sum(modType: string, cfg?: calculator.ListCfg, names?: Array<string>): number;
  }
  interface ModStore {
    Parent?: unknown;
    Child?: unknown;
    Actor?: calculator.Actor;
    Multipliers?: Record<string, number>;
    Conditions?: Record<string, boolean>;
    Clone(): (calculator.ModStore | undefined);
    EvalMod(m?: unknown, cfg?: calculator.ListCfg): (unknown | undefined);
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
    Tree(): (data.Tree | undefined);
  }
  interface RequirementsTableGems {
    Source: string;
    SourceGem: pob.Gem;
    Str: number;
    Dex: number;
    Int: number;
  }
  function NewCalculator(build: pob.PathOfBuilding): (calculator.Calculator | undefined);
}
export declare namespace config {
  function InitLogging(withTime: boolean): void;
}
export declare namespace data {
  interface Ascendancy {
    ID: string;
    Name: string;
    FlavourText?: string;
    FlavourTextColour?: string;
    FlavourTextRect?: data.FlavourTextRect;
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
  interface MasteryEffect {
    Effect: number;
    Stats?: Array<string>;
    ReminderText?: Array<string>;
  }
  interface Node {
    Skill?: number;
    Name?: string;
    Icon?: string;
    IsNotable?: boolean;
    Recipe?: Array<string>;
    Stats?: Array<string>;
    Group?: number;
    Orbit?: number;
    OrbitIndex?: number;
    Out?: Array<string>;
    In?: Array<string>;
    ReminderText?: Array<string>;
    IsMastery?: boolean;
    InactiveIcon?: string;
    ActiveIcon?: string;
    ActiveEffectImage?: string;
    MasteryEffects?: Array<data.MasteryEffect>;
    GrantedStrength?: number;
    AscendancyName?: string;
    GrantedDexterity?: number;
    IsAscendancyStart?: boolean;
    IsMultipleChoice?: boolean;
    GrantedIntelligence?: number;
    IsJewelSocket?: boolean;
    ExpansionJewel?: data.ExpansionJewel;
    GrantedPassivePoints?: number;
    IsKeystone?: boolean;
    FlavourText?: Array<string>;
    IsProxy?: boolean;
    IsMultipleChoiceOption?: boolean;
    IsBlighted?: boolean;
    ClassStartIndex?: number;
  }
  interface Points {
    TotalPoints: number;
    AscendancyPoints: number;
  }
  interface Sprite {
    Filename: string;
    W: number;
    H: number;
    Coords?: Record<string, data.Coord>;
  }
  interface Sprites {
    Background?: Record<number, data.Sprite>;
    NormalActive?: Record<number, data.Sprite>;
    NotableActive?: Record<number, data.Sprite>;
    KeystoneActive?: Record<number, data.Sprite>;
    NormalInactive?: Record<number, data.Sprite>;
    NotableInactive?: Record<number, data.Sprite>;
    KeystoneInactive?: Record<number, data.Sprite>;
    Mastery?: Record<number, data.Sprite>;
    MasteryConnected?: Record<number, data.Sprite>;
    MasteryActiveSelected?: Record<number, data.Sprite>;
    MasteryInactive?: Record<number, data.Sprite>;
    MasteryActiveEffect?: Record<number, data.Sprite>;
    AscendancyBackground?: Record<number, data.Sprite>;
    Ascendancy?: Record<number, data.Sprite>;
    StartNode?: Record<number, data.Sprite>;
    GroupBackground?: Record<number, data.Sprite>;
    Frame?: Record<number, data.Sprite>;
    Jewel?: Record<number, data.Sprite>;
    Line?: Record<number, data.Sprite>;
    JewelRadius?: Record<number, data.Sprite>;
  }
  interface Tree {
    Tree: string;
    Classes?: Array<data.Class>;
    Groups?: Record<string, data.Group>;
    Nodes?: Record<string, data.Node>;
    ExtraImages?: Record<string, data.ExtraImage>;
    JewelSlots?: Array<number>;
    MinX: number;
    MinY: number;
    MaxX: number;
    MaxY: number;
    Constants: data.Constants;
    Sprites: data.Sprites;
    ImageZoomLevels?: Array<number>;
    Points: data.Points;
  }
}
export declare namespace exposition {
  interface GemPart {
    Name: string;
    Description: string;
  }
  interface SkillGem {
    MaxLevel: number;
    ID: string;
    GemType: string;
    Base: exposition.GemPart;
    Vaal: exposition.GemPart;
    Support: boolean;
    CalculateStuff(): void;
  }
  function GetRawTree(version: string): Promise<(Uint8Array | undefined)>;
  function GetSkillGems(): (Array<exposition.SkillGem> | undefined);
  function GetStatByIndex(id: number): (raw.Stat | undefined);
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
    AddNewSocketGroup(): void;
    DeleteAllSocketGroups(): void;
    DeleteSocketGroup(index: number): void;
    GetStringOption(name: string): string;
    RemoveConfigOption(name: string): void;
    SetConfigOption(value: pob.Input): void;
    SetDefaultGemLevel(gemLevel: number): void;
    SetDefaultGemQuality(gemQuality: number): void;
    SetMainSocketGroup(mainSocketGroup: number): void;
    SetMatchGemLevelToCharacterLevel(enabled: boolean): void;
    SetShowAltQualityGems(enabled: boolean): void;
    SetShowSupportGemTypes(gemTypes: string): void;
    SetSkillGroupName(skillSet: number, socketGroup: number, label: string): void;
    SetSocketGroupGems(skillSet: number, socketGroup: number, gems?: Array<pob.Gem>): void;
    SetSortGemsByDPS(enabled: boolean): void;
    SetSortGemsByDPSField(field: string): void;
    WithMainSocketGroup(mainSocketGroup: number): (pob.PathOfBuilding | undefined);
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
    GetActiveSkillTypes(): (Array<raw.ActiveSkillType | undefined> | undefined);
    GetWeaponRestrictions(): (Array<raw.ItemClass | undefined> | undefined);
  }
  interface ActiveSkillType {
    FlagStat?: number;
    ID: string;
    Key: number;
  }
  interface BaseItemType {
    DropLevel: number;
    EquipAchievementItemsKey?: number;
    FlavourTextKey?: number;
    FragmentBaseItemTypesKey?: number;
    Hash: number;
    Height: number;
    ID: string;
    IdentifyMagicAchievementItems?: Array<unknown | undefined>;
    IdentifyAchievementItems?: Array<unknown | undefined>;
    ImplicitModsKeys?: Array<number>;
    Inflection: string;
    InheritsFrom: string;
    IsCorrupted: boolean;
    ItemClassesKey: number;
    ItemVisualIdentity: number;
    ModDomain: number;
    Name: string;
    SiteVisibility: number;
    SizeOnGround: number;
    SoundEffect?: number;
    TagsKeys?: Array<number>;
    VendorRecipeAchievementItems?: Array<number>;
    Width: number;
    Key: number;
    SkillGem(): (raw.SkillGem | undefined);
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
    GetActiveSkill(): (raw.ActiveSkill | undefined);
    GetCalculatedConstantStats(): (Record<string, number> | undefined);
    GetCalculatedLevels(): (Record<number, raw.CalculatedLevel | undefined> | undefined);
    GetCalculatedStatMap(): (cache.ComputationCache | undefined);
    GetCalculatedStats(): (Array<string> | undefined);
    GetEffectQualityStats(): (Record<number, raw.GrantedEffectQualityStat | undefined> | undefined);
    GetEffectStatSetsPerLevel(): (Record<number, raw.GrantedEffectStatSetsPerLevel | undefined> | undefined);
    GetEffectsPerLevel(): (Record<number, raw.GrantedEffectsPerLevel | undefined> | undefined);
    GetExcludeTypes(): (Array<raw.ActiveSkillType | undefined> | undefined);
    GetGrantedEffectStatSet(): (raw.GrantedEffectStatSet | undefined);
    GetSkillGem(): (raw.SkillGem | undefined);
    GetSupportTypes(): (Array<raw.ActiveSkillType | undefined> | undefined);
    HasGlobalEffect(): boolean;
    Levels(): (Record<number, raw.GrantedEffectsPerLevel | undefined> | undefined);
  }
  interface GrantedEffectQualityStat {
    GrantedEffectsKey: number;
    SetID: number;
    StatsKeys?: Array<number>;
    StatsValuesPermille?: Array<number>;
    Weight: number;
    Key: number;
    GetStats(): (Array<raw.Stat | undefined> | undefined);
  }
  interface GrantedEffectStatSet {
    Key: number;
    ID: string;
    ImplicitStats?: Array<number>;
    ConstantStats?: Array<number>;
    ConstantStatsValues?: Array<number>;
    BaseEffectiveness: number;
    IncrementalEffectiveness: number;
    GetConstantStats(): (Array<raw.Stat | undefined> | undefined);
    GetImplicitStats(): (Array<raw.Stat | undefined> | undefined);
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
    GetAdditionalBooleanStats(): (Array<raw.Stat | undefined> | undefined);
    GetAdditionalStats(): (Array<raw.Stat | undefined> | undefined);
    GetFloatStats(): (Array<raw.Stat | undefined> | undefined);
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
    GetCostTypes(): (Array<raw.CostType | undefined> | undefined);
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
    GetBaseItemType(): (raw.BaseItemType | undefined);
    GetGrantedEffect(): (raw.GrantedEffect | undefined);
    GetGrantedEffects(): (Array<raw.GrantedEffect | undefined> | undefined);
    GetNonVaal(): (raw.SkillGem | undefined);
    GetSecondaryGrantedEffect(): (raw.GrantedEffect | undefined);
    GetTags(): (Record<string, raw.Tag | undefined> | undefined);
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
    Clone(): (raw.StatMap | undefined);
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