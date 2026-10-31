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
export declare namespace config {
  function InitLogging(withTime: boolean): void;
}
export declare namespace exposition {
  interface Calculator {
    Calculate(): Promise<Result<(exposition.Outputs | undefined)>>;
    Serialize(): Result<string>;
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
  }
  function ParseBuild(xml: string): Result<(exposition.Calculator | undefined)>;
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