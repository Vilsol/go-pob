import { derived, get } from 'svelte/store';
import { currentBuild, outputs } from '$lib/global';
import { sidebar } from '$lib/build_warnings';
import { sidebarBreakdown } from '$lib/display/sidebar_breakdown';
import { calcSections } from '$lib/calcs/calc_sections';
import { checkFlag } from '$lib/calcs/check_flag';
import type { StatLine } from '$lib/display/stat_list';
import type { SpectreRow } from '$lib/design/spectres';
import { MainGroup } from '$lib/shell/main_group.svelte';
import { writeMainSkill } from '$lib/shell/main_skill';
import type { calculator } from '$lib/types';
import type { BuildEngine } from './engine';
import { syncWrap } from './worker';

const row = (r: calculator.SpectreRow): SpectreRow => ({ id: r.ID, name: r.Name, skills: [...(r.Skills ?? [])], tooltip: [...(r.Tooltip ?? [])] });

export const poe1Engine: BuildEngine = {
  game: 'poe1',
  build: currentBuild,
  calculated: outputs,
  sidebar,
  mainSkill: derived(outputs, (o) => o?.MainSkill),
  mainGroup: () => new MainGroup(),
  setMainSkill: (field, value) => writeMainSkill(get(outputs)?.MainSkill, field, value),
  classInfo: () => syncWrap?.GetBuildInfo().then((i) => (i ? { className: i.className, ascendClassName: i.ascendClassName } : undefined)),
  save: (path) => syncWrap.SaveBuildAs(path),
  clear: () => Promise.resolve(syncWrap?.ClearBuild()),
  // CalcBreakdownControl.lua:75-83
  breakdownSteps: async (line: StatLine) => {
    if (!line.bd || !syncWrap) return undefined;
    const minion = !!line.minion;
    const key = await syncWrap.SidebarBreakdownKey({ ...line.bd, minion });
    if (key === undefined) return undefined;
    const out = get(outputs);
    const actor = minion ? out?.Minion : out;
    const cells = sidebarBreakdown(calcSections, key, line.bd.modNames, line.bd.ignoredSections).filter((c) => checkFlag(c, actor));
    return syncWrap.SidebarBreakdownSections(minion, cells);
  },
  spectres: {
    read: () => syncWrap?.GetSpectreLibrary().then((l) => ({ library: l.library.map(row), build: l.build.map(row) })),
    save: (ids) => syncWrap?.SetSpectres(ids)
  },
  undo: {
    history: () => syncWrap?.GetUndoHistoryWhenRecorded(),
    undo: () => syncWrap?.Undo(),
    redo: () => syncWrap?.Redo(),
    undoTo: (i) => syncWrap?.UndoTo(i)
  },
  loadouts: {
    list: () => syncWrap?.GetLoadouts(),
    select: (name) => syncWrap?.SelectLoadout(name).then(() => currentBuild.set(get(currentBuild))),
    create: (name) => syncWrap?.NewLoadout(name).then(() => currentBuild.set(get(currentBuild)))
  },
  conversion: {
    check: () => syncWrap?.GetVersionConversion(),
    convert: () => syncWrap?.ConvertBuildVersion()
  }
};

// the app's own version and game data, which the PoE1 runtime reports whichever game is open
export const runtime = {
  buildInfo: () => syncWrap?.BuildInfo(),
  gameDataVersion: () => syncWrap?.GameDataVersion()
};
