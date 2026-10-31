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
};

/* tree2.ClickResult */
export const ClickResult = { Nothing: 0, Changed: 1, ChooseAttribute: 2, ConfirmClassChange: 3 } as const;

// everything the PoE2 sidebar and config page read, as plain data
export type Poe2Calc = Poe2View & {
  minion?: Record<string, number>;
  skillFlags: Record<string, boolean>;
  minionSkillFlags?: Record<string, boolean>;
  socketGroups: Poe2SocketGroup[];
  mainSocketGroup: number;
  mainActiveSkill: number;
  configInputs: Record<string, ConfigValue2>;
  relevance: calculator2.ConfigRelevance;
  tree: Poe2TreeState;
};

export type Poe2Click = { result: number; calc?: Poe2Calc };

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
    return {
      className: out.ClassName,
      ascendClassName: out.AscendClassName,
      level: out.Level,
      output: { ...out.Output },
      minion: out.Minion ? { ...out.Minion } : undefined,
      skillFlags: { ...out.SkillFlags },
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
        ascUsed: st.AscUsed
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

  /* PassiveTreeView.lua:356-373 (PoB-PoE2) */
  TreeHover(id: number): { path: number[]; depends: number[] } {
    const h = this.calc?.HoverNode(id);
    return { path: [...(h?.Path ?? [])], depends: [...(h?.Depends ?? [])] };
  }

  /* PassiveTreeView.lua:425-594 (PoB-PoE2); hotkey is the attribute index a held S, D or I picks */
  async TreeClick(id: number, right: boolean, hotkey: number, confirmClassChange: boolean): Promise<Poe2Click> {
    if (!this.calc) throw new Error('no PoE2 build is open');
    const result = this.calc.ClickNode(id, right, hotkey, confirmClassChange).unwrap();
    return result === ClickResult.Changed ? { result, calc: await this.view() } : { result };
  }

  /* TreeTab.lua:851-876 (PoB-PoE2) */
  async TreeAllocateAttribute(id: number, attributeIndex: number): Promise<Poe2Calc> {
    this.calc?.AllocateAttributeNode(id, attributeIndex).unwrap();
    return this.view();
  }

  async ClearDataCache(): Promise<number> {
    await this.booted;
    const names = await fs.promises.readdir('/cache');
    await tabs!.write({ kind: 'clear', bucket: 'cache' });
    return names.length;
  }

  Serialize(): string {
    if (!this.calc) throw new Error('no PoE2 build is open');
    return this.calc.Serialize().unwrap();
  }
}

export const worker2Instance = new PoB2Worker();

expose(worker2Instance);

export type Worker2Type = PoB2Worker;
