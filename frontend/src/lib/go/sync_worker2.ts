import { expose } from 'comlink';
import { configure, fs } from '@zenfs/core';
import { IndexedDB } from '@zenfs/dom';
import '$lib/console_hook';
import '../../wasm_exec.js';
import { boot, config, exposition, storage, type calculator2 } from '../types2';
import type { Poe2View } from '../global';
import { defaultStateOf, relevanceQuery2, type ConfigValue2 } from '../display/configurations2';
import { DESKTOP } from '$lib/pwa/desktop';
import type { Tree2 } from '../components/skill-tree/scene2';
import { TabStorage, type StorageWrite } from '$lib/storage/tabs';
import { remountStore } from './storage_config';
import type { SavedStats2 } from '../shell/poe2_save';
import { calcSections2 } from '../calcs/calc_sections2';
import { calcsModProps2 } from '../display/calcs2';

// only the data cache bucket; on the web it is its own IndexedDB database, since zenfs allocates inode ids per
// instance and two instances writing the PoE1 worker's database overwrite each other's files
const disk = (bucket: string, key: string) => `/desktop/fs/${bucket}/${encodeURIComponent(key)}`;

// every tab's PoE2 worker shares the database, so one of them writes it, as the PoE1 workers do theirs
let tabs: TabStorage | undefined;

const cacheGet = async (bucket: string, key: string) => {
  if (DESKTOP) {
    const res = await fetch(disk(bucket, key));
    if (!res.ok) throw new Error(`read ${bucket}/${key}: ${res.status}`);
    return new Uint8Array(await res.arrayBuffer());
  }
  return new Uint8Array(await fs.promises.readFile(`/${bucket}/${key}`));
};

const cacheSet = async (bucket: string, key: string, value: Uint8Array | undefined) => {
  if (!value || !key) return;
  if (DESKTOP) {
    const res = await fetch(disk(bucket, key), { method: 'PUT', body: value });
    if (!res.ok) throw new Error(`write ${bucket}/${key}: ${res.status}`);
    return;
  }
  await tabs!.write({ kind: 'set', bucket, key, value });
};

const cacheExists = async (bucket: string, key: string) => {
  if (DESKTOP) return (await fetch(disk(bucket, key), { method: 'HEAD' })).ok;
  return fs.promises.exists(`/${bucket}/${key}`);
};

const cacheWrite = async (w: StorageWrite) => {
  if (w.kind === 'set') await fs.promises.writeFile(`/${w.bucket}/${w.key}`, w.value);
  else if (w.kind === 'clear')
    for (const n of await fs.promises.readdir(`/${w.bucket}`)) await fs.promises.rm(`/${w.bucket}/${n}`, { recursive: true, force: true });
};

const initializeStorage = async () => {
  if (!DESKTOP) {
    tabs = new TabStorage('go-pob-storage-poe2', {
      buckets: ['cache'],
      apply: cacheWrite,
      remount: () => remountStore('/cache', 'cache2'),
      lazy: { cache: 1000 }
    });
    await tabs.start();
    await configure({ mounts: { '/cache': await IndexedDB.create({ storeName: 'cache2' }) } });
  }
  await storage.InitializeStorage(
    () => [],
    cacheGet,
    cacheSet,
    cacheExists,
    () => {},
    () => {}
  );
};

export type Poe2SocketGroup = { label: string; skills: string[] };

// the spec as the tree view draws it; text holds the nodes whose name or stats the spec replaced
export type Poe2TreeState = {
  alloc: number[];
  allocMode: Record<number, number>;
  text: Record<number, { name: string; stats: string[] }>;
  className: string;
  ascendClassBaseName: string;
  used: number;
  ascUsed: number;
  // the jewelRadiusIndex of the jewel in each socket whose jewel has a radius
  jewelRadiusIndex: Record<number, number>;
};

/* tree2.ClickResult */
export const ClickResult = { Nothing: 0, Changed: 1, ChooseAttribute: 2, ConfirmClassChange: 3 } as const;

// everything the PoE2 sidebar and config page read, as plain data
export type Poe2Calc = Poe2View & {
  minion?: Record<string, number>;
  skillFlags: Record<string, boolean>;
  skillData: Record<string, boolean>;
  minionSkillFlags?: Record<string, boolean>;
  socketGroups: Poe2SocketGroup[];
  mainSocketGroup: number;
  mainActiveSkill: number;
  configInputs: Record<string, ConfigValue2>;
  relevance: calculator2.ConfigRelevance;
  tree: Poe2TreeState;
};

export type Poe2Click = { result: number; calc?: Poe2Calc };

/* SkillListControl.lua:78-96, SkillsTab.lua:1351-1361 (PoB-PoE2) */
export type Poe2SkillGroup = {
  label: string;
  slot: string;
  source: string;
  enabled: boolean;
  includeInFullDPS: boolean;
  gems: { name: string; level: number; quality: number; count: number; enabled: boolean; support: boolean; color: string }[];
};

// the Calcs page's {p:mod:n} values by cell entry id, the true and string outputs, and output.MainHand / output.OffHand
export type Poe2CalcsPage = {
  mods: Record<string, number>;
  flags: Record<string, boolean>;
  strings: Record<string, string>;
  mainHand?: Record<string, number>;
  offHand?: Record<string, number>;
};

/* ItemsTab:AddItemTooltip's title and mod line groups (PoB-PoE2) */
export type Poe2Item = { slot: string; id: number; rarity: string; name: string[]; lines: string[][] };
export type Poe2Items = { useSecondWeaponSet: boolean; items: Poe2Item[] };

export type Poe2Notes = { status: string; plain: string; doc: string };

const toValue = (v: calculator2.ConfigValue): ConfigValue2 | undefined => (v.IsNumber ? v.Number : v.IsBoolean ? v.Boolean : v.IsString ? v.String : undefined);

const fromValue = (v: ConfigValue2 | undefined): calculator2.ConfigValue => ({
  Number: typeof v === 'number' ? v : 0,
  String: typeof v === 'string' ? v : '',
  Boolean: typeof v === 'boolean' ? v : false,
  IsNumber: typeof v === 'number',
  IsString: typeof v === 'string',
  IsBoolean: typeof v === 'boolean'
});

class PoB2Worker {
  private booted?: Promise<unknown>;
  private calc?: exposition.Calculator;
  private treeData?: Promise<Tree2>;

  boot(wasm: ArrayBuffer) {
    this.booted ??= boot(wasm).then(async () => {
      config.InitLogging(false);
      await initializeStorage();
      // not awaited, so Open is not held behind it; a load error reaches Open through Calculate, which runs the same loads
      void exposition.Preload();
    });
    return this.booted;
  }

  private async view(): Promise<Poe2Calc> {
    const calc = this.calc;
    if (!calc) throw new Error('no PoE2 build is open');
    const out = (await calc.Calculate()).unwrap();
    if (!out) throw new Error('the PoE2 build did not calculate');
    const inputs: Record<string, ConfigValue2> = {};
    for (const [k, v] of Object.entries(out.ConfigInputs ?? {})) {
      const value = toValue(v);
      if (value !== undefined) inputs[k] = value;
    }
    const rel = calc.ConfigRelevance(relevanceQuery2).unwrap();
    const st = calc.TreeState().unwrap();
    if (!st) throw new Error('the PoE2 build has no passive tree');
    const allocMode: Record<number, number> = {};
    const modes = st.AllocMode ?? [];
    for (let i = 0; i + 1 < modes.length; i += 2) allocMode[modes[i]] = modes[i + 1];
    const jewelRadiusIndex: Record<number, number> = {};
    const radii = st.JewelRadiusIndex ?? [];
    for (let i = 0; i + 1 < radii.length; i += 2) jewelRadiusIndex[radii[i]] = radii[i + 1];
    return {
      className: out.ClassName,
      ascendClassName: out.AscendClassName,
      level: out.Level,
      output: { ...out.Output },
      minion: out.Minion ? { ...out.Minion } : undefined,
      skillFlags: { ...out.SkillFlags },
      skillData: { ...out.SkillData },
      minionSkillFlags: out.MinionSkillFlags ? { ...out.MinionSkillFlags } : undefined,
      socketGroups: (out.SocketGroups ?? []).map((g) => ({ label: g.Label, skills: [...(g.Skills ?? [])] })),
      mainSocketGroup: out.MainSocketGroup,
      mainActiveSkill: out.MainActiveSkill,
      configInputs: inputs,
      relevance: Object.fromEntries(Object.entries(rel).map(([k, v]) => [k, [...((v as string[] | undefined) ?? [])]])) as calculator2.ConfigRelevance,
      tree: {
        alloc: [...(st.Alloc ?? [])],
        allocMode,
        text: Object.fromEntries((st.Text ?? []).map((t) => [t.ID, { name: t.Name, stats: [...(t.Stats ?? [])] }])),
        className: st.ClassName,
        ascendClassBaseName: st.AscendClassBaseName,
        used: st.Used,
        ascUsed: st.AscUsed,
        jewelRadiusIndex
      }
    };
  }

  async Open(xml: string): Promise<Poe2Calc> {
    await this.booted;
    const calc = exposition.ParseBuild(xml).unwrap();
    if (!calc) throw new Error('the PoE2 build did not parse');
    this.calc?.release();
    this.calc = calc;
    try {
      return await this.view();
    } catch (err) {
      this.calc = undefined;
      calc.release();
      throw err;
    }
  }

  /* ConfigTab.lua:308-331, 1009-1021 (PoB-PoE2) */
  async SetConfigOption(name: string, value: ConfigValue2 | undefined): Promise<Poe2Calc> {
    this.calc?.SetConfigInput(name, fromValue(value), value === undefined || value === defaultStateOf(name, value));
    return this.view();
  }

  /* Build.lua:364-368 (PoB-PoE2) */
  async SetMainSocketGroup(index: number): Promise<Poe2Calc> {
    this.calc?.SetMainSocketGroup(index);
    return this.view();
  }

  /* Build.lua:376-381 (PoB-PoE2) */
  async SetMainActiveSkill(index: number): Promise<Poe2Calc> {
    this.calc?.SetMainActiveSkill(index);
    return this.view();
  }

  TreeData(): Promise<Tree2> {
    this.treeData ??= (async () => {
      await this.booted;
      return JSON.parse((await exposition.PassiveTreeJSON()).unwrap()) as Tree2;
    })().catch((err: unknown) => {
      this.treeData = undefined;
      throw err;
    });
    return this.treeData;
  }

  async TreeClasses(): Promise<string[]> {
    await this.booted;
    return [...((await exposition.TreeClasses()).unwrap() ?? [])];
  }

  /* PassiveTreeView.lua:356-373 (PoB-PoE2); allocMode is spec.allocMode, the weapon set Alt+wheel picked */
  TreeHover(id: number, allocMode: number): { path: number[]; depends: number[] } {
    const h = this.calc?.HoverNode(id, allocMode);
    return { path: [...(h?.Path ?? [])], depends: [...(h?.Depends ?? [])] };
  }

  /* PassiveTreeView.lua:425-594 (PoB-PoE2); hotkey is the attribute index a held S, D or I picks */
  async TreeClick(id: number, right: boolean, hotkey: number, confirmClassChange: boolean, allocMode: number): Promise<Poe2Click> {
    if (!this.calc) throw new Error('no PoE2 build is open');
    const result = this.calc.ClickNode(id, right, hotkey, confirmClassChange, allocMode).unwrap();
    return result === ClickResult.Changed ? { result, calc: await this.view() } : { result };
  }

  /* TreeTab.lua:851-876 (PoB-PoE2) */
  async TreeAllocateAttribute(id: number, attributeIndex: number, allocMode: number): Promise<Poe2Calc> {
    this.calc?.AllocateAttributeNode(id, attributeIndex, allocMode).unwrap();
    return this.view();
  }

  /* TreeTab.lua:355-364 (PoB-PoE2): Ctrl+Z / Ctrl+Y; no calc when there was nothing to undo or redo */
  async TreeUndo(redo: boolean): Promise<Poe2Calc | undefined> {
    if (!this.calc) throw new Error('no PoE2 build is open');
    const changed = (redo ? this.calc.Redo() : this.calc.Undo()).unwrap();
    return changed ? this.view() : undefined;
  }

  private calcsQuery?: exposition.CalcsModQuery;

  /* CalcFormat.lua:46-60 (PoB-PoE2): every Calcs page {p:mod:n} entry over the last calculation, and the outputs the
     Calcs page reads beside the sidebar's */
  CalcsPage(): Poe2CalcsPage {
    if (!this.calc) throw new Error('no PoE2 build is open');
    this.calcsQuery ??= { Entries: calcsModProps2(calcSections2) };
    const mods = { ...this.calc.CalcsMods(this.calcsQuery).unwrap() };
    const out = this.calc.CalcsOutputs().unwrap();
    return {
      mods,
      flags: { ...out?.Flags },
      strings: { ...out?.Strings },
      mainHand: out?.MainHand ? { ...out.MainHand } : undefined,
      offHand: out?.OffHand ? { ...out.OffHand } : undefined
    };
  }

  SkillGroups(): Poe2SkillGroup[] {
    if (!this.calc) throw new Error('no PoE2 build is open');
    return (this.calc.SkillGroups().unwrap() ?? []).map((g) => ({
      label: g.Label,
      slot: g.Slot,
      source: g.Source,
      enabled: g.Enabled,
      includeInFullDPS: g.IncludeInFullDPS,
      gems: (g.Gems ?? []).map((gem) => ({
        name: gem.Name,
        level: gem.Level,
        quality: gem.Quality,
        count: gem.Count,
        enabled: gem.Enabled,
        support: gem.Support,
        color: gem.Color
      }))
    }));
  }

  EquippedItems(): Poe2Items {
    if (!this.calc) throw new Error('no PoE2 build is open');
    const v = this.calc.EquippedItems().unwrap();
    return {
      useSecondWeaponSet: !!v?.UseSecondWeaponSet,
      items: (v?.Items ?? []).map((i) => ({
        slot: i.Slot,
        id: i.ID,
        rarity: i.Rarity,
        name: [...(i.Name ?? [])],
        lines: (i.Lines ?? []).map((g) => [...(g ?? [])])
      }))
    };
  }

  async ClearDataCache(): Promise<number> {
    await this.booted;
    const names = await fs.promises.readdir('/cache');
    await tabs!.write({ kind: 'clear', bucket: 'cache' });
    return names.length;
  }

  /* Build.lua:1180-1244 (PoB-PoE2) */
  Serialize(stats?: SavedStats2): string {
    if (!this.calc) throw new Error('no PoE2 build is open');
    if (!stats) return this.calc.Serialize().unwrap();
    return this.calc.SerializeWithStats(stats.player, stats.fullDPS, stats.minion).unwrap();
  }

  /* NotesTab.lua:75-82 (PoB-PoE2), with the Notes page's rich document */
  Notes(): Poe2Notes {
    if (!this.calc) throw new Error('no PoE2 build is open');
    return { status: this.calc.RichNotesStatus(), plain: this.calc.Notes(), doc: this.calc.RichNotesDoc() };
  }

  /* NotesTab.lua:84-88 (PoB-PoE2) */
  SetNotes(plain: string, doc: string) {
    if (!this.calc) throw new Error('no PoE2 build is open');
    this.calc.SetRichNotes(plain, doc);
  }

  /* ImportTab.lua:135-137 (PoB-PoE2) */
  ExportCode(stats: SavedStats2): string {
    if (!this.calc) throw new Error('no PoE2 build is open');
    return this.calc.ExportCode(stats.player, stats.fullDPS, stats.minion).unwrap();
  }
}

export const worker2Instance = new PoB2Worker();

expose(worker2Instance);

export type Worker2Type = PoB2Worker;
