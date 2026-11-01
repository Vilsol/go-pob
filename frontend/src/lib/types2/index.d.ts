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
    AllocateAttributeNode(id: number, attributeIndex: number, allocMode: number): Result<void>;
    CalcsMods(query: exposition.CalcsModQuery): Result<(Record<string, number> | undefined)>;
    CalcsOutputs(): Result<(exposition.CalcsOutputs | undefined)>;
    Calculate(): Promise<Result<(exposition.Outputs | undefined)>>;
    ClickNode(id: number, right: boolean, hotkey: number, confirmClassChange: boolean, allocMode: number): Result<number>;
    ConfigRelevance(query: calculator2.ConfigRelevance): Result<calculator2.ConfigRelevance>;
    EquippedItems(): Result<(exposition.ItemsView | undefined)>;
    HoverNode(id: number, allocMode: number): exposition.TreeHover;
    Redo(): Result<boolean>;
    Serialize(): Result<string>;
    SerializeWithStats(player: Array<exposition.SavedStat> | undefined, fullDPS: Array<exposition.SavedFullDPS> | undefined, minion: Array<exposition.SavedStat> | undefined): Result<string>;
    SetConfigInput(name: string, value: calculator2.ConfigValue, isDefault: boolean): void;
    SetMainActiveSkill(index: number): void;
    SetMainSocketGroup(index: number): void;
    SkillGroups(): Result<(Array<exposition.SkillGroupView> | undefined)>;
    TreeState(): Result<(exposition.TreeState | undefined)>;
    Undo(): Result<boolean>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface ItemView {
    readonly Slot: string;
    readonly ID: number;
    readonly Name?: Array<string>;
    readonly Rarity: string;
    readonly Lines?: Array<Array<string> | undefined>;
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
    readonly Name: string;
    readonly Level: number;
    readonly Quality: number;
    readonly Count: number;
    readonly Enabled: boolean;
    readonly Support: boolean;
    readonly Color: string;
  }
  interface SkillGroupView {
    readonly Label: string;
    readonly Slot: string;
    readonly Source: string;
    readonly Enabled: boolean;
    readonly IncludeInFullDPS: boolean;
    readonly Set1: boolean;
    readonly Set2: boolean;
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