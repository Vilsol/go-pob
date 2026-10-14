/* eslint-disable */

/// <reference lib="es2018" />
/// <reference lib="dom" />
/// <reference lib="esnext.disposable" />
export type Result<T> = (
  | {
      /** Whether the call succeeded. */
      readonly ok: true;
      /** The value the call produced. */
      readonly value: T;
    }
  | {
      /** Whether the call succeeded. */
      readonly ok: false;
      /** Why the call failed. */
      readonly error: Error;
    }
) & {
  /** Returns the value, throwing the error if the call failed. */
  unwrap(): T;
  /** Returns the value, or the fallback if the call failed. */
  unwrapOr(fallback: T): T;
};
export declare namespace builds {
  interface SavedBuild {
    ClassName: string;
    Level: number;
    Ascendancy: string;
    MainSkill: string;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  function EmptyBuild(): (pob.PathOfBuilding | undefined);
  function ParseBuild(rawXML: Uint8Array | undefined): Result<(pob.PathOfBuilding | undefined)>;
  function ParseBuildOver(live: pob.PathOfBuilding | undefined, liveXML: string, rawXML: string): Result<(pob.PathOfBuilding | undefined)>;
  function ParseBuildStr(rawXML: string): Result<(pob.PathOfBuilding | undefined)>;
  function SavedBuildSummary(rawXML: string): Result<builds.SavedBuild>;
  function SerializeBuild(build: pob.PathOfBuilding | undefined): Result<(Uint8Array | undefined)>;
  function SerializeBuildWithStats(build: pob.PathOfBuilding | undefined, player: Array<pob.PlayerStat> | undefined, fullDPS: Array<pob.FullDPSSkill> | undefined, minion: Array<pob.PlayerStat> | undefined): Result<(Uint8Array | undefined)>;
}
export declare namespace calculator {
  interface ActiveSkill {
    SkillFlags?: Record<calculator.SkillFlag, boolean>;
    readonly SkillModList?: moddb.ModList;
    readonly SkillCfg?: moddb.ListCfg;
    SkillTypes?: Record<data.SkillType, boolean>;
    SkillData?: calculator.SkillData;
    readonly ActiveEffect?: calculator.GemEffect;
    readonly Weapon1Cfg?: moddb.ListCfg;
    readonly Weapon2Cfg?: moddb.ListCfg;
    readonly SupportList?: Array<calculator.GemEffect | undefined>;
    readonly TriggeredBy?: calculator.GemEffect;
    readonly Actor?: calculator.Actor;
    SocketGroupIndex: number;
    readonly SummonSkill?: calculator.ActiveSkill;
    ConversionTable?: Record<data.DamageType, Readonly<calculator.ConversionTable>>;
    readonly Minion?: calculator.ActorMinion;
    MinionList?: Array<string>;
    Weapon1Flags: mod.MFlag;
    Weapon2Flags: mod.MFlag;
    readonly EffectList?: Array<calculator.GemEffect | undefined>;
    DisableReason: string;
    readonly BaseSkillModList?: moddb.ModList;
    SlotName: string;
    MinionSkillTypes?: Record<data.SkillType, boolean>;
    readonly BleedCfg?: moddb.ListCfg;
    readonly OHBleedCfg?: moddb.ListCfg;
    readonly PoisonCfg?: moddb.ListCfg;
    readonly OHPoisonCfg?: moddb.ListCfg;
    readonly IgniteCfg?: moddb.ListCfg;
    readonly OHIgniteCfg?: moddb.ListCfg;
    readonly DecayCfg?: moddb.ListCfg;
    readonly DotCfg?: moddb.ListCfg;
    readonly DotTypeCfg?: Record<data.DamageType, moddb.ListCfg | undefined>;
    SkillTotemId: number;
    SkillPart: number;
    SkillPartName: string;
    ActiveMineCount?: number;
    ActiveStageCount: number;
    readonly BuffList?: Array<calculator.Buff | undefined>;
    BuffSkill: boolean;
    MinionBuffSkill: boolean;
    DebuffSkill: boolean;
    Mirage?: calculator.MirageResult;
    Marked: boolean;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface Actor {
    readonly ModDB?: moddb.ModDB;
    Level: number;
    readonly Enemy?: calculator.Actor;
    readonly Player?: calculator.Actor;
    readonly Parent?: calculator.Actor;
    readonly MinionData?: raw.MinionInfo;
    readonly ItemList?: Record<string, calculator.ItemData | undefined>;
    readonly ActiveSkillList?: Array<calculator.ActiveSkill | undefined>;
    Output?: Record<string, number>;
    OutputTable?: Record<calculator.OutTable, Record<string, number> | undefined>;
    OutputStrings?: Record<string, string>;
    SkillDPS?: Array<calculator.SkillDPS>;
    readonly MainSkill?: calculator.ActiveSkill;
    readonly Breakdown?: calculator.Breakdown;
    WeaponData1?: calculator.SkillData;
    WeaponData2?: calculator.SkillData;
    WeaponRange1: number;
    WeaponRange2: number;
    StrDmgBonus: number;
    DamageShiftTable?: Record<data.DamageType, Record<data.DamageType, number> | undefined>;
    DamageOverTimeShiftTable?: Record<data.DamageType, Record<data.DamageType, number> | undefined>;
    AddedUsingCondition(v: string): [boolean, boolean];
    AuraLifeReservedBase(skillName: string, summonSkillName: string): [number, boolean];
    AuraManaReservedBase(skillName: string, summonSkillName: string): [number, boolean];
    EquippedSlots(): (Array<string> | undefined);
    GetOutput(stat: string): [number, boolean];
    ItemCorruptedIs(slot: string, corrupted: boolean): boolean;
    ItemElderIs(slot: string, elder: boolean): boolean;
    ItemInSlot(slot: string): boolean;
    ItemModifierMatches(slot: string, searchCond: string): boolean;
    ItemNameIs(slot: string, name: string): boolean;
    ItemRarityIs(slot: string, rarity: string): boolean;
    ItemShaperIs(slot: string, shaper: boolean): boolean;
    MonsterTags(): [(Array<string> | undefined), boolean];
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ActorMinion {
    readonly Actor?: calculator.Actor;
    Type: string;
    LifeTable?: Array<number>;
    Uses?: Record<string, boolean>;
    ItemSet?: pob.ItemSet;
    AddedUsingCondition(v: string): [boolean, boolean];
    AuraLifeReservedBase(skillName: string, summonSkillName: string): [number, boolean];
    AuraManaReservedBase(skillName: string, summonSkillName: string): [number, boolean];
    EquippedSlots(): (Array<string> | undefined);
    GetOutput(stat: string): [number, boolean];
    ItemCorruptedIs(slot: string, corrupted: boolean): boolean;
    ItemElderIs(slot: string, elder: boolean): boolean;
    ItemInSlot(slot: string): boolean;
    ItemModifierMatches(slot: string, searchCond: string): boolean;
    ItemNameIs(slot: string, name: string): boolean;
    ItemRarityIs(slot: string, rarity: string): boolean;
    ItemShaperIs(slot: string, shaper: boolean): boolean;
    MonsterTags(): [(Array<string> | undefined), boolean];
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface AffixControl {
    readonly Type: string;
    readonly Series?: Array<calculator.AffixSeries>;
    readonly Sel: number;
    readonly SliderShown: boolean;
    readonly SliderVal: number;
    readonly DivCount: number;
  }
  interface AffixControls {
    readonly Sorting: boolean;
    readonly Controls?: Array<calculator.AffixControl>;
  }
  interface AffixSeries {
    readonly Label: string;
    readonly ModList?: Array<string>;
    readonly HaveRange: boolean;
    readonly Tooltip?: Array<string>;
    readonly Mid?: Array<string>;
  }
  interface AnointNotable {
    readonly ID: number;
    readonly Name: string;
    readonly Stats?: Array<string>;
    readonly ReminderText?: Array<string>;
    readonly Recipe?: Array<string>;
    readonly Unsupported?: Array<boolean>;
  }
  interface ArmourData {
    Ward: number;
    EnergyShield: number;
    Armour: number;
    Evasion: number;
    BlockChance: number;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface BCol {
    Label: string;
    Key: string;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface BDamageType {
    Source: string;
    ConvSrc?: string;
    Total: string;
    ConvDst?: string;
    GainDst?: string;
    Base?: string;
    Inc?: string;
    More?: string;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface BMultiChain {
    Base?: calculator.BMultiChainItem;
    Label: string;
    Total: string;
    NoTotal: boolean;
    Items?: Array<calculator.BMultiChainItem>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface BMultiChainItem {
    Format: string;
    Value: number;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface BReservation {
    SkillName: string;
    Base: string;
    Mult?: string;
    More?: string;
    Inc?: string;
    Efficiency?: string;
    EfficiencyMore?: string;
    Total: string;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface BSlot {
    Base: number;
    Inc: string;
    More: string;
    Total: string;
    Source: string;
    SourceName?: string;
    readonly Item?: calculator.ItemData;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface Breakdown {
    readonly ModDB?: moddb.ModDB;
    Output?: Record<string, number>;
    readonly Actor?: calculator.Actor;
    readonly Data?: Record<string, calculator.Entry | undefined>;
    AddCol(key: string, cols: Array<calculator.BCol> | undefined): void;
    AddLine(key: string, s: Array<string> | undefined): void;
    AddRow(key: string, rows: Array<Record<string, string> | undefined> | undefined): void;
    DamageType(key: string, damageTypes: Array<calculator.BDamageType> | undefined): void;
    EffMult(key: string, damageType: data.DamageType, resist: number, pen: number, taken: number, mult: number, takenMore: number, sourceRes: data.DamageType, useRes: boolean, invertChance: number, effectiveResist: number | undefined): void;
    GetData(): (Record<string, calculator.EntryPlain | undefined> | undefined);
    Has(key: string): boolean;
    MultiChain(key: string, chains: Array<calculator.BMultiChain> | undefined): void;
    Reservation(key: string, reservation: Array<calculator.BReservation> | undefined): void;
    SetLabel(key: string, label: string): void;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface Buff {
    Type: string;
    Name: string;
    ActiveSkillBuff: boolean;
    ApplyNotPlayer: boolean;
    ApplyMinions: boolean;
    ApplyAllies: boolean;
    AllowTotemBuff: boolean;
    Cond: string;
    EnemyCond: string;
    StackVar: string;
    StackLimitVar: string;
    StackLimit: number;
    readonly ModList?: moddb.ModList;
    readonly UnscalableModList?: moddb.ModList;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface Calculator {
    PoB?: pob.PathOfBuilding;
    AlignCalcsSkillToMain: boolean;
    BuildOutput(mode: calculator.OutputMode): Promise<(calculator.Environment | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ClassChange {
    readonly Changed: boolean;
    readonly NeedsConfirm: boolean;
    readonly ClassName: string;
    readonly Handled: boolean;
  }
  interface ClusterInfo {
    readonly Skills?: Array<calculator.ClusterSkillChoice>;
    readonly Skill: string;
    readonly NodeCount: number;
    readonly Min: number;
    readonly Max: number;
  }
  interface ClusterSkillChoice {
    readonly ID: string;
    readonly Label: string;
  }
  interface ColProps {
    readonly Cfg: string;
    readonly ModSource?: string;
    readonly Enemy?: boolean;
    readonly ModName?: Array<string>;
    readonly ModType: mod.Type;
    readonly Actor: string;
  }
  interface CompareJewelSlot {
    readonly Label: string;
    readonly NodeID: number;
    readonly PText: string;
    readonly CText: string;
    readonly PAllocated: boolean;
    readonly CAllocated: boolean;
  }
  interface CompareModRow {
    readonly Key: string;
    readonly Value: number;
    readonly Display: string;
    readonly SourceType: string;
    readonly SourceName: string;
    readonly ModName: string;
  }
  interface ComparePowerBuilder {
    Progress(): number;
    Report(): (Array<calculator.ComparePowerEntry> | undefined);
    Stats(): (Array<calculator.PowerStat> | undefined);
    StepMs(budgetMs: number): number;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ComparePowerCategories {
    readonly TreeNodes: boolean;
    readonly Items: boolean;
    readonly SkillGems: boolean;
    readonly SupportGems: boolean;
    readonly Config: boolean;
  }
  interface ComparePowerEntry {
    readonly Category: string;
    readonly Name: string;
    readonly NodeID: number;
    readonly SlotName: string;
    readonly Var: string;
    readonly Rarity: string;
    readonly PathDist: number;
    readonly Impact?: Array<number>;
  }
  interface CompareSkillBreakdownRow {
    readonly Value: string;
    readonly Name: string;
    readonly Source: string;
    readonly SourceName: string;
  }
  interface ConfigRelevance {
    readonly Conds?: Array<string>;
    readonly MinionConds?: Array<string>;
    readonly EnemyConds?: Array<string>;
    readonly CondTrue?: Array<string>;
    readonly Mults?: Array<string>;
    readonly EnemyMults?: Array<string>;
    readonly Stats?: Array<string>;
    readonly EnemyStats?: Array<string>;
    readonly TagTypes?: Array<string>;
    readonly Flags?: Array<string>;
    readonly Mods?: Array<string>;
    readonly Skills?: Array<string>;
    readonly TransfiguredSkills?: Array<string>;
    readonly SkillFlags?: Array<string>;
    readonly SkillData?: Array<string>;
  }
  interface ConversionTable {
    Targets?: Record<data.DamageType, number>;
    Mult: number;
    Conversion?: Record<data.DamageType, number>;
    Gain?: Record<data.DamageType, number>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface CorruptChoices {
    readonly Unique: boolean;
    readonly ImplicitsCannotBeChanged: boolean;
    readonly Corrupted?: Array<calculator.CorruptMod>;
    readonly ScourgeUpside?: Array<calculator.CorruptMod>;
    readonly ScourgeDownside?: Array<calculator.CorruptMod>;
    readonly Rolls?: Array<calculator.CorruptRoll>;
  }
  interface CorruptMod {
    readonly ID: string;
    readonly Label: string;
    readonly Lines?: Array<string>;
    readonly Group: string;
    readonly ModTags?: Array<string>;
  }
  interface CorruptRoll {
    readonly Index: number;
    readonly Line: string;
    readonly Value: number;
  }
  interface CraftBaseGroup {
    readonly Type: string;
    readonly Bases?: Array<string>;
  }
  interface CrucibleChoice {
    readonly ID: string;
    readonly Label: string;
    readonly Lines?: Array<string>;
    readonly Type: string;
  }
  interface CrucibleChoices {
    readonly Nodes?: Array<Array<calculator.CrucibleChoice> | undefined>;
    readonly Selected?: Array<string>;
  }
  interface CustomMod {
    readonly ID: string;
    readonly Label: string;
    readonly Lines?: Array<string>;
    readonly Type: string;
    readonly ModTags?: Array<string>;
  }
  interface CustomModChoices {
    readonly Sources?: Array<calculator.ImplicitSource>;
    readonly Mods?: Record<string, Array<calculator.CustomMod> | undefined>;
  }
  interface DisplayItemInfo {
    readonly Quality: number;
    readonly HasQuality: boolean;
    readonly Variants?: Array<string>;
    readonly Variant: number;
    readonly AltVariants?: Array<number>;
    readonly VariantGroups?: calculator.DisplayVariantGroups;
    readonly Sockets: string;
    readonly SocketLimit: number;
    readonly DefaultColor: string;
    readonly CanInfluence: boolean;
    readonly Influence?: Array<string>;
    readonly ShowsCatalyst: boolean;
    readonly Catalyst: number;
    readonly CatalystQuality: number;
    readonly Mods?: Array<calculator.DisplayModLine>;
    readonly Enchants: number;
    readonly Anoints: number;
    readonly Anointed?: Array<string>;
    readonly Corruptible: boolean;
    readonly AddImplicit: boolean;
    readonly AddCustom: boolean;
    readonly AddCrucible: boolean;
    readonly Cluster?: calculator.ClusterInfo;
  }
  interface DisplayModLine {
    readonly Line: string;
    readonly Kind: string;
    readonly Section: string;
    readonly Raw: number;
    readonly HasRange: boolean;
    readonly Range: number;
    readonly Disabled: boolean;
    readonly Removable: boolean;
    readonly Removal: string;
    readonly Mutable: boolean;
    readonly Mutated: boolean;
  }
  interface DisplayVariantGroup {
    readonly Group: number;
    readonly Options?: Array<calculator.DisplayVariantOption>;
    readonly Selected: number;
    readonly Enabled: boolean;
  }
  interface DisplayVariantGroups {
    readonly Versions?: Array<string>;
    readonly Version: number;
    readonly Groups?: Array<calculator.DisplayVariantGroup>;
  }
  interface DisplayVariantOption {
    readonly Variant: number;
    readonly Label: string;
  }
  interface EnchantChoices {
    readonly HaveSkills: boolean;
    readonly Skills?: Array<calculator.EnchantSkill>;
    readonly Used?: Array<string>;
  }
  interface EnchantSkill {
    readonly Skill: string;
    readonly Sources?: Array<calculator.EnchantSource>;
  }
  interface EnchantSource {
    readonly Name: string;
    readonly Label: string;
    readonly Lines?: Array<string>;
  }
  interface Entry {
    Columns?: Array<calculator.BCol>;
    Rows?: Array<Record<string, string> | undefined>;
    Lines?: Array<string>;
    readonly Slots?: Array<calculator.BSlot>;
    Reservations?: Array<calculator.BReservation>;
    Label: string;
    DamageTypes?: Array<calculator.BDamageType>;
    Radius: number;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface Environment {
    Cache?: calculator.EnvironmentCache;
    Build?: pob.PathOfBuilding;
    Mode: calculator.OutputMode;
    readonly Spec?: calculator.PassiveSpec;
    JewelRadiusData?: Record<string, Record<string, number> | undefined>;
    readonly ModDB?: moddb.ModDB;
    readonly EnemyModDB?: moddb.ModDB;
    readonly ItemModDB?: moddb.ModDB;
    EnemyLevel: number;
    ConfigPlaceholder?: Record<string, number>;
    readonly Player?: calculator.Actor;
    readonly Enemy?: calculator.Actor;
    readonly Minion?: calculator.ActorMinion;
    readonly RequirementsTable?: Array<calculator.RequirementsTable | undefined>;
    readonly RequirementsTableItems?: Array<calculator.RequirementsTable | undefined>;
    readonly RequirementsTableGems?: Array<calculator.RequirementsTable | undefined>;
    readonly Flasks?: Array<calculator.Item | undefined>;
    readonly Tinctures?: Array<calculator.Item | undefined>;
    AllocatedNodes?: Record<string, Readonly<data.Node>>;
    GrantedAscendancyNodes?: Record<string, boolean>;
    ItemClassRestrictions?: Record<string, string>;
    readonly AuxSkillList?: Array<calculator.ActiveSkill | undefined>;
    readonly Buffs?: Record<string, moddb.ModList | undefined>;
    readonly Debuffs?: Record<string, moddb.ModList | undefined>;
    readonly MinionBuffs?: Record<string, moddb.ModList | undefined>;
    ModeBuffs: boolean;
    ModeCombat: boolean;
    ModeEffective: boolean;
    MainSocketGroup: number;
    SkillsUsed?: Record<string, boolean>;
    DebugErrors?: Array<string>;
    CalcProps?: Record<string, number>;
    readonly AegisModList?: moddb.ModList;
    readonly TheIronMass?: moddb.ModList;
    TracePoolWalk: boolean;
    readonly PoolWalkTrace?: Array<calculator.PoolWalkStep>;
    readonly InitialNodeModDB?: moddb.ModList;
    readonly WeaponModList1?: moddb.ModList;
    CalcsBreakdownModList(name: string, showMinion: boolean): (Array<calculator.ModSectionRow> | undefined);
    CalcsModSection(sd: calculator.ModSectionData, showMinion: boolean): (Array<calculator.ModSectionRow> | undefined);
    CalcsSkillControls(): (calculator.MainSkillControls | undefined);
    CompareModRows(sd: calculator.ModSectionData, minion: boolean): (Array<calculator.CompareModRow> | undefined);
    CompareSkillBreakdown(name: string): (Array<calculator.CompareSkillBreakdownRow> | undefined);
    ConfigRelevance(query: calculator.ConfigRelevance): calculator.ConfigRelevance;
    CountAllocNodes(): [number, number, number, number];
    GemDisplays(): (Array<calculator.GemDisplayRow> | undefined);
    GemSortCache(group: number, gems: Array<pob.Gem> | undefined): (Array<calculator.GemSortRow> | undefined);
    GemSupportPairs(group: number): (Array<calculator.GemSupportPair> | undefined);
    ItemTooltipStats(): calculator.ItemTooltipStats;
    MainSkillControls(): (calculator.MainSkillControls | undefined);
    NewSidebarWarningsRun(): (calculator.SidebarWarningsRun | undefined);
    SidebarWarnings(): (calculator.SidebarWarnings | undefined);
    SocketGroupTooltip(group: number): (Array<calculator.TooltipLine> | undefined);
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface EnvironmentCache {
    TreeVersion: string;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ExplodeSource {
    ModSource: string;
    NodeID: string;
    Rarity: calculator.ItemRarity;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface GemDisplay {
    readonly Level: number;
    readonly Quality: number;
    readonly SocketQuality: number;
    readonly ItemQuality: number;
    readonly SupportQuality: number;
    readonly GlobalQuality: number;
  }
  interface GemDisplayRow {
    readonly Group: number;
    readonly Gem: number;
    readonly Display: calculator.GemDisplay;
  }
  interface GemEffect {
    readonly GrantedEffect?: calculator.GrantedEffect;
    Level: number;
    Quality: number;
    QualityID: string;
    SrcInstance?: pob.Gem;
    GemData?: poe.SkillGem;
    SocketQuality: number;
    ItemQuality: number;
    GlobalQuality: number;
    SupportQuality: number;
    ActorLevel?: number;
    GrantedEffectLevel?: raw.CalculatedLevel;
    CostEmptied: boolean;
    ExplodeSource?: calculator.ExplodeSource;
    Superseded: boolean;
    Values?: Record<string, number>;
    readonly MainSkill?: calculator.ActiveSkill;
    ActivationFreqInc: number;
    ActivationFreqMore: number;
    AttachedBrandCount: number;
    IgnoresTickRate: boolean;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface GemSortRow {
    readonly CanSupport: boolean;
    readonly Pending: boolean;
  }
  interface GemSupportPair {
    readonly Support: number;
    readonly Active: number;
  }
  interface GrantedEffect {
    Raw?: poe.GrantedEffect;
    Parts?: Array<raw.SkillPart>;
    SkillTypes?: Record<data.SkillType, boolean>;
    BaseMultiplier(): number;
    CastTime(): number;
    DamageEffectiveness(): number;
    FromItem(): boolean;
    ID(): string;
    IsSupport(): boolean;
    Name(): string;
    WeaponTypes(): (Array<data.ItemClassName> | undefined);
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface GroupSourceName {
    readonly Name: string;
    readonly Rarity: string;
  }
  interface ImplicitChoices {
    readonly Sources?: Array<calculator.ImplicitSource>;
    readonly Groups?: Record<string, Array<calculator.ImplicitGroup> | undefined>;
  }
  interface ImplicitGroup {
    readonly Label: string;
    readonly Mods?: Array<calculator.ImplicitMod>;
  }
  interface ImplicitMod {
    readonly ID: string;
    readonly Label: string;
    readonly Lines?: Array<string>;
    readonly Type: string;
    readonly ModTags?: Array<string>;
  }
  interface ImplicitSource {
    readonly ID: string;
    readonly Label: string;
  }
  interface Item {
    Raw: string;
    Name: string;
    Title: string;
    BaseName: string;
    Rarity: calculator.ItemRarity;
    Type: string;
    Base?: calculator.ItemBase;
    Quality?: number;
    ItemLevel?: number;
    UniqueID: string;
    League: string;
    Note: string;
    Source: string;
    ClassRestriction: string;
    Unreleased: boolean;
    UpgradePaths?: Array<string>;
    Corrupted: boolean;
    Mirrored: boolean;
    RareLikeUnique: boolean;
    FoilType: string;
    Split: boolean;
    Fractured: boolean;
    Foulborn: boolean;
    Synthesised: boolean;
    Crafted: boolean;
    Scourge: boolean;
    CheckSection: boolean;
    Influence: calculator.ItemInfluence;
    Sockets?: Array<calculator.ItemSocket>;
    Requirements?: Record<string, number>;
    readonly ImplicitModLines?: Array<calculator.ModLine>;
    readonly ExplicitModLines?: Array<calculator.ModLine>;
    readonly CrucibleModLines?: Array<calculator.ModLine>;
    readonly EnchantModLines?: Array<calculator.ModLine>;
    readonly ScourgeModLines?: Array<calculator.ModLine>;
    readonly ClassRequirementModLines?: Array<calculator.ModLine>;
    AdvancedCopy: boolean;
    ModMagnitudeMods?: Array<calculator.ModMagnitudeMod>;
    VariantList?: Array<string>;
    Variant?: number;
    VersionList?: Array<string>;
    SelectedVersion?: number;
    VariantGroups?: Record<number, Record<number, Record<number, boolean> | undefined> | undefined>;
    VariantGroupSelections?: Record<number, number>;
    UsesVariantGroups: boolean;
    AllowDuplicateVariants: boolean;
    Catalyst?: number;
    CatalystQuality?: number;
    HasAltVariant: boolean;
    HasAltVariant2: boolean;
    HasAltVariant3: boolean;
    HasAltVariant4: boolean;
    HasAltVariant5: boolean;
    VariantAlt?: number;
    VariantAlt2?: number;
    VariantAlt3?: number;
    VariantAlt4?: number;
    VariantAlt5?: number;
    NamePrefix: string;
    NameSuffix: string;
    TalismanTier?: number;
    MemoryStrands?: number;
    Intangibility?: number;
    Limit?: number;
    JewelRadiusLabel: string;
    CanBeAnointed: boolean;
    CanHaveTwoEnchants: boolean;
    CanHaveThreeEnchants: boolean;
    CanHaveFourEnchants: boolean;
    GrantedSkills?: Array<calculator.ItemGrantedSkill>;
    AbyssalSocketCount: number;
    SocketedJewelEffectModifier: number;
    SocketColourAlwaysMatches: boolean;
    WeaponData?: Record<number, Record<string, number> | undefined>;
    ArmourData?: Record<string, number>;
    FlaskData?: Record<string, number>;
    TinctureData?: Record<string, number>;
    BuildModList(): void;
    CfgCorruptedIs(corrupted: boolean): boolean;
    CfgElderIs(elder: boolean): boolean;
    CfgModifierMatches(searchCond: string, slot: string): boolean;
    CfgNameIs(name: string): boolean;
    CfgRarityIs(rarity: string): boolean;
    CfgShaperIs(shaper: boolean): boolean;
    Clone(): (calculator.Item | undefined);
    NormaliseQuality(): void;
    ParseRaw(raw: string): void;
    PrimarySlot(): string;
    VariantGroupOptions(groupID: number, excludeSelected: boolean): (Array<number> | undefined);
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ItemBase {
    Name: string;
    Type: string;
    SubType: string;
    SocketLimit: number;
    Req: calculator.ItemBaseReq;
    Armour?: calculator.ItemBaseArmour;
    Weapon?: calculator.ItemBaseWeapon;
    Flask?: calculator.ItemBaseFlask;
    Tincture?: calculator.ItemBaseTincture;
    IsWeaponType(): boolean;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ItemBaseArmour {
    ArmourMin: number;
    ArmourMax: number;
    EvasionMin: number;
    EvasionMax: number;
    EnergyShieldMin: number;
    EnergyShieldMax: number;
    WardMin: number;
    WardMax: number;
    MovementPenalty: number;
    BlockChance: number;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ItemBaseFlask {
    Life: number;
    Mana: number;
    Duration: number;
    ChargesMax: number;
    ChargesUsed: number;
    Buff?: Array<string>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ItemBaseReq {
    Level: number;
    Str: number;
    Dex: number;
    Int: number;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ItemBaseTincture {
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ItemBaseWeapon {
    PhysicalMin: number;
    PhysicalMax: number;
    CritChanceBase: number;
    AttackRateBase: number;
    Range: number;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ItemDBFilter {
    readonly Slot: number;
    readonly Type: number;
    readonly League: number;
    readonly Requirement: number;
    readonly Obtainable: number;
    readonly Search: string;
    readonly SearchMode: number;
  }
  interface ItemData {
    ArmourData?: calculator.ArmourData;
    Type: string;
    Name: string;
    readonly ModList?: moddb.ModList;
    readonly SlotModList?: Record<number, moddb.ModList | undefined>;
    Rarity: string;
    Corrupted: boolean;
    SocketColors?: Array<string>;
    SocketColourAlwaysMatches: boolean;
    ExplicitModLines?: Array<string>;
    OtherModLines?: Array<string>;
    GrantedSkills?: Array<calculator.ItemGrantedSkill>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ItemEdit {
    readonly Kind: string;
    readonly Index: number;
    readonly Value: number;
    readonly Second: number;
    readonly Float: number;
    readonly Text: string;
  }
  interface ItemGrantedSkill {
    SkillID: string;
    Level: number;
    NoSupports: boolean;
    Source: string;
    Triggered: boolean;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ItemInfluence {
    Shaper: boolean;
    Elder: boolean;
    Adjudicator: boolean;
    Basilisk: boolean;
    Crusader: boolean;
    Eyrie: boolean;
    Cleansing: boolean;
    Tangle: boolean;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  type ItemRarity = "NORMAL" | "MAGIC" | "RARE" | "UNIQUE" | "RELIC";
  const ItemRarity: {
    readonly RarityNormal: "NORMAL";
    readonly RarityMagic: "MAGIC";
    readonly RarityRare: "RARE";
    readonly RarityUnique: "UNIQUE";
    readonly RarityRelic: "RELIC";
  };
  interface ItemSocket {
    Color: string;
    Group: number;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ItemTooltip {
    readonly Influence?: Array<string>;
    readonly Title?: Array<string>;
    readonly Lines?: Array<calculator.TooltipLine>;
    readonly Skills?: Array<calculator.TooltipSkill>;
    readonly Children?: Array<Array<calculator.TooltipLine> | undefined>;
  }
  interface ItemTooltipStats {
    readonly Str: number;
    readonly Dex: number;
    readonly Int: number;
    readonly Omni: number;
    readonly OmniscienceRequirements: boolean;
    readonly OmniAttributeRequirements: number;
  }
  interface LoadoutList {
    readonly Loadouts?: Array<string>;
    readonly Selected: number;
  }
  interface MainSkillControls {
    SkillSet: number;
    SocketGroup: number;
    GemIndex: number;
    Skills?: Array<string>;
    SkillEnabled: boolean;
    MainActiveSkill: number;
    PartShown: boolean;
    Parts?: Array<string>;
    SkillPart: number;
    StageShown: boolean;
    StageCount: string;
    MineShown: boolean;
    MineCount: string;
    MinionShown: boolean;
    MinionEnabled: boolean;
    MinionLibraryShown: boolean;
    Minions?: Array<calculator.MinionOption>;
    Minion: number;
    MinionUses?: Array<string>;
    MinionSkillShown: boolean;
    MinionSkillEnabled: boolean;
    MinionSkills?: Array<string>;
    MinionSkill: number;
    SkillDisabled: boolean;
    DisableReason: string;
    InfoMessage: string;
    InfoMessage2: string;
    MinionInfoMessage: string;
    MinionInfoMessage2: string;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface MasteryOption {
    readonly Effect: number;
    readonly Label: string;
    readonly Selected: boolean;
  }
  interface MinionOption {
    Label: string;
    MinionID: string;
    ItemSetID: number;
    Tooltip?: Array<string>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface MirageResult {
    Name: string;
    Count: number;
    SkillPartName: string;
    Output?: Record<string, number>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ModLine {
    Line: string;
    Extra: string;
    Range?: number;
    ModTags?: Array<string>;
    VariantList?: Record<number, boolean>;
    Crafted: boolean;
    Custom: boolean;
    Fractured: boolean;
    Mutated: boolean;
    Vestigial: boolean;
    Scourge: boolean;
    Implicit: boolean;
    ValueScalar: number;
    CorruptedRange?: number;
    Prefix: boolean;
    Suffix: boolean;
    Unveiled: boolean;
    Disabled: boolean;
    Order?: number;
    ModID: string;
    NewModID: string;
    ModGroup: string;
    Crucible: boolean;
    VersionList?: Record<number, boolean>;
    VariantGroupList?: Record<number, boolean>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ModMagnitudeMod {
    Tags?: Array<string>;
    AnyTags?: Array<string>;
    Quality: number;
    Multiplier?: number;
    ModType: string;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ModSectionData {
    readonly ModName?: Array<string>;
    readonly Multi: boolean;
    readonly ModType: string;
    readonly Cfg: string;
    readonly ModSource: string;
    readonly Enemy: boolean;
    readonly Actor: string;
  }
  interface ModSectionRow {
    readonly Value: string;
    readonly Name: string;
    readonly Flags: string;
    readonly Tags: string;
    readonly Source: string;
    readonly SourceName: string;
    readonly Totals?: Array<string>;
    readonly ItemName: string;
    readonly NodeID: number;
  }
  type OutTable = "MainHand" | "OffHand";
  const OutTable: {
    readonly OutTableMainHand: "MainHand";
    readonly OutTableOffHand: "OffHand";
  };
  type OutputMode = "MAIN" | "CALCS" | "CALCULATOR";
  const OutputMode: {
    readonly OutputModeMain: "MAIN";
    readonly OutputModeCalcs: "CALCS";
    readonly OutputModeCalculator: "CALCULATOR";
  };
  interface PartySummaryEntry {
    readonly Group: string;
    readonly Name: string;
    readonly Value: string;
  }
  interface PassiveSpec {
    Build?: pob.PathOfBuilding;
    TreeVersion: string;
    AllocNodes?: Record<string, Readonly<data.Node>>;
    MasterySelections?: Record<number, number>;
    ClassID: pob.ClassID;
    ClassName: string;
    AscendancyID: pob.AscendancyID;
    AscendancyName: string;
    AllocatedNotableCount: number;
    AllocatedMasteryCount: number;
    DistanceToClassStart?: Record<string, number>;
    Class(): data.Class;
    SelectAscendancyClass(ascendancyID: pob.AscendancyID): void;
    SelectClass(classID: pob.ClassID): void;
    Tree(): (data.Tree | undefined);
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface PoolWalkStep {
    N: number;
    Damage?: Record<data.DamageType, number>;
    LifeIn: number;
    ManaIn: number;
    EnergyShieldIn: number;
    WardIn: number;
    LifeOut: number;
    ManaOut: number;
    EnergyShieldOut: number;
    WardOut: number;
    RecoupableIn?: Record<data.DamageType, number>;
    RecoupableOut?: Record<data.DamageType, number>;
    AegisIn?: Record<string, number>;
    AegisOut?: Record<string, number>;
    GuardIn?: Record<string, number>;
    GuardOut?: Record<string, number>;
    AllyIn?: Record<string, number>;
    AllyOut?: Record<string, number>;
    readonly Actor?: calculator.Actor;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface PowerReportEntry {
    readonly ID: number;
    readonly Name: string;
    readonly Type: string;
    readonly Power: number;
    readonly PathPower: number;
    readonly PathDist: number;
    readonly Allocated: boolean;
    readonly Stats?: Array<string>;
    readonly Cluster: boolean;
    readonly Offence: number;
    readonly Defence: number;
    readonly Effect: number;
    readonly Picked: boolean;
  }
  interface PowerShardRows {
    readonly Rows?: Array<calculator.PowerReportEntry>;
    readonly Owned?: Array<number>;
    readonly Ascendancy?: Array<calculator.PowerReportEntry>;
  }
  interface PowerStat {
    readonly Stat: string;
    readonly Label: string;
    readonly Negate: boolean;
    readonly Combined: boolean;
  }
  interface RebaseResult {
    readonly Text: string;
    readonly Dropped?: Array<string>;
  }
  interface RequirementsTable {
    Source: string;
    SourceGem?: pob.Gem;
    Str: number;
    Dex: number;
    Int: number;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface SecondaryAscendOption {
    readonly Label: string;
    readonly AscendClassID: number;
  }
  interface SecondaryAscendancyDropdown {
    readonly Entries?: Array<calculator.SecondaryAscendOption>;
    readonly Selected: number;
    readonly Enabled: boolean;
  }
  interface SidebarWarnings {
    EnergyShieldProtectsMana: boolean;
    CostWarning?: Array<string>;
    LifeCostWarningList?: Array<string>;
    ManaCostWarningList?: Array<string>;
    RageCostWarningList?: Array<string>;
    ESCostWarningList?: Array<string>;
    LifePercentCostPercentCostWarningList?: Array<string>;
    ManaPercentCostPercentCostWarningList?: Array<string>;
    PlayerAspectCount: number;
    MinionAspectCount: number;
    JewelLimitWarning?: Array<string>;
    SocketLimitWarning?: Array<string>;
    MissingAnointWarning?: Array<string>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface SidebarWarningsRun {
    Result(): (calculator.SidebarWarnings | undefined);
    Step(): boolean;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface SkillDPS {
    Name: string;
    DPS: number;
    Count: number;
    Trigger: string;
    SkillPart: string;
    Source: string;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface SkillData {
    StagesMax: number;
    PulseInterval: number;
    PulseDamage: number;
    SupportBonechill: boolean;
    Cooldown: number;
    StoredUses?: number;
    Triggered: boolean;
    TriggeredByBrand: boolean;
    TriggeredByBattleMageCry: boolean;
    TriggerOnCrit: boolean;
    CountsAttachedBrandsInDamage: boolean;
    AttachedBrandCount: number;
    TriggeredOnDeath: boolean;
    TriggeredByAutoexertion: boolean;
    TriggeredByDamageTaken: number;
    TriggeredWhileChannelling: boolean;
    TriggeredByMeleeKill: boolean;
    TriggeredByCospris: boolean;
    TriggeredByMjolner: boolean;
    TriggeredBySevenTeachings: boolean;
    TriggeredByKineticFlux: boolean;
    TriggeredByUnique: boolean;
    TriggeredByFocus: boolean;
    TriggeredByCraft: boolean;
    TriggeredByManaSpent: boolean;
    TriggeredByManaforged?: number;
    ManaForgedArrowsPercentThreshold: number;
    TriggeredByParentAttack: boolean;
    TriggeredByGeneralsCry: boolean;
    TriggerTime: number;
    TriggeredBySaviour: boolean;
    CritChance: number;
    SetOffHandPhysicalMin: number;
    SetOffHandPhysicalMax: number;
    PercentageLifeSacrificed: number;
    PercentageSacrificedDealtAsChaos: number;
    AdditionalRuinChance: number;
    ManabondMissingManaGainPercent: number;
    ManabondMissingUnreservedManaPercentage: number;
    SetOffHandFireMin: number;
    SetOffHandFireMax: number;
    SetOffHandColdMin: number;
    SetOffHandColdMax: number;
    AttackTime: number;
    CastTimeOverride: number;
    CastTimeOverridesAttackTime: boolean;
    TimeOverride: number;
    FixedCastTime: boolean;
    TriggerRate?: number;
    ShowAverage: boolean;
    ManaReservationPercent?: number;
    TotemLevel: number;
    CannotBeEvaded: boolean;
    DoubleHitsWhenDualWielding: boolean;
    DpsMultiplier: number;
    DpsBaseMultiplier: number;
    PodOverlapMultiplier: number;
    DamagePerProjectile: number;
    DelayPerProjectile: number;
    MaxRageVortexSacrificePercentage: number;
    FifthStrikeDamage: number;
    FifthStrikeProjectiles: number;
    BaseMultiplier: number;
    DamageEffectiveness: number;
    LifeLeechPerUse: number;
    ManaLeechPerUse: number;
    BleedDurationIsSkillDuration: boolean;
    BleedIsSkillEffect: boolean;
    Duration: number;
    FrostBombDamagePer100ms: number;
    BleedBasePercent: number;
    Type: data.ItemClassName;
    AttackRate: number;
    PhysicalMin: number;
    PhysicalMax: number;
    AttackSpeedInc: number;
    CountsAsAll1H: boolean;
    CountsAsDualWielding: boolean;
    AddedUsingAxe?: boolean;
    AddedUsingSword?: boolean;
    AddedUsingDagger?: boolean;
    AddedUsingMace?: boolean;
    AddedUsingClaw?: boolean;
    PhysicalBonusMin: number;
    PhysicalBonusMax: number;
    LightningMin: number;
    LightningMax: number;
    LightningBonusMin: number;
    LightningBonusMax: number;
    ColdMin: number;
    ColdMax: number;
    ColdBonusMin: number;
    ColdBonusMax: number;
    FireMin: number;
    FireMax: number;
    FireBonusMin: number;
    FireBonusMax: number;
    ChaosMin: number;
    ChaosMax: number;
    ChaosBonusMin: number;
    ChaosBonusMax: number;
    LifeDealtAsChaos: number;
    EnergyShieldDealtAsChaos: number;
    LifeDealtAsFire: number;
    SkeletonLife: number;
    SelfDamageTakenLife: number;
    SelfDamageTakenES: number;
    Radius: number;
    RadiusSecondary: number;
    RadiusTertiary: number;
    RadiusExtra: number;
    RadiusTertiaryBaseMargin: number;
    ProjectileSpeedAppliesToMSAreaOfEffect: boolean;
    DurationSecondary: number;
    DurationTertiary: number;
    MinionLevel: number;
    MinionLevelIsEnemyLevel: boolean;
    MinionLevelIsPlayerLevel: number;
    MinionDamageEffectiveness: number;
    FireDot: number;
    RFLifeMultiplier: number;
    RFESMultiplier: number;
    RFManaMultiplier: number;
    RFMultiplier: number;
    PercentSacrificed: number;
    TriggeredByCoC: boolean;
    DotIsSpell: boolean;
    RepeatFrequency: number;
    RepeatInterval: number;
    HitFrequency: number;
    CremationFireRate: number;
    CremationFireRateIncrease: number;
    DamageInterval: number;
    HitFrequencyPerBlade: number;
    IncVolleyFrequency: number;
    VaalBladeVortexBlade: number;
    RepeatFrequencyIncrease: number;
    BeamOverlapMultiplier: number;
    ActiveArrowMultiplier?: number;
    TornadoShotSecondaryHitChance?: number;
    ShrapnelBallistaProjectileOverlap?: number;
    HoaOverkill?: number;
    HoaMoreBurn: number;
    HoaOverkillPercent: number;
    MoltenShellDamageMitigated?: number;
    VaalMoltenShellDamageMitigated?: number;
    MoltenShellReflect: number;
    TriggeredByMirageArcher: boolean;
    TriggeredBySnipe: boolean;
    TriggeredBySacredWisps: boolean;
    ChanceToTriggerOnCrit: number;
    ColdDot: number;
    CorpseExplosionLifeMultiplier: number;
    SelfFireExplosionLifeMultiplier: number;
    CorpseLife: number;
    CorpseExplosionDamageType: string;
    ChaosDot: number;
    BaseManaCostIsAtLeastPercentUnreservedMana: number;
    ManaReservationFlat?: number;
    LifeReservationFlat?: number;
    LifeReservationPercent?: number;
    ManaReservationFlatForced?: number;
    LifeReservationFlatForced?: number;
    ManaReservationPercentForced?: number;
    LifeReservationPercentForced?: number;
    ManaReservedBase: number;
    ManaReservedPercent: number;
    LifeReservedBase: number;
    LifeReservedPercent: number;
    AttackSpeedMultiplier: number;
    MinionUseBowAndQuiver: boolean;
    MinionUseMainHandWeapon: boolean;
    ArrowSpeedAppliesToAreaOfEffect: boolean;
    GainPercentBaseWandDamageToSpells: number;
    HitTimeOverride: number;
    TrapCooldown: number;
    MineDurationAppliesToSkill: boolean;
    Debuff: boolean;
    DebuffSecondary: boolean;
    DebuffTertiary: boolean;
    ReserveDuration: number;
    AuraDuration: number;
    SoulPreventionDuration: number;
    SkillEffectAppliesToSoulGainPrevention: boolean;
    AuraCannotAffectSelf: boolean;
    ThisIsNotABuff: boolean;
    BuffNotPlayer: boolean;
    BuffMinions: boolean;
    BuffAllies: boolean;
    AllowTotemBuff: boolean;
    Enable: boolean;
    DotIsArea: boolean;
    DotIsProjectile: boolean;
    DotIsAttack: boolean;
    DotIsHit: boolean;
    PhysicalDot: number;
    LightningDot: number;
    ExplodeCorpse: boolean;
    HitChanceIsExplodeChance: boolean;
    PhysicalEffectiveExplodePercentage: number;
    LightningEffectiveExplodePercentage: number;
    ColdEffectiveExplodePercentage: number;
    FireEffectiveExplodePercentage: number;
    ChaosEffectiveExplodePercentage: number;
    PoisonIsSkillEffect: boolean;
    PoisonDurationIsSkillDuration: boolean;
    StackMultiplier: number;
    RadiusIsWeaponRange: boolean;
    Decay: number;
    BaseStunDuration: number;
    Range: number;
    RangeBonus: number;
    HitTimeMultiplier: number;
    TotemFireOnce: boolean;
    NumStages: number;
    ProjectileSpeed: number;
    StackCount?: number;
    SummonSpeed?: number;
    StrikeInterval: number;
    RemainingDurationDamage: number;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  type SkillFlag = "brand" | "arrow" | "hex" | "curse" | "attack" | "weapon1Attack" | "weapon2Attack" | "selfCast" | "notAverage" | "showAverage" | "hit" | "projectile" | "trap" | "mine" | "totem" | "bothWeaponAttack" | "buffs" | "combat" | "effective" | "spell" | "melee" | "chaining" | "area" | "monsterExplode" | "cast" | "shieldAttack" | "forceMainHand" | "disable" | "bleed" | "duration" | "igniteCanStack" | "igniteToChaos" | "DotCanStack" | "dot" | "dotFromAttack" | "ignite" | "poison" | "decay" | "impale" | "ballista" | "minion" | "permanentMinion" | "randomPhys" | "forking" | "piercing" | "warcry" | "haveMinion" | "minionSkill" | "spectre" | "stages" | "multiPart" | "multiStage" | "triggered" | "focused";
  const SkillFlag: {
    readonly SkillFlagBrand: "brand";
    readonly SkillFlagArrow: "arrow";
    readonly SkillFlagHex: "hex";
    readonly SkillFlagCurse: "curse";
    readonly SkillFlagAttack: "attack";
    readonly SkillFlagWeapon1Attack: "weapon1Attack";
    readonly SkillFlagWeapon2Attack: "weapon2Attack";
    readonly SkillFlagSelfCast: "selfCast";
    readonly SkillFlagNotAverage: "notAverage";
    readonly SkillFlagShowAverage: "showAverage";
    readonly SkillFlagHit: "hit";
    readonly SkillFlagProjectile: "projectile";
    readonly SkillFlagTrap: "trap";
    readonly SkillFlagMine: "mine";
    readonly SkillFlagTotem: "totem";
    readonly SkillFlagBothWeaponAttack: "bothWeaponAttack";
    readonly SkillFlagBuffs: "buffs";
    readonly SkillFlagCombat: "combat";
    readonly SkillFlagEffective: "effective";
    readonly SkillFlagSpell: "spell";
    readonly SkillFlagMelee: "melee";
    readonly SkillFlagChaining: "chaining";
    readonly SkillFlagArea: "area";
    readonly SkillFlagMonsterExplode: "monsterExplode";
    readonly SkillFlagCast: "cast";
    readonly SkillFlagShieldAttack: "shieldAttack";
    readonly SkillFlagForceMainHand: "forceMainHand";
    readonly SkillFlagDisable: "disable";
    readonly SkillFlagBleed: "bleed";
    readonly SkillFlagDuration: "duration";
    readonly SkillFlagIgniteCanStack: "igniteCanStack";
    readonly SkillFlagIgniteToChaos: "igniteToChaos";
    readonly SkillFlagDotCanStack: "DotCanStack";
    readonly SkillFlagDot: "dot";
    readonly SkillFlagDotFromAttack: "dotFromAttack";
    readonly SkillFlagIgnite: "ignite";
    readonly SkillFlagPoison: "poison";
    readonly SkillFlagDecay: "decay";
    readonly SkillFlagImpale: "impale";
    readonly SkillFlagBallista: "ballista";
    readonly SkillFlagMinion: "minion";
    readonly SkillFlagPermanentMinion: "permanentMinion";
    readonly SkillFlagRandomPhys: "randomPhys";
    readonly SkillFlagForking: "forking";
    readonly SkillFlagPiercing: "piercing";
    readonly SkillFlagWarcry: "warcry";
    readonly SkillFlagHaveMinion: "haveMinion";
    readonly SkillFlagMinionSkill: "minionSkill";
    readonly SkillFlagSpectre: "spectre";
    readonly SkillFlagStages: "stages";
    readonly SkillFlagMultiPart: "multiPart";
    readonly SkillFlagMultiStage: "multiStage";
    readonly SkillFlagTriggered: "triggered";
    readonly SkillFlagFocused: "focused";
  };
  interface SpecNodeRewrite {
    readonly NodeID: number;
    readonly Name: string;
    readonly Stats?: Array<string>;
    readonly Replaced: boolean;
  }
  interface SpectreRow {
    readonly ID: string;
    readonly Name: string;
    readonly Skills?: Array<string>;
    readonly Tooltip?: Array<string>;
  }
  interface TattooCountLine {
    readonly Line: string;
    readonly Count: number;
  }
  interface TimelessDesiredNode {
    readonly ID: string;
    readonly DisplayName: string;
    readonly DesiredIdx: number;
  }
  interface TimelessJewelType {
    readonly Label: string;
    readonly Name: string;
    readonly ID: number;
  }
  interface TimelessMod {
    readonly Label: string;
    readonly ID: string;
    readonly Descriptions?: Array<string>;
    readonly TotalMod: boolean;
  }
  interface TimelessNodeMatch {
    readonly ID: string;
    readonly TargetNodeNames?: Array<string>;
    readonly TotalWeight: number;
  }
  interface TimelessSearchQuery {
    readonly JewelType: number;
    readonly ConquerorType: number;
    readonly SocketID: number;
    readonly SocketFilter: boolean;
    readonly SocketFilterDistance: number;
    readonly ProtectedNodes?: Array<string>;
    readonly SearchList: string;
    readonly SearchListFallback: string;
    readonly TotalMinimumWeight: number;
    readonly HasTotalMinimumWeight: boolean;
  }
  interface TimelessSearchResult {
    readonly Label: string;
    readonly Seed: number;
    readonly Total: number;
    readonly SocketID: number;
    readonly SocketLabel: string;
    readonly Nodes?: Array<calculator.TimelessNodeMatch>;
  }
  interface TimelessSocket {
    readonly Label: string;
    readonly Keystone: string;
    readonly ID: number;
  }
  interface TimelessTrade {
    readonly URL: string;
    readonly SelIndex: number;
    readonly HighlightIndex: number;
    readonly LastSearchStart: number;
    readonly LastSearchCount: number;
    readonly LastSearchPresent: boolean;
  }
  interface TimelessTradeQuery {
    readonly JewelType: number;
    readonly ConquerorType: number;
    readonly DevotionVariant1: number;
    readonly DevotionVariant2: number;
    readonly Seeds?: Array<number>;
    readonly SelIndex: number;
    readonly HighlightIndex: number;
    readonly SearchMore: boolean;
    readonly TradeTypeIndex: number;
    readonly Realm: string;
    readonly League: string;
    readonly LastSearchStart: number;
    readonly LastSearchCount: number;
    readonly LastSearchPresent: boolean;
  }
  interface TooltipLine {
    readonly Text: string;
    readonly Separator: boolean;
  }
  interface TooltipSkill {
    readonly SkillID: string;
    readonly Level: number;
  }
  interface TreeClusterNode {
    readonly ID: number;
    readonly Type: string;
    readonly Name: string;
    readonly Icon: string;
    readonly Stats?: Array<string>;
    readonly Group: number;
    readonly Orbit: number;
    readonly OrbitIndex: number;
    readonly X: number;
    readonly Y: number;
    readonly Linked?: Array<number>;
  }
  interface TreeCompareMastery {
    readonly Node: number;
    readonly Effect: number;
    readonly Stats?: Array<string>;
  }
  interface TreeCompareSpec {
    readonly Nodes?: Array<number>;
    readonly Masteries?: Array<calculator.TreeCompareMastery>;
  }
  interface TreeJewelOverlay {
    readonly NodeID: number;
    readonly Art: string;
  }
  interface TreeJewelRing {
    readonly NodeID: number;
    readonly Outer: number;
    readonly Inner: number;
    readonly Art: string;
  }
  interface TreeSocketRadius {
    readonly Inner: number;
    readonly Outer: number;
    readonly Col: string;
    readonly Nodes?: Array<number>;
  }
  interface TreeSpecPoints {
    readonly Used: number;
    readonly AscendUsed: number;
    readonly Sockets: number;
  }
  interface TreeSpecRespec {
    readonly Gold: number;
    readonly Passives: number;
    readonly Ascendancy: number;
  }
  interface TreeSpecRow {
    readonly Label: string;
    readonly Title: string;
    readonly ClassName: string;
    readonly AscendClassName: string;
    readonly TreeVersion: string;
    readonly VersionLabel: string;
  }
  interface ArmourDataPlain {
    readonly Ward: number;
    readonly EnergyShield: number;
    readonly Armour: number;
    readonly Evasion: number;
    readonly BlockChance: number;
  }
  interface BColPlain {
    readonly Label: string;
    readonly Key: string;
  }
  interface BDamageTypePlain {
    readonly Source: string;
    readonly ConvSrc?: string;
    readonly Total: string;
    readonly ConvDst?: string;
    readonly GainDst?: string;
    readonly Base?: string;
    readonly Inc?: string;
    readonly More?: string;
  }
  interface BReservationPlain {
    readonly SkillName: string;
    readonly Base: string;
    readonly Mult?: string;
    readonly More?: string;
    readonly Inc?: string;
    readonly Efficiency?: string;
    readonly EfficiencyMore?: string;
    readonly Total: string;
  }
  interface BSlotPlain {
    readonly Base: number;
    readonly Inc: string;
    readonly More: string;
    readonly Total: string;
    readonly Source: string;
    readonly SourceName?: string;
    readonly Item?: calculator.ItemDataPlain;
  }
  interface EntryPlain {
    readonly Columns?: Array<calculator.BColPlain>;
    readonly Rows?: Array<Record<string, string> | undefined>;
    readonly Lines?: Array<string>;
    readonly Slots?: Array<calculator.BSlotPlain>;
    readonly Reservations?: Array<calculator.BReservationPlain>;
    readonly Label: string;
    readonly DamageTypes?: Array<calculator.BDamageTypePlain>;
    readonly Radius: number;
  }
  interface ItemDataPlain {
    readonly ArmourData?: calculator.ArmourDataPlain;
    readonly Type: string;
    readonly Name: string;
    readonly ModList?: moddb.ModListPlain;
    readonly SlotModList?: Record<number, moddb.ModListPlain | undefined>;
    readonly Rarity: string;
    readonly Corrupted: boolean;
    readonly SocketColors?: Array<string>;
    readonly SocketColourAlwaysMatches: boolean;
    readonly ExplicitModLines?: Array<string>;
    readonly OtherModLines?: Array<string>;
    readonly GrantedSkills?: Array<calculator.ItemGrantedSkillPlain>;
  }
  interface ItemGrantedSkillPlain {
    readonly SkillID: string;
    readonly Level: number;
    readonly NoSupports: boolean;
    readonly Source: string;
    readonly Triggered: boolean;
  }
  function CompareItemExpanded(raw: string, other: string): Promise<(Array<string> | undefined)>;
  function CompareJewelSlots(primary: pob.PathOfBuilding | undefined, compare: pob.PathOfBuilding | undefined): Promise<(Array<calculator.CompareJewelSlot> | undefined)>;
  function NewCalculator(build: pob.PathOfBuilding): (calculator.Calculator | undefined);
  function NewCompareJewelPowerBuilder(primary: pob.PathOfBuilding | undefined, compare: pob.PathOfBuilding | undefined, stats: Array<string> | undefined): (calculator.ComparePowerBuilder | undefined);
  function NewComparePowerBuilder(primary: pob.PathOfBuilding | undefined, compare: pob.PathOfBuilding | undefined, stats: Array<string> | undefined, categories: calculator.ComparePowerCategories, config: Array<pob.Input> | undefined): (calculator.ComparePowerBuilder | undefined);
  function SetDefaultItemAffixQuality(q: number): Promise<void>;
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
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface CharacterAttributes {
    Strength: number;
    Dexterity: number;
    Intelligence: number;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface Class {
    Name: string;
    BaseStr: number;
    BaseDex: number;
    BaseInt: number;
    Ascendancies?: Array<data.Ascendancy>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface Classes {
    StrDexIntClass: number;
    StrClass: number;
    DexClass: number;
    IntClass: number;
    StrDexClass: number;
    StrIntClass: number;
    DexIntClass: number;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface Constants {
    Classes: data.Classes;
    CharacterAttributes: data.CharacterAttributes;
    PSSCentreInnerRadius: number;
    SkillsPerOrbit?: Array<number>;
    OrbitRadii?: Array<number>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface Coord {
    X: number;
    Y: number;
    W: number;
    H: number;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  type DamageType = "Physical" | "Lightning" | "Cold" | "Fire" | "Elemental" | "Chaos";
  const DamageType: {
    readonly DamageTypePhysical: "Physical";
    readonly DamageTypeLightning: "Lightning";
    readonly DamageTypeCold: "Cold";
    readonly DamageTypeFire: "Fire";
    readonly DamageTypeElemental: "Elemental";
    readonly DamageTypeChaos: "Chaos";
  };
  interface ExpansionJewel {
    Size: number;
    Index: number;
    Proxy: string;
    Parent?: string;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ExtraImage {
    X: number;
    Y: number;
    Image: string;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface FlavourTextRect {
    X: number;
    Y: number;
    Width: number;
    Height: number;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface Group {
    X: number;
    Y: number;
    Orbits?: Array<number>;
    Nodes?: Array<string>;
    IsProxy?: boolean;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  type ItemClassName = "None" | "Bow" | "Claw" | "Dagger" | "Fishing Rod" | "One Handed Axe" | "One Handed Mace" | "One Handed Sword" | "Rune Dagger" | "Sceptre" | "Shield" | "Staff" | "Thrusting One Handed Sword" | "Two Handed Axe" | "Two Handed Mace" | "Two Handed Sword" | "Unarmed" | "Wand" | "Warstaff";
  const ItemClassName: {
    readonly None: "None";
    readonly Bow: "Bow";
    readonly Claw: "Claw";
    readonly Dagger: "Dagger";
    readonly FishingRod: "Fishing Rod";
    readonly OneHandAxe: "One Handed Axe";
    readonly OneHandMace: "One Handed Mace";
    readonly OneHandSword: "One Handed Sword";
    readonly RuneDagger: "Rune Dagger";
    readonly Sceptre: "Sceptre";
    readonly Shield: "Shield";
    readonly Staff: "Staff";
    readonly ThrustingOneHandSword: "Thrusting One Handed Sword";
    readonly TwoHandAxe: "Two Handed Axe";
    readonly TwoHandMace: "Two Handed Mace";
    readonly TwoHandSword: "Two Handed Sword";
    readonly Unarmed: "Unarmed";
    readonly Wand: "Wand";
    readonly Warstaff: "Warstaff";
  };
  interface MasteryEffect {
    Effect: number;
    Stats?: Array<string>;
    ReminderText?: Array<string>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
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
    IsBloodline?: boolean;
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
    ClusterJewelIncEffect: number;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface Points {
    TotalPoints: number;
    AscendancyPoints: number;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  type SkillType = "Attack" | "Spell" | "Projectile" | "DualWieldOnly" | "Buff" | "Removed6" | "MainHandOnly" | "Removed8" | "Minion" | "Damage" | "Area" | "Duration" | "RequiresShield" | "ProjectileSpeed" | "HasReservation" | "ReservationBecomesCost" | "Trappable" | "Totemable" | "Mineable" | "ElementalStatus" | "MinionsCanExplode" | "Removed22" | "Chains" | "Melee" | "MeleeSingleTarget" | "Multicastable" | "TotemCastsAlone" | "Multistrikeable" | "CausesBurning" | "SummonsTotem" | "TotemCastsWhenNotDetached" | "Fire" | "Cold" | "Lightning" | "Triggerable" | "Trapped" | "Movement" | "Removed39" | "DamageOverTime" | "RemoteMined" | "Triggered" | "Vaal" | "Aura" | "Removed45" | "CanTargetUnusableCorpse" | "Removed47" | "RangedAttack" | "Removed49" | "Chaos" | "FixedSpeedProjectile" | "Removed52" | "ThresholdJewelArea" | "ThresholdJewelProjectile" | "ThresholdJewelDuration" | "ThresholdJewelRangedAttack" | "Removed57" | "Channel" | "DegenOnlySpellDamage" | "Removed60" | "InbuiltTrigger" | "Golem" | "Herald" | "AuraAffectsEnemies" | "NoRuthless" | "ThresholdJewelSpellDamage" | "Cascadable" | "ProjectilesFromUser" | "MirageArcherCanUse" | "ProjectileSpiral" | "SingleMainProjectile" | "MinionsPersistWhenSkillRemoved" | "ProjectileNumber" | "Warcry" | "Instant" | "Brand" | "DestroysCorpse" | "NonHitChill" | "ChillingArea" | "AppliesCurse" | "CanRapidFire" | "AuraDuration" | "AreaSpell" | "OR" | "AND" | "NOT" | "Physical" | "AppliesMaim" | "CreatesMinion" | "Guard" | "Travel" | "Blink" | "CanHaveBlessing" | "ProjectilesNotFromUser" | "AttackInPlaceIsDefault" | "Nova" | "InstantNoRepeatWhenHeld" | "InstantShiftAttackForLeftMouse" | "AuraNotOnCaster" | "Banner" | "Rain" | "Cooldown" | "ThresholdJewelChaining" | "Slam" | "Stance" | "NonRepeatable" | "OtherThingUsesSkill" | "Steel" | "Hex" | "Mark" | "Aegis" | "Orb" | "KillNoDamageModifiers" | "RandomElement" | "LateConsumeCooldown" | "Arcane" | "FixedCastTime" | "RequiresOffHandNotWeapon" | "Link" | "Blessing" | "ZeroReservation" | "Pact" | "Retaliation" | "NeverExertable" | "MinionsAreUndamagable";
  const SkillType: {
    readonly SkillTypeAttack: "Attack";
    readonly SkillTypeSpell: "Spell";
    readonly SkillTypeProjectile: "Projectile";
    readonly SkillTypeDualWieldOnly: "DualWieldOnly";
    readonly SkillTypeBuff: "Buff";
    readonly SkillTypeRemoved6: "Removed6";
    readonly SkillTypeMainHandOnly: "MainHandOnly";
    readonly SkillTypeRemoved8: "Removed8";
    readonly SkillTypeMinion: "Minion";
    readonly SkillTypeDamage: "Damage";
    readonly SkillTypeArea: "Area";
    readonly SkillTypeDuration: "Duration";
    readonly SkillTypeRequiresShield: "RequiresShield";
    readonly SkillTypeProjectileSpeed: "ProjectileSpeed";
    readonly SkillTypeHasReservation: "HasReservation";
    readonly SkillTypeReservationBecomesCost: "ReservationBecomesCost";
    readonly SkillTypeTrappable: "Trappable";
    readonly SkillTypeTotemable: "Totemable";
    readonly SkillTypeMineable: "Mineable";
    readonly SkillTypeElementalStatus: "ElementalStatus";
    readonly SkillTypeMinionsCanExplode: "MinionsCanExplode";
    readonly SkillTypeRemoved22: "Removed22";
    readonly SkillTypeChains: "Chains";
    readonly SkillTypeMelee: "Melee";
    readonly SkillTypeMeleeSingleTarget: "MeleeSingleTarget";
    readonly SkillTypeMulticastable: "Multicastable";
    readonly SkillTypeTotemCastsAlone: "TotemCastsAlone";
    readonly SkillTypeMultistrikeable: "Multistrikeable";
    readonly SkillTypeCausesBurning: "CausesBurning";
    readonly SkillTypeSummonsTotem: "SummonsTotem";
    readonly SkillTypeTotemCastsWhenNotDetached: "TotemCastsWhenNotDetached";
    readonly SkillTypeFire: "Fire";
    readonly SkillTypeCold: "Cold";
    readonly SkillTypeLightning: "Lightning";
    readonly SkillTypeTriggerable: "Triggerable";
    readonly SkillTypeTrapped: "Trapped";
    readonly SkillTypeMovement: "Movement";
    readonly SkillTypeRemoved39: "Removed39";
    readonly SkillTypeDamageOverTime: "DamageOverTime";
    readonly SkillTypeRemoteMined: "RemoteMined";
    readonly SkillTypeTriggered: "Triggered";
    readonly SkillTypeVaal: "Vaal";
    readonly SkillTypeAura: "Aura";
    readonly SkillTypeRemoved45: "Removed45";
    readonly SkillTypeCanTargetUnusableCorpse: "CanTargetUnusableCorpse";
    readonly SkillTypeRemoved47: "Removed47";
    readonly SkillTypeRangedAttack: "RangedAttack";
    readonly SkillTypeRemoved49: "Removed49";
    readonly SkillTypeChaos: "Chaos";
    readonly SkillTypeFixedSpeedProjectile: "FixedSpeedProjectile";
    readonly SkillTypeRemoved52: "Removed52";
    readonly SkillTypeThresholdJewelArea: "ThresholdJewelArea";
    readonly SkillTypeThresholdJewelProjectile: "ThresholdJewelProjectile";
    readonly SkillTypeThresholdJewelDuration: "ThresholdJewelDuration";
    readonly SkillTypeThresholdJewelRangedAttack: "ThresholdJewelRangedAttack";
    readonly SkillTypeRemoved57: "Removed57";
    readonly SkillTypeChannel: "Channel";
    readonly SkillTypeDegenOnlySpellDamage: "DegenOnlySpellDamage";
    readonly SkillTypeRemoved60: "Removed60";
    readonly SkillTypeInbuiltTrigger: "InbuiltTrigger";
    readonly SkillTypeGolem: "Golem";
    readonly SkillTypeHerald: "Herald";
    readonly SkillTypeAuraAffectsEnemies: "AuraAffectsEnemies";
    readonly SkillTypeNoRuthless: "NoRuthless";
    readonly SkillTypeThresholdJewelSpellDamage: "ThresholdJewelSpellDamage";
    readonly SkillTypeCascadable: "Cascadable";
    readonly SkillTypeProjectilesFromUser: "ProjectilesFromUser";
    readonly SkillTypeMirageArcherCanUse: "MirageArcherCanUse";
    readonly SkillTypeProjectileSpiral: "ProjectileSpiral";
    readonly SkillTypeSingleMainProjectile: "SingleMainProjectile";
    readonly SkillTypeMinionsPersistWhenSkillRemoved: "MinionsPersistWhenSkillRemoved";
    readonly SkillTypeProjectileNumber: "ProjectileNumber";
    readonly SkillTypeWarcry: "Warcry";
    readonly SkillTypeInstant: "Instant";
    readonly SkillTypeBrand: "Brand";
    readonly SkillTypeDestroysCorpse: "DestroysCorpse";
    readonly SkillTypeNonHitChill: "NonHitChill";
    readonly SkillTypeChillingArea: "ChillingArea";
    readonly SkillTypeAppliesCurse: "AppliesCurse";
    readonly SkillTypeCanRapidFire: "CanRapidFire";
    readonly SkillTypeAuraDuration: "AuraDuration";
    readonly SkillTypeAreaSpell: "AreaSpell";
    readonly SkillTypeOR: "OR";
    readonly SkillTypeAND: "AND";
    readonly SkillTypeNOT: "NOT";
    readonly SkillTypePhysical: "Physical";
    readonly SkillTypeAppliesMaim: "AppliesMaim";
    readonly SkillTypeCreatesMinion: "CreatesMinion";
    readonly SkillTypeGuard: "Guard";
    readonly SkillTypeTravel: "Travel";
    readonly SkillTypeBlink: "Blink";
    readonly SkillTypeCanHaveBlessing: "CanHaveBlessing";
    readonly SkillTypeProjectilesNotFromUser: "ProjectilesNotFromUser";
    readonly SkillTypeAttackInPlaceIsDefault: "AttackInPlaceIsDefault";
    readonly SkillTypeNova: "Nova";
    readonly SkillTypeInstantNoRepeatWhenHeld: "InstantNoRepeatWhenHeld";
    readonly SkillTypeInstantShiftAttackForLeftMouse: "InstantShiftAttackForLeftMouse";
    readonly SkillTypeAuraNotOnCaster: "AuraNotOnCaster";
    readonly SkillTypeBanner: "Banner";
    readonly SkillTypeRain: "Rain";
    readonly SkillTypeCooldown: "Cooldown";
    readonly SkillTypeThresholdJewelChaining: "ThresholdJewelChaining";
    readonly SkillTypeSlam: "Slam";
    readonly SkillTypeStance: "Stance";
    readonly SkillTypeNonRepeatable: "NonRepeatable";
    readonly SkillTypeOtherThingUsesSkill: "OtherThingUsesSkill";
    readonly SkillTypeSteel: "Steel";
    readonly SkillTypeHex: "Hex";
    readonly SkillTypeMark: "Mark";
    readonly SkillTypeAegis: "Aegis";
    readonly SkillTypeOrb: "Orb";
    readonly SkillTypeKillNoDamageModifiers: "KillNoDamageModifiers";
    readonly SkillTypeRandomElement: "RandomElement";
    readonly SkillTypeLateConsumeCooldown: "LateConsumeCooldown";
    readonly SkillTypeArcane: "Arcane";
    readonly SkillTypeFixedCastTime: "FixedCastTime";
    readonly SkillTypeRequiresOffHandNotWeapon: "RequiresOffHandNotWeapon";
    readonly SkillTypeLink: "Link";
    readonly SkillTypeBlessing: "Blessing";
    readonly SkillTypeZeroReservation: "ZeroReservation";
    readonly SkillTypePact: "Pact";
    readonly SkillTypeRetaliation: "Retaliation";
    readonly SkillTypeNeverExertable: "NeverExertable";
    readonly SkillTypeMinionsAreUndamagable: "MinionsAreUndamagable";
  };
  interface Sprite {
    Filename: string;
    W: number;
    H: number;
    Coords?: Record<string, Readonly<data.Coord>>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface Sprites {
    Background?: Record<string, Readonly<data.Sprite>>;
    NormalActive?: Record<string, Readonly<data.Sprite>>;
    NotableActive?: Record<string, Readonly<data.Sprite>>;
    KeystoneActive?: Record<string, Readonly<data.Sprite>>;
    NormalInactive?: Record<string, Readonly<data.Sprite>>;
    NotableInactive?: Record<string, Readonly<data.Sprite>>;
    KeystoneInactive?: Record<string, Readonly<data.Sprite>>;
    Mastery?: Record<string, Readonly<data.Sprite>>;
    MasteryConnected?: Record<string, Readonly<data.Sprite>>;
    MasteryActiveSelected?: Record<string, Readonly<data.Sprite>>;
    MasteryInactive?: Record<string, Readonly<data.Sprite>>;
    MasteryActiveEffect?: Record<string, Readonly<data.Sprite>>;
    AscendancyBackground?: Record<string, Readonly<data.Sprite>>;
    Ascendancy?: Record<string, Readonly<data.Sprite>>;
    StartNode?: Record<string, Readonly<data.Sprite>>;
    GroupBackground?: Record<string, Readonly<data.Sprite>>;
    Frame?: Record<string, Readonly<data.Sprite>>;
    Jewel?: Record<string, Readonly<data.Sprite>>;
    Line?: Record<string, Readonly<data.Sprite>>;
    JewelRadius?: Record<string, Readonly<data.Sprite>>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface SubgraphNode {
    readonly ID: number;
    readonly Type: string;
    readonly Linked?: Array<number>;
  }
  interface Tree {
    Tree: string;
    Classes?: Array<data.Class>;
    Groups?: Record<string, Readonly<data.Group>>;
    Nodes?: Record<string, Readonly<data.Node>>;
    ExtraImages?: Record<string, Readonly<data.ExtraImage>>;
    JewelSlots?: Array<number>;
    MinX: number;
    MinY: number;
    MaxX: number;
    MaxY: number;
    Constants: data.Constants;
    Sprites: data.Sprites;
    ImageZoomLevels?: Array<number>;
    Points: data.Points;
    AlternateAscendancies?: Array<data.Ascendancy>;
    JewelRadiusNodes(): (Record<number, Array<Array<number> | undefined> | undefined> | undefined);
    NodePosition(id: number): [number, number, boolean];
    NodesInRadiusOf(centre: number, radiusIndex: number): (Record<number, boolean> | undefined);
    OrbitPosition(group: number, orbit: number, oidx: number): [number, number, boolean];
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
}
export declare namespace debug {
  interface BuildInfo {
    GoVersion: string;
    Path: string;
    Main: debug.Module;
    Deps?: Array<debug.Module | undefined>;
    Settings?: Array<debug.BuildSetting>;
    String(): string;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface BuildSetting {
    Key: string;
    Value: string;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface Module {
    Path: string;
    Version: string;
    Sum: string;
    Replace?: debug.Module;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
}
export declare namespace exposition {
  interface AddedItem {
    readonly ID: number;
    readonly SlotName: string;
  }
  interface AddedModCompare {
    readonly Base: exposition.StatOutputs;
    readonly Added: exposition.StatOutputs;
  }
  interface AnointCompare {
    readonly Note: string;
    readonly Header: string;
    readonly Base: exposition.StatOutputs;
    readonly New: exposition.StatOutputs;
  }
  interface ElementWrapper {
    readonly Elements?: Record<string, Readonly<calculator.ColProps>>;
  }
  interface EquippedSlot {
    readonly SlotName: string;
    readonly ItemID: number;
    readonly Name: string;
    readonly Rarity: string;
    readonly Flask: boolean;
    readonly Active: boolean;
    readonly Text: string;
  }
  interface FlaskCompare {
    readonly Active: boolean;
    readonly Tincture: boolean;
    readonly Output: exposition.StatOutputs;
    readonly Effective?: Array<string>;
  }
  interface GemPart {
    readonly Name: string;
    readonly Description: string;
  }
  type GemType = "STR" | "DEX" | "INT" | "NONE";
  const GemType: {
    readonly GemTypeStrength: "STR";
    readonly GemTypeDexterity: "DEX";
    readonly GemTypeIntelligence: "INT";
    readonly GemTypeNone: "NONE";
  };
  interface ItemDBInfo {
    readonly Slots?: Array<string>;
    readonly Types?: Array<string>;
    readonly Leagues?: Array<string>;
    readonly Requirements?: Array<string>;
    readonly Obtainable?: Array<string>;
    readonly SearchModes?: Array<string>;
    readonly SortStats?: Array<exposition.ItemDBSortStat>;
  }
  interface ItemDBRow {
    readonly Name: string;
    readonly Title: string;
    readonly BaseName: string;
    readonly Rarity: string;
    readonly Type: string;
    readonly PrimarySlot: string;
    readonly Level: number;
    readonly League: string;
    readonly Source: string;
    readonly Power: number;
  }
  interface ItemDBSortStat {
    readonly Stat: string;
    readonly Label: string;
  }
  interface ItemModLine {
    readonly Line: string;
    readonly Kind: string;
  }
  interface ItemProperty {
    readonly Label: string;
    readonly Value: string;
  }
  interface ItemRef {
    readonly Name: string;
    readonly Rarity: string;
  }
  interface ItemSetTip {
    readonly ID: string;
    readonly Lines?: Array<string>;
  }
  interface ItemsTabSlotList {
    readonly Abyssal?: Array<string>;
    readonly Sockets?: Array<exposition.SocketSlot>;
  }
  interface MiscCalculator {
    AddItem(text: string): Promise<Result<exposition.AddedItem>>;
    AffixPowers(text: string, slot: number, stat: string): (Array<number> | undefined);
    AnointPowers(text: string, slot: number, ids: Array<number> | undefined, stat: string): (Array<number> | undefined);
    Base(): exposition.StatOutputs;
    CalcNodes(addNodes: Array<number> | undefined, removeNodes: Array<number> | undefined): Promise<exposition.StatOutputs>;
    CalcToggleFlask(text: string): Promise<Result<exposition.FlaskCompare>>;
    CalcUnequip(slotName: string): Promise<exposition.StatOutputs>;
    CompareAddedMod(text: string, lines: Array<string> | undefined, modType: string): [exposition.AddedModCompare, boolean];
    CompareAnoint(text: string, slot: number, id: number): [exposition.AnointCompare, boolean];
    CompareItem(text: string): Promise<Result<(Array<exposition.SlotCompare> | undefined)>>;
    ConditionalSlots(): (Array<string> | undefined);
    CorruptSortValues(text: string, source: string, modType: string, ids: Array<string> | undefined, stat: string): (Array<number> | undefined);
    CustomModSortValues(text: string, sourceID: string, ids: Array<string> | undefined, stat: string): (Array<number> | undefined);
    EnchantSortValues(text: string, slot: number, lines: Array<string> | undefined, stat: string): (Array<number> | undefined);
    ImplicitSortValues(text: string, sourceID: string, ids: Array<string> | undefined, stat: string): (Array<number> | undefined);
    ItemDBRows(kind: string, f: calculator.ItemDBFilter, level: number, str: number, dex: number, intel: number, stat: string): (Array<exposition.ItemDBRow> | undefined);
    ReplaceItem(id: number, text: string): Promise<Result<void>>;
    SocketDistanceToClassStart(nodeID: number): number;
    SocketJewelTooltip(nodeID: number, stats: calculator.ItemTooltipStats): [calculator.ItemTooltip, boolean];
    SplitPersonalityPath(): (Array<number> | undefined);
    TimelessFallbackWeights(jewelType: number, stat: string): [string, boolean];
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ModLine {
    readonly Status: exposition.ModLineStatus;
    readonly Extra: string;
    readonly Mods?: Array<exposition.ParsedMod>;
  }
  type ModLineStatus = "blank" | "comment" | "parsed" | "unsupported";
  const ModLineStatus: {
    readonly ModLineBlank: "blank";
    readonly ModLineComment: "comment";
    readonly ModLineParsed: "parsed";
    readonly ModLineUnsupported: "unsupported";
  };
  interface ParsedItem {
    readonly Name: string;
    readonly Title: string;
    readonly BaseName: string;
    readonly Type: string;
    readonly Rarity: string;
    readonly Quality: number;
    readonly ItemLevel: number;
    readonly Corrupted: boolean;
    readonly Mirrored: boolean;
    readonly Split: boolean;
    readonly Fractured: boolean;
    readonly Synthesised: boolean;
    readonly Influences?: Array<string>;
    readonly Sockets: string;
    readonly Properties?: Array<exposition.ItemProperty>;
    readonly Level: number;
    readonly Str: number;
    readonly Dex: number;
    readonly Int: number;
    readonly Enchants?: Array<exposition.ItemModLine>;
    readonly Scourge?: Array<exposition.ItemModLine>;
    readonly Implicits?: Array<exposition.ItemModLine>;
    readonly Explicits?: Array<exposition.ItemModLine>;
    readonly Flask: boolean;
    readonly FitSlots?: Array<string>;
    readonly Tooltip: calculator.ItemTooltip;
    readonly ArmourData?: Record<string, number>;
  }
  interface ParsedMod {
    readonly Name: string;
    readonly Type: string;
    readonly Value: string;
    readonly Tags?: Array<string>;
  }
  interface PowerBuilder {
    Current(): number;
    Report(): (Array<calculator.PowerReportEntry> | undefined);
    Shard(): calculator.PowerShardRows;
    Step(budgetMs: number): number;
    Tasks(): number;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface SharedItemSetRow {
    readonly Title: string;
    readonly Tip?: Array<string>;
  }
  interface SidebarPass {
    Breakdown(minion: boolean, stat: string): (Array<string> | undefined);
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface SkillGem {
    readonly MaxLevel: number;
    readonly ID: string;
    readonly GemType: exposition.GemType;
    readonly Base: exposition.GemPart;
    readonly Vaal: exposition.GemPart;
    readonly Support: boolean;
    readonly Name: string;
    readonly Tags?: Array<string>;
    readonly LevelRequirement: number;
    readonly GlobalEffects?: Array<string>;
    readonly Legacy: boolean;
    readonly HideFromGemList: boolean;
    readonly LegacyAwakened: boolean;
  }
  interface SlotCompare {
    readonly SlotName: string;
    readonly Label: string;
    readonly Replacing?: exposition.ItemRef;
    readonly Output: exposition.StatOutputs;
  }
  interface SlotItems {
    readonly SlotName: string;
    readonly ItemIDs?: Array<number>;
  }
  interface SocketSlot {
    readonly NodeID: number;
    readonly ItemID: number;
    readonly Choices?: Array<number>;
  }
  interface StatOutputs {
    readonly Output?: Record<string, number>;
    readonly Minion?: Record<string, number>;
    readonly SkillFlags?: Record<calculator.SkillFlag, boolean>;
    readonly GrantedPassives?: Array<number>;
  }
  interface TattooArt {
    readonly NodeID: number;
    readonly Icon: string;
    readonly ActiveEffectImage: string;
  }
  interface TattooChoice {
    readonly DN: string;
    readonly Label: string;
    readonly Descriptions?: Array<string>;
    readonly Legacy: boolean;
  }
  interface TattooNodeTip {
    readonly NodeID: number;
    readonly Name: string;
    readonly Stats?: Array<string>;
    readonly Reminder?: Array<string>;
  }
  interface TimelessSearch {
    AddItem(build: pob.PathOfBuilding | undefined, row: number, devotion1: number, devotion2: number, socketLabel: string): number;
    Count(): number;
    Desired(): (Array<calculator.TimelessDesiredNode> | undefined);
    Item(row: number, devotion1: number, devotion2: number, socketLabel: string): string;
    Rows(start: number, count: number): (Array<calculator.TimelessSearchResult> | undefined);
    Step(): number;
    TradeURL(q: calculator.TimelessTradeQuery): calculator.TimelessTrade;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface TreeJewel {
    readonly NodeID: number;
    readonly ItemID: number;
    readonly Name: string;
    readonly Rarity: string;
  }
  interface TreeSpecList {
    readonly Active: number;
    readonly Specs?: Array<calculator.TreeSpecRow>;
  }
  interface TreeVersionOption {
    readonly Value: string;
    readonly Label: string;
  }
  function AddDBItem(build: pob.PathOfBuilding | undefined, kind: string, name: string, equip: boolean, second: boolean): Promise<[number, boolean]>;
  function AddSharedItem(build: pob.PathOfBuilding | undefined, index: number, slotName: string): Promise<number>;
  function AddTattoo(build: pob.PathOfBuilding | undefined, nodeID: number, dn: string): Result<void>;
  function AffixEditText(text: string, slot: number, series: number, slider: number): Promise<[string, boolean]>;
  function AffixRollTooltip(text: string, slot: number, slider: number): Promise<(Array<string> | undefined)>;
  function AnointNotables(): Promise<(Array<calculator.AnointNotable> | undefined)>;
  function CalcConfigStats(m: exposition.MiscCalculator | undefined, input: pob.Input, remove: boolean): Promise<exposition.StatOutputs>;
  function CalcCustomModGroupStats(m: exposition.MiscCalculator | undefined, i: number): Promise<exposition.StatOutputs>;
  function CalcGemStats(m: exposition.MiscCalculator | undefined, socketGroup: number, gemIndex: number, gem: pob.Gem): Promise<exposition.StatOutputs>;
  function CalcMasteryEffect(m: exposition.MiscCalculator | undefined, nodeID: number, effect: number, addNodes: Array<number> | undefined): Promise<exposition.StatOutputs>;
  function CalcTreeSpec(m: exposition.MiscCalculator | undefined, specID: number): Promise<exposition.StatOutputs>;
  function CalcTreeVersion(): string;
  function CalculateAllocationPaths(version: string, activeNodes: Array<number> | undefined, rootNodes: Array<number> | undefined): (Record<number, number> | undefined);
  function CalculateDependentNodes(version: string, activeNodes: Array<number> | undefined, target: number, rootNodes: Array<number> | undefined): (Array<number> | undefined);
  function CalculateDependentNodesWith(version: string, activeNodes: Array<number> | undefined, target: number, rootNodes: Array<number> | undefined, subgraph: Array<data.SubgraphNode> | undefined): (Array<number> | undefined);
  function CalculatePrunableNodes(version: string, activeNodes: Array<number> | undefined, rootNodes: Array<number> | undefined): (Array<number> | undefined);
  function ChangeSecondaryAscendancy(build: pob.PathOfBuilding | undefined, ascendClassID: number): Promise<void>;
  function ChangeTreeAscendancy(build: pob.PathOfBuilding | undefined, ascendancy: string): Promise<void>;
  function ChangeTreeClass(build: pob.PathOfBuilding | undefined, className: string, decision: number): Promise<calculator.ClassChange>;
  function ClickTreeAscendancyNode(build: pob.PathOfBuilding | undefined, nodeID: number, decision: number): Promise<calculator.ClassChange>;
  function CompareSpecJewelTooltip(build: pob.PathOfBuilding | undefined, specID: number, nodeID: number, stats: calculator.ItemTooltipStats): [calculator.ItemTooltip, boolean];
  function ConfigPlaceholders(build: pob.PathOfBuilding | undefined): (Record<string, number> | undefined);
  function ConvertAllTreeSpecs(build: pob.PathOfBuilding | undefined, version: string): Promise<string>;
  function ConvertTreeSpec(build: pob.PathOfBuilding | undefined, version: string, replace: boolean): Promise<string>;
  function CopyItemSet(build: pob.PathOfBuilding | undefined, itemSetID: number, title: string): Promise<number>;
  function CorruptItemText(text: string, source: string, ids: Array<string> | undefined, downside: Array<string> | undefined, rolls: boolean, rollIndex: Array<number> | undefined, rollValue: Array<number> | undefined): Promise<[string, boolean]>;
  function CorruptedRangeLine(text: string, index: number, value: number): Promise<string>;
  function CraftBaseGroups(): Promise<(Array<calculator.CraftBaseGroup> | undefined)>;
  function CraftItemText(rarity: number, title: string, baseName: string): Promise<[string, boolean]>;
  function CrucibleText(text: string, ids: Array<string> | undefined): Promise<[string, boolean]>;
  function CurrentGameVersionLabel(): string;
  function CustomModText(text: string, sourceID: string, modID: string, custom: string): Promise<[string, boolean]>;
  function DefaultGem(gemID: string, nameSpec: string, skillID: string, level: number, quality: number): pob.Gem;
  function DeleteAllItems(build: pob.PathOfBuilding | undefined): Promise<void>;
  function DeleteItem(build: pob.PathOfBuilding | undefined, itemID: number): Promise<boolean>;
  function DeleteItemPrompt(build: pob.PathOfBuilding | undefined, itemID: number): Promise<string>;
  function DeleteItemSet(build: pob.PathOfBuilding | undefined, itemSetID: number): Promise<void>;
  function DeleteSharedItem(index: number): Promise<boolean>;
  function DeleteSharedItemSet(index: number): Promise<boolean>;
  function DeleteTreeSpec(build: pob.PathOfBuilding | undefined, specID: number): Promise<void>;
  function DeleteUnusedItems(build: pob.PathOfBuilding | undefined): Promise<boolean>;
  function DisplayItem(text: string): Promise<[calculator.DisplayItemInfo, boolean]>;
  function DropItemOnSlot(build: pob.PathOfBuilding | undefined, slotName: string, id: number, kind: string, name: string): Promise<boolean>;
  function EditItem(text: string, edit: calculator.ItemEdit): Promise<Result<string>>;
  function EquipItem(build: pob.PathOfBuilding | undefined, text: string, slotName: string): Promise<Result<number>>;
  function EquipItemID(build: pob.PathOfBuilding | undefined, slotName: string, itemID: number): Promise<Result<boolean>>;
  function EquipItemInSet(build: pob.PathOfBuilding | undefined, setID: number, id: number, kind: string, name: string, second: boolean): Promise<boolean>;
  function EquippedItems(build: pob.PathOfBuilding | undefined): Promise<(Array<exposition.EquippedSlot> | undefined)>;
  function ExportTreeLink(build: pob.PathOfBuilding | undefined): Promise<string>;
  function GemDefaultLevel(build: pob.PathOfBuilding | undefined, gemID: string): number;
  function GemDefaultLevelMode(build: pob.PathOfBuilding | undefined): string;
  function GemError(gemID: string, skillID: string, nameSpec: string): string;
  function GemQualityTooltip(gem: pob.Gem): (Array<calculator.TooltipLine> | undefined);
  function GemTooltip(build: pob.PathOfBuilding | undefined, gem: pob.Gem, display: calculator.GemDisplay, hasDisplay: boolean, stats: calculator.ItemTooltipStats): (Array<calculator.TooltipLine> | undefined);
  function GetLoadouts(build: pob.PathOfBuilding | undefined): Promise<calculator.LoadoutList>;
  function GetMasteryOptions(build: pob.PathOfBuilding | undefined, nodeID: number): Promise<(Array<calculator.MasteryOption> | undefined)>;
  function GetMasterySelections(build: pob.PathOfBuilding | undefined): (Record<number, number> | undefined);
  function GetMiscCalculator(build: pob.PathOfBuilding | undefined): Promise<(exposition.MiscCalculator | undefined)>;
  function GetPartySummaries(build: pob.PathOfBuilding | undefined): (Record<string, Array<calculator.PartySummaryEntry> | undefined> | undefined);
  function GetRawTree(version: string): Promise<(Uint8Array | undefined)>;
  function GetSkillGems(): (Array<exposition.SkillGem> | undefined);
  function GetStatByIndex(id: number): (poe.Stat | undefined);
  function GetTattooChoices(build: pob.PathOfBuilding | undefined, nodeID: number, showLegacy: boolean): (Array<exposition.TattooChoice> | undefined);
  function GetTattooCountLines(build: pob.PathOfBuilding | undefined): Promise<(Array<calculator.TattooCountLine> | undefined)>;
  function GetTimelessData(build: pob.PathOfBuilding | undefined): pob.TimelessData;
  function GetTreeClusterNodes(build: pob.PathOfBuilding | undefined): Promise<(Array<calculator.TreeClusterNode> | undefined)>;
  function GetTreeJewelOverlays(build: pob.PathOfBuilding | undefined): Promise<(Array<calculator.TreeJewelOverlay> | undefined)>;
  function GetTreeJewelRings(build: pob.PathOfBuilding | undefined, compared: Array<number> | undefined): Promise<(Array<calculator.TreeJewelRing> | undefined)>;
  function GetTreeSocketRadii(build: pob.PathOfBuilding | undefined, nodeID: number): Promise<(Array<calculator.TreeSocketRadius> | undefined)>;
  function GetTreeSpecClusterNodes(build: pob.PathOfBuilding | undefined, specID: number): Promise<(Array<calculator.TreeClusterNode> | undefined)>;
  function GetTreeSpecs(build: pob.PathOfBuilding | undefined): Promise<exposition.TreeSpecList>;
  function GetTreeSpecsPoints(build: pob.PathOfBuilding | undefined): Promise<(Array<calculator.TreeSpecPoints> | undefined)>;
  function GetTreeVersionOptions(): (Array<exposition.TreeVersionOption> | undefined);
  function GoldRespecPrice(build: pob.PathOfBuilding | undefined): number;
  function ImplicitText(text: string, sourceID: string, modID: string, custom: string): Promise<[string, boolean]>;
  function ImportAccountItems(build: pob.PathOfBuilding | undefined, characterJSON: string, itemsJSON: string, clearItems: boolean, clearSkills: boolean, ignoreWeaponSwap: boolean): Promise<Result<string>>;
  function ImportAccountPassiveTree(build: pob.PathOfBuilding | undefined, characterJSON: string, passivesJSON: string, deleteJewels: boolean): Promise<Result<string>>;
  function ImportItemsAndSkills(build: pob.PathOfBuilding | undefined, characterJSON: string, clearItems: boolean, clearSkills: boolean, ignoreWeaponSwap: boolean): Promise<Result<string>>;
  function ImportPassiveTreeAndJewels(build: pob.PathOfBuilding | undefined, characterJSON: string, deleteJewels: boolean): Promise<Result<string>>;
  function ImportSharedItemSet(build: pob.PathOfBuilding | undefined, index: number): Promise<number>;
  function ImportTreeLink(build: pob.PathOfBuilding | undefined, title: string, link: string): Promise<string>;
  function ItemAffixControls(text: string): Promise<[calculator.AffixControls, boolean]>;
  function ItemCorruptChoices(text: string): Promise<[calculator.CorruptChoices, boolean]>;
  function ItemCrucibleChoices(text: string): Promise<[calculator.CrucibleChoices, boolean]>;
  function ItemCustomModChoices(text: string): Promise<[calculator.CustomModChoices, boolean]>;
  function ItemDBLists(kind: string): Promise<exposition.ItemDBInfo>;
  function ItemDBRaw(kind: string, name: string): Promise<[string, boolean]>;
  function ItemDBText(kind: string, name: string): Promise<[string, boolean]>;
  function ItemDBTooltip(build: pob.PathOfBuilding | undefined, kind: string, name: string, stats: calculator.ItemTooltipStats): Promise<[calculator.ItemTooltip, boolean]>;
  function ItemEnchantChoices(build: pob.PathOfBuilding | undefined, text: string): Promise<[calculator.EnchantChoices, boolean]>;
  function ItemImplicitChoices(text: string): Promise<[calculator.ImplicitChoices, boolean]>;
  function ItemListFiltered(build: pob.PathOfBuilding | undefined, filter: string): Promise<(Array<number> | undefined)>;
  function ItemListLoadoutFilters(build: pob.PathOfBuilding | undefined): Promise<(Array<string> | undefined)>;
  function ItemPrimarySlot(build: pob.PathOfBuilding | undefined, id: number, kind: string, name: string): Promise<string>;
  function ItemSetTooltips(build: pob.PathOfBuilding | undefined): Promise<(Array<exposition.ItemSetTip> | undefined)>;
  function ItemSourceTooltip(build: pob.PathOfBuilding | undefined, name: string, stats: calculator.ItemTooltipStats): Promise<[calculator.ItemTooltip, boolean]>;
  function ItemsTabSlots(build: pob.PathOfBuilding | undefined): Promise<exposition.ItemsTabSlotList>;
  function LeapHeldNodes(build: pob.PathOfBuilding | undefined): Promise<(Array<number> | undefined)>;
  function LiveHeapMB(): number;
  function MergePowerAscendancy(shards: Array<calculator.PowerShardRows> | undefined): (Array<calculator.PowerReportEntry> | undefined);
  function MergePowerReports(shards: Array<calculator.PowerShardRows> | undefined): (Array<calculator.PowerReportEntry> | undefined);
  function MiscCalculatorFor(build: pob.PathOfBuilding | undefined, calc: calculator.Calculator | undefined): (exposition.MiscCalculator | undefined);
  function MiscGemDisplays(m: exposition.MiscCalculator | undefined): (Array<calculator.GemDisplayRow> | undefined);
  function MiscGemSortCache(m: exposition.MiscCalculator | undefined, group: number, gems: Array<pob.Gem> | undefined): (Array<calculator.GemSortRow> | undefined);
  function MiscGemSupportPairs(m: exposition.MiscCalculator | undefined, group: number): (Array<calculator.GemSupportPair> | undefined);
  function MiscSocketGroupTooltip(m: exposition.MiscCalculator | undefined, group: number): (Array<calculator.TooltipLine> | undefined);
  function ModBrowserList(search: string): (Array<string> | undefined);
  function ModBrowserSupported(line: string): boolean;
  function ModDebugDump(c: calculator.Calculator | undefined): Promise<string>;
  function MoveItemSet(build: pob.PathOfBuilding | undefined, from: number, to: number): Promise<void>;
  function MoveTreeSpec(build: pob.PathOfBuilding | undefined, from: number, to: number): Promise<void>;
  function NewItemSet(build: pob.PathOfBuilding | undefined, title: string): Promise<number>;
  function NewLoadout(build: pob.PathOfBuilding | undefined, name: string): Promise<void>;
  function NewPowerBuilder(m: exposition.MiscCalculator | undefined, stat: string, maxDepth: number): (exposition.PowerBuilder | undefined);
  function NewPowerBuilderShard(m: exposition.MiscCalculator | undefined, stat: string, maxDepth: number, index: number, count: number): (exposition.PowerBuilder | undefined);
  function NewSidebarPass(build: pob.PathOfBuilding | undefined): (exposition.SidebarPass | undefined);
  function NewTimelessSearch(m: exposition.MiscCalculator | undefined, q: calculator.TimelessSearchQuery): (exposition.TimelessSearch | undefined);
  function NewTreeSpec(build: pob.PathOfBuilding | undefined, title: string): Promise<void>;
  function NodePowerStats(): (Array<calculator.PowerStat> | undefined);
  function NodeSkillTooltips(build: pob.PathOfBuilding | undefined, nodeID: number, stats: calculator.ItemTooltipStats): (Array<Array<calculator.TooltipLine> | undefined> | undefined);
  function NormaliseDisplayItem(build: pob.PathOfBuilding | undefined, text: string, copyEldritchImplicits: boolean): Promise<string>;
  function OptimiseSockets(build: pob.PathOfBuilding | undefined, slotName: string): Promise<boolean>;
  function ParseItem(build: pob.PathOfBuilding | undefined, text: string, stats: calculator.ItemTooltipStats): Promise<Result<exposition.ParsedItem>>;
  function ParseModLines(lines: Array<string> | undefined): (Array<exposition.ModLine> | undefined);
  function PowerAscendancy(p: exposition.PowerBuilder | undefined): (Array<calculator.PowerReportEntry> | undefined);
  function PowerStats(): (Array<calculator.PowerStat> | undefined);
  function RebaseChoices(text: string): Promise<(Array<string> | undefined)>;
  function RebaseItemText(text: string, baseName: string): Promise<[calculator.RebaseResult, boolean]>;
  function RemoveAllTreeTattoos(build: pob.PathOfBuilding | undefined): Promise<void>;
  function RenameItemSet(build: pob.PathOfBuilding | undefined, itemSetID: number, title: string): Promise<void>;
  function RenameSharedItemSet(index: number, title: string): Promise<boolean>;
  function ResetTreeNodes(build: pob.PathOfBuilding | undefined): Promise<void>;
  function SecondaryAscendancies(build: pob.PathOfBuilding | undefined): Promise<calculator.SecondaryAscendancyDropdown>;
  function SelectLoadout(build: pob.PathOfBuilding | undefined, name: string): Promise<boolean>;
  function SelectMasteryEffect(build: pob.PathOfBuilding | undefined, nodeID: number, effectID: number, path: Array<number> | undefined): Promise<Result<void>>;
  function SetActiveItemSet(build: pob.PathOfBuilding | undefined, itemSetID: number): Promise<void>;
  function SetActiveTreeSpec(build: pob.PathOfBuilding | undefined, specID: number): Promise<void>;
  function SetCalcTabElements(elements: exposition.ElementWrapper | undefined): void;
  function SetItemModDebug(on: boolean): Promise<void>;
  function SetShowFlavourText(on: boolean): Promise<void>;
  function SetTimelessData(build: pob.PathOfBuilding | undefined, d: pob.TimelessData): void;
  function SetWeaponSet(build: pob.PathOfBuilding | undefined, second: boolean): Promise<boolean>;
  function ShareItem(build: pob.PathOfBuilding | undefined, id: number, kind: string, name: string, index: number): Promise<boolean>;
  function ShareItemSet(build: pob.PathOfBuilding | undefined, setID: number, index: number): Promise<boolean>;
  function SharedItemSets(): Promise<(Array<exposition.SharedItemSetRow> | undefined)>;
  function SharedItems(): Promise<(Array<string> | undefined)>;
  function SlotItemLists(build: pob.PathOfBuilding | undefined): Promise<(Array<exposition.SlotItems> | undefined)>;
  function SocketGroupSourceNames(build: pob.PathOfBuilding | undefined, setIndex: number, groupIndex: number): Promise<(Array<calculator.GroupSourceName> | undefined)>;
  function SocketTreeJewel(build: pob.PathOfBuilding | undefined, nodeID: number, itemID: number): Promise<Result<boolean>>;
  function SocketedInItemNames(build: pob.PathOfBuilding | undefined, slotNames: Array<string> | undefined): Promise<(Record<string, string> | undefined)>;
  function SortItems(build: pob.PathOfBuilding | undefined): Promise<void>;
  function SpectreLibrary(): (Array<calculator.SpectreRow> | undefined);
  function SpectreRows(ids: Array<string> | undefined): (Array<calculator.SpectreRow> | undefined);
  function TattooEditKind(build: pob.PathOfBuilding | undefined, nodeID: number): string;
  function TimelessConquerors(jewelType: number): (Array<string> | undefined);
  function TimelessDevotionVariants(): (Array<string> | undefined);
  function TimelessFallbackModes(): (Array<calculator.PowerStat> | undefined);
  function TimelessJewelTypes(): (Array<calculator.TimelessJewelType> | undefined);
  function TimelessMods(jewelType: number): (Array<calculator.TimelessMod> | undefined);
  function TimelessProtectableNodes(m: exposition.MiscCalculator | undefined, socketID: number): (Array<calculator.TimelessMod> | undefined);
  function TimelessRealms(): (Array<string> | undefined);
  function TimelessSockets(build: pob.PathOfBuilding | undefined): (Array<calculator.TimelessSocket> | undefined);
  function TimelessTradeTypes(): (Array<string> | undefined);
  function ToggleEquipItem(build: pob.PathOfBuilding | undefined, itemID: number, second: boolean): Promise<boolean>;
  function TreeConqueredNodes(m: exposition.MiscCalculator | undefined): (Array<calculator.SpecNodeRewrite> | undefined);
  function TreeSocketChoices(build: pob.PathOfBuilding | undefined, nodeID: number): Promise<(Array<exposition.TreeJewel> | undefined)>;
  function TreeSocketedJewels(build: pob.PathOfBuilding | undefined): Promise<(Array<exposition.TreeJewel> | undefined)>;
  function TreeSpecCompare(build: pob.PathOfBuilding | undefined, specID: number): calculator.TreeCompareSpec;
  function TreeSpecRespecCost(build: pob.PathOfBuilding | undefined, index: number): calculator.TreeSpecRespec;
  function TreeTattooArt(build: pob.PathOfBuilding | undefined): (Array<exposition.TattooArt> | undefined);
  function TreeTattooNodes(build: pob.PathOfBuilding | undefined): (Array<exposition.TattooNodeTip> | undefined);
}
export declare namespace fwd {
  interface Reader {
    BufferSize(): number;
    Buffered(): number;
    InputOffset(): number;
    Next(n: number): Result<(Uint8Array | undefined)>;
    Peek(n: number): Result<(Uint8Array | undefined)>;
    PeekByte(): Result<number>;
    Read(b: Uint8Array | undefined): Result<number>;
    ReadByte(): Result<number>;
    ReadFull(b: Uint8Array | undefined): Result<number>;
    Skip(n: number): Result<number>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
}
export declare namespace io {
  interface Reader {
    Read(p: Uint8Array | undefined): Result<number>;
  }
  interface Writer {
    Write(p: Uint8Array | undefined): Result<number>;
  }
}
export declare namespace mod {
  type KeywordFlag = 1 | 2 | 4 | 8 | 16 | 32 | 64 | 128 | 256 | 512 | 1024 | 2048 | 4096 | 8192 | 16384 | 32768 | 65536 | 131072 | 262144 | 524288 | 1048576 | 2097152 | 4194304 | 8388608 | 16777216 | 33554432 | 67108864 | 134217728 | 268435456 | 1073741824 | -1073741825;
  const KeywordFlag: {
    readonly KeywordFlagAura: 1;
    readonly KeywordFlagCurse: 2;
    readonly KeywordFlagWarcry: 4;
    readonly KeywordFlagMovement: 8;
    readonly KeywordFlagPhysical: 16;
    readonly KeywordFlagFire: 32;
    readonly KeywordFlagCold: 64;
    readonly KeywordFlagLightning: 128;
    readonly KeywordFlagChaos: 256;
    readonly KeywordFlagVaal: 512;
    readonly KeywordFlagBow: 1024;
    readonly KeywordFlagArrow: 2048;
    readonly KeywordFlagTrap: 4096;
    readonly KeywordFlagMine: 8192;
    readonly KeywordFlagTotem: 16384;
    readonly KeywordFlagMinion: 32768;
    readonly KeywordFlagAttack: 65536;
    readonly KeywordFlagSpell: 131072;
    readonly KeywordFlagHit: 262144;
    readonly KeywordFlagAilment: 524288;
    readonly KeywordFlagBrand: 1048576;
    readonly KeywordFlagPoison: 2097152;
    readonly KeywordFlagBleed: 4194304;
    readonly KeywordFlagIgnite: 8388608;
    readonly KeywordFlagPhysicalDot: 16777216;
    readonly KeywordFlagLightningDot: 33554432;
    readonly KeywordFlagColdDot: 67108864;
    readonly KeywordFlagFireDot: 134217728;
    readonly KeywordFlagChaosDot: 268435456;
    readonly KeywordFlagMatchAll: 1073741824;
    readonly MatchAllMask: -1073741825;
  };
  type MFlag = 1 | 2 | 4 | 8 | 16 | 256 | 512 | 1024 | 1536 | 2048 | 4096 | 8192 | 65536 | 131072 | 262144 | 524288 | 1048576 | 2097152 | 4194304 | 8388608 | 16777216 | 33554432 | 67108864 | 134217728 | 268435456 | 536870912 | 1073676288;
  const MFlag: {
    readonly MFlagAttack: 1;
    readonly MFlagSpell: 2;
    readonly MFlagHit: 4;
    readonly MFlagDot: 8;
    readonly MFlagCast: 16;
    readonly MFlagMelee: 256;
    readonly MFlagArea: 512;
    readonly MFlagProjectile: 1024;
    readonly MFlagSourceMask: 1536;
    readonly MFlagAilment: 2048;
    readonly MFlagMeleeHit: 4096;
    readonly MFlagWeapon: 8192;
    readonly MFlagAxe: 65536;
    readonly MFlagBow: 131072;
    readonly MFlagClaw: 262144;
    readonly MFlagDagger: 524288;
    readonly MFlagMace: 1048576;
    readonly MFlagStaff: 2097152;
    readonly MFlagSword: 4194304;
    readonly MFlagWand: 8388608;
    readonly MFlagUnarmed: 16777216;
    readonly MFlagFishing: 33554432;
    readonly MFlagWeaponMelee: 67108864;
    readonly MFlagWeaponRanged: 134217728;
    readonly MFlagWeapon1H: 268435456;
    readonly MFlagWeapon2H: 536870912;
    readonly MFlagWeaponMask: 1073676288;
  };
  interface ModValueMulti {
    ValueFloat: number;
    ValueFlag: boolean;
    Clone(): (mod.ModValueMulti | undefined);
    Flag(): boolean;
    Float(): number;
    SetFlag(v: boolean): void;
    SetFloat(v: number): void;
    Type(): mod.ModValueMultiType;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  type ModValueMultiType = "float" | "flag" | "list";
  const ModValueMultiType: {
    readonly ModValueMultiTypeFloat: "float";
    readonly ModValueMultiTypeFlag: "flag";
    readonly ModValueMultiTypeList: "list";
  };
  type Source = "Base" | "Config" | "Chill" | "Bonechill" | "Shock" | "Scorch" | "Brittle" | "Sap" | "Feeding Frenzy" | "Skill";
  const Source: {
    readonly SourceBase: "Base";
    readonly SourceConfig: "Config";
    readonly SourceChill: "Chill";
    readonly SourceBonechill: "Bonechill";
    readonly SourceShock: "Shock";
    readonly SourceScorch: "Scorch";
    readonly SourceBrittle: "Brittle";
    readonly SourceSap: "Sap";
    readonly SourceFeedingFrenzy: "Feeding Frenzy";
    readonly SourceSkill: "Skill";
  };
  interface Tag {
    Type(): mod.Type;
  }
  type Type = "INC" | "MORE" | "BASE" | "FLAG" | "OVERRIDE" | "LIST" | "SocketedIn" | "PerStat" | "PercentStat" | "Condition" | "ActorCondition" | "Multiplier" | "ModFlag" | "SkillType" | "SkillId" | "Global" | "SkillName" | "MAX" | "CHANCE" | "StatThreshold" | "SlotNumber" | "SlotName" | "ItemCondition" | "DistanceRamp" | "MultiplierThreshold" | "ModFlagOr" | "KeywordFlagAnd" | "InSlot" | "DUMMY" | "GlobalEffect" | "MeleeProximity" | "IgnoreCond" | "SkillPart" | "Limit" | "DisablesItem" | "MonsterTag" | "Flag";
  const Type: {
    readonly TypeIncrease: "INC";
    readonly TypeMore: "MORE";
    readonly TypeBase: "BASE";
    readonly TypeFlag: "FLAG";
    readonly TypeOverride: "OVERRIDE";
    readonly TypeList: "LIST";
    readonly TypeSocketedIn: "SocketedIn";
    readonly TypePerStat: "PerStat";
    readonly TypePercentStat: "PercentStat";
    readonly TypeCondition: "Condition";
    readonly TypeActorCondition: "ActorCondition";
    readonly TypeMultiplier: "Multiplier";
    readonly TypeModFlag: "ModFlag";
    readonly TypeSkillType: "SkillType";
    readonly TypeSkillID: "SkillId";
    readonly TypeGlobal: "Global";
    readonly TypeSkillName: "SkillName";
    readonly TypeMAX: "MAX";
    readonly TypeChance: "CHANCE";
    readonly TypeStatThreshold: "StatThreshold";
    readonly TypeSlotNumber: "SlotNumber";
    readonly TypeSlotName: "SlotName";
    readonly TypeItemCondition: "ItemCondition";
    readonly TypeDistanceRamp: "DistanceRamp";
    readonly TypeMultiplierThreshold: "MultiplierThreshold";
    readonly TypeModFlagOr: "ModFlagOr";
    readonly TypeKeywordFlagAnd: "KeywordFlagAnd";
    readonly TypeInSlot: "InSlot";
    readonly TypeDummy: "DUMMY";
    readonly TypeGlobalEffect: "GlobalEffect";
    readonly TypeMeleeProximity: "MeleeProximity";
    readonly TypeIgnoreCond: "IgnoreCond";
    readonly TypeSkillPart: "SkillPart";
    readonly TypeLimit: "Limit";
    readonly TypeDisablesItem: "DisablesItem";
    readonly TypeMonsterTag: "MonsterTag";
    readonly TypeDisablerFlag: "Flag";
  };
}
export declare namespace moddb {
  interface Actor {
    GetOutput(arg1: string): [number, boolean];
  }
  interface CfgItem {
    CfgCorruptedIs(corrupted: boolean): boolean;
    CfgModifierMatches(searchCond: string, slot: string): boolean;
    CfgNameIs(name: string): boolean;
    CfgRarityIs(rarity: string): boolean;
  }
  interface ListCfg {
    Actor: string;
    Flags?: mod.MFlag;
    KeywordFlags?: mod.KeywordFlag;
    Source?: mod.Source;
    SkillStats?: Record<string, number>;
    SkillCond?: Record<string, boolean>;
    SlotName: string;
    SkillName: string;
    SummonSkillName: string;
    SkillGem?: poe.SkillGem;
    SkillPart: number;
    SkillTypes?: Record<data.SkillType, boolean>;
    SkillDist: number;
    SkillDistSet: boolean;
    SocketColor: string;
    SocketNum: number;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ModDB {
    readonly ModStore?: moddb.ModStore;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ModList {
    readonly ModStore?: moddb.ModStore;
    NewMod(name: string, modType: mod.Type, value: number, source: mod.Source, tags: Array<mod.Tag> | undefined): void;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ModResult {
    Value: number;
    Mod: mod.Mod;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ModStore {
    Multipliers?: Record<string, number>;
    Conditions?: Record<string, boolean>;
    Clone(): (moddb.ModStore | undefined);
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ModListPlain {
    readonly ModStore?: moddb.ModStorePlain;
  }
  interface ModStorePlain {
    readonly Multipliers?: Record<string, number>;
    readonly Conditions?: Record<string, boolean>;
  }
}
export declare namespace msgp {
  interface Extension {
    ExtensionType(): number;
    Len(): number;
    MarshalBinaryTo(arg1: Uint8Array | undefined): Result<void>;
    UnmarshalBinary(arg1: Uint8Array | undefined): Result<void>;
  }
  interface Reader {
    R?: fwd.Reader;
    BufferSize(): number;
    Buffered(): number;
    IsNil(): boolean;
    NextType(): Result<msgp.Type>;
    Read(p: Uint8Array | undefined): Result<number>;
    ReadArrayHeader(): Result<number>;
    ReadBool(): Result<boolean>;
    ReadByte(): Result<number>;
    ReadBytes(scratch: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    ReadBytesHeader(): Result<number>;
    ReadDuration(): Result<number>;
    ReadExactBytes(into: Uint8Array | undefined): Result<void>;
    ReadExtensionRaw(): Result<[number, (Uint8Array | undefined)]>;
    ReadFloat32(): Result<number>;
    ReadFloat64(): Result<number>;
    ReadFull(p: Uint8Array | undefined): Result<number>;
    ReadInt(): Result<number>;
    ReadInt16(): Result<number>;
    ReadInt32(): Result<number>;
    ReadInt64(): Result<number>;
    ReadInt8(): Result<number>;
    ReadJSONNumber(): Result<string>;
    ReadMapHeader(): Result<number>;
    ReadMapKey(scratch: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    ReadMapKeyPtr(): Result<(Uint8Array | undefined)>;
    ReadNil(): Result<void>;
    ReadString(): Result<string>;
    ReadStringAsBytes(scratch: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    ReadStringHeader(): Result<number>;
    ReadTime(): Result<Date>;
    ReadUint(): Result<number>;
    ReadUint16(): Result<number>;
    ReadUint32(): Result<number>;
    ReadUint64(): Result<number>;
    ReadUint8(): Result<number>;
    Skip(): Result<void>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  type Type = 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12 | 13 | 14 | 15 | 16;
  const Type: {
    readonly InvalidType: 0;
    readonly StrType: 1;
    readonly BinType: 2;
    readonly MapType: 3;
    readonly ArrayType: 4;
    readonly Float64Type: 5;
    readonly Float32Type: 6;
    readonly BoolType: 7;
    readonly IntType: 8;
    readonly UintType: 9;
    readonly NilType: 10;
    readonly DurationType: 11;
    readonly ExtensionType: 12;
    readonly Complex64Type: 13;
    readonly Complex128Type: 14;
    readonly TimeType: 15;
    readonly NumberType: 16;
  };
  interface Writer {
    Append(b: Uint8Array | undefined): Result<void>;
    Buffered(): number;
    Flush(): Result<void>;
    Write(p: Uint8Array | undefined): Result<number>;
    WriteArrayHeader(sz: number): Result<void>;
    WriteBool(b: boolean): Result<void>;
    WriteByte(u: number): Result<void>;
    WriteBytes(b: Uint8Array | undefined): Result<void>;
    WriteBytesHeader(sz: number): Result<void>;
    WriteDuration(d: number): Result<void>;
    WriteExtensionRaw(extType: number, payload: Uint8Array | undefined): Result<void>;
    WriteFloat(f: number): Result<void>;
    WriteFloat32(f: number): Result<void>;
    WriteFloat64(f: number): Result<void>;
    WriteInt(i: number): Result<void>;
    WriteInt16(i: number): Result<void>;
    WriteInt32(i: number): Result<void>;
    WriteInt64(i: number): Result<void>;
    WriteInt8(i: number): Result<void>;
    WriteJSONNumber(n: string): Result<void>;
    WriteMapHeader(sz: number): Result<void>;
    WriteMapStrStr(mp: Record<string, string> | undefined): Result<void>;
    WriteNil(): Result<void>;
    WriteString(s: string): Result<void>;
    WriteStringFromBytes(str: Uint8Array | undefined): Result<void>;
    WriteStringHeader(sz: number): Result<void>;
    WriteTime(t: Date): Result<void>;
    WriteUint(u: number): Result<void>;
    WriteUint16(u: number): Result<void>;
    WriteUint32(u: number): Result<void>;
    WriteUint64(u: number): Result<void>;
    WriteUint8(u: number): Result<void>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
}
export declare namespace pob {
  type AscendancyID = 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12 | 13 | 14 | 15 | 16 | 17 | 18;
  const AscendancyID: {
    readonly Ascendant: 0;
    readonly Assassin: 1;
    readonly Berserker: 2;
    readonly Champion: 3;
    readonly Chieftain: 4;
    readonly Deadeye: 5;
    readonly Elementalist: 6;
    readonly Gladiator: 7;
    readonly Guardian: 8;
    readonly Hierophant: 9;
    readonly Inquisitor: 10;
    readonly Juggernaut: 11;
    readonly Necromancer: 12;
    readonly Occultist: 13;
    readonly Pathfinder: 14;
    readonly Raider: 15;
    readonly Saboteur: 16;
    readonly Slayer: 17;
    readonly Trickster: 18;
  };
  interface Build {
    readonly PantheonMinorGod: string;
    readonly PantheonMajorGod: string;
    readonly Bandit: string;
    readonly ViewMode: pob.BuildViewMode;
    readonly ClassName: string;
    readonly AscendClassName: string;
    readonly Level: number;
    readonly CharacterLevelAutoMode: boolean;
    readonly MainSocketGroup: number;
    readonly TargetVersion: pob.GameVersion;
    readonly PassiveNodes?: Array<number>;
    readonly PlayerStats: Array<pob.PlayerStat>;
    readonly FullDPSSkills?: Array<pob.FullDPSSkill>;
    readonly MinionStats?: Array<pob.PlayerStat>;
    readonly Spectres: Array<pob.Spectre>;
    readonly TimelessData?: pob.TimelessData;
  }
  type BuildViewMode = "TREE" | "ITEMS" | "IMPORT" | "NOTES" | "CONFIG" | "SKILLS" | "CALC";
  const BuildViewMode: {
    readonly ViewModeTree: "TREE";
    readonly ViewModeItems: "ITEMS";
    readonly ViewModeImport: "IMPORT";
    readonly ViewModeNotes: "NOTES";
    readonly ViewModeConfig: "CONFIG";
    readonly ViewModeSkills: "SKILLS";
    readonly ViewModeCalc: "CALC";
  };
  interface Calcs {
    readonly Inputs: Array<pob.Input>;
    readonly Sections: Array<pob.Section>;
  }
  type ClassID = 0 | 1 | 2 | 3 | 4 | 5 | 6;
  const ClassID: {
    readonly Scion: 0;
    readonly Marauder: 1;
    readonly Ranger: 2;
    readonly Witch: 3;
    readonly Duelist: 4;
    readonly Templar: 5;
    readonly Shadow: 6;
  };
  interface Config {
    readonly Inputs: Array<pob.Input>;
    readonly Placeholders: Array<pob.Input>;
    readonly ActiveConfigSet: number;
    readonly ConfigSets: Array<pob.ConfigSet>;
    readonly CustomModifierBlocks: Array<pob.CustomModifierBlock>;
  }
  interface ConfigSet {
    readonly ID: number;
    readonly Title?: string;
    readonly Inputs: Array<pob.Input>;
    readonly Placeholders: Array<pob.Input>;
    readonly CustomModifierBlocks: Array<pob.CustomModifierBlock>;
  }
  interface CustomModifierBlock {
    readonly Title?: string;
    readonly Enabled?: string;
    readonly Text: string;
  }
  interface FullDPSSkill {
    readonly Stat: string;
    readonly Value: string;
    readonly SkillPart: string;
    readonly Source: string;
  }
  type GameVersion = "3_0" | "3_0";
  const GameVersion: {
    readonly GameVersion3_0: "3_0";
    readonly LiveTargetVersion: "3_0";
  };
  interface Gem {
    readonly Quality: number;
    readonly SkillPart: number;
    readonly EnableGlobal2: boolean;
    readonly SkillPartCalcs: number;
    readonly QualityID: string;
    readonly GemID: string;
    readonly Enabled: boolean;
    readonly Count?: number;
    readonly EnableGlobal1: boolean;
    readonly NameSpec: string;
    readonly VariantID?: string;
    readonly Level: number;
    readonly SkillID: string;
    readonly SkillMinionItemSet: number;
    readonly SkillMinion: string;
    readonly SkillMinionSkill: number;
    readonly SkillStageCount: number;
    readonly SkillStageCountCalcs: number;
    readonly SkillMineCount?: number;
    readonly SkillMineCountCalcs?: number;
    readonly SkillMinionCalcs?: string;
    readonly SkillMinionSkillCalcs?: number;
    readonly SkillMinionItemSetCalcs?: number;
  }
  interface Import {
    ExportParty: boolean;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface Input {
    readonly Name: string;
    readonly Boolean?: boolean;
    readonly Number?: number;
    readonly String?: string;
  }
  interface Item {
    readonly ID: number;
    readonly Variant?: number;
    readonly VariantAlt?: number;
    readonly VariantAlt2?: number;
    readonly VariantAlt3?: number;
    readonly VariantAlt4?: number;
    readonly VariantAlt5?: number;
    readonly Text: string;
  }
  interface ItemSet {
    readonly ID: string;
    readonly Title?: string;
    readonly UseSecondWeaponSet?: boolean;
    readonly Slots?: Array<pob.Slot>;
  }
  interface Items {
    readonly ActiveItemSet: number;
    readonly UseSecondWeaponSet?: boolean;
    readonly ShowStatDifferences?: boolean;
    readonly ItemSets: Array<pob.ItemSet>;
    readonly Items?: Array<pob.Item>;
  }
  interface Override {
    readonly NodeID: number;
    readonly DN: string;
    readonly Icon?: string;
    readonly ActiveEffectImage?: string;
  }
  interface Party {
    Destination: string;
    Append: string;
    ShowAdvanceTools: string;
    ImportedBuffs: Array<pob.PartyBuffs>;
    ExportedBuffs: Array<pob.PartyBuffs>;
    Members?: Array<pob.PartyMember>;
    Applied(): (Array<pob.PartyBuffs> | undefined);
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface PartyBuffs {
    Name: string;
    Text: string;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface PartyMember {
    Name: string;
    ClassName: string;
    Ascendancy: string;
    Level: number;
    Destination: string;
    Active: boolean;
    Buffs?: Array<pob.PartyBuffs>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
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
    Party: pob.Party;
    Import?: pob.Import;
    RichNotes?: pob.RichNotes;
    ActiveConfigSetIndex(): number;
    ActiveSkillSetIndex(): number;
    ActiveSpecIndex(): number;
    AddItem(text: string): number;
    AddNewSocketGroup(): void;
    AddPartyMember(xmlText: string, destination: string, appendMode: boolean): Result<void>;
    AddSpec(spec: pob.Spec): void;
    AllocateNodes(nodeIds: Array<number> | undefined): void;
    ApplyNewBuildDefaults(defaultCharLevel: number, defaultGemQuality: number): void;
    CalcsInputBool(name: string): boolean;
    CalcsInputNumber(name: string, def: number): number;
    CalcsInputString(name: string, def: string): string;
    CalcsSectionCollapsed(id: string, subsection: string): [boolean, boolean];
    ClearParty(destination: string): void;
    ConvertToLiveVersion(): void;
    CopyCompareSpec(from: pob.PathOfBuilding | undefined, andUse: boolean): void;
    CopyConfigSet(configSetID: number, title: string): number;
    CopyItemSet(itemSetID: number, title: string): number;
    CopySkillSet(skillSetID: number, title: string): number;
    CopySocketGroup(socketGroup: number): string;
    CopySpec(specID: number, title: string): void;
    DeallocateNodes(nodeIds: Array<number> | undefined): void;
    DeallocateSpecNodes(specIndex: number, nodeIDs: Array<number> | undefined): void;
    DeleteAllItems(): void;
    DeleteAllSocketGroups(): void;
    DeleteConfigSet(configSetID: number): void;
    DeleteItem(itemID: number): boolean;
    DeleteItemSet(itemSetID: number): boolean;
    DeleteSkillSet(skillSetID: number): void;
    DeleteSocketGroup(index: number): void;
    DeleteSpec(specID: number): void;
    DisablePartyEffects(): void;
    ExportParty(): boolean;
    GetBooleanOption(name: string): boolean;
    GetCustomModifierBlocks(): (Array<pob.CustomModifierBlock> | undefined);
    GetNumberOption(name: string): number;
    GetNumberOptionOK(name: string): [number, boolean];
    GetNumberPlaceholder(name: string): number;
    GetStringOption(name: string): string;
    GetTimelessData(): pob.TimelessData;
    GetTreeView(): pob.TreeView;
    GuessMainSocketGroup(): number;
    ImportParty(xmlText: string, destination: string, appendMode: boolean): Result<void>;
    MoveConfigSet(from: number, to: number): void;
    MoveItem(id: number, beforeID: number): void;
    MoveItemSet(from: number, to: number): void;
    MoveSocketGroup(from: number, to: number): void;
    MoveSpec(from: number, to: number): void;
    NeedsVersionConversion(): boolean;
    NewConfigSet(title: string): number;
    NewItemSet(title: string): number;
    NewSkillSet(title: string): number;
    PartyBuffers(): (Record<string, string> | undefined);
    PartyEffectsApplied(): boolean;
    PartyEffectsDisabled(): boolean;
    PartyMembers(): (Array<pob.PartyMember> | undefined);
    PartyOptions(): [string, boolean, boolean];
    PasteSocketGroup(text: string): number;
    RebuildParty(): void;
    RemoveConfigOption(name: string): void;
    RemovePartyMember(i: number): void;
    RemoveTattoo(nodeID: number): boolean;
    RenameConfigSet(configSetID: number, title: string): void;
    RenameItemSet(itemSetID: number, title: string): void;
    RenameSkillSet(skillSetID: number, title: string): void;
    RenameSpec(specID: number, title: string): void;
    ResetConfigOptions(): void;
    RichNotesDoc(): string;
    RichNotesStatus(): string;
    SetActiveConfigSet(configSetID: number): void;
    SetActiveItemSet(itemSetID: number): void;
    SetActiveSkillSet(skillSetID: number): void;
    SetActiveSpec(specID: number): void;
    SetAscendancy(ascendancyName: string): void;
    SetCalcsInputBool(name: string, value: boolean): void;
    SetCalcsInputNumber(name: string, value: number): void;
    SetCalcsInputString(name: string, value: string): void;
    SetCalcsSectionCollapsed(id: string, subsection: string, collapsed: boolean): void;
    SetCharacterLevelAutoMode(auto: boolean): void;
    SetClass(className: string): void;
    SetConfigOption(value: pob.Input): void;
    SetCustomModifierBlocks(blocks: Array<pob.CustomModifierBlock> | undefined): void;
    SetDefaultGemLevel(gemLevel: number): void;
    SetDefaultGemLevelMode(mode: string): void;
    SetDefaultGemQuality(gemQuality: number): void;
    SetExportParty(on: boolean): void;
    SetExportedBuffs(texts: Record<string, string> | undefined): void;
    SetFlaskActive(slotName: string, active: boolean): void;
    SetGemSkillMineCount(skillSet: number, socketGroup: number, gem: number, mineCount: number): void;
    SetGemSkillMineCountCalcs(skillSet: number, socketGroup: number, gem: number, mineCount: number): void;
    SetGemSkillMinion(skillSet: number, socketGroup: number, gem: number, minionID: string): void;
    SetGemSkillMinionCalcs(skillSet: number, socketGroup: number, gem: number, minionID: string): void;
    SetGemSkillMinionItemSet(skillSet: number, socketGroup: number, gem: number, itemSetID: number): void;
    SetGemSkillMinionItemSetCalcs(skillSet: number, socketGroup: number, gem: number, itemSetID: number): void;
    SetGemSkillMinionSkill(skillSet: number, socketGroup: number, gem: number, minionSkill: number): void;
    SetGemSkillMinionSkillCalcs(skillSet: number, socketGroup: number, gem: number, minionSkill: number): void;
    SetGemSkillPart(skillSet: number, socketGroup: number, gem: number, skillPart: number): void;
    SetGemSkillPartCalcs(skillSet: number, socketGroup: number, gem: number, skillPart: number): void;
    SetGemSkillStageCount(skillSet: number, socketGroup: number, gem: number, stageCount: number): void;
    SetGemSkillStageCountCalcs(skillSet: number, socketGroup: number, gem: number, stageCount: number): void;
    SetItemText(id: number, text: string): boolean;
    SetLevel(level: number): void;
    SetMainActiveSkill(skillSet: number, socketGroup: number, mainActiveSkill: number): void;
    SetMainActiveSkillCalcs(skillSet: number, socketGroup: number, mainActiveSkill: number): void;
    SetMainSocketGroup(mainSocketGroup: number): void;
    SetMasteryEffect(nodeID: number, effectID: number): void;
    SetMatchGemLevelToCharacterLevel(enabled: boolean): void;
    SetNotes(notes: string): void;
    SetPartyBuffer(name: string, text: string): void;
    SetPartyMemberActive(i: number, active: boolean): void;
    SetPartyOptions(destination: string, appendMode: boolean, showAdvanceTools: boolean): void;
    SetRichNotes(plain: string, doc: string): void;
    SetSetSlotItem(setID: number, slotName: string, itemID: number): boolean;
    SetShowAltQualityGems(enabled: boolean): void;
    SetShowLegacyGems(enabled: boolean): void;
    SetShowStatDifferences(on: boolean): void;
    SetShowSupportGemTypes(gemTypes: string): void;
    SetSkillGroupName(skillSet: number, socketGroup: number, label: string): void;
    SetSlotItem(slotName: string, itemID: number): void;
    SetSlotItemID(setID: number, slotName: string): number;
    SetSocketGroupGems(skillSet: number, socketGroup: number, gems: Array<pob.Gem> | undefined): void;
    SetSocketGroupImbuedSupport(skillSet: number, socketGroup: number, name: string): void;
    SetSocketGroupSlot(skillSet: number, socketGroup: number, slot: string): void;
    SetSortGemsByDPS(enabled: boolean): void;
    SetSortGemsByDPSField(field: string): void;
    SetSpectres(ids: Array<string> | undefined): void;
    SetTattoo(o: pob.Override): void;
    SetTimelessData(d: pob.TimelessData): void;
    SetTreeView(v: pob.TreeView): void;
    SetUseSecondWeaponSet(second: boolean): boolean;
    SocketJewel(nodeID: number, itemID: number): void;
    SocketedJewel(nodeID: number): number;
    SpectreIDs(): (Array<string> | undefined);
    StatDifferencesShown(): boolean;
    TattooCount(): number;
    TattooOn(nodeID: number): string;
    WithMainSkill(socketGroup: number, mainActiveSkill: number): (pob.PathOfBuilding | undefined);
    WithMainSocketGroup(mainSocketGroup: number): (pob.PathOfBuilding | undefined);
    WithoutPartyEffects(): (pob.PathOfBuilding | undefined);
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface PlayerStat {
    readonly Value: number;
    readonly Stat: string;
  }
  interface RichNotes {
    Format: string;
    V: number;
    PlainHash: string;
    Doc: string;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface Section {
    readonly Collapsed: boolean;
    readonly ID: string;
    readonly Subsection: string;
  }
  interface Skill {
    MainActiveSkillCalcs: number;
    MainActiveSkill: number;
    Label: string;
    Enabled: boolean;
    IncludeInFullDPS?: boolean;
    GroupCount?: number;
    Gems: Array<pob.Gem>;
    Slot: string;
    ImbuedSupport?: string;
    SourceAttr?: string;
    Source: string;
    SlotEnabled: boolean;
    DisplayLabel: string;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface SkillSet {
    ID: number;
    Title?: string;
    Skills: Array<pob.Skill>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface Skills {
    SortGemsByDPSField: string;
    ShowSupportGemTypes: string;
    DefaultGemLevel?: string;
    MatchGemLevelToCharacterLevel: boolean;
    ShowAltQualityGems: boolean;
    DefaultGemQuality?: number;
    ActiveSkillSet: number;
    SortGemsByDPS: boolean;
    ShowLegacyGems: boolean;
    SkillSets: Array<pob.SkillSet>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface Slot {
    readonly ItemID: number;
    readonly Name: string;
    readonly Active: boolean;
  }
  interface Socket {
    readonly ItemID: number;
    readonly NodeID: number;
  }
  interface Spec {
    readonly ClassID: pob.ClassID;
    readonly AscendClassID: pob.AscendancyID;
    readonly SecondaryAscendClassID?: number;
    readonly ClusterHashFormatVersion?: number;
    readonly Title?: string;
    readonly TreeVersion: string;
    readonly NodesAttr: string;
    readonly MasteryEffects: string;
    readonly URL: string;
    readonly Sockets: Array<pob.Socket>;
    readonly Overrides: Array<pob.Override>;
  }
  interface Spectre {
    readonly ID: string;
  }
  interface TimelessData {
    readonly JewelTypeID: string;
    readonly ConquerorTypeID: string;
    readonly DevotionVariant1: string;
    readonly DevotionVariant2: string;
    readonly JewelSocketID: string;
    readonly FallbackWeightModeIdx: string;
    readonly SocketFilter: string;
    readonly SocketFilterDistance: string;
    readonly SearchList: string;
    readonly SearchListFallback: string;
  }
  interface Tree {
    readonly ActiveSpec: number;
    readonly Specs: Array<pob.Spec>;
  }
  interface TreeView {
    readonly ZoomLevel: number;
    readonly ZoomX: number;
    readonly ZoomY: number;
    readonly SearchStr: string;
    readonly ShowHeatMap?: boolean;
    readonly ShowStatDifferences: boolean;
  }
  const BuildInfo: debug.BuildInfo | undefined;
  function CompressEncode(xml: string): Result<string>;
  function DecodeDecompress(code: string): Result<string>;
}
export declare namespace poe {
  interface ActiveSkill {
    ActiveSkill: raw.ActiveSkill;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    GetActiveSkillBaseFlagsAndTypes(): [(Record<poe.SkillFlag, boolean> | undefined), (Record<poe.SkillType, boolean> | undefined)];
    GetActiveSkillTypes(): (Array<poe.ActiveSkillType | undefined> | undefined);
    GetWeaponRestrictions(): (Array<poe.ItemClass | undefined> | undefined);
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ActiveSkillType {
    ActiveSkillType: raw.ActiveSkillType;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface BaseItemType {
    readonly BaseItemType: raw.BaseItemType;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    SkillGem(): (poe.SkillGem | undefined);
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface CostType {
    CostType: raw.CostType;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalJSON(b: Uint8Array | undefined): Result<void>;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface EffectivenessCostConstant {
    EffectivenessCostConstant: raw.EffectivenessCostConstant;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface GrantedEffect {
    GrantedEffect: raw.GrantedEffect;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    GetActiveSkill(): (poe.ActiveSkill | undefined);
    GetAdditionalStatSets(): (Array<poe.GrantedEffectStatSet | undefined> | undefined);
    GetEffectQualityStats(): (Record<number, poe.GrantedEffectQualityStat | undefined> | undefined);
    GetEffectStatSetsPerLevel(): (Record<number, poe.GrantedEffectStatSetsPerLevel | undefined> | undefined);
    GetEffectsPerLevel(): (Record<number, poe.GrantedEffectsPerLevel | undefined> | undefined);
    GetExcludeTypes(): (Array<poe.ActiveSkillType | undefined> | undefined);
    GetGrantedEffectStatSet(): (poe.GrantedEffectStatSet | undefined);
    GetSkillGem(): (poe.SkillGem | undefined);
    GetSupportTypes(): (Array<poe.ActiveSkillType | undefined> | undefined);
    HasGlobalEffect(): boolean;
    Levels(): (Record<number, poe.GrantedEffectsPerLevel | undefined> | undefined);
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalJSON(b: Uint8Array | undefined): Result<void>;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface GrantedEffectQualityStat {
    GrantedEffectQualityStat: raw.GrantedEffectQualityStat;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    GetStats(): (Array<poe.Stat | undefined> | undefined);
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface GrantedEffectStatSet {
    GrantedEffectStatSet: raw.GrantedEffectStatSet;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    GetConstantStats(): (Array<poe.Stat | undefined> | undefined);
    GetImplicitStats(): (Array<poe.Stat | undefined> | undefined);
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface GrantedEffectStatSetsPerLevel {
    GrantedEffectStatSetsPerLevel: raw.GrantedEffectStatSetsPerLevel;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    GetAdditionalBooleanStats(): (Array<poe.Stat | undefined> | undefined);
    GetAdditionalStats(): (Array<poe.Stat | undefined> | undefined);
    GetFloatStats(): (Array<poe.Stat | undefined> | undefined);
    GetInterpolationBases(): (Array<poe.EffectivenessCostConstant | undefined> | undefined);
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface GrantedEffectsPerLevel {
    GrantedEffectsPerLevel: raw.GrantedEffectsPerLevel;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    GetCostTypes(): (Array<poe.CostType | undefined> | undefined);
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalJSON(b: Uint8Array | undefined): Result<void>;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ItemClass {
    ItemClass: raw.ItemClass;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  type SkillFlag = "brand" | "hex" | "curse" | "attack" | "weapon1Attack" | "weapon2Attack" | "selfCast" | "notAverage" | "showAverage" | "hit" | "projectile" | "trap" | "mine" | "totem" | "bothWeaponAttack" | "buffs" | "combat" | "effective" | "spell" | "melee" | "chaining" | "area" | "cast" | "shieldAttack" | "forceMainHand" | "disable";
  const SkillFlag: {
    readonly SkillFlagBrand: "brand";
    readonly SkillFlagHex: "hex";
    readonly SkillFlagCurse: "curse";
    readonly SkillFlagAttack: "attack";
    readonly SkillFlagWeapon1Attack: "weapon1Attack";
    readonly SkillFlagWeapon2Attack: "weapon2Attack";
    readonly SkillFlagSelfCast: "selfCast";
    readonly SkillFlagNotAverage: "notAverage";
    readonly SkillFlagShowAverage: "showAverage";
    readonly SkillFlagHit: "hit";
    readonly SkillFlagProjectile: "projectile";
    readonly SkillFlagTrap: "trap";
    readonly SkillFlagMine: "mine";
    readonly SkillFlagTotem: "totem";
    readonly SkillFlagBothWeaponAttack: "bothWeaponAttack";
    readonly SkillFlagBuffs: "buffs";
    readonly SkillFlagCombat: "combat";
    readonly SkillFlagEffective: "effective";
    readonly SkillFlagSpell: "spell";
    readonly SkillFlagMelee: "melee";
    readonly SkillFlagChaining: "chaining";
    readonly SkillFlagArea: "area";
    readonly SkillFlagCast: "cast";
    readonly SkillFlagShieldAttack: "shieldAttack";
    readonly SkillFlagForceMainHand: "forceMainHand";
    readonly SkillFlagDisable: "disable";
  };
  interface SkillGem {
    SkillGem: raw.SkillGem;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    DefaultLevel(): number;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    GetBaseItemType(): (poe.BaseItemType | undefined);
    GetGrantedEffect(): (poe.GrantedEffect | undefined);
    GetGrantedEffects(): (Array<poe.GrantedEffect | undefined> | undefined);
    GetNonVaal(): (poe.SkillGem | undefined);
    GetSecondaryGrantedEffect(): (poe.GrantedEffect | undefined);
    GetTags(): (Record<raw.TagName, poe.Tag | undefined> | undefined);
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalJSON(b: Uint8Array | undefined): Result<void>;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  type SkillType = "Attack" | "Spell" | "Projectile" | "DualWieldOnly" | "Buff" | "Removed6" | "MainHandOnly" | "Removed8" | "Minion" | "Damage" | "Area" | "Duration" | "RequiresShield" | "ProjectileSpeed" | "HasReservation" | "ReservationBecomesCost" | "Trappable" | "Totemable" | "Mineable" | "ElementalStatus" | "MinionsCanExplode" | "Removed22" | "Chains" | "Melee" | "MeleeSingleTarget" | "Multicastable" | "TotemCastsAlone" | "Multistrikeable" | "CausesBurning" | "SummonsTotem" | "TotemCastsWhenNotDetached" | "Fire" | "Cold" | "Lightning" | "Triggerable" | "Trapped" | "Movement" | "Removed39" | "DamageOverTime" | "RemoteMined" | "Triggered" | "Vaal" | "Aura" | "Removed45" | "CanTargetUnusableCorpse" | "Removed47" | "RangedAttack" | "Removed49" | "Chaos" | "FixedSpeedProjectile" | "Removed52" | "ThresholdJewelArea" | "ThresholdJewelProjectile" | "ThresholdJewelDuration" | "ThresholdJewelRangedAttack" | "Removed57" | "Channel" | "DegenOnlySpellDamage" | "Removed60" | "InbuiltTrigger" | "Golem" | "Herald" | "AuraAffectsEnemies" | "NoRuthless" | "ThresholdJewelSpellDamage" | "Cascadable" | "ProjectilesFromUser" | "MirageArcherCanUse" | "ProjectileSpiral" | "SingleMainProjectile" | "MinionsPersistWhenSkillRemoved" | "ProjectileNumber" | "Warcry" | "Instant" | "Brand" | "DestroysCorpse" | "NonHitChill" | "ChillingArea" | "AppliesCurse" | "CanRapidFire" | "AuraDuration" | "AreaSpell" | "OR" | "AND" | "NOT" | "Physical" | "AppliesMaim" | "CreatesMinion" | "Guard" | "Travel" | "Blink" | "CanHaveBlessing" | "ProjectilesNotFromUser" | "AttackInPlaceIsDefault" | "Nova" | "InstantNoRepeatWhenHeld" | "InstantShiftAttackForLeftMouse" | "AuraNotOnCaster" | "Banner" | "Rain" | "Cooldown" | "ThresholdJewelChaining" | "Slam" | "Stance" | "NonRepeatable" | "OtherThingUsesSkill" | "Steel" | "Hex" | "Mark" | "Aegis" | "Orb" | "KillNoDamageModifiers" | "RandomElement" | "LateConsumeCooldown" | "Arcane" | "FixedCastTime" | "RequiresOffHandNotWeapon" | "Link" | "Blessing" | "ZeroReservation";
  const SkillType: {
    readonly SkillTypeAttack: "Attack";
    readonly SkillTypeSpell: "Spell";
    readonly SkillTypeProjectile: "Projectile";
    readonly SkillTypeDualWieldOnly: "DualWieldOnly";
    readonly SkillTypeBuff: "Buff";
    readonly SkillTypeRemoved6: "Removed6";
    readonly SkillTypeMainHandOnly: "MainHandOnly";
    readonly SkillTypeRemoved8: "Removed8";
    readonly SkillTypeMinion: "Minion";
    readonly SkillTypeDamage: "Damage";
    readonly SkillTypeArea: "Area";
    readonly SkillTypeDuration: "Duration";
    readonly SkillTypeRequiresShield: "RequiresShield";
    readonly SkillTypeProjectileSpeed: "ProjectileSpeed";
    readonly SkillTypeHasReservation: "HasReservation";
    readonly SkillTypeReservationBecomesCost: "ReservationBecomesCost";
    readonly SkillTypeTrappable: "Trappable";
    readonly SkillTypeTotemable: "Totemable";
    readonly SkillTypeMineable: "Mineable";
    readonly SkillTypeElementalStatus: "ElementalStatus";
    readonly SkillTypeMinionsCanExplode: "MinionsCanExplode";
    readonly SkillTypeRemoved22: "Removed22";
    readonly SkillTypeChains: "Chains";
    readonly SkillTypeMelee: "Melee";
    readonly SkillTypeMeleeSingleTarget: "MeleeSingleTarget";
    readonly SkillTypeMulticastable: "Multicastable";
    readonly SkillTypeTotemCastsAlone: "TotemCastsAlone";
    readonly SkillTypeMultistrikeable: "Multistrikeable";
    readonly SkillTypeCausesBurning: "CausesBurning";
    readonly SkillTypeSummonsTotem: "SummonsTotem";
    readonly SkillTypeTotemCastsWhenNotDetached: "TotemCastsWhenNotDetached";
    readonly SkillTypeFire: "Fire";
    readonly SkillTypeCold: "Cold";
    readonly SkillTypeLightning: "Lightning";
    readonly SkillTypeTriggerable: "Triggerable";
    readonly SkillTypeTrapped: "Trapped";
    readonly SkillTypeMovement: "Movement";
    readonly SkillTypeRemoved39: "Removed39";
    readonly SkillTypeDamageOverTime: "DamageOverTime";
    readonly SkillTypeRemoteMined: "RemoteMined";
    readonly SkillTypeTriggered: "Triggered";
    readonly SkillTypeVaal: "Vaal";
    readonly SkillTypeAura: "Aura";
    readonly SkillTypeRemoved45: "Removed45";
    readonly SkillTypeCanTargetUnusableCorpse: "CanTargetUnusableCorpse";
    readonly SkillTypeRemoved47: "Removed47";
    readonly SkillTypeRangedAttack: "RangedAttack";
    readonly SkillTypeRemoved49: "Removed49";
    readonly SkillTypeChaos: "Chaos";
    readonly SkillTypeFixedSpeedProjectile: "FixedSpeedProjectile";
    readonly SkillTypeRemoved52: "Removed52";
    readonly SkillTypeThresholdJewelArea: "ThresholdJewelArea";
    readonly SkillTypeThresholdJewelProjectile: "ThresholdJewelProjectile";
    readonly SkillTypeThresholdJewelDuration: "ThresholdJewelDuration";
    readonly SkillTypeThresholdJewelRangedAttack: "ThresholdJewelRangedAttack";
    readonly SkillTypeRemoved57: "Removed57";
    readonly SkillTypeChannel: "Channel";
    readonly SkillTypeDegenOnlySpellDamage: "DegenOnlySpellDamage";
    readonly SkillTypeRemoved60: "Removed60";
    readonly SkillTypeInbuiltTrigger: "InbuiltTrigger";
    readonly SkillTypeGolem: "Golem";
    readonly SkillTypeHerald: "Herald";
    readonly SkillTypeAuraAffectsEnemies: "AuraAffectsEnemies";
    readonly SkillTypeNoRuthless: "NoRuthless";
    readonly SkillTypeThresholdJewelSpellDamage: "ThresholdJewelSpellDamage";
    readonly SkillTypeCascadable: "Cascadable";
    readonly SkillTypeProjectilesFromUser: "ProjectilesFromUser";
    readonly SkillTypeMirageArcherCanUse: "MirageArcherCanUse";
    readonly SkillTypeProjectileSpiral: "ProjectileSpiral";
    readonly SkillTypeSingleMainProjectile: "SingleMainProjectile";
    readonly SkillTypeMinionsPersistWhenSkillRemoved: "MinionsPersistWhenSkillRemoved";
    readonly SkillTypeProjectileNumber: "ProjectileNumber";
    readonly SkillTypeWarcry: "Warcry";
    readonly SkillTypeInstant: "Instant";
    readonly SkillTypeBrand: "Brand";
    readonly SkillTypeDestroysCorpse: "DestroysCorpse";
    readonly SkillTypeNonHitChill: "NonHitChill";
    readonly SkillTypeChillingArea: "ChillingArea";
    readonly SkillTypeAppliesCurse: "AppliesCurse";
    readonly SkillTypeCanRapidFire: "CanRapidFire";
    readonly SkillTypeAuraDuration: "AuraDuration";
    readonly SkillTypeAreaSpell: "AreaSpell";
    readonly SkillTypeOR: "OR";
    readonly SkillTypeAND: "AND";
    readonly SkillTypeNOT: "NOT";
    readonly SkillTypePhysical: "Physical";
    readonly SkillTypeAppliesMaim: "AppliesMaim";
    readonly SkillTypeCreatesMinion: "CreatesMinion";
    readonly SkillTypeGuard: "Guard";
    readonly SkillTypeTravel: "Travel";
    readonly SkillTypeBlink: "Blink";
    readonly SkillTypeCanHaveBlessing: "CanHaveBlessing";
    readonly SkillTypeProjectilesNotFromUser: "ProjectilesNotFromUser";
    readonly SkillTypeAttackInPlaceIsDefault: "AttackInPlaceIsDefault";
    readonly SkillTypeNova: "Nova";
    readonly SkillTypeInstantNoRepeatWhenHeld: "InstantNoRepeatWhenHeld";
    readonly SkillTypeInstantShiftAttackForLeftMouse: "InstantShiftAttackForLeftMouse";
    readonly SkillTypeAuraNotOnCaster: "AuraNotOnCaster";
    readonly SkillTypeBanner: "Banner";
    readonly SkillTypeRain: "Rain";
    readonly SkillTypeCooldown: "Cooldown";
    readonly SkillTypeThresholdJewelChaining: "ThresholdJewelChaining";
    readonly SkillTypeSlam: "Slam";
    readonly SkillTypeStance: "Stance";
    readonly SkillTypeNonRepeatable: "NonRepeatable";
    readonly SkillTypeOtherThingUsesSkill: "OtherThingUsesSkill";
    readonly SkillTypeSteel: "Steel";
    readonly SkillTypeHex: "Hex";
    readonly SkillTypeMark: "Mark";
    readonly SkillTypeAegis: "Aegis";
    readonly SkillTypeOrb: "Orb";
    readonly SkillTypeKillNoDamageModifiers: "KillNoDamageModifiers";
    readonly SkillTypeRandomElement: "RandomElement";
    readonly SkillTypeLateConsumeCooldown: "LateConsumeCooldown";
    readonly SkillTypeArcane: "Arcane";
    readonly SkillTypeFixedCastTime: "FixedCastTime";
    readonly SkillTypeRequiresOffHandNotWeapon: "RequiresOffHandNotWeapon";
    readonly SkillTypeLink: "Link";
    readonly SkillTypeBlessing: "Blessing";
    readonly SkillTypeZeroReservation: "ZeroReservation";
  };
  interface Stat {
    Stat: raw.Stat;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalJSON(b: Uint8Array | undefined): Result<void>;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface Tag {
    Tag: raw.Tag;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
}
export declare namespace raw {
  interface ActiveSkill {
    AlternateSkillTargetingBehavioursKey?: number;
    AIFile: string;
    WebsiteImage: string;
    Description: string;
    DisplayedName: string;
    IconDDSFile: string;
    ID: string;
    WebsiteDescription: string;
    SkillID: string;
    WeaponRestrictionItemClassesKeys?: Array<number>;
    MinionActiveSkillTypes?: Array<number>;
    OutputStatKeys?: Array<number>;
    InputStatKeys?: Array<number>;
    SecondarySkillSpecificStats?: Array<number>;
    ActiveSkillTypes?: Array<number>;
    ActiveSkillTargetTypes?: Array<number>;
    SkillTotemID: number;
    Key: number;
    IsManuallyCasted: boolean;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ActiveSkillType {
    FlagStat?: number;
    ID: string;
    Key: number;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface BaseItemType {
    SoundEffect?: number;
    EquipAchievementItemsKey?: number;
    FlavourTextKey?: number;
    FragmentBaseItemTypesKey?: number;
    ID: string;
    Name: string;
    Inflection: string;
    InheritsFrom: string;
    TagsKeys?: Array<number>;
    EnchantMods?: Array<number>;
    ImplicitModsKeys?: Array<number>;
    VendorRecipeAchievementItems?: Array<number>;
    SizeOnGround: number;
    ItemVisualIdentity: number;
    ModDomain: number;
    ItemClassesKey: number;
    SiteVisibility: number;
    DropLevel: number;
    Height: number;
    Hash: number;
    Width: number;
    Key: number;
    IsCorrupted: boolean;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
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
    PvPDamageMultiplier?: number;
    SoulCost?: number;
    SkillUseStorage?: number;
    StoredUses?: number;
    SoulPreventionDuration?: number;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface CostType {
    FormatText: string;
    ID: string;
    StatsKey: number;
    Divisor: number;
    Key: number;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalJSON(b: Uint8Array | undefined): Result<void>;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface EffectivenessCostConstant {
    ID: string;
    Multiplier: number;
    Key: number;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface GrantedEffect {
    ActiveSkill?: number;
    PlusVersionOf?: number;
    Animation?: number;
    SupportGemLetter: string;
    ID: string;
    SupportTypes?: Array<number>;
    AddTypes?: Array<number>;
    ExcludeTypes?: Array<number>;
    WeaponRestrictions?: Array<number>;
    AddMinionTypes?: Array<number>;
    StatSet2?: Array<number>;
    Attribute: number;
    CastTime: number;
    GrantedEffectStatSets: number;
    Key: number;
    IgnoreMinionTypes: boolean;
    CannotBeSupported: boolean;
    SupportsGemsOnly: boolean;
    IsSupport: boolean;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalJSON(b: Uint8Array | undefined): Result<void>;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface GrantedEffectQualityStat {
    StatsKeys?: Array<number>;
    StatsValuesPermille?: Array<number>;
    GrantedEffectsKey: number;
    SetID: number;
    Weight: number;
    Key: number;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface GrantedEffectStatSet {
    ID: string;
    ImplicitStats?: Array<number>;
    ConstantStats?: Array<number>;
    ConstantStatsValues?: Array<number>;
    Key: number;
    BaseEffectiveness: number;
    IncrementalEffectiveness: number;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface GrantedEffectStatSetsPerLevel {
    GrantedEffects?: Array<number>;
    AdditionalStats?: Array<number>;
    AdditionalStatsValues?: Array<number>;
    StatInterpolations?: Array<number>;
    AdditionalBooleanStats?: Array<number>;
    BaseResolvedValues?: Array<number>;
    InterpolationBases?: Array<number>;
    FloatStats?: Array<number>;
    FloatStatsValues?: Array<number>;
    BaseMultiplier: number;
    GemLevel: number;
    DamageEffectiveness: number;
    PlayerLevelReq: number;
    OffhandCritChance: number;
    AttackCritChance: number;
    StatSet: number;
    Key: number;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface GrantedEffectsPerLevel {
    CostAmounts?: Array<number>;
    CostTypes?: Array<number>;
    LifeReservationFlat: number;
    LifeReservationPercent: number;
    CooldownGroup: number;
    Cooldown: number;
    CostMultiplier: number;
    AttackTime: number;
    GrantedEffect: number;
    Level: number;
    AttackSpeedMultiplier: number;
    CooldownBypassType: number;
    ManaReservationFlat: number;
    ManaReservationPercent: number;
    PlayerLevelReq: number;
    SoulGainPreventionDuration: number;
    StoredUses: number;
    VaalSouls: number;
    VaalStoredUses: number;
    PvPDamageMultiplier: number;
    Key: number;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalJSON(b: Uint8Array | undefined): Result<void>;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ItemClass {
    ItemStance?: number;
    ItemClassCategory?: number;
    Name: string;
    ID: string;
    Flags?: Array<number>;
    Key: number;
    CanBeDoubleCorrupted: boolean;
    CanHaveInfluence: boolean;
    CanHaveVeiledMods: boolean;
    CanScourge: boolean;
    CanTransferSkin: boolean;
    CanHaveIncubators: boolean;
    CanHaveAspects: boolean;
    AllocateToMapOwner: boolean;
    CanBeCorrupted: boolean;
    AlwaysShow: boolean;
    RemovedIfLeavesArea: boolean;
    AlwaysAllocate: boolean;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface MinionInfo {
    Name: string;
    Life: number;
    EnergyShield: number;
    Armour: number;
    Evasion: number;
    FireResist: number;
    ColdResist: number;
    LightningResist: number;
    ChaosResist: number;
    Damage: number;
    DamageSpread: number;
    DamageFixup: number;
    AttackTime: number;
    AttackRange: number;
    Accuracy: number;
    WeaponType1: string;
    WeaponType2: string;
    Limit: string;
    LifeScaling: string;
    MonsterTags?: Array<string>;
    BaseDamageIgnoresAttackSpeed: boolean;
    Hostile: boolean;
    SkillList?: Array<string>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface SkillGem {
    VaalGem?: number;
    RegularVariant?: number;
    AwakenedVariant?: number;
    GlobalGemLevelStat?: number;
    SecondaryGrantedEffect?: number;
    HungryLoopMod?: number;
    GemLevelProgression?: number;
    SecondarySupportName: string;
    Description: string;
    Tags?: Array<number>;
    GemVariants?: Array<number>;
    Int: number;
    Dex: number;
    BaseItemType: number;
    Str: number;
    GrantedEffect: number;
    GemColour: number;
    Key: number;
    IsVaalGem: boolean;
    IsSupport: boolean;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalJSON(b: Uint8Array | undefined): Result<void>;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface SkillPart {
    Name: string;
    Flags?: Record<string, boolean>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface Stat {
    MainHandAliasStatsKey?: number;
    Category?: number;
    OffHandAliasStatsKey?: number;
    ID: string;
    Text: string;
    ContextFlags?: Array<number>;
    BelongsStatsKey?: Array<string>;
    Hash32: number;
    Semantics: number;
    Key: number;
    IsWeaponLocal: boolean;
    IsVirtual: boolean;
    IsScalable: boolean;
    IsLocal: boolean;
    CannotGrantToMinion: boolean;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalJSON(b: Uint8Array | undefined): Result<void>;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface Tag {
    DisplayString: string;
    ID: string;
    Name: raw.TagName;
    Key: number;
    DecodeMsg(dc: msgp.Reader | undefined): Result<void>;
    EncodeMsg(en: msgp.Writer | undefined): Result<void>;
    MarshalMsg(b: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    Msgsize(): number;
    UnmarshalMsg(bts: Uint8Array | undefined): Result<(Uint8Array | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  type TagName = "Ailment" | "Attack" | "Attribute" | "Aura" | "Caster" | "Chaos" | "Cold" | "Critical" | "Curse" | "Damage" | "Defences" | "Elemental" | "Fire" | "Gem" | "Life" | "Lightning" | "Mana" | "Minion" | "Physical" | "Resistance" | "Speed";
  const TagName: {
    readonly TagAilment: "Ailment";
    readonly TagAttack: "Attack";
    readonly TagAttribute: "Attribute";
    readonly TagAura: "Aura";
    readonly TagCaster: "Caster";
    readonly TagChaos: "Chaos";
    readonly TagCold: "Cold";
    readonly TagCritical: "Critical";
    readonly TagCurse: "Curse";
    readonly TagDamage: "Damage";
    readonly TagDefences: "Defences";
    readonly TagElemental: "Elemental";
    readonly TagFire: "Fire";
    readonly TagGem: "Gem";
    readonly TagLife: "Life";
    readonly TagLightning: "Lightning";
    readonly TagMana: "Mana";
    readonly TagMinion: "Minion";
    readonly TagPhysical: "Physical";
    readonly TagResistance: "Resistance";
    readonly TagSpeed: "Speed";
  };
  function GameDataVersion(): string;
  function InitializeAll(version: string, updateFunc: (data: string) => void): Promise<Result<void>>;
  function LoadDeferred(): Promise<Result<void>>;
}
export declare namespace storage {
  interface DirEntry {
    readonly Name: string;
    readonly Type: storage.DirEntryType;
    readonly Class: string;
    readonly LastEdit: string;
    readonly Level: number;
    readonly Ascendancy: string;
    readonly MainSkill: string;
  }
  type DirEntryType = "file" | "dir";
  const DirEntryType: {
    readonly DirEntryTypeFile: "file";
    readonly DirEntryTypeDir: "dir";
  };
  function DeleteBuild(path: string): Promise<Result<void>>;
  function GetBuild(path: string): Promise<Result<string>>;
  function InitializeStorage(list: (bucket: string, dir: string) => (Array<storage.DirEntry> | undefined) | PromiseLike<(Array<storage.DirEntry> | undefined)>, get: (bucket: string, key: string) => (Uint8Array | undefined) | PromiseLike<(Uint8Array | undefined)>, set: (bucket: string, key: string, value: Uint8Array | undefined) => void, exists: (bucket: string, key: string) => boolean | PromiseLike<boolean>, createFolder: (bucket: string, dir: string) => void, del: (bucket: string, path: string) => void): Promise<void>;
  function ListBuilds(dir: string): Promise<Result<(Array<storage.DirEntry> | undefined)>>;
  function NewFolder(path: string): Promise<Result<void>>;
  function SetBuild(path: string, value: string): Promise<Result<void>>;
}
export function boot(wasm: string | URL | BufferSource): Promise<{ builds: typeof builds; calculator: typeof calculator; config: typeof config; data: typeof data; exposition: typeof exposition; mod: typeof mod; msgp: typeof msgp; pob: typeof pob; poe: typeof poe; raw: typeof raw; storage: typeof storage }>;
export const initializeCrystalline: () => void;