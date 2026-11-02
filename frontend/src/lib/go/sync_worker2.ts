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
  | { op: 'removeGem'; group: number; gem: number }
  | { op: 'level' | 'quality' | 'count' | 'corrupt'; group: number; gem: number; value: number }
  | { op: 'gemEnabled'; group: number; gem: number; state: boolean };

/* GemSelectControl.lua:104-133 (PoB-PoE2): what the gem list's filters and search read */
export type Poe2Gem = { id: string; name: string; tags: string[]; support: boolean; isLineage: boolean; legacy: boolean; levelReq: number; color: number };

// the Calcs page's {p:mod:n} values by cell entry id, the true and string outputs, and output.MainHand / output.OffHand
export type Poe2CalcsPage = {
  mods: Record<string, number>;
  flags: Record<string, boolean>;
  strings: Record<string, string>;
  mainHand?: Record<string, number>;
  offHand?: Record<string, number>;
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

class PoB2Worker {
  private booted?: Promise<unknown>;
  private calc?: exposition.Calculator;
  private treeData?: Promise<Tree2>;

  boot(wasm: ArrayBuffer) {
    this.booted ??= boot(wasm).then(async () => {
      config.InitLogging(false);
      await initializeStorage();
      // not awaited, so Open is not held behind it; a load error reaches Open through Calculate, which runs the same loads
      void exposition.Preload().then(() => console.log('PoE2 game data preloaded'));
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
      userLabel: g.UserLabel,
      groupCount: g.GroupCount,
      slot: g.Slot,
      source: g.Source,
      enabled: g.Enabled,
      includeInFullDPS: g.IncludeInFullDPS,
      set1: g.Set1,
      set2: g.Set2,
      weaponSetLocked: g.WeaponSetLocked,
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
        corruptIndex: gem.CorruptIndex
      }))
    }));
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
    }
    return this.view();
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
    return this.view();
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
