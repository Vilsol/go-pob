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
  interface SocketGroupView {
    readonly Label: string;
    readonly Skills?: Array<string>;
  }
}
export declare namespace config {
  function InitLogging(withTime: boolean): void;
}
export declare namespace exposition {
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
    AddDisplayItem(slot: string): Result<number>;
    AllocateAttributeNode(id: number, attributeIndex: number, allocMode: number): Result<void>;
    CalcsMods(query: exposition.CalcsModQuery): Result<(Record<string, number> | undefined)>;
    CalcsOutputs(): Result<(exposition.CalcsOutputs | undefined)>;
    Calculate(): Promise<Result<(exposition.Outputs | undefined)>>;
    ClickNode(id: number, right: boolean, hotkey: number, confirmClassChange: boolean, allocMode: number): Result<number>;
    CloseDisplayItem(): Result<void>;
    ConfigRelevance(query: calculator2.ConfigRelevance): Result<calculator2.ConfigRelevance>;
    CopyItemSet(id: number, title: string): Result<void>;
    DeleteAllSocketGroups(): Result<void>;
    DeleteItem(id: number): Result<void>;
    DeleteItemSet(id: number): Result<void>;
    DeleteSocketGroup(index: number): Result<void>;
    EditDisplayItemText(text: string, rarity: string): Result<void>;
    EditItem(id: number): Result<void>;
    EquippedItems(): Result<(exposition.ItemsView | undefined)>;
    ExportCode(player: Array<exposition.SavedStat> | undefined, fullDPS: Array<exposition.SavedFullDPS> | undefined, minion: Array<exposition.SavedStat> | undefined): Result<string>;
    GemOptions(): calculator2.GemOptions;
    HoverNode(id: number, allocMode: number): exposition.TreeHover;
    ItemText(id: number): Result<string>;
    ItemsPage(): Result<(exposition.ItemsPageView | undefined)>;
    MoveItemSet(from: number, to: number): Result<void>;
    MoveSocketGroup(from: number, to: number): Result<void>;
    NewItemSet(title: string): Result<void>;
    NewSocketGroup(): Result<number>;
    Notes(): string;
    PasteItem(raw: string): Result<void>;
    Redo(): Result<boolean>;
    RemoveGem(group: number, gem: number): Result<void>;
    RenameItemSet(id: number, title: string): Result<void>;
    RichNotesDoc(): string;
    RichNotesStatus(): string;
    Serialize(): Result<string>;
    SerializeWithStats(player: Array<exposition.SavedStat> | undefined, fullDPS: Array<exposition.SavedFullDPS> | undefined, minion: Array<exposition.SavedStat> | undefined): Result<string>;
    SetActiveItemSet(id: number): Result<void>;
    SetCharacterRune(slot: string, runeName: string): Result<void>;
    SetConfigInput(name: string, value: calculator2.ConfigValue, isDefault: boolean): void;
    SetDisplayItemRune(socket: number, runeName: string): Result<void>;
    SetDisplayItemSocketCount(count: number): Result<void>;
    SetGem(group: number, gem: number, gemID: string): Result<void>;
    SetGemCorruption(group: number, gem: number, option: number): Result<void>;
    SetGemCount(group: number, gem: number, count: number): Result<void>;
    SetGemEnabled(group: number, gem: number, state: boolean): Result<void>;
    SetGemLevel(group: number, gem: number, level: number): Result<void>;
    SetGemOptions(options: calculator2.GemOptions): void;
    SetGemQuality(group: number, gem: number, quality: number): Result<void>;
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
    SetUseSecondWeaponSet(second: boolean): Result<void>;
    SkillGroups(): Result<(Array<exposition.SkillGroupView> | undefined)>;
    TreeState(): Result<(exposition.TreeState | undefined)>;
    Undo(): Result<boolean>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
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
    readonly Gems?: Array<exposition.SkillGemView>;
  }
  interface TreeHover {
    readonly Path?: Array<number>;
    readonly Depends?: Array<number>;
  }
  interface TreeNodeText {
    readonly ID: number;
    readonly Name: string;
    readonly Stats?: Array<string>;
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
  }
  function GemList(): Promise<Result<(Array<calculator2.GemListEntry> | undefined)>>;
  function ParseBuild(xml: string): Result<(exposition.Calculator | undefined)>;
  function PassiveTreeJSON(): Promise<Result<string>>;
  function Preload(): Promise<Result<void>>;
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