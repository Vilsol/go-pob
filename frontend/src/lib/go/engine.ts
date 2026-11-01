import { derived, get, type Readable } from 'svelte/store';
import { currentBuild, poe2Build } from '$lib/global';
import type { Sidebar } from '$lib/display/sidebar';
import type { StatLine } from '$lib/display/stat_list';
import type { SidebarSection } from '$lib/display/sidebar_breakdown';
import type { SpectreRow } from '$lib/design/spectres';
import type { MainSkillField } from '$lib/shell/main_skill';
import type { calculator } from '$lib/types';
import type { UndoView } from './undo';
import { poe1Engine } from './engine_poe1';
import { poe2Engine } from './engine_poe2.svelte';

export { runtime } from './engine_poe1';

export type Game = 'poe1' | 'poe2';

// the sidebar's socket group picker; made while a component initialises, since it follows the build for as long as
// that component lives
export type MainGroupView = { readonly groups: string[]; readonly index: number; select(index: number): void };

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
};

export const engines: Record<Game, BuildEngine> = { poe1: poe1Engine, poe2: poe2Engine };

// the game of the open build; undefined while none is open
export const game: Readable<Game | undefined> = derived([currentBuild, poe2Build], ([b, v]) => (b ? 'poe1' : v ? 'poe2' : undefined));

// the frame is keyed on the game, so a component reads its engine once, while it initialises; with no build open
// the PoE1 engine answers, as the frame did before there were two
export const currentEngine = (): BuildEngine => engines[get(game) ?? 'poe1'];
