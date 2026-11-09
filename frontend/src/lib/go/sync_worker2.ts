import { expose } from 'comlink';
import { configure, fs } from '@zenfs/core';
import { IndexedDB } from '@zenfs/dom';
import '$lib/console_hook';
import { useEngine, watchTraps } from './wasm_engine';
import type { WasmEngine } from './wasm_file';
import { boot, config, exposition, initializeCrystalline, storage, type calculator2 } from '../types2';
import type { Poe2View } from '../global';
import type { SkillDPSEntry } from '../display/stat_list';
import { defaultStateOf, relevanceQuery2, type ConfigValue2 } from '../display/configurations2';
import { DESKTOP } from '$lib/pwa/desktop';
import type { Tree2 } from '../components/skill-tree/scene2';
import { TabStorage, type StorageWrite } from '$lib/storage/tabs';
import { remountStore } from './storage_config';
import type { SavedStats2 } from '../shell/poe2_save';
import { calcSections2 } from '../calcs/calc_sections2';
import { calcsBreakdowns2, calcsModProps2, modSectionData2 } from '../display/calcs2';
import type { SidebarCell, SidebarSection as BreakdownSection, SidebarStatRef } from '../display/sidebar_breakdown';
import type { CalcSectionEntry2 } from '../calcs/calc_section_types2';
import type { calculator } from '$lib/types';
import type { CalcBreakdown } from '../display/calcs';

import type { ItemDBFilterInput, ItemDBKind, ItemDBLists, ItemDBRowPlain } from '../design/item_db';
import { addStatCompares2 } from '../display/stat_compare2';
import type { StatCompareSection } from '../skill_tree/stat_compare';
import type { ConfigSetEdit } from './engine';
import type { PartyMemberView, PartyView } from './party_view';
import type { CompareInfo } from '../shell/compare_types';
import type { SpecOp } from '../design/tree';
import { DIFF_LABELS, UNRECORDED, UndoHistory, stepLabel, unrecordedLabel, type LabelOf, type Snapshot, type UndoView } from './undo';
import { activeSpecNodes, configChangeLabel, itemTextName, skillsChangeLabel, treeChangeLabel, type StepDelta, type TreeNodeInfo } from './step_labels';
import { options2 } from '../display/configurations2';

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
    await tabs.start(async () => configure({ mounts: { '/cache': await IndexedDB.create({ storeName: 'cache2' }) } }));
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
  // the jewel in each socket that holds one (PassiveTreeView.lua:961-969, PoB-PoE2)
  jewels?: { socket: number; title: string; baseName: string; unique: boolean }[];
  // each node's author note (PassiveTreeView.lua:2086-2093, PoB-PoE2)
  notes?: Record<number, string>;
  // Build.lua:202-216, 1025-1103 (PoB-PoE2)
  levelAuto?: boolean;
  progress?: Poe2Progress;
};

export type Poe2Progress = {
  level: number;
  act: string;
  extra: number;
  normalPassives: number;
  usedMax: number;
  weaponSet1Used: number;
  weaponSet2Used: number;
  weaponSetMax: number;
  ascUsed: number;
  ascMax: number;
  warnings: string[];
};

/* tree2.ClickResult */
export const ClickResult = { Nothing: 0, Changed: 1, ChooseAttribute: 2, ConfirmClassChange: 3 } as const;

// everything the PoE2 sidebar and config page read, as plain data
export type Poe2Calc = Poe2View & {
  minion?: Record<string, number>;
  skillFlags: Record<string, boolean>;
  skillData: Record<string, boolean>;
  minionSkillFlags?: Record<string, boolean>;
  // Calcs.lua:525-531 (PoB-PoE2): output.SkillDPS, the Full DPS rows
  skillDPS?: SkillDPSEntry[];
  socketGroups: Poe2SocketGroup[];
  mainSocketGroup: number;
  mainActiveSkill: number;
  configInputs: Record<string, ConfigValue2>;
  relevance: calculator2.ConfigRelevance;
  tree: Poe2TreeState;
  // ConfigSetListControl.lua:93-99 (PoB-PoE2)
  configSets?: { sets: { id: number; title: string }[]; active: number };
  // ConfigTab.lua:24-98 (PoB-PoE2)
  customBlocks?: { Title: string; Enabled: string; Text: string }[];
  // TreeTab.lua:485-491 (PoB-PoE2): the tree list's labels and the active spec, 1-based
  treeSpecs?: { labels: string[]; active: number };
};

/* PassiveSpecListControl.lua:77-85, TreeTab.lua:67-78 (PoB-PoE2) */
export type Poe2TreeSpecRow = { title: string; className: string; ascendClassName: string; used: number; sockets: number; version: string };

/* Build.lua:1519-1540 (PoB-PoE2): a class dropdown entry and its ascendancies, by ascendClassId */
export type Poe2TreeClass = { label: string; classId: number; ascendancies: string[] };

export type Poe2Click = { result: number; calc?: Poe2Calc };

/* SkillListControl.lua:78-96, SkillsTab.lua:1351-1361 (PoB-PoE2); label is the display label, userLabel the group's own */
export type Poe2SkillGroup = {
  label: string;
  userLabel: string;
  groupCount: number;
  slot: string;
  source: string;
  enabled: boolean;
  includeInFullDPS: boolean;
  set1: boolean;
  set2: boolean;
  weaponSetLocked: boolean;
  // SkillsTab.lua:172-201 (PoB-PoE2): per weapon set, whether the group can be used there and whether its box can change
  weaponSetValid: [boolean, boolean];
  weaponSetEnabled: [boolean, boolean];
  forcedBoth: boolean;
  gems: {
    id: string;
    name: string;
    level: number;
    quality: number;
    count: number;
    enabled: boolean;
    support: boolean;
    color: string;
    // SkillsTab.lua:764-766, 1086-1100, 1134-1149 (PoB-PoE2)
    generated: boolean;
    countShown: boolean;
    corruptShown: boolean;
    corruptIndex: number;
    // SkillsTab.lua:1157-1162 (PoB-PoE2)
    error: string;
  }[];
};

/* SkillsTab.lua (PoB-PoE2): one edit of the socket group list; groups and gems count from 1 */
export type Poe2SkillsEdit =
  | { op: 'newGroup' }
  | { op: 'deleteGroup'; group: number }
  | { op: 'deleteAll' }
  | { op: 'moveGroup'; from: number; to: number }
  | { op: 'label'; group: number; label: string }
  | { op: 'enabled' | 'fullDPS'; group: number; state: boolean }
  | { op: 'groupCount'; group: number; count: number }
  | { op: 'weaponSet'; group: number; set: 1 | 2; state: boolean }
  | { op: 'gem'; group: number; gem: number; id: string }
  | { op: 'addGem'; group: number; id: string }
  | { op: 'removeGem'; group: number; gem: number }
  | { op: 'level' | 'quality' | 'count' | 'corrupt'; group: number; gem: number; value: number }
  | { op: 'gemEnabled'; group: number; gem: number; state: boolean }
  | { op: 'activeSet' | 'deleteSet'; set: number }
  | { op: 'newSet'; title: string }
  | { op: 'copySet' | 'renameSet'; set: number; title: string }
  | { op: 'paste'; text: string };

/* SkillsTab.lua:85-96, SkillSetListControl.lua:103-108 (PoB-PoE2): the skill sets in order and the active one's id */
export type Poe2SkillSets = { sets: { id: number; title: string }[]; active: number };

/* GemSelectControl.lua:104-133 (PoB-PoE2): what the gem list's filters and search read */
export type Poe2Gem = { id: string; name: string; tags: string[]; support: boolean; isLineage: boolean; legacy: boolean; levelReq: number; color: number };

// the Calcs page's {p:mod:n} values by cell entry id, the true and string outputs, and output.MainHand / output.OffHand
export type Poe2CalcsPage = {
  mods: Record<string, number>;
  flags: Record<string, boolean>;
  strings: Record<string, string>;
  mainHand?: Record<string, number>;
  offHand?: Record<string, number>;
  /* CalcBreakdownControl.lua:147-297 (PoB-PoE2) */
  breakdowns: Record<string, CalcBreakdown>;
};

/* ItemsTab:AddItemTooltip's title and mod line groups (PoB-PoE2) */
export type Poe2Item = { id: number; rarity: string; name: string[]; lines: string[][] };

/* ItemsTab.lua, ItemSlotControl.lua (PoB-PoE2): the items tab's sets, its shown slots, the build's items, the
   character rune slots and the item open in the editor */
export type Poe2ItemsPage = {
  activeSet: number;
  sets: { id: number; title: string }[];
  useSecondWeaponSet: boolean;
  slots: { name: string; label: string; parent: string; nodeId: number; itemId: number; choices: number[]; activatable: boolean; active: boolean }[];
  items: Poe2Item[];
  runeSlots: { name: string; label: string; rune: string; choices: string[] }[];
  display?: {
    id: number;
    item: Poe2Item;
    text: string;
    rarity: string;
    fits: string[];
    canHaveSockets: boolean;
    socketCount: number;
    runes: string[];
    runeChoices: { name: string; lines: string[] }[];
  };
};

/* ItemsTab.lua, ItemSlotControl.lua, ItemSetService.lua (PoB-PoE2): one edit of the build's items */
export type Poe2ItemsEdit =
  | { op: 'add'; slot?: string }
  | { op: 'delete'; id: number }
  | { op: 'equip'; slot: string; id: number }
  | { op: 'active'; slot: string; state: boolean }
  | { op: 'swap'; second: boolean }
  | { op: 'characterRune'; slot: string; rune: string }
  | { op: 'set'; id: number }
  | { op: 'newSet'; title: string }
  | { op: 'copySet' | 'renameSet'; id: number; title: string }
  | { op: 'deleteSet'; id: number }
  | { op: 'moveSet'; from: number; to: number };

/* ItemsTab.lua (PoB-PoE2): one edit of the item open in the editor */
export type Poe2DisplayEdit =
  | { op: 'paste'; text: string }
  | { op: 'edit'; id: number }
  | { op: 'text'; text: string; rarity: string }
  | { op: 'rune'; socket: number; rune: string }
  | { op: 'sockets'; count: number }
  | { op: 'close' };

/* ItemDBControl.lua (PoB-PoE2): the item database list, as the shared Browse items page shows it */
export type Poe2ItemDBKind = ItemDBKind;
export type Poe2ItemDBLists = ItemDBLists;
export type Poe2ItemDBFilter = ItemDBFilterInput;
export type Poe2ItemDBRow = ItemDBRowPlain;

/* ItemsTab.lua:4382-4538 (PoB-PoE2): the item a tooltip's stat differences are for */
export type Poe2CompareTarget =
  | { kind: 'display' }
  | { kind: 'item'; id: number }
  | { kind: 'db'; db: Poe2ItemDBKind; name: string }
  | { kind: 'craft'; rarity: number; title: string; base: string };
export type Poe2CompareBlock = { header: string; slot?: string; sections: StatCompareSection[]; calcMs: number };

export type Poe2NodeCompare = { blocks: { header: string; sections: StatCompareSection[]; calcMs: number }[]; noChange: string };
export type Poe2NodeTip = { pathLines: string[]; distanceToStart: number; granted: boolean };
export type Poe2PowerProgress = { done: number; tasks: number; total: number; finished: boolean };
/* node.power (CalcsTab.lua:551-709, PoB-PoE2); single and path are unset where PoB leaves them nil */
export type Poe2NodePower = {
  nodes: Map<number, { distance: number; single?: number; path?: number; offence: number; defence: number }>;
  /* calcsTab.powerMax: singleStat, offence, offencePerPoint, defence, defencePerPoint */
  max: number[];
};
export type Poe2PowerRow = { ID: number; Name: string; Type: string; Allocated: boolean; Power: number; PathPower: number; PathDist: number };

const powerProgress = (p: exposition.PowerProgress | undefined): Poe2PowerProgress => ({
  done: p?.Done ?? 0,
  tasks: p?.Tasks ?? 0,
  total: p?.Total ?? 0,
  finished: p?.Finished ?? true
});

/* ItemsTab.lua:2232-2246, 2368-2518 (PoB-PoE2): the editor's affix drop-downs; list[0] is "None", sel is 0-based */
export type Poe2AffixControl = {
  type: string;
  list: { label: string; modList: string[]; haveRange: boolean; tooltip: string[] }[];
  sel: number;
  sliderShown: boolean;
  sliderVal: number;
};

const item2 = (i: exposition.ItemView) => ({
  id: i.ID,
  rarity: i.Rarity,
  name: [...(i.Name ?? [])],
  lines: (i.Lines ?? []).map((g) => [...(g ?? [])])
});

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

const SPEC_SOURCES: Record<SpecOp['op'], string> = {
  new: 'NewTreeSpec',
  copy: 'CopyTreeSpec',
  rename: 'RenameTreeSpec',
  move: 'MoveTreeSpec',
  delete: 'DeleteTreeSpec'
};

// the PoE1 history's step for each Skills page edit; a gem edit is a currentGemGroup step
const SKILLS_SOURCES: Partial<Record<Poe2SkillsEdit['op'], string>> = {
  newGroup: 'socketGroups',
  deleteGroup: 'socketGroups',
  deleteAll: 'socketGroups',
  moveGroup: 'MoveSocketGroup',
  label: 'socketGroupLabel',
  paste: 'PasteSocketGroup',
  activeSet: 'SetActiveSkillSet',
  newSet: 'NewSkillSet',
  copySet: 'CopySkillSet',
  renameSet: 'RenameSkillSet',
  deleteSet: 'DeleteSkillSet'
};

// a step of the whole-build history, and the build it shows when it moved
export type Poe2Restore = { history: UndoView; calc?: Poe2Calc };

// PassiveTree.lua node types (PoB-PoE2) ranked as nodeRank ranks PoE1's, for a step's name
const configSetsOf = (calc: exposition.Calculator | undefined): { sets: { id: number; title: string }[]; active: number } => {
  const v = calc?.ConfigSets();
  return { sets: (v?.Sets ?? []).map((s) => ({ id: s.ID, title: s.Title })), active: v?.Active ?? 1 };
};

const customBlocksOf = (calc: exposition.Calculator | undefined): { Title: string; Enabled: string; Text: string }[] =>
  (calc?.CustomModifierBlocks() ?? []).map((b) => ({ Title: b.Title, Enabled: String(b.Enabled), Text: b.Text }));

const treeSpecsOf = (calc: exposition.Calculator | undefined): { labels: string[]; active: number } => {
  const v = calc?.TreeSpecs();
  return { labels: [...(v?.Labels ?? [])], active: v?.Active ?? 1 };
};

const skillGroupsOf = (calc: exposition.Calculator): Poe2SkillGroup[] =>
  (calc.SkillGroups().unwrap() ?? []).map((g) => ({
    label: g.Label,
    userLabel: g.UserLabel,
    groupCount: g.GroupCount,
    slot: g.Slot,
    source: g.Source,
    enabled: g.Enabled,
    includeInFullDPS: g.IncludeInFullDPS,
    set1: g.Set1,
    set2: g.Set2,
    weaponSetLocked: g.WeaponSetLocked,
    weaponSetValid: [g.Set1Valid, g.Set2Valid],
    weaponSetEnabled: [g.Set1Enabled, g.Set2Enabled],
    forcedBoth: g.ForcedBoth,
    gems: (g.Gems ?? []).map((gem) => ({
      id: gem.ID,
      name: gem.Name,
      level: gem.Level,
      quality: gem.Quality,
      count: gem.Count,
      enabled: gem.Enabled,
      support: gem.Support,
      color: gem.Color,
      generated: gem.Generated,
      countShown: gem.CountShown,
      corruptShown: gem.CorruptShown,
      corruptIndex: gem.CorruptIndex,
      error: gem.Error
    }))
  }));

const skillSetsOf = (calc: exposition.Calculator): Poe2SkillSets => {
  const v = calc.SkillSets();
  return { sets: (v.Sets ?? []).map((s) => ({ id: s.ID, title: s.Title })), active: v.Active };
};

/* CompareTab.lua (PoB-PoE2): what the Compare page reads of one build besides its calculation */
export type Poe2CompareParts = {
  groups: Poe2SkillGroup[];
  skillSets: Poe2SkillSets;
  items: { slot: string; name: string[]; rarity: string; lines: string[][] }[];
  itemSets: { sets: { id: number; title: string }[]; active: number };
  // the active spec's tree version
  treeVersion: string;
};

const compareParts = (calc: exposition.Calculator): Poe2CompareParts => {
  const page = calc.ItemsPage().unwrap();
  return {
    groups: skillGroupsOf(calc),
    skillSets: skillSetsOf(calc),
    items: (calc.EquippedItems().unwrap()?.Items ?? []).map((i) => ({ slot: i.Slot, ...item2(i) })),
    itemSets: { sets: (page?.Sets ?? []).map((s) => ({ id: s.ID, title: s.Title })), active: page?.ActiveItemSet ?? 1 },
    treeVersion: (calc.TreeSpecRows().unwrap() ?? [])[treeSpecsOf(calc).active - 1]?.Version ?? ''
  };
};

export type Poe2CompareSetKind = 'tree' | 'skill' | 'item' | 'config' | 'group' | 'skills';

const NODE_RANK2: Record<string, number> = { Keystone: 5, Notable: 3, Mastery: 2, Socket: 1 };

class PoB2Worker {
  private booted?: Promise<unknown>;
  private calc?: exposition.Calculator;
  private treeData?: Promise<Tree2>;
  private treeNames?: Map<number, TreeNodeInfo>;
  // the whole-build history PoE1's top bar shows; PoB-PoE2 keeps only the tree and skills tabs' own
  private history = new UndoHistory();
  // the calculation each kept state showed, for a step's stat changes
  private shown = new Map<string, Poe2Calc>();

  boot(wasm: ArrayBuffer, engine: WasmEngine = 'go', preloadFailed?: (message: string) => void) {
    this.booted ??= useEngine(engine).then(async () => {
      if (engine === 'tinygo') {
        const go = new Go();
        const { instance } = await WebAssembly.instantiate(wasm, go.importObject);
        void go.run(watchTraps(instance));
        initializeCrystalline();
      } else await boot(wasm);
      config.InitLogging(false);
      await initializeStorage();
      // not awaited, so Open is not held behind it; a load error reaches Open through Calculate, which runs the same loads
      void exposition
        .Preload()
        .then((r) => r.unwrap())
        .then(
          () => console.log('PoE2 game data preloaded'),
          (e: unknown) => preloadFailed?.(e instanceof Error ? e.message : String(e))
        );
    });
    return this.booted;
  }

  private async view(): Promise<Poe2Calc> {
    const calc = this.calc;
    if (!calc) throw new Error('no PoE2 build is open');
    return this.viewOf(calc);
  }

  private async viewOf(calc: exposition.Calculator): Promise<Poe2Calc> {
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
      skillDPS: (out.SkillDPS ?? []).map((r) => ({ ...r })),
      socketGroups: (out.SocketGroups ?? []).map((g) => ({ label: g.Label, skills: [...(g.Skills ?? [])] })),
      mainSocketGroup: out.MainSocketGroup,
      mainActiveSkill: out.MainActiveSkill,
      configInputs: inputs,
      configSets: configSetsOf(calc),
      treeSpecs: treeSpecsOf(calc),
      customBlocks: customBlocksOf(calc),
      relevance: Object.fromEntries(Object.entries(rel).map(([k, v]) => [k, [...((v as string[] | undefined) ?? [])]])) as calculator2.ConfigRelevance,
      tree: {
        alloc: [...(st.Alloc ?? [])],
        allocMode,
        text: Object.fromEntries((st.Text ?? []).map((t) => [t.ID, { name: t.Name, stats: [...(t.Stats ?? [])] }])),
        className: st.ClassName,
        ascendClassBaseName: st.AscendClassBaseName,
        used: st.Used,
        ascUsed: st.AscUsed,
        jewelRadiusIndex,
        jewels: (st.Jewels ?? []).map((j) => ({ socket: j.Socket, title: j.Title, baseName: j.BaseName, unique: j.Unique })),
        notes: Object.fromEntries((st.Notes ?? []).map((n) => [n.ID, n.Text])),
        levelAuto: st.LevelAutoMode,
        progress: {
          level: st.Progress.Level,
          act: st.Progress.Act,
          extra: st.Progress.Extra,
          normalPassives: st.Progress.NormalPassives,
          usedMax: st.Progress.UsedMax,
          weaponSet1Used: st.Progress.WeaponSet1Used,
          weaponSet2Used: st.Progress.WeaponSet2Used,
          weaponSetMax: st.Progress.WeaponSetMax,
          ascUsed: st.Progress.AscUsed,
          ascMax: st.Progress.AscMax,
          warnings: [...(st.Progress.Warnings ?? [])]
        }
      }
    };
  }

  async Open(xml: string): Promise<Poe2Calc> {
    await this.booted;
    const calc = exposition.ParseBuild(xml).unwrap();
    if (!calc) throw new Error('the PoE2 build did not parse');
    this.calc?.release();
    this.calc = calc;
    // CompareTab.lua (PoB-PoE2) is created with each build
    this.clearCompare();
    let view: Poe2Calc;
    try {
      view = await this.view();
    } catch (err) {
      this.calc = undefined;
      this.history.clear();
      calc.release();
      throw err;
    }
    this.shown.clear();
    const saved = this.saved();
    if (saved) {
      this.history.reset(saved.snap);
      this.remember(saved.xml, view);
    } else this.history.clear();
    return view;
  }

  private compareList: { calc: exposition.Calculator; name: string; seq: number; view: Poe2Calc }[] = [];
  private compareB?: (typeof this.compareList)[number];
  private compareSeq = 0;

  /* CompareTab.lua:1312-1325 (PoB-PoE2) */
  async LoadCompare(xml: string, name: string): Promise<CompareInfo> {
    await this.booted;
    const calc = exposition.ParseBuild(xml).unwrap();
    if (!calc) throw new Error('the build did not parse');
    let view: Poe2Calc;
    try {
      view = await this.viewOf(calc);
    } catch (err) {
      calc.release();
      throw err;
    }
    this.compareB = { calc, name, seq: ++this.compareSeq, view };
    this.compareList.push(this.compareB);
    return this.GetCompareInfo();
  }

  /* CompareTab.lua:229-234 (PoB-PoE2) */
  SetActiveCompare(index: number): CompareInfo {
    this.compareB = this.compareList[index] ?? this.compareB;
    return this.GetCompareInfo();
  }

  /* CompareTab.lua:1340-1351 (PoB-PoE2) */
  RemoveCompare(): CompareInfo {
    const i = this.compareB ? this.compareList.indexOf(this.compareB) : -1;
    if (i < 0) return this.GetCompareInfo();
    const [old] = this.compareList.splice(i, 1);
    this.compareB = this.compareList[Math.min(i, this.compareList.length - 1)];
    old.calc.release();
    return this.GetCompareInfo();
  }

  private clearCompare() {
    for (const e of this.compareList) e.calc.release();
    this.compareList = [];
    this.compareB = undefined;
  }

  GetCompareInfo(): CompareInfo {
    const names = this.compareList.map((e) => e.name);
    return this.compareB
      ? { name: this.compareB.name, seq: this.compareB.seq, names, active: this.compareList.indexOf(this.compareB) }
      : { name: '', seq: 0, names, active: -1 };
  }

  // the open build's parts, and the compared build's with its calculation
  CompareSides(): { a: Poe2CompareParts; b: Poe2CompareParts & { view: Poe2Calc }; bName: string; seq: number } | undefined {
    const b = this.compareB;
    if (!b || !this.calc) return undefined;
    return { a: compareParts(this.calc), b: { ...compareParts(b.calc), view: b.view }, bName: b.name, seq: b.seq };
  }

  // the compared build changes, so it calculates again and takes a new seq, which the page reads anew
  private async compareEdited(b: NonNullable<typeof this.compareB>): Promise<CompareInfo> {
    b.view = await this.viewOf(b.calc);
    b.seq = ++this.compareSeq;
    return this.GetCompareInfo();
  }

  /* CompareTab.lua:1180-1224 (PoB-PoE2): a config control of the compared build */
  async SetCompareConfigOption(name: string, value: ConfigValue2 | undefined): Promise<CompareInfo> {
    const b = this.compareB;
    if (!b) return this.GetCompareInfo();
    b.calc.SetConfigInput(name, fromValue(value), value === undefined || value === defaultStateOf(name, value));
    return this.compareEdited(b);
  }

  /* CompareTab.lua:279-389 (PoB-PoE2): the compared build's tree, skill, item and config sets and main skill, by
     index in their lists */
  async SetCompareSet(kind: Poe2CompareSetKind, index: number): Promise<CompareInfo> {
    const b = this.compareB;
    if (!b) return this.GetCompareInfo();
    const c = b.calc;
    if (kind === 'tree') c.SetActiveTreeSpec(index + 1);
    else if (kind === 'skill') c.SetActiveSkillSet(skillSetsOf(c).sets[index]?.id ?? 0).unwrap();
    else if (kind === 'item') c.SetActiveItemSet(compareParts(c).itemSets.sets[index]?.id ?? 0).unwrap();
    else if (kind === 'config') c.SetActiveConfigSet(configSetsOf(c).sets[index]?.id ?? 1);
    else if (kind === 'group') c.SetMainSocketGroup(index + 1);
    else c.SetMainActiveSkill(index + 1);
    return this.compareEdited(b);
  }

  /* CompareTab.lua:1180-1224 (PoB-PoE2): one option of the open build takes the compared build's value */
  async UseCompareConfig(name: string): Promise<Poe2Calc> {
    const b = this.compareB;
    if (!b || !this.calc) return this.view();
    const value = b.view.configInputs[name];
    this.calc.SetConfigInput(name, fromValue(value), value === undefined || value === defaultStateOf(name, value));
    return this.changed('UseCompareConfig', `Used ${(options2[name]?.label ?? name).replace(/:$/, '')} from ${b.name}`);
  }

  /* CompareTab.lua:1298-1309 (PoB-PoE2) */
  async CopyCompareConfig(): Promise<Poe2Calc> {
    const b = this.compareB;
    if (!b || !this.calc) return this.view();
    for (const [k, v] of Object.entries(b.view.configInputs)) this.calc.SetConfigInput(k, fromValue(v), false);
    return this.changed('CopyCompareConfig', `Copied the config of ${b.name}`);
  }

  /* CompareTab.lua:1386-1423 (PoB-PoE2) */
  async CopyCompareSpec(andUse: boolean): Promise<Poe2Calc> {
    const b = this.compareB;
    if (!b || !this.calc) return this.view();
    this.calc.CopyCompareSpec(b.calc, andUse).unwrap();
    return this.changed('CopyCompareSpec', `Copied the passive tree of ${b.name}`);
  }

  /* CompareTab.lua:1494-1514 (PoB-PoE2) */
  async CopyCompareItem(slot: string, andUse: boolean): Promise<Poe2Calc> {
    const b = this.compareB;
    if (!b || !this.calc) return this.view();
    this.calc.CopyCompareItem(b.calc, slot, andUse).unwrap();
    return this.changed('CopyCompareItem', `Copied the ${slot} item of ${b.name}`);
  }

  private saved(): { xml: string; snap: Snapshot } | undefined {
    const xml = this.calc?.Serialize().unwrap();
    return xml ? { xml, snap: this.history.table.encode(xml) } : undefined;
  }

  private remember(xml: string, view: Poe2Calc) {
    this.shown.delete(xml);
    this.shown.set(xml, view);
    for (const old of this.shown.keys()) {
      if (this.shown.size <= this.history.max + 2) break;
      this.shown.delete(old);
    }
  }

  // an edit's recalculation, then its step in the history, named as PoE1's history names the same edit
  private async changed(source: string, action?: string | ((view: Poe2Calc) => string)): Promise<Poe2Calc> {
    const view = await this.view();
    const saved = this.saved();
    if (saved && this.history.open) {
      const text = typeof action === 'function' ? action(view) : action;
      this.history.record(source, (before) => this.describe(source, text, before, saved.xml, view), saved.snap, Date.now());
      this.remember(saved.xml, view);
    }
    return view;
  }

  private describe(source: string, action: string | undefined, before: Snapshot, after: string, view: Poe2Calc): LabelOf {
    const was = this.history.table.decode(before);
    const diff = DIFF_LABELS[source];
    const text =
      action ??
      (diff === 'tree'
        ? treeChangeLabel(activeSpecNodes(was), activeSpecNodes(after), (id) => this.treeNames?.get(id))
        : diff === 'skills'
          ? skillsChangeLabel(was, after)
          : undefined);
    const label = stepLabel(source);
    return { ...label, action: text ?? label.action, delta: this.delta(this.shown.get(was), view) };
  }

  private delta(before: Poe2Calc | undefined, after: Poe2Calc): StepDelta[] | undefined {
    if (!before) return undefined;
    return addStatCompares2(
      after.skillFlags,
      after.minionSkillFlags ?? {},
      !!after.minion,
      { output: before.output, minion: before.minion },
      { output: after.output, minion: after.minion }
    ).flatMap((s) => s.lines.map(({ label, value, better }) => ({ label, value, better, section: s.actor ?? 'Player' })));
  }

  // a change no recalculation recorded (the notes, a node's note) becomes a step of its own before the history moves
  private recordUnrecorded(): boolean {
    const before = this.history.current();
    const now = this.saved();
    if (!before || !now) return false;
    return this.history.record(UNRECORDED, unrecordedLabel(this.history.table.decode(before), now.xml), now.snap, Date.now(), false);
  }

  private async restore(step: () => Snapshot | undefined): Promise<Poe2Restore> {
    const snap = step();
    if (!snap) return { history: this.history.view() };
    const calc = exposition.ParseBuild(this.history.table.decode(snap)).unwrap();
    if (!calc) throw new Error('undo: failed parsing a recorded state');
    this.calc?.release();
    this.calc = calc;
    return { history: this.history.view(), calc: await this.view() };
  }

  History(): UndoView {
    return this.history.view();
  }

  async Undo(): Promise<Poe2Restore> {
    if (!this.calc || !this.history.open) return { history: this.history.view() };
    this.recordUnrecorded();
    return this.restore(() => this.history.undo());
  }

  async Redo(): Promise<Poe2Restore> {
    if (!this.calc || !this.history.open || this.recordUnrecorded()) return { history: this.history.view() };
    return this.restore(() => this.history.redo());
  }

  async UndoTo(i: number): Promise<Poe2Restore> {
    if (!this.calc || !this.history.open || this.recordUnrecorded()) return { history: this.history.view() };
    return this.restore(() => this.history.goTo(i));
  }

  /* ConfigTab.lua:308-331, 1009-1021 (PoB-PoE2) */
  async SetConfigOption(name: string, value: ConfigValue2 | undefined): Promise<Poe2Calc> {
    const remove = value === undefined || value === defaultStateOf(name, value);
    this.calc?.SetConfigInput(name, fromValue(value), remove);
    const opt = options2[name] as (typeof options2)[string] | undefined;
    const v = opt && { label: opt.label, type: opt.type, list: opt.list?.map((o) => ({ label: o.label, value: o.val })) };
    return this.changed(remove ? 'SetConfigOption: remove' : 'SetConfigOption: change', configChangeLabel(name, value ?? '', remove, v));
  }

  /* ConfigTab.lua:152-178, ConfigSetListControl.lua:93-99 (PoB-PoE2) */
  ConfigSets(): { sets: { id: number; title: string }[]; active: number } {
    return configSetsOf(this.calc);
  }

  /* ConfigTab.lua:24-98, 1200-1223 (PoB-PoE2): the active config set's custom modifier blocks */
  CustomModifierBlocks(): { Title: string; Enabled: string; Text: string }[] {
    return customBlocksOf(this.calc);
  }

  async SetCustomModifierBlocks(blocks: { Title?: string; Enabled?: string; Text: string }[]): Promise<Poe2Calc> {
    this.calc?.SetCustomModifierBlocks(blocks.map((b) => ({ Title: b.Title ?? 'Default', Enabled: b.Enabled !== 'false', Text: b.Text })));
    return this.changed('SetCustomModifierBlocks', `Edited custom modifiers (${blocks.length} block${blocks.length === 1 ? '' : 's'})`);
  }

  ParseModLines(
    lines: string[]
  ): { Status: exposition.ModLineStatus; Extra: string; Mods?: { Name: string; Type: string; Value: string; Tags?: string[] }[] }[] {
    return (exposition.ParseModLines(lines) ?? []).map((l) => ({
      Status: l.Status,
      Extra: l.Extra,
      Mods: l.Mods?.map((m) => ({ Name: m.Name, Type: m.Type, Value: m.Value, Tags: m.Tags && [...m.Tags] }))
    }));
  }

  /* ConfigTab.lua:167-170 (PoB-PoE2) */
  async SelectConfigSet(id: number): Promise<Poe2Calc> {
    this.calc?.SetActiveConfigSet(id);
    return this.changed('SetActiveConfigSet', `Switched config set → ${this.ConfigSets().sets.find((s) => s.id === id)?.title || 'Default'}`);
  }

  /* ConfigSetService.lua:17-46, ListControl.lua:441-449 (PoB-PoE2) */
  async EditConfigSet(e: ConfigSetEdit): Promise<Poe2Calc> {
    const calc = this.calc;
    if (!calc) throw new Error('no PoE2 build is open');
    if (e.op === 'new') calc.NewConfigSet(e.title);
    else if (e.op === 'delete') calc.DeleteConfigSet(e.id);
    else if (e.op === 'move') calc.MoveConfigSet(e.from, e.to);
    else if (e.op === 'copy') calc.CopyConfigSet(e.id, e.title);
    else calc.RenameConfigSet(e.id, e.title);
    return this.changed(`EditConfigSets: ${e.op}`);
  }

  /* Build.lua:364-368 (PoB-PoE2) */
  async SetMainSocketGroup(index: number): Promise<Poe2Calc> {
    this.calc?.SetMainSocketGroup(index);
    return this.changed('SetMainSocketGroup', (v) => `Set the main skill to ${v.socketGroups[index - 1]?.label || `socket group ${index}`}`);
  }

  /* Build.lua:376-381 (PoB-PoE2) */
  async SetMainActiveSkill(index: number): Promise<Poe2Calc> {
    this.calc?.SetMainActiveSkill(index);
    return this.changed('SetMainActiveSkill');
  }

  /* Build.lua:209-216 (PoB-PoE2): the level box, which turns the automatic level off */
  async SetLevel(level: number): Promise<Poe2Calc> {
    this.calc?.SetLevel(level);
    return this.changed('SetLevel', `Set the character level to ${level}`);
  }

  /* Build.lua:202-208 (PoB-PoE2) */
  async SetLevelAutoMode(auto: boolean): Promise<Poe2Calc> {
    this.calc?.SetCharacterLevelAutoMode(auto);
    return this.changed('SetCharacterLevelAutoMode', auto ? 'Switched the character level to automatic' : 'Switched the character level to manual');
  }

  TreeData(): Promise<Tree2> {
    this.treeData ??= (async () => {
      await this.booted;
      const tree = JSON.parse((await exposition.PassiveTreeJSON()).unwrap()) as Tree2;
      this.treeNames = new Map(
        tree.nodes.filter((n) => n.name).map((n) => [n.id, { name: n.name, rank: (n.ascendancy && n.type === 'Notable' ? 1 : 0) + (NODE_RANK2[n.type] ?? 0) }])
      );
      return tree;
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

  /* PassiveTreeView.lua:425-594 (PoB-PoE2); hotkey is the attribute index a held S, D or I picks; trace is the
     Shift-traced path */
  async TreeClick(id: number, right: boolean, hotkey: number, confirmClassChange: boolean, allocMode: number, trace?: number[]): Promise<Poe2Click> {
    if (!this.calc) throw new Error('no PoE2 build is open');
    const result = this.calc.ClickNode(id, right, hotkey, confirmClassChange, allocMode, trace ?? []).unwrap();
    return result === ClickResult.Changed ? { result, calc: await this.changed('AllocateNode') } : { result };
  }

  /* PassiveTreeView.lua:520-528 (PoB-PoE2): the Class Change popup's Connect Path for a click on another class's
     ascendancy node */
  async TreeClickConnect(id: number, hotkey: number, allocMode: number): Promise<Poe2Click> {
    if (!this.calc) throw new Error('no PoE2 build is open');
    const result = this.calc.ClickNodeConnectPath(id, hotkey, allocMode).unwrap();
    return result === ClickResult.Changed ? { result, calc: await this.changed('AllocateNode') } : { result };
  }

  /* TreeTab.lua:851-876 (PoB-PoE2) */
  async TreeAllocateAttribute(id: number, attributeIndex: number, allocMode: number): Promise<Poe2Calc> {
    this.calc?.AllocateAttributeNode(id, attributeIndex, allocMode).unwrap();
    return this.changed('AllocateNode');
  }

  /* TreeTab.lua:145-161 (PoB-PoE2) */
  async TreeReset(): Promise<Poe2Calc> {
    this.calc?.ResetTree().unwrap();
    return this.changed('ResetTreeNodes', `Reset passive tree ${this.activeSpecTitle()}`);
  }

  /* TreeTab.lua:355-364 (PoB-PoE2): Ctrl+Z / Ctrl+Y; no calc when there was nothing to undo or redo */
  async TreeUndo(redo: boolean): Promise<Poe2Calc | undefined> {
    if (!this.calc) throw new Error('no PoE2 build is open');
    const changed = (redo ? this.calc.Redo() : this.calc.Undo()).unwrap();
    return changed ? this.changed('AllocateNode') : undefined;
  }

  /* TreeTab.lua:485-491 (PoB-PoE2) */
  TreeSpecs(): { labels: string[]; active: number } {
    return treeSpecsOf(this.calc);
  }

  private activeSpecTitle() {
    const v = this.TreeSpecs();
    return v.labels[v.active - 1] || 'Default';
  }

  /* TreeTab.lua:444, 581-586 (PoB-PoE2): the compared spec's allocated nodes, specID 1-based */
  TreeSpecAlloc(specID: number): number[] {
    return [...(this.calc?.TreeSpecAlloc(specID).unwrap() ?? [])];
  }

  /* PassiveSpecListControl.lua:77-85, TreeTab.lua:67-78 (PoB-PoE2) */
  TreeSpecRows(): Poe2TreeSpecRow[] {
    return (this.calc?.TreeSpecRows().unwrap() ?? []).map((r) => ({
      title: r.Title,
      className: r.ClassName,
      ascendClassName: r.AscendClassName,
      used: r.Used,
      sockets: r.Sockets,
      version: r.Version
    }));
  }

  /* TreeTab.lua:55-62, 365-376 (PoB-PoE2); index is 1-based */
  async SetActiveTreeSpec(index: number): Promise<Poe2Calc> {
    this.calc?.SetActiveTreeSpec(index);
    return this.changed('SetActiveTreeSpec', `Switched passive tree → ${this.activeSpecTitle()}`);
  }

  /* PassiveSpecListControl.lua:16-122, ListControl.lua:440-448 (PoB-PoE2); the op's indices are 0-based */
  async EditTreeSpecs(op: SpecOp): Promise<Poe2Calc> {
    const calc = this.calc;
    if (!calc) throw new Error('no PoE2 build is open');
    if (op.op === 'new') calc.NewTreeSpec(op.title).unwrap();
    else if (op.op === 'copy') calc.CopyTreeSpec(op.index + 1, op.title).unwrap();
    else if (op.op === 'rename') calc.RenameTreeSpec(op.index + 1, op.title);
    else if (op.op === 'move') calc.MoveTreeSpec(op.from + 1, op.to + 1);
    else calc.DeleteTreeSpec(op.index + 1);
    return this.changed(SPEC_SOURCES[op.op]);
  }

  /* PassiveSpec.lua:161-171 (PoB-PoE2) */
  TreeNodeNote(nodeId: number): string {
    return this.calc?.TreeNodeNote(nodeId) ?? '';
  }

  /* PassiveTreeView.lua:556-564, Main.lua:1818-1823 (PoB-PoE2): an empty note removes it; the note changes no stat */
  SetTreeNodeNote(nodeId: number, text: string) {
    this.calc?.SetTreeNodeNote(nodeId, text);
  }

  /* Build.lua:1519-1540 (PoB-PoE2) */
  async TreeClassList(): Promise<Poe2TreeClass[]> {
    await this.booted;
    return ((await exposition.TreeClassList()).unwrap() ?? []).map((c) => ({ label: c.Label, classId: c.ClassID, ascendancies: [...(c.Ascendancies ?? [])] }));
  }

  /* Build.lua:246-272 (PoB-PoE2); choice 0 asks first, 1 is Continue, 2 is Connect Path */
  async ChangeTreeClass(classId: number, choice: number): Promise<{ ask: boolean; calc?: Poe2Calc }> {
    if (!this.calc) throw new Error('no PoE2 build is open');
    const ask = this.calc.ChangeTreeClass(classId, choice).unwrap() ?? false;
    return ask ? { ask } : { ask, calc: await this.changed('ChangeTreeClass', (v) => `Changed class to ${v.className}`) };
  }

  /* Build.lua:273-278 (PoB-PoE2) */
  async ChangeTreeAscendancy(ascendClassId: number): Promise<Poe2Calc> {
    this.calc?.ChangeTreeAscendancy(ascendClassId).unwrap();
    return this.changed('ChangeTreeAscendancy', (v) => `Changed ascendancy to ${v.ascendClassName}`);
  }

  /* SkillsTab.lua:594-599 (PoB-PoE2): Ctrl+Z / Ctrl+Y on the Skills tab; no calc when there was nothing to undo or redo */
  async SkillsUndo(redo: boolean): Promise<Poe2Calc | undefined> {
    if (!this.calc) throw new Error('no PoE2 build is open');
    return this.calc.SkillsUndo(redo).unwrap() ? this.changed('socketGroups') : undefined;
  }

  private calcsQuery?: exposition.CalcsModQuery;

  /* CalcFormat.lua:46-60 (PoB-PoE2): every Calcs page {p:mod:n} entry over the last calculation, and the outputs the
     Calcs page reads beside the sidebar's */
  CalcsPage(): Poe2CalcsPage {
    if (!this.calc) throw new Error('no PoE2 build is open');
    return this.calcsPageOf(this.calc);
  }

  private calcsPageOf(calc: exposition.Calculator): Poe2CalcsPage {
    this.calcsQuery ??= { Entries: calcsModProps2(calcSections2) };
    const mods = { ...calc.CalcsMods(this.calcsQuery).unwrap() };
    const out = calc.CalcsOutputs().unwrap();
    return {
      mods,
      flags: { ...out?.Flags },
      strings: { ...out?.Strings },
      mainHand: out?.MainHand ? { ...out.MainHand } : undefined,
      offHand: out?.OffHand ? { ...out.OffHand } : undefined,
      breakdowns: calcsBreakdowns2(calc.CalcsBreakdowns().unwrap() ?? {})
    };
  }

  /* CompareTab.lua:4583-4652 (PoB-PoE2): both builds' Calcs page values */
  CompareCalcsPages(): { a: Poe2CalcsPage; b: Poe2CalcsPage } | undefined {
    const b = this.compareB;
    if (!b || !this.calc) return undefined;
    return { a: this.calcsPageOf(this.calc), b: this.calcsPageOf(b.calc) };
  }

  /* CalcBreakdownControl.lua:300-548 (PoB-PoE2): a cell entry's mod table over the last calculation */
  CalcsModSection(sd: calculator2.ModSectionData): calculator2.ModSectionRow[] {
    if (!this.calc) throw new Error('no PoE2 build is open');
    return [...(this.calc.CalcsModSection(sd).unwrap() ?? [])];
  }

  /* CalcBreakdownControl.lua:293-296 (PoB-PoE2): the SkillBuffs or SkillDebuffs list's mod table over the last calculation */
  CalcsBreakdownModList(name: string): calculator2.ModSectionRow[] {
    if (!this.calc) throw new Error('no PoE2 build is open');
    return [...(this.calc.CalcsBreakdownModList(name, false).unwrap() ?? [])];
  }

  /* Build.lua:2334-2370 (PoB-PoE2): the breakdown a sidebar stat's line names, undefined for none */
  SidebarBreakdownKey(ref: SidebarStatRef): { key: string; source: string } | undefined {
    if (!this.calc) throw new Error('no PoE2 build is open');
    const got = this.calc.SidebarBreakdownKey(ref.stat, ref.childStat ?? '', ref.breakdown ?? '', ref.breakdown !== undefined, !!ref.modNames).unwrap();
    return got?.Key ? { key: got.Key, source: got.Source } : undefined;
  }

  /* CalcBreakdownControl.lua:60-93 (PoB-PoE2): the sections a sidebar breakdown's cells show */
  SidebarBreakdownSections(cells: SidebarCell[]): BreakdownSection[] {
    if (!this.calc) throw new Error('no PoE2 build is open');
    const bds = calcsBreakdowns2(this.calc.CalcsBreakdowns().unwrap() ?? {});
    return cells.flatMap((cell): BreakdownSection[] => {
      if (cell.breakdown !== undefined) {
        const bd = (cell.gearOnly && bds[`${cell.breakdown}#gearOnly`]) || bds[cell.breakdown];
        return bd ? [{ label: cell.label, bd }] : [];
      }
      if (!cell.modName) return [];
      const rows = this.calc!.CalcsModSection(modSectionData2(cell as CalcSectionEntry2)).unwrap();
      return rows?.length ? [{ label: cell.label, mods: [...rows] as unknown as calculator.ModSectionRow[] }] : [];
    });
  }

  SkillGroups(): Poe2SkillGroup[] {
    if (!this.calc) throw new Error('no PoE2 build is open');
    return skillGroupsOf(this.calc);
  }

  SkillSets(): Poe2SkillSets {
    if (!this.calc) throw new Error('no PoE2 build is open');
    return skillSetsOf(this.calc);
  }

  /* SkillsTab.lua:639-657 (PoB-PoE2): the text Ctrl+C copies for a group; "" for a group a source creates */
  CopySocketGroup(group: number): string {
    if (!this.calc) throw new Error('no PoE2 build is open');
    return this.calc.CopySocketGroup(group).unwrap() ?? '';
  }

  /* GemSelectControl.lua:278-348 (PoB-PoE2): per gem, whether it supports the group and whether the sort calculates it */
  GemSortCache(group: number, gemIds: string[]): { canSupport: boolean[]; pending: boolean[] } {
    if (!this.calc) throw new Error('no PoE2 build is open');
    const v = this.calc.GemSortCache(group, gemIds).unwrap();
    return { canSupport: [...(v?.CanSupport ?? [])], pending: [...(v?.Pending ?? [])] };
  }

  /* GemSelectControl.lua:61-102, 323-331, 405-409 (PoB-PoE2): each gem's DPS in the group's new gem slot */
  async GemSortDPS(group: number, gemIds: string[], field: string): Promise<{ base: number; dps: number[] }> {
    if (!this.calc) throw new Error('no PoE2 build is open');
    const v = (await this.calc.GemSortDPS(group, gemIds, field)).unwrap();
    return { base: v?.Base ?? 0, dps: [...(v?.DPS ?? [])] };
  }

  /* SkillsTab.lua, SkillListControl.lua (PoB-PoE2): an edit of the socket group list, then the recalculation */
  async EditSkills(e: Poe2SkillsEdit): Promise<Poe2Calc> {
    const c = this.calc;
    if (!c) throw new Error('no PoE2 build is open');
    switch (e.op) {
      case 'newGroup':
        c.NewSocketGroup().unwrap();
        break;
      case 'deleteGroup':
        c.DeleteSocketGroup(e.group).unwrap();
        break;
      case 'deleteAll':
        c.DeleteAllSocketGroups().unwrap();
        break;
      case 'moveGroup':
        c.MoveSocketGroup(e.from, e.to).unwrap();
        break;
      case 'label':
        c.SetSocketGroupLabel(e.group, e.label).unwrap();
        break;
      case 'enabled':
        c.SetSocketGroupEnabled(e.group, e.state).unwrap();
        break;
      case 'fullDPS':
        c.SetSocketGroupIncludeInFullDPS(e.group, e.state).unwrap();
        break;
      case 'groupCount':
        c.SetSocketGroupCount(e.group, e.count).unwrap();
        break;
      case 'weaponSet':
        c.SetSocketGroupWeaponSet(e.group, e.set, e.state).unwrap();
        break;
      case 'gem':
        c.SetGem(e.group, e.gem, e.id).unwrap();
        break;
      case 'addGem':
        c.AddGem(e.group, e.id).unwrap();
        break;
      case 'removeGem':
        c.RemoveGem(e.group, e.gem).unwrap();
        break;
      case 'level':
        c.SetGemLevel(e.group, e.gem, e.value).unwrap();
        break;
      case 'quality':
        c.SetGemQuality(e.group, e.gem, e.value).unwrap();
        break;
      case 'count':
        c.SetGemCount(e.group, e.gem, e.value).unwrap();
        break;
      case 'corrupt':
        c.SetGemCorruption(e.group, e.gem, e.value).unwrap();
        break;
      case 'gemEnabled':
        c.SetGemEnabled(e.group, e.gem, e.state).unwrap();
        break;
      case 'activeSet':
        c.SetActiveSkillSet(e.set).unwrap();
        break;
      case 'newSet':
        c.NewSkillSet(e.title).unwrap();
        break;
      case 'copySet':
        c.CopySkillSet(e.set, e.title).unwrap();
        break;
      case 'renameSet':
        c.RenameSkillSet(e.set, e.title).unwrap();
        break;
      case 'deleteSet':
        c.DeleteSkillSet(e.set).unwrap();
        break;
      case 'paste':
        // SkillsTab.lua:753-760 (PoB-PoE2): a paste with no gem adds nothing
        if (!c.PasteSocketGroup(e.text).unwrap()) return this.view();
        break;
    }
    c.AddSkillsUndoState();
    return this.changed(SKILLS_SOURCES[e.op] ?? 'currentGemGroup');
  }

  private gemList?: Promise<Poe2Gem[]>;

  GemList(): Promise<Poe2Gem[]> {
    this.gemList ??= (async () => {
      await this.booted;
      return ((await exposition.GemList()).unwrap() ?? []).map((g) => ({
        id: g.ID,
        name: g.Name,
        tags: [...(g.Tags ?? [])],
        support: g.Support,
        isLineage: g.IsLineage,
        legacy: g.Legacy,
        levelReq: g.LevelRequirement,
        color: g.Color
      }));
    })().catch((err: unknown) => {
      this.gemList = undefined;
      throw err;
    });
    return this.gemList;
  }

  /* SkillsTab.lua:405-427 (PoB-PoE2) */
  GemOptions(): calculator2.GemOptions {
    if (!this.calc) throw new Error('no PoE2 build is open');
    return { ...this.calc.GemOptions() };
  }

  /* SkillsTab.lua:114-140 (PoB-PoE2) */
  SetGemOptions(options: calculator2.GemOptions) {
    this.calc?.SetGemOptions(options);
  }

  ItemsPage(): Poe2ItemsPage {
    if (!this.calc) throw new Error('no PoE2 build is open');
    const v = this.calc.ItemsPage().unwrap();
    if (!v) throw new Error('the PoE2 build has not calculated');
    const d = v.Display;
    return {
      activeSet: v.ActiveItemSet,
      sets: (v.Sets ?? []).map((s) => ({ id: s.ID, title: s.Title })),
      useSecondWeaponSet: v.UseSecondWeaponSet,
      slots: (v.Slots ?? []).map((s) => ({
        name: s.Name,
        label: s.Label,
        parent: s.Parent,
        nodeId: s.NodeID,
        itemId: s.ItemID,
        choices: [...(s.Choices ?? [])],
        activatable: s.Activatable,
        active: s.Active
      })),
      items: (v.Items ?? []).map(item2),
      runeSlots: (v.RuneSlots ?? []).map((r) => ({ name: r.Name, label: r.Label, rune: r.Rune, choices: [...(r.Choices ?? [])] })),
      display: d && {
        id: d.ID,
        item: item2(d.Item),
        text: d.Text,
        rarity: d.Rarity,
        fits: [...(d.Fits ?? [])],
        canHaveSockets: d.CanHaveSockets,
        socketCount: d.SocketCount,
        runes: [...(d.Runes ?? [])],
        runeChoices: (d.RuneChoices ?? []).map((r) => ({ name: r.Name, lines: [...(r.Lines ?? [])] }))
      }
    };
  }

  /* ItemsTab.lua, ItemSlotControl.lua, ItemSetService.lua (PoB-PoE2): an edit of the build's items, then the
     recalculation */
  async EditItems(e: Poe2ItemsEdit): Promise<Poe2Calc> {
    const c = this.calc;
    if (!c) throw new Error('no PoE2 build is open');
    const [source, action] = this.itemsStep(e);
    switch (e.op) {
      case 'add':
        c.AddDisplayItem(e.slot ?? '').unwrap();
        break;
      case 'delete':
        c.DeleteItem(e.id).unwrap();
        break;
      case 'equip':
        c.SetSlotItem(e.slot, e.id).unwrap();
        break;
      case 'active':
        c.SetSlotActive(e.slot, e.state).unwrap();
        break;
      case 'swap':
        c.SetUseSecondWeaponSet(e.second).unwrap();
        break;
      case 'characterRune':
        c.SetCharacterRune(e.slot, e.rune).unwrap();
        break;
      case 'set':
        c.SetActiveItemSet(e.id).unwrap();
        break;
      case 'newSet':
        c.NewItemSet(e.title).unwrap();
        break;
      case 'copySet':
        c.CopyItemSet(e.id, e.title).unwrap();
        break;
      case 'renameSet':
        c.RenameItemSet(e.id, e.title).unwrap();
        break;
      case 'deleteSet':
        c.DeleteItemSet(e.id).unwrap();
        break;
      case 'moveSet':
        c.MoveItemSet(e.from, e.to).unwrap();
        break;
    }
    return this.changed(source, action);
  }

  private itemName(id: number) {
    const text = id ? this.calc?.ItemText(id).unwrap() : undefined;
    return text ? itemTextName(text) : 'an item';
  }

  // the step an items edit makes, named before the edit changes what it names
  private itemsStep(e: Poe2ItemsEdit): [string, string?] {
    switch (e.op) {
      case 'add':
        return ['SaveItem'];
      case 'delete':
        return ['DeleteItem', `Deleted ${this.itemName(e.id)}`];
      case 'equip':
        return ['EquipItemID', e.id ? `Equipped ${this.itemName(e.id)} in ${e.slot}` : `Emptied ${e.slot}`];
      case 'active':
        return ['SetFlaskActive', `${e.state ? 'Activated' : 'Deactivated'} ${e.slot}`];
      case 'swap':
        return ['SetWeaponSet', `Switched to weapon set ${e.second ? 2 : 1}`];
      case 'characterRune':
        return ['SetCharacterRune'];
      case 'set':
        return ['SetActiveItemSet'];
      case 'newSet':
        return ['NewItemSet'];
      case 'copySet':
        return ['CopyItemSet'];
      case 'renameSet':
        return ['RenameItemSet'];
      case 'deleteSet':
        return ['DeleteItemSet'];
      case 'moveSet':
        return ['MoveItemSet'];
    }
  }

  /* ItemsTab.lua (PoB-PoE2): an edit of the editor's item; the build is unchanged, so there is no recalculation */
  EditDisplayItem(e: Poe2DisplayEdit) {
    const c = this.calc;
    if (!c) throw new Error('no PoE2 build is open');
    switch (e.op) {
      case 'paste':
        c.PasteItem(e.text).unwrap();
        break;
      case 'edit':
        c.EditItem(e.id).unwrap();
        break;
      case 'text':
        c.EditDisplayItemText(e.text, e.rarity).unwrap();
        break;
      case 'rune':
        c.SetDisplayItemRune(e.socket, e.rune).unwrap();
        break;
      case 'sockets':
        c.SetDisplayItemSocketCount(e.count).unwrap();
        break;
      case 'close':
        c.CloseDisplayItem().unwrap();
        break;
    }
  }

  /* ItemListControl.lua:332-335 (PoB-PoE2) */
  ItemText(id: number): string {
    if (!this.calc) throw new Error('no PoE2 build is open');
    return this.calc.ItemText(id).unwrap();
  }

  /* ItemsTab.lua:2838-2846 (PoB-PoE2): the Craft Item popup's types and their bases */
  async CraftBases(): Promise<{ type: string; bases: string[] }[]> {
    return ((await exposition.CraftBases()) ?? []).map((g) => ({ type: g.Type, bases: [...(g.Bases ?? [])] }));
  }

  /* ItemsTab.lua:2847-2852 (PoB-PoE2): the item Create would make; rarity is 1-based */
  async CraftItemPreview(rarity: number, title: string, base: string): Promise<Poe2Item> {
    return item2((await exposition.CraftItemPreview(rarity, title, base)).unwrap());
  }

  /* ItemsTab.lua:2853-2863 (PoB-PoE2): Create opens the item in the editor */
  CraftDisplayItem(rarity: number, title: string, base: string) {
    if (!this.calc) throw new Error('no PoE2 build is open');
    this.calc.CraftDisplayItem(rarity, title, base).unwrap();
  }

  /* ItemDBControl.lua:25-92 (PoB-PoE2): the item database list's dropdowns */
  async ItemDBLists(kind: Poe2ItemDBKind): Promise<Poe2ItemDBLists> {
    const l = await exposition.ItemDBLists(kind);
    const list = (v: readonly string[] | undefined) => [...(v ?? [])];
    return {
      slots: list(l.Slots),
      types: list(l.Types),
      leagues: list(l.Leagues),
      requirements: list(l.Requirements),
      obtainable: list(l.Obtainable),
      searchModes: list(l.SearchModes),
      sortStats: []
    };
  }

  /* ItemDBControl.lua:94-298 (PoB-PoE2): the rows the filters keep, by name */
  ItemDBRows(kind: Poe2ItemDBKind, f: Poe2ItemDBFilter): Poe2ItemDBRow[] {
    if (!this.calc) throw new Error('no PoE2 build is open');
    const rows = this.calc
      .ItemDBRows(kind, {
        Slot: f.slot,
        Type: f.type,
        League: f.league,
        Requirement: f.requirement,
        Obtainable: f.obtainable,
        Search: f.search,
        SearchMode: f.searchMode
      })
      .unwrap();
    return (rows ?? []).map((r) => ({
      name: r.Name,
      title: r.Title,
      base: r.BaseName,
      rarity: r.Rarity,
      type: r.Type,
      slot: r.PrimarySlot,
      level: r.Level,
      league: r.League,
      source: r.Source,
      power: 0
    }));
  }

  /* ItemDBControl.lua:333-341 (PoB-PoE2) */
  async ItemDBTooltip(kind: Poe2ItemDBKind, name: string): Promise<Poe2Item> {
    return item2((await exposition.ItemDBTooltip(kind, name)).unwrap());
  }

  /* ItemDBControl.lua:384-386 (PoB-PoE2) */
  async ItemDBRaw(kind: Poe2ItemDBKind, name: string): Promise<string> {
    return (await exposition.ItemDBRaw(kind, name)).unwrap();
  }

  /* ItemDBControl.lua:374-381 (PoB-PoE2): a double click opens the item in the editor */
  OpenDBItem(kind: Poe2ItemDBKind, name: string) {
    if (!this.calc) throw new Error('no PoE2 build is open');
    this.calc.OpenDBItem(kind, name).unwrap();
  }

  /* ItemDBControl.lua:348-373 (PoB-PoE2): Ctrl+click adds the item and equips it, then the recalculation */
  async AddDBItem(kind: Poe2ItemDBKind, name: string, equip: boolean, second: boolean): Promise<Poe2Calc> {
    if (!this.calc) throw new Error('no PoE2 build is open');
    this.calc.AddDBItem(kind, name, equip, second).unwrap();
    return this.changed('AddDBItem', equip ? `Equipped ${name}` : `Added ${name}`);
  }

  /* ItemsTab.lua:4382-4538, Build.lua:2585-2640 (PoB-PoE2): an item tooltip's stat differences, one block per slot it
     fits (or its flask or charm toggle), measured against the calculator's base pass */
  // async: a compare can be the first calculation to need a minion table, which loads over fetch
  async ItemCompare(t: Poe2CompareTarget): Promise<Poe2CompareBlock[]> {
    const c = this.calc;
    if (!c) throw new Error('no PoE2 build is open');
    const start = performance.now();
    const v = (
      await (t.kind === 'display'
        ? c.DisplayItemCompare()
        : t.kind === 'item'
          ? c.BuildItemCompare(t.id)
          : t.kind === 'db'
            ? c.DBItemCompare(t.db, t.name)
            : c.CraftItemCompare(t.rarity, t.title, t.base))
    ).unwrap();
    if (!v) return [];
    const calcMs = performance.now() - start;
    const outputs = (o: exposition.CompareOutputs) => ({ output: { ...(o.Output ?? {}) }, minion: o.Minion ? { ...o.Minion } : undefined });
    const base = outputs(v.Base);
    return (v.Blocks ?? []).map((b) => ({
      header: b.Header,
      slot: b.Slot || undefined,
      sections: addStatCompares2({ ...(v.SkillFlags ?? {}) }, { ...(v.MinionSkillFlags ?? {}) }, v.Minion, base, outputs(b.Compare)),
      calcMs
    }));
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

  /* PartyTab.lua:516-573, 897-980 (PoB-PoE2): the text boxes, their summaries and the import options */
  GetParty(): PartyView | undefined {
    const c = this.calc;
    if (!c) return undefined;
    const summaries = c.PartySummaries() ?? {};
    const [destination, append, advanced] = c.PartyOptions();
    return {
      buffers: { ...(c.PartyBuffers() ?? {}) },
      summaries: Object.fromEntries(Object.entries(summaries).map(([k, v]) => [k, (v ?? []).map((e) => ({ group: e.Group, name: e.Name, value: e.Value }))])),
      destination,
      append,
      advanced,
      applied: c.PartyEffectsApplied(),
      disabled: c.PartyEffectsDisabled()
    };
  }

  GetPartyMembers(): PartyMemberView[] {
    return (this.calc?.PartyMembers() ?? []).map((m) => ({
      name: m.Name,
      className: m.ClassName,
      ascendancy: m.Ascendancy,
      level: m.Level,
      active: m.Active,
      gives: [...(m.Gives ?? [])]
    }));
  }

  async GetPartyImpact(stats: string[]): Promise<{ stat: string; with: number; without: number }[]> {
    const c = this.calc;
    if (!c) return [];
    return ((await c.PartyImpact(stats)).unwrap() ?? []).map((r) => ({ stat: r.Stat, with: r.With, without: r.Without }));
  }

  private partyWrite(source: string, action: string, write: (c: exposition.Calculator) => void): Promise<Poe2Calc> {
    if (!this.calc) throw new Error('no PoE2 build is open');
    write(this.calc);
    return this.changed(source, action);
  }

  SetPartyBuffer(name: string, text: string): Promise<Poe2Calc> {
    return this.partyWrite('SetPartyBuffer', `Edited the party ${name}`, (c) => c.SetPartyBuffer(name, text));
  }

  SetPartyOptions(destination: string, append: boolean, advanced: boolean): Promise<Poe2Calc> {
    return this.partyWrite('SetPartyOptions', 'Changed the party options', (c) => c.SetPartyOptions(destination, append, advanced));
  }

  /* PartyTab.lua:162-285 (PoB-PoE2): xml is the member's decoded build code */
  async ImportPartyMember(xml: string, destination: string, append: boolean): Promise<Poe2Calc> {
    let err: Error | undefined;
    const view = await this.partyWrite('ImportParty', `Imported a party member into ${destination}`, (c) => {
      try {
        c.AddPartyMember(xml, destination, append).unwrap();
      } catch (e) {
        err = e as Error;
      }
    });
    if (err) throw err;
    return view;
  }

  ClearParty(destination: string): Promise<Poe2Calc> {
    return this.partyWrite('ClearParty', `Cleared the party ${destination}`, (c) => c.ClearParty(destination));
  }

  DisablePartyEffects(): Promise<Poe2Calc> {
    return this.partyWrite('DisablePartyEffects', 'Disabled the party effects', (c) => c.DisablePartyEffects());
  }

  RebuildParty(): Promise<Poe2Calc> {
    return this.partyWrite('RebuildParty', 'Rebuilt the party', (c) => c.RebuildParty());
  }

  SetPartyMemberActive(i: number, active: boolean): Promise<Poe2Calc> {
    const name = this.GetPartyMembers()[i]?.name ?? String(i + 1);
    return this.partyWrite('SetPartyMemberActive', `${active ? 'Enabled' : 'Disabled'} party member ${name}`, (c) => c.SetPartyMemberActive(i, active));
  }

  RemovePartyMember(i: number): Promise<Poe2Calc> {
    const name = this.GetPartyMembers()[i]?.name ?? String(i + 1);
    return this.partyWrite('RemovePartyMember', `Removed party member ${name}`, (c) => c.RemovePartyMember(i));
  }

  /* ImportTab.lua:135-137 (PoB-PoE2) */
  ExportCode(stats: SavedStats2): string {
    if (!this.calc) throw new Error('no PoE2 build is open');
    return this.calc.ExportCode(stats.player, stats.fullDPS, stats.minion).unwrap();
  }

  /* ImportTab.lua:494-531 (PoB-PoE2): the build planner export's Tree, Skill and Item dropdowns */
  BuildPlannerSets(): exposition.BuildPlannerSets {
    if (!this.calc) throw new Error('no PoE2 build is open');
    return this.calc.BuildPlannerSets();
  }

  /* BuildExportPoE2.lua:286-313 (PoB-PoE2): one tree, skill set and item set as a .build file */
  BuildPlannerExport(meta: exposition.BuildPlannerMeta, specIndex: number, skillSetId: number, itemSetId: number): exposition.BuildPlannerFile {
    if (!this.calc) throw new Error('no PoE2 build is open');
    return this.calc.BuildPlannerExport(meta, specIndex, skillSetId, itemSetId).unwrap();
  }

  /* ItemsTab.lua:2232-2246 (PoB-PoE2): the editor's affix drop-downs, prefixes first */
  DisplayItemAffixes(): Poe2AffixControl[] {
    if (!this.calc) throw new Error('no PoE2 build is open');
    return (this.calc.DisplayItemAffixes().unwrap() ?? []).map((c) => ({
      type: c.Type,
      list: (c.List ?? []).map((e) => ({ label: e.Label, modList: [...(e.ModList ?? [])], haveRange: e.HaveRange, tooltip: [...(e.Tooltip ?? [])] })),
      sel: c.SelIndex - 1,
      sliderShown: c.SliderShown,
      sliderVal: c.SliderVal
    }));
  }

  /* ItemsTab.lua:932-947, 1067-1075 (PoB-PoE2): affix drop-down #control (0-based) picks entry sel at the slider's
     value, or its slider moves to slider when sel is already picked */
  EditDisplayItemAffix(control: number, sel: number, slider: number) {
    const c = this.calc;
    if (!c) throw new Error('no PoE2 build is open');
    const cur = c.DisplayItemAffixes().unwrap()?.[control];
    if (cur && sel > 0 && sel === cur.SelIndex - 1 && cur.SliderShown) c.SetDisplayItemAffixRange(control + 1, slider).unwrap();
    else c.SelectDisplayItemAffix(control + 1, sel + 1, slider).unwrap();
  }

  /* ItemsTab.lua:678-694, 2564-2577 (PoB-PoE2): the editor's modifier range lines and its quality, when it shows one */
  DisplayItemRanges(): { lines: { label: string; range: number }[]; quality?: number } {
    if (!this.calc) throw new Error('no PoE2 build is open');
    const [lines, quality] = this.calc.DisplayItemRanges().unwrap();
    return { lines: (lines ?? []).map((l) => ({ label: l.Label, range: l.Range })), quality: quality < 0 ? undefined : quality };
  }

  /* ItemsTab.lua:687-691, 1147-1166 (PoB-PoE2): range line #line (1-based) moves to value, or the quality is set */
  EditDisplayItemRange(e: { op: 'range'; line: number; value: number } | { op: 'quality'; value: number }) {
    const c = this.calc;
    if (!c) throw new Error('no PoE2 build is open');
    if (e.op === 'range') c.SetDisplayItemRangeLine(e.line, e.value).unwrap();
    else c.SetDisplayItemQuality(e.value).unwrap();
  }

  /* ItemsTab.lua:1079-1104 (PoB-PoE2): affix drop-down #control's (0-based) roll slider tooltip */
  DisplayItemAffixRoll(control: number, slider: number): string[] {
    if (!this.calc) throw new Error('no PoE2 build is open');
    return [...(this.calc.DisplayItemAffixRoll(control + 1, slider).unwrap() ?? [])];
  }

  /* PassiveTreeView.lua:1967-2008, Build.lua:2585-2640 (PoB-PoE2): a node tooltip's stat differences against the
     calculator's base pass; trace is the Shift-traced path while tracing */
  // async: a compare can be the first calculation to need a minion table, which loads over fetch
  async TreeNodeCompare(id: number, trace?: number[]): Promise<Poe2NodeCompare> {
    const c = this.calc;
    if (!c) throw new Error('no PoE2 build is open');
    const start = performance.now();
    const v = (await c.NodeCompare(id, trace ?? [], trace !== undefined)).unwrap();
    if (!v) throw new Error('no node compare');
    const calcMs = performance.now() - start;
    const outputs = (o: exposition.CompareOutputs) => ({ output: { ...(o.Output ?? {}) }, minion: o.Minion ? { ...o.Minion } : undefined });
    const base = outputs(v.Base);
    return {
      blocks: (v.Blocks ?? []).map((b) => ({
        header: b.Header,
        sections: addStatCompares2(
          { ...(v.SkillFlags ?? {}) },
          { ...(v.MinionSkillFlags ?? {}) },
          v.Minion,
          base,
          outputs(b.Compare),
          b.PathLength || undefined
        ),
        calcMs
      })),
      noChange: v.NoChange
    };
  }

  /* PassiveTreeView.lua:1712-1714, 2011-2053 (PoB-PoE2): a node tooltip's pathing lines and a socket's distance */
  TreeNodeTip(id: number, trace?: number[]): Poe2NodeTip {
    const v = this.calc?.NodeTip(id, trace ?? [], trace !== undefined).unwrap();
    return { pathLines: [...(v?.PathLines ?? [])], distanceToStart: v?.DistanceToStart ?? 0, granted: v?.Granted ?? false };
  }

  /* PassiveTreeView.lua:321-350 (PoB-PoE2): one hover while Shift is held; an empty trace starts one */
  TreeTraceNode(trace: number[], id: number): { trace: number[]; hover: boolean } {
    const v = this.calc?.TraceNode(trace, id).unwrap();
    return { trace: [...(v?.Trace ?? trace)], hover: v?.Hover ?? false };
  }

  /* TreeTab.lua:260-265 (PoB-PoE2): the heat map's stats; stat "" is Offence/Defence */
  async TreePowerStats(): Promise<{ Stat: string; Label: string }[]> {
    await this.booted;
    return (exposition.PowerStats() ?? []).map((s) => ({ Stat: s.Stat, Label: s.Label }));
  }

  /* TreeTab.lua:924-930 (PoB-PoE2): starts the power builder over the last calculation, maxDepth 0 for All */
  async TreePowerStart(stat: string, maxDepth: number): Promise<Poe2PowerProgress> {
    if (!this.calc) throw new Error('no PoE2 build is open');
    return powerProgress((await this.calc.StartNodePower(stat, maxDepth)).unwrap());
  }

  /* CalcsTab.lua:617-709 (PoB-PoE2): runs the power builder for about budgetMs, a node at a time, so the worker
     answers other requests between steps; the node powers come with the progress when asked or when it finishes. A
     recalculation stops it (the call then throws). */
  async TreePowerStep(budgetMs: number, withPower: boolean): Promise<{ progress: Poe2PowerProgress; power?: Poe2NodePower }> {
    const c = this.calc;
    if (!c) throw new Error('no PoE2 build is open');
    const start = performance.now();
    let p: exposition.PowerProgress | undefined;
    do p = (await c.StepNodePower(1)).unwrap();
    while (p && !p.Finished && performance.now() - start < budgetMs);
    const progress = powerProgress(p);
    if (!withPower && !progress.finished) return { progress };
    const v = c.NodePower().unwrap();
    const power: Poe2NodePower = { nodes: new Map(), max: [...(v?.Max ?? [0, 0, 0, 0, 0])] };
    (v?.IDs ?? []).forEach((id, i) =>
      power.nodes.set(id, {
        distance: v!.Distance![i],
        single: v!.HasSingle![i] ? v!.Single![i] : undefined,
        path: v!.HasPath![i] ? v!.Path![i] : undefined,
        offence: v!.Offence![i],
        defence: v!.Defence![i]
      })
    );
    return { progress, power };
  }

  /* TreeTab.lua:932-1048 (PoB-PoE2): the power report's rows, before the stat's display scale and format */
  TreePowerReport(): Poe2PowerRow[] {
    return (this.calc?.NodePowerReport().unwrap() ?? []).map((r) => ({ ...r }));
  }
}

export const worker2Instance = new PoB2Worker();

type Serial<T> = { [K in keyof T]: T[K] extends (...args: infer A) => infer R ? (...args: A) => Promise<Awaited<R>> : T[K] };

// each call starts once the one before it settles: a Go call answered by a promise can wait on a fetch part way
// through, and a call run meanwhile would read or edit the build it has half changed, or wait on the same table load
// and stop the Go runtime
const serial = <T extends object>(target: T): Serial<T> => {
  let last: Promise<unknown> = Promise.resolve();
  return new Proxy(target, {
    get: (t, prop) => {
      const v = Reflect.get(t, prop) as unknown;
      if (typeof v !== 'function') return v;
      return (...args: unknown[]) => {
        const run = last.then(() => (v as (...a: unknown[]) => unknown).apply(t, args));
        last = run.catch(() => undefined);
        return run;
      };
    }
  }) as unknown as Serial<T>;
};

export const worker2 = serial(worker2Instance);

expose(worker2);

export type Worker2Type = PoB2Worker;
