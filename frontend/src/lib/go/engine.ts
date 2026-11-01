import { derived, get, type Readable } from 'svelte/store';
import { currentBuild, poe2Build } from '$lib/global';
import type { Sidebar } from '$lib/display/sidebar';
import type { StatLine } from '$lib/display/stat_list';
import type { SidebarSection } from '$lib/display/sidebar_breakdown';
import type { SpectreRow } from '$lib/design/spectres';
import type { MainSkillField } from '$lib/shell/main_skill';
import type { calculator, exposition, pob } from '$lib/types';
import type { CalcsView } from '$lib/display/calcs';
import type { ConfigValue } from '$lib/display/configurations';
import type { ConfigTable, Inputs } from '$lib/design/config';
import type { ItemTip } from '$lib/design/fixtures';
import type { StatCompareSection } from '$lib/skill_tree/stat_compare';
import type { Site } from '$lib/design/share';
import type { UndoView } from './undo';
import { poe1Engine } from './engine_poe1';
import { poe2Engine } from './engine_poe2.svelte';

export { runtime } from './engine_poe1';

export type Game = 'poe1' | 'poe2';

// the sidebar's socket group picker; made while a component initialises, since it follows the build for as long as
// that component lives
export type MainGroupView = { readonly groups: string[]; readonly index: number; select(index: number): void };

type Async<T> = Promise<T> | undefined;
type Tooltip = { Text: string; Separator: boolean }[];

export type ConfigSetEdit =
  | { op: 'new'; title: string }
  | { op: 'copy' | 'rename'; id: number; title: string }
  | { op: 'delete'; id: number }
  | { op: 'move'; from: number; to: number };

// ConfigTab.lua: what the Config page reads and writes; a part a game has none of is absent, and the page hides it
export type ConfigEngine = {
  table(): Promise<ConfigTable>;
  // the page reads the options again whenever this changes
  revision: Readable<unknown>;
  inputs(): Async<Inputs>;
  placeholders(): Async<Record<string, number | string>>;
  relevance: Readable<unknown>;
  output: Readable<Record<string, number> | undefined>;
  // undefined removes the option from the build
  set(key: string, value: ConfigValue | undefined): Async<unknown>;
  reset?(): Async<unknown>;
  // ConfigTab.lua:152-155, ConfigSetListControl.lua:13-110
  sets?: {
    read(): Async<{ sets: { id: number; title?: string }[]; active: number }>;
    select(id: number): Async<unknown>;
    edit(op: ConfigSetEdit): Async<unknown>;
  };
  // ConfigTab.lua:15-86, 1305-1550
  custom?: {
    read(): Async<pob.CustomModifierBlock[]>;
    write(blocks: pob.CustomModifierBlock[]): Async<unknown>;
    parse(lines: string[]): Async<exposition.ModLine[]>;
    compare(group: number): Async<StatCompareSection[] | undefined>;
    browser: { list(search: string): Async<string[]>; sources(line: string): Async<string[]> };
  };
  // ConfigTab.lua:721-766
  compare?(key: string, value: ConfigValue): Async<StatCompareSection[] | undefined>;
};

// CalcsTab.lua:40-119 over the latest calculation
export type CalcsTabView = {
  sections: CalcsView;
  controls?: calculator.MainSkillControls;
  // CalcsTab.lua:135-137
  lists?: { BuffList?: string; CombatList?: string; CurseList?: string };
  haveMinion?: boolean;
};

// CalcsTab.lua: what the Calcs page reads and writes; a part a game has none of is absent, and the page hides it
export type CalcsEngine = {
  tab: Readable<CalcsTabView | undefined>;
  setMainSkill(field: MainSkillField, value: number | undefined): void;
  // CalcsTab.lua:12-17, 87-91: the tab's own skill number, buff mode and minion toggle; without them the tab shows the
  // sidebar's main group
  inputs?: {
    read(): Async<{ buffMode: string; showMinion: boolean; skillNumber: number }>;
    buffMode(mode: string): void;
    showMinion(show: boolean): void;
    skillNumber(n: number): void;
  };
  // CalcsTab.lua:174-189, 208-216
  collapse?: { revision: Readable<unknown>; read(): Async<Record<string, boolean> | undefined>; set(id: string, value: boolean): void };
  // the worker's handle on the tab's cells while the page is open; returns the detach
  attach?(): () => void;
  // CalcsTab.lua:46-51
  grouptip?(group: number): Async<Tooltip>;
  modrows?(sd: unknown): Async<calculator.ModSectionRow[]>;
  listrows?(bd: string): Async<calculator.ModSectionRow[]>;
  // Calcs.lua:37-70
  moddebug?(): Async<string>;
  // CalcBreakdownControl.lua:400-403
  itemtip?(name: string): Async<ItemTip | undefined>;
};

// what the app frame (top bar, sidebar, phone frame, layout) needs from the open build, whichever game it is
export type BuildEngine = {
  game: Game;
  // emits on every recalculation; a different object than the last means another build opened
  build: Readable<object | undefined>;
  // emits after every recalculation
  calculated: Readable<unknown>;
  sidebar: Readable<Sidebar | undefined>;
  mainSkill: Readable<calculator.MainSkillControls | undefined>;
  mainGroup(): MainGroupView;
  setMainSkill(field: MainSkillField, value: number | undefined): void;
  // Main.lua:1750-1756
  classInfo(): Promise<{ className: string; ascendClassName: string } | undefined> | undefined;
  save(path: string): Promise<unknown>;
  clear(): Promise<unknown>;
  // NotesTab.lua:75-88: the notes text and the rich document the Notes page keeps beside it; revision changes when
  // another build opens, and a read's write goes to the build it was read from
  notes: {
    revision: Readable<string>;
    read(): Async<{ status: string; plain: string; doc: string; write: (plain: string, doc: string) => Async<unknown> }>;
  };
  // ImportTab.lua:132-203: the build's code and the sites it can be shared to
  share: { code(): Async<string>; sites: Site[] };
  // Build.lua:1764-1908
  breakdownSteps?: (line: StatLine) => Promise<SidebarSection[] | undefined>;
  // Build.lua:1453-1482
  spectres?: { read(): Promise<{ library: SpectreRow[]; build: SpectreRow[] }> | undefined; save(ids: string[]): Promise<unknown> | undefined };
  undo?: {
    history(): Promise<UndoView> | undefined;
    undo(): Promise<UndoView> | undefined;
    redo(): Promise<UndoView> | undefined;
    undoTo(index: number): Promise<UndoView> | undefined;
  };
  // Build.lua:301-415, 753-878
  loadouts?: {
    list(): Promise<{ loadouts: string[]; selected: number } | undefined> | undefined;
    select(name: string): Promise<unknown> | undefined;
    create(name: string): Promise<unknown> | undefined;
  };
  // Build.lua:99-106
  conversion?: { check(): Promise<string | undefined> | string | undefined; convert(): Promise<unknown> | undefined };
  // an edit in flight, and the last one's error, for an engine whose edits the page waits on
  busy?: Readable<boolean>;
  error?: Readable<string | undefined>;
  config: ConfigEngine;
  calcs: CalcsEngine;
};

export const engines: Record<Game, BuildEngine> = { poe1: poe1Engine, poe2: poe2Engine };

// the game of the open build; undefined while none is open
export const game: Readable<Game | undefined> = derived([currentBuild, poe2Build], ([b, v]) => (b ? 'poe1' : v ? 'poe2' : undefined));

// the frame is keyed on the game, so a component reads its engine once, while it initialises; with no build open
// the PoE1 engine answers, as the frame did before there were two
export const currentEngine = (): BuildEngine => engines[get(game) ?? 'poe1'];
