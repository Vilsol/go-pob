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
export declare namespace calculator2 {
  interface ConfigRelevance {
    readonly Conds?: Array<string>;
    readonly MinionConds?: Array<string>;
    readonly EnemyConds?: Array<string>;
    readonly Mults?: Array<string>;
    readonly EnemyMults?: Array<string>;
    readonly Stats?: Array<string>;
    readonly EnemyStats?: Array<string>;
    readonly TagTypes?: Array<string>;
    readonly Flags?: Array<string>;
    readonly PlayerFlags?: Array<string>;
    readonly MinionFlags?: Array<string>;
    readonly Mods?: Array<string>;
    readonly Skills?: Array<string>;
    readonly TransfiguredSkills?: Array<string>;
    readonly SkillTypes?: Array<string>;
    readonly SkillFlags?: Array<string>;
    readonly SkillData?: Array<string>;
    readonly GemFamilies?: Array<string>;
  }
  interface ConfigSetView {
    readonly ID: number;
    readonly Title: string;
  }
  interface ConfigValue {
    readonly Number: number;
    readonly String: string;
    readonly Boolean: boolean;
    readonly IsNumber: boolean;
    readonly IsString: boolean;
    readonly IsBoolean: boolean;
  }
  interface GemListEntry {
    readonly ID: string;
    readonly Name: string;
    readonly Tags?: Array<string>;
    readonly Support: boolean;
    readonly IsLineage: boolean;
    readonly Legacy: boolean;
    readonly LevelRequirement: number;
    readonly Color: number;
  }
  interface GemOptions {
    readonly DefaultGemLevel: string;
    readonly DefaultGemQuality: number;
    readonly ShowSupportGemTypes: string;
    readonly ShowLegacyGems: boolean;
    readonly SortGemsByDPS: boolean;
    readonly SortGemsByDPSField: string;
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
    readonly ItemID: number;
    readonly NodeID: number;
  }
  interface PlayerProgress {
    readonly Level: number;
    readonly Act: string;
    readonly Extra: number;
    readonly NormalPassives: number;
    readonly UsedMax: number;
    readonly WeaponSet1Used: number;
    readonly WeaponSet2Used: number;
    readonly WeaponSetMax: number;
    readonly AscUsed: number;
    readonly AscMax: number;
    readonly Warnings?: Array<string>;
  }
  interface PowerReportRow {
    readonly ID: number;
    readonly Name: string;
    readonly Type: string;
    readonly Allocated: boolean;
    readonly Power: number;
    readonly PathPower: number;
    readonly PathDist: number;
  }
  interface SkillDPS {
    readonly Name: string;
    readonly DPS: number;
    readonly Count: number;
    readonly Trigger: string;
    readonly SkillPart: string;
    readonly Source: string;
  }
  interface SkillSetView {
    readonly ID: number;
    readonly Title: string;
  }
  interface SocketGroupView {
    readonly Label: string;
    readonly Skills?: Array<string>;
  }
}
export declare namespace config {
  function InitLogging(withTime: boolean): void;
}
export declare namespace exposition {
  interface AffixControlView {
    readonly Type: string;
    readonly List?: Array<exposition.AffixEntryView>;
    readonly SelIndex: number;
    readonly SliderShown: boolean;
    readonly SliderVal: number;
  }
  interface AffixEntryView {
    readonly Label: string;
    readonly ModList?: Array<string>;
    readonly HaveRange: boolean;
    readonly Tooltip?: Array<string>;
  }
  interface CalcsBreakdown {
    readonly Lines?: Array<string>;
    readonly Radius: number;
    readonly Tables?: Array<exposition.CalcsBreakdownTable>;
    readonly Gear: boolean;
    readonly GearTable: exposition.CalcsBreakdownTable;
  }
  interface CalcsBreakdownCol {
    readonly Label: string;
    readonly Key: string;
  }
  interface CalcsBreakdownTable {
    readonly Label: string;
    readonly Footer: string;
    readonly Columns?: Array<exposition.CalcsBreakdownCol>;
    readonly Rows?: Array<Record<string, string> | undefined>;
  }
  interface CalcsModEntry {
    readonly Cfg: string;
    readonly ModSource: string;
    readonly Enemy: boolean;
    readonly ModName?: Array<string>;
    readonly ModType: string;
    readonly Actor: string;
  }
  interface CalcsModQuery {
    readonly Entries?: Record<string, Readonly<exposition.CalcsModEntry>>;
  }
  interface CalcsOutputs {
    readonly Flags?: Record<string, boolean>;
    readonly Strings?: Record<string, string>;
    readonly MainHand?: Record<string, number>;
    readonly OffHand?: Record<string, number>;
  }
  interface Calculator {
    AddDBItem(kind: string, name: string, equip: boolean, second: boolean): Result<number>;
    AddDisplayItem(slot: string): Result<number>;
    AddGem(group: number, gemID: string): Result<void>;
    AddSkillsUndoState(): void;
    AllocateAttributeNode(id: number, attributeIndex: number, allocMode: number): Result<void>;
    BuildItemCompare(id: number): Promise<Result<(exposition.ItemCompareView | undefined)>>;
    CalcsBreakdowns(): Result<(Record<string, Readonly<exposition.CalcsBreakdown>> | undefined)>;
    CalcsModSection(sd: calculator2.ModSectionData): Result<(Array<calculator2.ModSectionRow> | undefined)>;
    CalcsMods(query: exposition.CalcsModQuery): Result<(Record<string, number> | undefined)>;
    CalcsOutputs(): Result<(exposition.CalcsOutputs | undefined)>;
    Calculate(): Promise<Result<(exposition.Outputs | undefined)>>;
    ChangeTreeAscendancy(ascendClassID: number): Result<void>;
    ChangeTreeClass(classID: number, choice: number): Result<boolean>;
    ClickNode(id: number, right: boolean, hotkey: number, confirmClassChange: boolean, allocMode: number, trace: Array<number> | undefined): Result<number>;
    ClickNodeConnectPath(id: number, hotkey: number, allocMode: number): Result<number>;
    CloseDisplayItem(): Result<void>;
    ConfigRelevance(query: calculator2.ConfigRelevance): Result<calculator2.ConfigRelevance>;
    ConfigSets(): exposition.ConfigSetsView;
    CopyConfigSet(id: number, title: string): number;
    CopyItemSet(id: number, title: string): Result<void>;
    CopySkillSet(id: number, title: string): Result<number>;
    CopySocketGroup(index: number): Result<string>;
    CopyTreeSpec(specID: number, title: string): Result<void>;
    CraftDisplayItem(rarity: number, title: string, base: string): Result<void>;
    CraftItemCompare(rarity: number, title: string, base: string): Promise<Result<(exposition.ItemCompareView | undefined)>>;
    CustomModifierBlocks(): (Array<exposition.CustomBlock> | undefined);
    DBItemCompare(kind: string, name: string): Promise<Result<(exposition.ItemCompareView | undefined)>>;
    DeleteAllSocketGroups(): Result<void>;
    DeleteConfigSet(id: number): void;
    DeleteItem(id: number): Result<void>;
    DeleteItemSet(id: number): Result<void>;
    DeleteSkillSet(id: number): Result<void>;
    DeleteSocketGroup(index: number): Result<void>;
    DeleteTreeSpec(specID: number): void;
    DisplayItemAffixRoll(control: number, val: number): Result<(Array<string> | undefined)>;
    DisplayItemAffixes(): Result<(Array<exposition.AffixControlView> | undefined)>;
    DisplayItemCompare(): Promise<Result<(exposition.ItemCompareView | undefined)>>;
    DisplayItemRanges(): Result<[(Array<exposition.RangeLineView> | undefined), number]>;
    EditDisplayItemText(text: string, rarity: string): Result<void>;
    EditItem(id: number): Result<void>;
    EquippedItems(): Result<(exposition.ItemsView | undefined)>;
    ExportCode(player: Array<exposition.SavedStat> | undefined, fullDPS: Array<exposition.SavedFullDPS> | undefined, minion: Array<exposition.SavedStat> | undefined): Result<string>;
    GemOptions(): calculator2.GemOptions;
    GemSortCache(group: number, gemIDs: Array<string> | undefined): Result<(exposition.GemSortCacheView | undefined)>;
    GemSortDPS(group: number, gemIDs: Array<string> | undefined, field: string): Promise<Result<(exposition.GemSortDPSView | undefined)>>;
    HoverNode(id: number, allocMode: number): exposition.TreeHover;
    ItemDBRows(kind: string, f: calculator2.ItemDBFilter): Result<(Array<exposition.ItemDBRow> | undefined)>;
    ItemText(id: number): Result<string>;
    ItemsPage(): Result<(exposition.ItemsPageView | undefined)>;
    MoveConfigSet(from: number, to: number): void;
    MoveItemSet(from: number, to: number): Result<void>;
    MoveSocketGroup(from: number, to: number): Result<void>;
    MoveTreeSpec(from: number, to: number): void;
    NewConfigSet(title: string): number;
    NewItemSet(title: string): Result<void>;
    NewSkillSet(title: string): Result<number>;
    NewSocketGroup(): Result<number>;
    NewTreeSpec(title: string): Result<void>;
    NodeCompare(id: number, trace: Array<number> | undefined, tracing: boolean): Promise<Result<(exposition.NodeCompareView | undefined)>>;
    NodePower(): Result<(exposition.NodePowerView | undefined)>;
    NodePowerReport(): Result<(Array<calculator2.PowerReportRow> | undefined)>;
    NodeTip(id: number, trace: Array<number> | undefined, tracing: boolean): Result<(exposition.NodeTipView | undefined)>;
    Notes(): string;
    OpenDBItem(kind: string, name: string): Result<void>;
    PasteItem(raw: string): Result<void>;
    PasteSocketGroup(text: string): Result<number>;
    Redo(): Result<boolean>;
    RemoveGem(group: number, gem: number): Result<void>;
    RenameConfigSet(id: number, title: string): void;
    RenameItemSet(id: number, title: string): Result<void>;
    RenameSkillSet(id: number, title: string): Result<void>;
    RenameTreeSpec(specID: number, title: string): void;
    ResetTree(): Result<void>;
    RichNotesDoc(): string;
    RichNotesStatus(): string;
    SelectDisplayItemAffix(control: number, entry: number, slider: number): Result<void>;
    Serialize(): Result<string>;
    SerializeWithStats(player: Array<exposition.SavedStat> | undefined, fullDPS: Array<exposition.SavedFullDPS> | undefined, minion: Array<exposition.SavedStat> | undefined): Result<string>;
    SetActiveConfigSet(id: number): void;
    SetActiveItemSet(id: number): Result<void>;
    SetActiveSkillSet(id: number): Result<void>;
    SetActiveTreeSpec(specID: number): void;
    SetCharacterLevelAutoMode(auto: boolean): void;
    SetCharacterRune(slot: string, runeName: string): Result<void>;
    SetConfigInput(name: string, value: calculator2.ConfigValue, isDefault: boolean): void;
    SetCustomModifierBlocks(blocks: Array<exposition.CustomBlock> | undefined): void;
    SetDisplayItemAffixRange(control: number, val: number): Result<void>;
    SetDisplayItemQuality(quality: number): Result<void>;
    SetDisplayItemRangeLine(line: number, val: number): Result<void>;
    SetDisplayItemRune(socket: number, runeName: string): Result<void>;
    SetDisplayItemSocketCount(count: number): Result<void>;
    SetGem(group: number, gem: number, gemID: string): Result<void>;
    SetGemCorruption(group: number, gem: number, option: number): Result<void>;
    SetGemCount(group: number, gem: number, count: number): Result<void>;
    SetGemEnabled(group: number, gem: number, state: boolean): Result<void>;
    SetGemLevel(group: number, gem: number, level: number): Result<void>;
    SetGemOptions(options: calculator2.GemOptions): void;
    SetGemQuality(group: number, gem: number, quality: number): Result<void>;
    SetLevel(level: number): void;
    SetMainActiveSkill(index: number): void;
    SetMainSocketGroup(index: number): void;
    SetRichNotes(plain: string, doc: string): void;
    SetSlotActive(slot: string, state: boolean): Result<void>;
    SetSlotItem(slot: string, id: number): Result<void>;
    SetSocketGroupCount(index: number, count: number): Result<void>;
    SetSocketGroupEnabled(index: number, state: boolean): Result<void>;
    SetSocketGroupIncludeInFullDPS(index: number, state: boolean): Result<void>;
    SetSocketGroupLabel(index: number, label: string): Result<void>;
    SetSocketGroupWeaponSet(index: number, set: number, state: boolean): Result<void>;
    SetTreeNodeNote(nodeID: number, text: string): void;
    SetUseSecondWeaponSet(second: boolean): Result<void>;
    SidebarBreakdownKey(stat: string, childStat: string, breakdown: string, hasBreakdown: boolean, modNames: boolean): Result<exposition.SidebarBreakdownRef>;
    SkillGroups(): Result<(Array<exposition.SkillGroupView> | undefined)>;
    SkillSets(): exposition.SkillSetsView;
    SkillsUndo(redo: boolean): Result<boolean>;
    StartNodePower(stat: string, maxDepth: number): Promise<Result<(exposition.PowerProgress | undefined)>>;
    StepNodePower(n: number): Promise<Result<(exposition.PowerProgress | undefined)>>;
    TraceNode(trace: Array<number> | undefined, id: number): Result<(exposition.TraceView | undefined)>;
    TreeNodeNote(nodeID: number): string;
    TreeSpecAlloc(specID: number): Result<(Array<number> | undefined)>;
    TreeSpecRows(): Result<(Array<exposition.TreeSpecRow> | undefined)>;
    TreeSpecs(): exposition.TreeSpecsView;
    TreeState(): Result<(exposition.TreeState | undefined)>;
    Undo(): Result<boolean>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface CompareOutputs {
    readonly Output?: Record<string, number>;
    readonly Minion?: Record<string, number>;
  }
  interface ConfigSetsView {
    readonly Sets?: Array<calculator2.ConfigSetView>;
    readonly Active: number;
  }
  interface CraftBaseGroup {
    readonly Type: string;
    readonly Bases?: Array<string>;
  }
  interface CustomBlock {
    readonly Title: string;
    readonly Enabled: boolean;
    readonly Text: string;
  }
  interface DisplayItemView {
    readonly ID: number;
    readonly Item: exposition.ItemView;
    readonly Text: string;
    readonly Rarity: string;
    readonly Fits?: Array<string>;
    readonly CanHaveSockets: boolean;
    readonly SocketCount: number;
    readonly Runes?: Array<string>;
    readonly RuneChoices?: Array<exposition.RuneChoiceView>;
  }
  interface GemSortCacheView {
    readonly CanSupport?: Array<boolean>;
    readonly Pending?: Array<boolean>;
  }
  interface GemSortDPSView {
    readonly Base: number;
    readonly DPS?: Array<number>;
  }
  interface ItemCompareBlock {
    readonly Header: string;
    readonly Slot: string;
    readonly Compare: exposition.CompareOutputs;
  }
  interface ItemCompareView {
    readonly Base: exposition.CompareOutputs;
    readonly SkillFlags?: Record<string, boolean>;
    readonly MinionSkillFlags?: Record<string, boolean>;
    readonly Minion: boolean;
    readonly Blocks?: Array<exposition.ItemCompareBlock>;
  }
  interface ItemDBInfo {
    readonly Slots?: Array<string>;
    readonly Types?: Array<string>;
    readonly Leagues?: Array<string>;
    readonly Requirements?: Array<string>;
    readonly Obtainable?: Array<string>;
    readonly SearchModes?: Array<string>;
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
  }
  interface ItemSetView {
    readonly ID: number;
    readonly Title: string;
  }
  interface ItemSlotView {
    readonly Name: string;
    readonly Label: string;
    readonly Parent: string;
    readonly NodeID: number;
    readonly ItemID: number;
    readonly Choices?: Array<number>;
    readonly Activatable: boolean;
    readonly Active: boolean;
  }
  interface ItemView {
    readonly Slot: string;
    readonly ID: number;
    readonly Name?: Array<string>;
    readonly Rarity: string;
    readonly Lines?: Array<Array<string> | undefined>;
  }
  interface ItemsPageView {
    readonly ActiveItemSet: number;
    readonly Sets?: Array<exposition.ItemSetView>;
    readonly UseSecondWeaponSet: boolean;
    readonly Slots?: Array<exposition.ItemSlotView>;
    readonly Items?: Array<exposition.ItemView>;
    readonly RuneSlots?: Array<exposition.RuneSlotView>;
    readonly Display?: exposition.DisplayItemView;
  }
  interface ItemsView {
    readonly UseSecondWeaponSet: boolean;
    readonly Items?: Array<exposition.ItemView>;
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
  interface NodeCompareBlockView {
    readonly Header: string;
    readonly PathLength: number;
    readonly Compare: exposition.CompareOutputs;
  }
  interface NodeCompareView {
    readonly Base: exposition.CompareOutputs;
    readonly SkillFlags?: Record<string, boolean>;
    readonly MinionSkillFlags?: Record<string, boolean>;
    readonly Minion: boolean;
    readonly Blocks?: Array<exposition.NodeCompareBlockView>;
    readonly NoChange: string;
  }
  interface NodePowerView {
    readonly IDs?: Array<number>;
    readonly Distance?: Array<number>;
    readonly Single?: Array<number>;
    readonly Path?: Array<number>;
    readonly Offence?: Array<number>;
    readonly Defence?: Array<number>;
    readonly HasSingle?: Array<boolean>;
    readonly HasPath?: Array<boolean>;
    readonly Max?: Array<number>;
  }
  interface NodeTipView {
    readonly PathLines?: Array<string>;
    readonly DistanceToStart: number;
    readonly Granted: boolean;
  }
  interface Outputs {
    readonly Output?: Record<string, number>;
    readonly Minion?: Record<string, number>;
    readonly ClassName: string;
    readonly AscendClassName: string;
    readonly Level: number;
    readonly SkillFlags?: Record<string, boolean>;
    readonly MinionSkillFlags?: Record<string, boolean>;
    readonly SkillData?: Record<string, boolean>;
    readonly SocketGroups?: Array<calculator2.SocketGroupView>;
    readonly MainSocketGroup: number;
    readonly MainActiveSkill: number;
    readonly ConfigInputs?: Record<string, Readonly<calculator2.ConfigValue>>;
    readonly SkillDPS?: Array<calculator2.SkillDPS>;
  }
  interface ParsedMod {
    readonly Name: string;
    readonly Type: string;
    readonly Value: string;
    readonly Tags?: Array<string>;
  }
  interface PowerProgress {
    readonly Done: number;
    readonly Tasks: number;
    readonly Total: number;
    readonly Finished: boolean;
  }
  interface PowerStatView {
    readonly Stat: string;
    readonly Label: string;
  }
  interface RangeLineView {
    readonly Label: string;
    readonly Range: number;
  }
  interface RuneChoiceView {
    readonly Name: string;
    readonly Lines?: Array<string>;
  }
  interface RuneSlotView {
    readonly Name: string;
    readonly Label: string;
    readonly Rune: string;
    readonly Choices?: Array<string>;
  }
  interface SavedFullDPS {
    readonly Stat: string;
    readonly Value: number;
    readonly SkillPart: string;
    readonly Source: string;
  }
  interface SavedStat {
    readonly Stat: string;
    readonly Value: number;
  }
  interface SidebarBreakdownRef {
    readonly Key: string;
    readonly Source: string;
  }
  interface SkillGemView {
    readonly ID: string;
    readonly Name: string;
    readonly Level: number;
    readonly Quality: number;
    readonly Count: number;
    readonly Enabled: boolean;
    readonly Support: boolean;
    readonly Color: string;
    readonly Generated: boolean;
    readonly CountShown: boolean;
    readonly CorruptShown: boolean;
    readonly CorruptIndex: number;
    readonly Error: string;
  }
  interface SkillGroupView {
    readonly Label: string;
    readonly UserLabel: string;
    readonly GroupCount: number;
    readonly Slot: string;
    readonly Source: string;
    readonly Enabled: boolean;
    readonly IncludeInFullDPS: boolean;
    readonly Set1: boolean;
    readonly Set2: boolean;
    readonly WeaponSetLocked: boolean;
    readonly Set1Valid: boolean;
    readonly Set2Valid: boolean;
    readonly Set1Enabled: boolean;
    readonly Set2Enabled: boolean;
    readonly ForcedBoth: boolean;
    readonly Gems?: Array<exposition.SkillGemView>;
  }
  interface SkillSetsView {
    readonly Sets?: Array<calculator2.SkillSetView>;
    readonly Active: number;
  }
  interface TraceView {
    readonly Trace?: Array<number>;
    readonly Hover: boolean;
  }
  interface TreeClassRow {
    readonly Label: string;
    readonly ClassID: number;
    readonly Ascendancies?: Array<string>;
  }
  interface TreeHover {
    readonly Path?: Array<number>;
    readonly Depends?: Array<number>;
  }
  interface TreeJewel {
    readonly Socket: number;
    readonly Title: string;
    readonly BaseName: string;
    readonly Unique: boolean;
  }
  interface TreeNodeText {
    readonly ID: number;
    readonly Name: string;
    readonly Stats?: Array<string>;
  }
  interface TreeNote {
    readonly ID: number;
    readonly Text: string;
  }
  interface TreeSpecRow {
    readonly Title: string;
    readonly ClassName: string;
    readonly AscendClassName: string;
    readonly Used: number;
    readonly Sockets: number;
    readonly Version: string;
  }
  interface TreeSpecsView {
    readonly Labels?: Array<string>;
    readonly Active: number;
  }
  interface TreeState {
    readonly Alloc?: Array<number>;
    readonly AllocMode?: Array<number>;
    readonly Text?: Array<exposition.TreeNodeText>;
    readonly ClassName: string;
    readonly AscendClassBaseName: string;
    readonly Used: number;
    readonly AscUsed: number;
    readonly AttributeIndex: number;
    readonly JewelRadiusIndex?: Array<number>;
    readonly Jewels?: Array<exposition.TreeJewel>;
    readonly Notes?: Array<exposition.TreeNote>;
    readonly LevelAutoMode: boolean;
    readonly Progress: calculator2.PlayerProgress;
  }
  function CraftBases(): Promise<(Array<exposition.CraftBaseGroup> | undefined)>;
  function CraftItemPreview(rarity: number, title: string, base: string): Promise<Result<exposition.ItemView>>;
  function GemList(): Promise<Result<(Array<calculator2.GemListEntry> | undefined)>>;
  function ItemDBLists(kind: string): Promise<exposition.ItemDBInfo>;
  function ItemDBRaw(kind: string, name: string): Promise<Result<string>>;
  function ItemDBTooltip(kind: string, name: string): Promise<Result<exposition.ItemView>>;
  function ParseBuild(xml: string): Result<(exposition.Calculator | undefined)>;
  function ParseModLines(lines: Array<string> | undefined): (Array<exposition.ModLine> | undefined);
  function PassiveTreeJSON(): Promise<Result<string>>;
  function PowerStats(): (Array<exposition.PowerStatView> | undefined);
  function Preload(): Promise<Result<void>>;
  function TreeClassList(): Promise<Result<(Array<exposition.TreeClassRow> | undefined)>>;
  function TreeClasses(): Promise<Result<(Array<string> | undefined)>>;
  const TreeVersion: string;
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
    readonly Game: string;
  }
  type DirEntryType = "file" | "dir";
  const DirEntryType: {
    readonly DirEntryTypeFile: "file";
    readonly DirEntryTypeDir: "dir";
  };
  function InitializeStorage(list: (bucket: string, dir: string) => (Array<storage.DirEntry> | undefined) | PromiseLike<(Array<storage.DirEntry> | undefined)>, get: (bucket: string, key: string) => (Uint8Array | undefined) | PromiseLike<(Uint8Array | undefined)>, set: (bucket: string, key: string, value: Uint8Array | undefined) => void, exists: (bucket: string, key: string) => boolean | PromiseLike<boolean>, createFolder: (bucket: string, dir: string) => void, del: (bucket: string, path: string) => void): Promise<void>;
}
export function boot(wasm: string | URL | BufferSource): Promise<{ config: typeof config; exposition: typeof exposition; storage: typeof storage }>;
export const initializeCrystalline: () => void;