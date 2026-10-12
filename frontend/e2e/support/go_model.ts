import { execFileSync } from 'node:child_process';
import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { DATA_HOST, cachedAsset } from './data_cache';
import { isTreeAsset } from './tree_assets';
import { reverseConfigOptions } from '../../src/lib/display/configurations';
import { autoLevel, estimatePlayerProgress } from '../../src/lib/display/points';
import { savedStats } from '../../src/lib/display/saved_stats';
import type * as Types from '../../src/lib/types';
import { modRowsEqual } from '../../src/lib/display/compare_mods';

const ROOT = resolve(import.meta.dirname, '../..');
const WASM = resolve(ROOT, 'static/go-pob.wasm');

type Mod = typeof Types;
type Build = Types.pob.PathOfBuilding;

export type ModelOutputs = {
  Output: Record<string, number>;
  OutputTable: Record<string, Record<string, number> | undefined>;
  OutputStrings: Record<string, string>;
  SkillFlags: Record<string, boolean>;
  SkillDPS: Types.calculator.SkillDPS[];
  Minion?: { Output: Record<string, number>; OutputTable: Record<string, Record<string, number> | undefined>; SkillFlags: Record<string, boolean> };
  MainSkill?: Types.calculator.MainSkillControls;
  PointsUsed: [number, number, number, number];
};

let booted: Promise<Mod> | undefined;
let loaded: Mod | undefined;

const patchFetch = () => {
  const realFetch = globalThis.fetch;
  globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
    if (url.startsWith(DATA_HOST) && !isTreeAsset(url)) {
      const asset = cachedAsset(url);
      return asset.status === 200 ? new Response(new Uint8Array(asset.body)) : new Response(null, { status: 404 });
    }
    return realFetch(input, init);
  }) as typeof fetch;
};

const boot = async (): Promise<Mod> => {
  if (!existsSync(WASM)) {
    throw new Error(`REFUSING: ${WASM} is missing — run \`mise run build-go\` (task e2e does)`);
  }

  const goroot = execFileSync('go', ['env', 'GOROOT'], { encoding: 'utf8' }).trim();
  await import(`file://${goroot}/lib/wasm/wasm_exec.js`);
  if (process.env.E2E_GO_LOG !== '1') {
    (globalThis as unknown as { fs: { writeSync: (fd: number, buf: Uint8Array) => number } }).fs.writeSync = (_fd, buf) => buf.length;
  }

  patchFetch();

  const go = new (globalThis as unknown as { Go: new () => { importObject: WebAssembly.Imports; run: (i: WebAssembly.Instance) => Promise<void> } }).Go();
  const { instance } = await WebAssembly.instantiate(readFileSync(WASM), go.importObject);
  void go.run(instance);
  while ((globalThis as unknown as { go?: Record<string, unknown> }).go?.['go-pob'] === undefined) {
    await new Promise((r) => setImmediate(r));
  }

  const mod = (await import('../../src/lib/types/index.js')) as unknown as Mod;
  mod.initializeCrystalline();
  mod.config.InitLogging(false);

  const result = await mod.raw.InitializeAll(mod.raw.GameDataVersion(), () => undefined);
  if (!result.ok) {
    throw new Error(`REFUSING: the model could not load game data: ${String(result.error)}`);
  }
  loaded = mod;
  return mod;
};

export const model = () => (booted ??= boot());

export const loadXml = async (xml: string): Promise<Build> => {
  const m = await model();
  const build = m.builds.ParseBuildStr(xml).unwrap();
  if (!build) {
    throw new Error('ParseBuildStr returned nothing');
  }
  return build;
};

export const encode = async (xml: string): Promise<string> => (await model()).pob.CompressEncode(xml).unwrap();

export const decode = async (code: string): Promise<string> => (await model()).pob.DecodeDecompress(code).unwrap();

/* Build.lua:1062-1077 */
export const withoutView = (xml: string): string =>
  xml.replace(/<TreeView\b[^>]*?>/, (t) => t.replace(/\s(?:zoomLevel|zoomX|zoomY|searchStr|showHeatMap)="[^"]*"/g, ''));

export const serialize = async (build: Build): Promise<string> => new TextDecoder().decode((await model()).builds.SerializeBuild(build).unwrap());

const plainControls = (c: Types.calculator.MainSkillControls | undefined) =>
  c && ({ ...c, Minions: c.Minions?.map((m) => ({ Label: m.Label, MinionID: m.MinionID, ItemSetID: m.ItemSetID })) } as Types.calculator.MainSkillControls);

/* Build.lua:892-896 */
export const outputs = async (build: Build): Promise<ModelOutputs> => {
  const m = await model();
  const first = await m.calculator.NewCalculator(build)!.BuildOutput('MAIN' as Types.calculator.OutputMode);
  if (first?.Player) {
    const calc = m.calculator.NewCalculator(build)!;
    const auto = { on: calc.PoB!.Build.CharacterLevelAutoMode, level: calc.PoB!.Build.Level };
    calc.release();
    const level = autoLevel(estimatePlayerProgress(first.CountAllocNodes(), first.Player.Output?.ExtraPoints), auto.on, auto.level);
    if (level !== undefined) build.SetLevel(level);
  }
  const env = await m.calculator.NewCalculator(build)!.BuildOutput('MAIN' as Types.calculator.OutputMode);
  if (!env?.Player?.MainSkill) {
    throw new Error('the model produced no main skill');
  }
  const minion = env.Minion?.Actor;
  return {
    Output: { ...(env.Player.Output as Record<string, number>) },
    OutputTable: JSON.parse(JSON.stringify(env.Player.OutputTable ?? {})) as ModelOutputs['OutputTable'],
    OutputStrings: { ...(env.Player.OutputStrings ?? {}) },
    SkillFlags: { ...(env.Player.MainSkill.SkillFlags as unknown as Record<string, boolean>) },
    SkillDPS: (env.Player.SkillDPS ?? []).map(
      (s) => ({ Name: s.Name, DPS: s.DPS, Count: s.Count, Trigger: s.Trigger, SkillPart: s.SkillPart, Source: s.Source }) as Types.calculator.SkillDPS
    ),
    Minion: minion?.MainSkill
      ? {
          Output: { ...(minion.Output as Record<string, number>) },
          OutputTable: JSON.parse(JSON.stringify(minion.OutputTable ?? {})) as ModelOutputs['OutputTable'],
          SkillFlags: { ...(minion.MainSkill.SkillFlags as unknown as Record<string, boolean>) }
        }
      : undefined,
    MainSkill: plainControls(env.MainSkillControls()),
    PointsUsed: env.CountAllocNodes()
  };
};

export const itemTooltipStats = async (build: Build): Promise<Types.calculator.ItemTooltipStats> => {
  const m = await model();
  const env = await m.calculator.NewCalculator(build)!.BuildOutput('MAIN' as Types.calculator.OutputMode);
  if (!env) {
    throw new Error('the model produced no environment');
  }
  return { ...env.ItemTooltipStats() };
};

export const NO_TIP_STATS: Types.calculator.ItemTooltipStats = {
  Str: 0,
  Dex: 0,
  Int: 0,
  Omni: 0,
  OmniscienceRequirements: false,
  OmniAttributeRequirements: 0
};

/* Build.lua:1014-1061 */
export const serializeSaved = async (build: Build): Promise<string> => {
  const m = await model();
  const s = savedStats(await outputs(build));
  const num = (v: string) => (v === 'inf' ? Infinity : v === '-inf' ? -Infinity : Number(v));
  const bytes = m.builds
    .SerializeBuildWithStats(
      build,
      s.player.map((p) => ({ Stat: p.stat, Value: num(p.value) })),
      s.fullDPS.map((f) => ({ Stat: f.stat, Value: f.value, SkillPart: f.skillPart, Source: f.source })),
      s.minion.map((p) => ({ Stat: p.stat, Value: num(p.value) }))
    )
    .unwrap();
  return new TextDecoder().decode(bytes);
};

export const warnings = async (build: Build): Promise<{ warnings: Types.calculator.SidebarWarnings; costWarning: string[] }> => {
  const m = await model();
  const env = await m.calculator.NewCalculator(build)!.BuildOutput('MAIN' as Types.calculator.OutputMode);
  if (!env) {
    throw new Error('the model produced no environment');
  }
  const w = JSON.parse(JSON.stringify(env.SidebarWarnings())) as Types.calculator.SidebarWarnings;
  return { warnings: w, costWarning: w.CostWarning ?? [] };
};

export const fresh = <T>(build: Build, read: (b: Build) => T): T => {
  if (!loaded) {
    throw new Error('fresh() before the model booted');
  }
  const c = loaded.calculator.NewCalculator(build)!;
  try {
    return read(c.PoB!);
  } finally {
    c.release();
  }
};

export const setGemField = <K extends 'Level' | 'Quality' | 'Enabled' | 'Count'>(
  build: Build,
  group: number,
  gem: number,
  field: K,
  value: Types.pob.Gem[K]
) => {
  const set = build.ActiveSkillSetIndex();
  const gems = build.Skills.SkillSets[set].Skills[group].Gems.map((g) => ({ ...g }));
  gems[gem] = { ...gems[gem], [field]: value };
  build.SetSocketGroupGems(set, group, gems);
};

export const setConfig = (build: Build, key: string, value: boolean | number | string) => {
  const v = reverseConfigOptions[key];
  let remove: boolean;
  switch (v.type) {
    case 'list':
      remove = value === v.list[(v.defaultIndex ?? 1) - 1].value;
      break;
    case 'check':
      remove = v.defaultState !== undefined ? value === v.defaultState : value === false;
      break;
    case 'text':
      remove = value === '';
      break;
    default:
      remove = value === null;
  }
  if (remove) {
    build.RemoveConfigOption(key);
    return;
  }
  build.SetConfigOption(
    typeof value === 'boolean' ? { Name: key, Boolean: value } : typeof value === 'string' ? { Name: key, String: value } : { Name: key, Number: value }
  );
};

/* Build.lua:279-284 */
export const changeAscendancy = async (build: Build, ascendancy: string) => {
  await (await model()).exposition.ChangeTreeAscendancy(build, ascendancy);
};

export const calcsOutputs = async (build: Build, elements: Record<string, Types.calculator.ColProps>) => {
  const m = await model();
  m.exposition.SetCalcTabElements({ Elements: elements });
  const calc = m.calculator.NewCalculator(build)!;
  calc.AlignCalcsSkillToMain = false;
  const env = await calc.BuildOutput('CALCS' as Types.calculator.OutputMode);
  m.exposition.SetCalcTabElements({ Elements: {} });
  if (!env?.Player?.MainSkill) throw new Error('the model produced no main skill');
  return {
    Output: { ...(env.Player.Output as Record<string, number>) },
    OutputTable: JSON.parse(JSON.stringify(env.Player.OutputTable ?? {})) as ModelOutputs['OutputTable'],
    SkillFlags: { ...(env.Player.MainSkill.SkillFlags as unknown as Record<string, boolean>) },
    Breakdown: JSON.parse(JSON.stringify(env.Player.Breakdown?.GetData() ?? {})) as Record<string, Types.calculator.EntryPlain | undefined>,
    Calcs: { ...(env.CalcProps ?? {}) }
  };
};

export const timelessSearch = async (build: Build, q: Types.calculator.TimelessSearchQuery) => {
  const m = await model();
  const misc = await m.exposition.GetMiscCalculator(build);
  const s = m.exposition.NewTimelessSearch(misc, q);
  if (!s) throw new Error('the model refused the timeless search');
  while (s.Step() < 1);
  const out = { count: s.Count(), rows: [...(s.Rows(0, 5) ?? [])], item: s.Item(0, 1, 1, '') };
  s.release();
  return out;
};

export const timelessFallback = async (build: Build, jewelType: number, stat: string) => {
  const m = await model();
  const misc = await m.exposition.GetMiscCalculator(build);
  const [list, ok] = misc!.TimelessFallbackWeights(jewelType, stat);
  misc!.release();
  if (!ok) throw new Error('the model refused the fallback weights');
  return list;
};

/* CompareTab.lua:4617-4633 */
export const compareModsEqual = async (a: Build, b: Build, elements: Record<string, Types.calculator.ColProps>, sds: Types.calculator.ModSectionData[]) => {
  const m = await model();
  m.exposition.SetCalcTabElements({ Elements: elements });
  const pass = async (build: Build) => {
    const calc = m.calculator.NewCalculator(build)!;
    calc.AlignCalcsSkillToMain = false;
    return calc.BuildOutput('CALCS' as Types.calculator.OutputMode);
  };
  const [ea, eb] = [await pass(a), await pass(b)];
  m.exposition.SetCalcTabElements({ Elements: {} });
  return new Map(sds.map((sd) => [JSON.stringify(sd), modRowsEqual([...(ea?.CompareModRows(sd, false) ?? [])], [...(eb?.CompareModRows(sd, false) ?? [])])]));
};
