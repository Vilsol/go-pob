import { expose } from 'comlink';
import { configure, fs } from '@zenfs/core';
import { IndexedDB } from '@zenfs/dom';
import '$lib/console_hook';
import '../../wasm_exec.js';
import { boot, config, exposition, storage, type calculator2 } from '../types2';
import type { Poe2View } from '../global';
import { defaultStateOf, relevanceQuery2, type ConfigValue2 } from '../display/configurations2';
import { DESKTOP } from '$lib/pwa/desktop';

// only the data cache bucket; on the web it is its own IndexedDB database, since zenfs allocates inode ids per
// instance and two instances writing the PoE1 worker's database overwrite each other's files
const disk = (bucket: string, key: string) => `/desktop/fs/${bucket}/${encodeURIComponent(key)}`;

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
  await fs.promises.writeFile(`/${bucket}/${key}`, value);
};

const cacheExists = async (bucket: string, key: string) => {
  if (DESKTOP) return (await fetch(disk(bucket, key), { method: 'HEAD' })).ok;
  return fs.promises.exists(`/${bucket}/${key}`);
};

const initializeStorage = async () => {
  if (!DESKTOP) await configure({ mounts: { '/cache': await IndexedDB.create({ storeName: 'cache2' }) } });
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
};

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
      relevance: Object.fromEntries(Object.entries(rel).map(([k, v]) => [k, [...((v as string[] | undefined) ?? [])]])) as calculator2.ConfigRelevance
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

  async ClearDataCache(): Promise<number> {
    await this.booted;
    const names = await fs.promises.readdir('/cache');
    for (const n of names) await fs.promises.rm(`/cache/${n}`, { recursive: true, force: true });
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
