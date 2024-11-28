/* eslint-disable */
export declare namespace builds {
  function ParseBuild(rawXML?: Uint8Array): [(pob.PathOfBuilding | undefined), Error];
  function ParseBuildStr(rawXML: string): [(pob.PathOfBuilding | undefined), Error];
}
export declare namespace cache {
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
    BleedCfg?: calculator.ListCfg;
    OHBleedCfg?: calculator.ListCfg;
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
    BuildOutput(mode: string): Promise<(calculator.Environment | undefined)>;
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
    AllocatedNodes?: Record<string, data.Node>;
    AuxSkillList?: Record<string, unknown | undefined>;
    ModeBuffs: boolean;
    ModeCombat: boolean;
    ModeEffective: boolean;
    KeystonesAdded?: Record<string, unknown | undefined>;
    MainSocketGroup: number;
    DebugErrors?: Array<string>;
  }
  interface GemEffect {
    GrantedEffect?: calculator.GrantedEffect;
    Level: number;
    Quality: number;
    QualityID: string;
    SrcInstance?: pob.Gem;
    GemData?: poe.SkillGem;
    GrantedEffectLevel?: raw.CalculatedLevel;
    Superseded: boolean;
    IsSupporting?: Record<pob.Gem | undefined, boolean>;
    Values?: Record<string, number>;
  }
  interface GrantedEffect {
    Raw?: poe.GrantedEffect;
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
    AllocNodes?: Record<string, data.Node>;
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
    Background?: Record<string, data.Sprite>;
    NormalActive?: Record<string, data.Sprite>;
    NotableActive?: Record<string, data.Sprite>;
    KeystoneActive?: Record<string, data.Sprite>;
    NormalInactive?: Record<string, data.Sprite>;
    NotableInactive?: Record<string, data.Sprite>;
    KeystoneInactive?: Record<string, data.Sprite>;
    Mastery?: Record<string, data.Sprite>;
    MasteryConnected?: Record<string, data.Sprite>;
    MasteryActiveSelected?: Record<string, data.Sprite>;
    MasteryInactive?: Record<string, data.Sprite>;
    MasteryActiveEffect?: Record<string, data.Sprite>;
    AscendancyBackground?: Record<string, data.Sprite>;
    Ascendancy?: Record<string, data.Sprite>;
    StartNode?: Record<string, data.Sprite>;
    GroupBackground?: Record<string, data.Sprite>;
    Frame?: Record<string, data.Sprite>;
    Jewel?: Record<string, data.Sprite>;
    Line?: Record<string, data.Sprite>;
    JewelRadius?: Record<string, data.Sprite>;
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
export declare namespace debug {
  interface BuildInfo {
    GoVersion: string;
    Path: string;
    Main: debug.Module;
    Deps?: Array<debug.Module | undefined>;
    Settings?: Array<debug.BuildSetting>;
    String(): string;
  }
  interface BuildSetting {
    Key: string;
    Value: string;
  }
  interface Module {
    Path: string;
    Version: string;
    Sum: string;
    Replace?: debug.Module;
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
  function CalculateTreePath(version: string, activeNodes?: Array<number>, target: number): (Array<number> | undefined);
  function GetRawTree(version: string): Promise<(Uint8Array | undefined)>;
  function GetSkillGems(): (Array<exposition.SkillGem> | undefined);
  function GetStatByIndex(id: number): (poe.Stat | undefined);
}
export declare namespace fwd {
  interface Reader {
    BufferSize(): number;
    Buffered(): number;
    InputOffset(): number;
    Next(n: number): [(Uint8Array | undefined), Error];
    Peek(n: number): [(Uint8Array | undefined), Error];
    PeekByte(): [number, Error];
    Read(b?: Uint8Array): [number, Error];
    ReadByte(): [number, Error];
    ReadFull(b?: Uint8Array): [number, Error];
    Reset(rd?: unknown): void;
    Skip(n: number): [number, Error];
    WriteTo(w?: unknown): [number, Error];
  }
}
export declare namespace msgp {
  interface Reader {
    R?: fwd.Reader;
    BufferSize(): number;
    Buffered(): number;
    CopyNext(w?: unknown): [number, Error];
    IsNil(): boolean;
    NextType(): [number, Error];
    Read(p?: Uint8Array): [number, Error];
    ReadArrayHeader(): [number, Error];
    ReadBool(): [boolean, Error];
    ReadByte(): [number, Error];
    ReadBytes(scratch?: Uint8Array): [(Uint8Array | undefined), Error];
    ReadBytesHeader(): [number, Error];
    ReadDuration(): [number, Error];
    ReadExactBytes(into?: Uint8Array): Error;
    ReadExtension(e?: unknown): Error;
    ReadExtensionRaw(): [number, (Uint8Array | undefined), Error];
    ReadFloat32(): [number, Error];
    ReadFloat64(): [number, Error];
    ReadFull(p?: Uint8Array): [number, Error];
    ReadInt(): [number, Error];
    ReadInt16(): [number, Error];
    ReadInt32(): [number, Error];
    ReadInt64(): [number, Error];
    ReadInt8(): [number, Error];
    ReadIntf(): [(unknown | undefined), Error];
    ReadJSONNumber(): [string, Error];
    ReadMapHeader(): [number, Error];
    ReadMapKey(scratch?: Uint8Array): [(Uint8Array | undefined), Error];
    ReadMapKeyPtr(): [(Uint8Array | undefined), Error];
    ReadMapStrIntf(mp?: Record<string, unknown | undefined>): Error;
    ReadNil(): Error;
    ReadString(): [string, Error];
    ReadStringAsBytes(scratch?: Uint8Array): [(Uint8Array | undefined), Error];
    ReadStringHeader(): [number, Error];
    ReadTime(): [time.Time, Error];
    ReadUint(): [number, Error];
    ReadUint16(): [number, Error];
    ReadUint32(): [number, Error];
    ReadUint64(): [number, Error];
    ReadUint8(): [number, Error];
    Reset(r?: unknown): void;
    Skip(): Error;
    WriteToJSON(w?: unknown): [number, Error];
  }
  interface Writer {
    Append(b?: Uint8Array): Error;
    Buffered(): number;
    Flush(): Error;
    Reset(w?: unknown): void;
    Write(p?: Uint8Array): [number, Error];
    WriteArrayHeader(sz: number): Error;
    WriteBool(b: boolean): Error;
    WriteByte(u: number): Error;
    WriteBytes(b?: Uint8Array): Error;
    WriteBytesHeader(sz: number): Error;
    WriteDuration(d: number): Error;
    WriteExtension(e?: unknown): Error;
    WriteExtensionRaw(extType: number, payload?: Uint8Array): Error;
    WriteFloat(f: number): Error;
    WriteFloat32(f: number): Error;
    WriteFloat64(f: number): Error;
    WriteInt(i: number): Error;
    WriteInt16(i: number): Error;
    WriteInt32(i: number): Error;
    WriteInt64(i: number): Error;
    WriteInt8(i: number): Error;
    WriteIntf(v?: unknown): Error;
    WriteJSONNumber(n: string): Error;
    WriteMapHeader(sz: number): Error;
    WriteMapStrIntf(mp?: Record<string, unknown | undefined>): Error;
    WriteMapStrStr(mp?: Record<string, string>): Error;
    WriteNil(): Error;
    WriteString(s: string): Error;
    WriteStringFromBytes(str?: Uint8Array): Error;
    WriteStringHeader(sz: number): Error;
    WriteTime(t: time.Time): Error;
    WriteUint(u: number): Error;
    WriteUint16(u: number): Error;
    WriteUint32(u: number): Error;
    WriteUint64(u: number): Error;
    WriteUint8(u: number): Error;
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
    PassiveNodes?: Array<number>;
    PassiveNodesStartPaths?: Record<number, Array<number> | undefined>;
    PlayerStats: Array<pob.PlayerStat>;
  }
  interface Calcs {
    Inputs: Array<pob.Input>;
    Sections: Array<pob.Section>;
  }
  interface Config {
    Inputs: Array<pob.Input>;
    Placeholders: Array<pob.Input>;
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
    ItemSets: Array<pob.ItemSet>;
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
    AllocateNodes(nodeIds?: Array<number>): void;
    DeallocateNodes(nodeId: number): void;
    DeleteAllSocketGroups(): void;
    DeleteSocketGroup(index: number): void;
    GetStringOption(name: string): string;
    RemoveConfigOption(name: string): void;
    SetAscendancy(ascendancy: string): void;
    SetClass(clazz: string): void;
    SetConfigOption(value: pob.Input): void;
    SetDefaultGemLevel(gemLevel: number): void;
    SetDefaultGemQuality(gemQuality: number): void;
    SetLevel(level: number): void;
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
    Gems: Array<pob.Gem>;
    Slot: string;
    SlotEnabled: boolean;
    Source?: unknown;
    DisplayLabel: string;
    DisplaySkillList?: unknown;
    DisplaySkillListCalcs?: unknown;
  }
  interface SkillSet {
    ID: number;
    Skills: Array<pob.Skill>;
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
    SkillSets: Array<pob.SkillSet>;
  }
  interface Slot {
    ItemID: number;
    Name: string;
  }
  interface Spec {
    ClassID: number;
    AscendClassID: number;
    TreeVersion: string;
    NodesAttr: string;
    MasteryEffects: string;
    URL: string;
  }
  interface Tree {
    ActiveSpec: number;
    Specs: Array<pob.Spec>;
  }
  interface TreeView {
    ZoomLevel: number;
    ZoomX: number;
    ZoomY: number;
    SearchStr: string;
    ShowHeatMap?: boolean;
    ShowStatDifferences: boolean;
  }
  const BuildInfo: debug.BuildInfo | undefined;
  function CompressEncode(xml: string): [string, Error];
  function DecodeDecompress(code: string): [string, Error];
}
export declare namespace poe {
  interface ActiveSkill {
    ActiveSkill: raw.ActiveSkill;
    DecodeMsg(arg1?: msgp.Reader): Error;
    EncodeMsg(arg1?: msgp.Writer): Error;
    GetActiveSkillBaseFlagsAndTypes(): [(Record<string, boolean> | undefined), (Record<string, boolean> | undefined)];
    GetActiveSkillTypes(): (Array<poe.ActiveSkillType | undefined> | undefined);
    GetWeaponRestrictions(): (Array<poe.ItemClass | undefined> | undefined);
    MarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
  }
  interface ActiveSkillType {
    ActiveSkillType: raw.ActiveSkillType;
    DecodeMsg(arg1?: msgp.Reader): Error;
    EncodeMsg(arg1?: msgp.Writer): Error;
    MarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
  }
  interface BaseItemType {
    BaseItemType: raw.BaseItemType;
    DecodeMsg(arg1?: msgp.Reader): Error;
    EncodeMsg(arg1?: msgp.Writer): Error;
    MarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    SkillGem(): (poe.SkillGem | undefined);
    UnmarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
  }
  interface CostType {
    CostType: raw.CostType;
    DecodeMsg(arg1?: msgp.Reader): Error;
    EncodeMsg(arg1?: msgp.Writer): Error;
    MarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
  }
  interface GrantedEffect {
    GrantedEffect: raw.GrantedEffect;
    DecodeMsg(arg1?: msgp.Reader): Error;
    EncodeMsg(arg1?: msgp.Writer): Error;
    GetActiveSkill(): (poe.ActiveSkill | undefined);
    GetEffectQualityStats(): (Record<number, poe.GrantedEffectQualityStat | undefined> | undefined);
    GetEffectStatSetsPerLevel(): (Record<number, poe.GrantedEffectStatSetsPerLevel | undefined> | undefined);
    GetEffectsPerLevel(): (Record<number, poe.GrantedEffectsPerLevel | undefined> | undefined);
    GetExcludeTypes(): (Array<poe.ActiveSkillType | undefined> | undefined);
    GetGrantedEffectStatSet(): (poe.GrantedEffectStatSet | undefined);
    GetSkillGem(): (poe.SkillGem | undefined);
    GetSupportTypes(): (Array<poe.ActiveSkillType | undefined> | undefined);
    HasGlobalEffect(): boolean;
    Levels(): (Record<number, poe.GrantedEffectsPerLevel | undefined> | undefined);
    MarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
  }
  interface GrantedEffectQualityStat {
    GrantedEffectQualityStat: raw.GrantedEffectQualityStat;
    DecodeMsg(arg1?: msgp.Reader): Error;
    EncodeMsg(arg1?: msgp.Writer): Error;
    GetStats(): (Array<poe.Stat | undefined> | undefined);
    MarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
  }
  interface GrantedEffectStatSet {
    GrantedEffectStatSet: raw.GrantedEffectStatSet;
    DecodeMsg(arg1?: msgp.Reader): Error;
    EncodeMsg(arg1?: msgp.Writer): Error;
    GetConstantStats(): (Array<poe.Stat | undefined> | undefined);
    GetImplicitStats(): (Array<poe.Stat | undefined> | undefined);
    MarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
  }
  interface GrantedEffectStatSetsPerLevel {
    GrantedEffectStatSetsPerLevel: raw.GrantedEffectStatSetsPerLevel;
    DecodeMsg(arg1?: msgp.Reader): Error;
    EncodeMsg(arg1?: msgp.Writer): Error;
    GetAdditionalBooleanStats(): (Array<poe.Stat | undefined> | undefined);
    GetAdditionalStats(): (Array<poe.Stat | undefined> | undefined);
    GetFloatStats(): (Array<poe.Stat | undefined> | undefined);
    MarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
  }
  interface GrantedEffectsPerLevel {
    GrantedEffectsPerLevel: raw.GrantedEffectsPerLevel;
    DecodeMsg(arg1?: msgp.Reader): Error;
    EncodeMsg(arg1?: msgp.Writer): Error;
    GetCostTypes(): (Array<poe.CostType | undefined> | undefined);
    MarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
  }
  interface ItemClass {
    ItemClass: raw.ItemClass;
    DecodeMsg(arg1?: msgp.Reader): Error;
    EncodeMsg(arg1?: msgp.Writer): Error;
    MarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
  }
  interface SkillGem {
    SkillGem: raw.SkillGem;
    DecodeMsg(arg1?: msgp.Reader): Error;
    DefaultLevel(): number;
    EncodeMsg(arg1?: msgp.Writer): Error;
    GetBaseItemType(): (poe.BaseItemType | undefined);
    GetGrantedEffect(): (poe.GrantedEffect | undefined);
    GetGrantedEffects(): (Array<poe.GrantedEffect | undefined> | undefined);
    GetNonVaal(): (poe.SkillGem | undefined);
    GetSecondaryGrantedEffect(): (poe.GrantedEffect | undefined);
    GetTags(): (Record<string, poe.Tag | undefined> | undefined);
    MarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
  }
  interface Stat {
    Stat: raw.Stat;
    DecodeMsg(arg1?: msgp.Reader): Error;
    EncodeMsg(arg1?: msgp.Writer): Error;
    MarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
  }
  interface Tag {
    Tag: raw.Tag;
    DecodeMsg(arg1?: msgp.Reader): Error;
    EncodeMsg(arg1?: msgp.Writer): Error;
    MarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(arg1?: Uint8Array): [(Uint8Array | undefined), Error];
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
    ActiveSkillTypes?: Array<number>;
    ActiveSkillTargetTypes?: Array<number>;
    SkillTotemID: number;
    Key: number;
    IsManuallyCasted: boolean;
    DecodeMsg(dc?: msgp.Reader): Error;
    EncodeMsg(en?: msgp.Writer): Error;
    MarshalMsg(b?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(bts?: Uint8Array): [(Uint8Array | undefined), Error];
  }
  interface ActiveSkillType {
    FlagStat?: number;
    ID: string;
    Key: number;
    DecodeMsg(dc?: msgp.Reader): Error;
    EncodeMsg(en?: msgp.Writer): Error;
    MarshalMsg(b?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(bts?: Uint8Array): [(Uint8Array | undefined), Error];
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
    IdentifyMagicAchievementItems?: Array<unknown | undefined>;
    IdentifyAchievementItems?: Array<unknown | undefined>;
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
    DecodeMsg(dc?: msgp.Reader): Error;
    EncodeMsg(en?: msgp.Writer): Error;
    MarshalMsg(b?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(bts?: Uint8Array): [(Uint8Array | undefined), Error];
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
    DecodeMsg(dc?: msgp.Reader): Error;
    EncodeMsg(en?: msgp.Writer): Error;
    MarshalMsg(b?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(bts?: Uint8Array): [(Uint8Array | undefined), Error];
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
    Attribute: number;
    CastTime: number;
    GrantedEffectStatSets: number;
    Key: number;
    IgnoreMinionTypes: boolean;
    CannotBeSupported: boolean;
    SupportsGemsOnly: boolean;
    IsSupport: boolean;
    DecodeMsg(dc?: msgp.Reader): Error;
    EncodeMsg(en?: msgp.Writer): Error;
    MarshalMsg(b?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(bts?: Uint8Array): [(Uint8Array | undefined), Error];
  }
  interface GrantedEffectQualityStat {
    StatsKeys?: Array<number>;
    StatsValuesPermille?: Array<number>;
    GrantedEffectsKey: number;
    SetID: number;
    Weight: number;
    Key: number;
    DecodeMsg(dc?: msgp.Reader): Error;
    EncodeMsg(en?: msgp.Writer): Error;
    MarshalMsg(b?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(bts?: Uint8Array): [(Uint8Array | undefined), Error];
  }
  interface GrantedEffectStatSet {
    ID: string;
    ImplicitStats?: Array<number>;
    ConstantStats?: Array<number>;
    ConstantStatsValues?: Array<number>;
    Key: number;
    BaseEffectiveness: number;
    IncrementalEffectiveness: number;
    DecodeMsg(dc?: msgp.Reader): Error;
    EncodeMsg(en?: msgp.Writer): Error;
    MarshalMsg(b?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(bts?: Uint8Array): [(Uint8Array | undefined), Error];
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
    DecodeMsg(dc?: msgp.Reader): Error;
    EncodeMsg(en?: msgp.Writer): Error;
    MarshalMsg(b?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(bts?: Uint8Array): [(Uint8Array | undefined), Error];
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
    Key: number;
    DecodeMsg(dc?: msgp.Reader): Error;
    EncodeMsg(en?: msgp.Writer): Error;
    MarshalMsg(b?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(bts?: Uint8Array): [(Uint8Array | undefined), Error];
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
    DecodeMsg(dc?: msgp.Reader): Error;
    EncodeMsg(en?: msgp.Writer): Error;
    MarshalMsg(b?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(bts?: Uint8Array): [(Uint8Array | undefined), Error];
  }
  interface SkillGem {
    VaalGem?: number;
    RegularVariant?: number;
    AwakenedVariant?: number;
    GlobalGemLevelStat?: number;
    SecondaryGrantedEffect?: number;
    HungryLoopMod?: number;
    SecondarySupportName: string;
    Description: string;
    Tags?: Array<number>;
    Int: number;
    Dex: number;
    BaseItemType: number;
    Str: number;
    GrantedEffect: number;
    Key: number;
    IsVaalGem: boolean;
    DecodeMsg(dc?: msgp.Reader): Error;
    EncodeMsg(en?: msgp.Writer): Error;
    MarshalMsg(b?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(bts?: Uint8Array): [(Uint8Array | undefined), Error];
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
    DecodeMsg(dc?: msgp.Reader): Error;
    EncodeMsg(en?: msgp.Writer): Error;
    MarshalMsg(b?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(bts?: Uint8Array): [(Uint8Array | undefined), Error];
  }
  interface Tag {
    DisplayString: string;
    ID: string;
    Name: string;
    Key: number;
    DecodeMsg(dc?: msgp.Reader): Error;
    EncodeMsg(en?: msgp.Writer): Error;
    MarshalMsg(b?: Uint8Array): [(Uint8Array | undefined), Error];
    Msgsize(): number;
    UnmarshalMsg(bts?: Uint8Array): [(Uint8Array | undefined), Error];
  }
  function InitializeAll(version: string, updateFunc: (arg1: string) => Promise<void>): Promise<Error>;
}
export declare namespace time {
  interface Location {
    String(): string;
  }
  interface Time {
    Add(d: number): time.Time;
    AddDate(years: number, months: number, days: number): time.Time;
    After(u: time.Time): boolean;
    AppendFormat(b?: Uint8Array, layout: string): (Uint8Array | undefined);
    Before(u: time.Time): boolean;
    Clock(): [number, number, number];
    Compare(u: time.Time): number;
    Date(): [number, number, number];
    Day(): number;
    Equal(u: time.Time): boolean;
    Format(layout: string): string;
    GoString(): string;
    GobDecode(data?: Uint8Array): Error;
    GobEncode(): [(Uint8Array | undefined), Error];
    Hour(): number;
    ISOWeek(): [number, number];
    In(loc?: time.Location): time.Time;
    IsDST(): boolean;
    IsZero(): boolean;
    Local(): time.Time;
    Location(): (time.Location | undefined);
    MarshalBinary(): [(Uint8Array | undefined), Error];
    MarshalJSON(): [(Uint8Array | undefined), Error];
    MarshalText(): [(Uint8Array | undefined), Error];
    Minute(): number;
    Month(): number;
    Nanosecond(): number;
    Round(d: number): time.Time;
    Second(): number;
    String(): string;
    Sub(u: time.Time): number;
    Truncate(d: number): time.Time;
    UTC(): time.Time;
    Unix(): number;
    UnixMicro(): number;
    UnixMilli(): number;
    UnixNano(): number;
    UnmarshalBinary(data?: Uint8Array): Error;
    UnmarshalJSON(data?: Uint8Array): Error;
    UnmarshalText(data?: Uint8Array): Error;
    Weekday(): number;
    Year(): number;
    YearDay(): number;
    Zone(): [string, number];
    ZoneBounds(): [time.Time, time.Time];
  }
}
export const initializeCrystalline: () => void;