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
  interface Calculator {
    AllocateAttributeNode(id: number, attributeIndex: number): Result<void>;
    Calculate(): Promise<Result<(exposition.Outputs | undefined)>>;
    ClickNode(id: number, right: boolean, hotkey: number, confirmClassChange: boolean): Result<number>;
    ConfigRelevance(query: calculator2.ConfigRelevance): Result<calculator2.ConfigRelevance>;
    HoverNode(id: number): exposition.TreeHover;
    Serialize(): Result<string>;
    SetConfigInput(name: string, value: calculator2.ConfigValue, isDefault: boolean): void;
    SetMainActiveSkill(index: number): void;
    SetMainSocketGroup(index: number): void;
    TreeState(): Result<(exposition.TreeState | undefined)>;
    /** Releases the Go resources behind this wrapper. */
    release(): void;
    [Symbol.dispose](): void;
  }
  interface Outputs {
    readonly Output?: Record<string, number>;
    readonly Minion?: Record<string, number>;
    readonly ClassName: string;
    readonly AscendClassName: string;
    readonly Level: number;
    readonly SkillFlags?: Record<string, boolean>;
    readonly MinionSkillFlags?: Record<string, boolean>;
    readonly SocketGroups?: Array<calculator2.SocketGroupView>;
    readonly MainSocketGroup: number;
    readonly MainActiveSkill: number;
    readonly ConfigInputs?: Record<string, Readonly<calculator2.ConfigValue>>;
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
  }
  function ParseBuild(xml: string): Result<(exposition.Calculator | undefined)>;
  function PassiveTreeJSON(): Promise<Result<string>>;
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