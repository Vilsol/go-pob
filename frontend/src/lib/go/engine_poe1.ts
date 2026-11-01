import { derived, get } from 'svelte/store';
import { backendLoaded, buildRevision, currentBuild, outputs } from '$lib/global';
import { sidebar } from '$lib/build_warnings';
import { sidebarBreakdown } from '$lib/display/sidebar_breakdown';
import { calcSections } from '$lib/calcs/calc_sections';
import { calcElements, calcsView } from '$lib/display/calcs';
import { poe1ConfigTable } from '$lib/design/config';
import { checkFlag } from '$lib/calcs/check_flag';
import type { StatLine } from '$lib/display/stat_list';
import type { SpectreRow } from '$lib/design/spectres';
import { MainGroup } from '$lib/shell/main_group.svelte';
import { writeMainSkill } from '$lib/shell/main_skill';
import type { calculator } from '$lib/types';
import type { BuildEngine, CalcsTabView } from './engine';
import { syncWrap } from './worker';
import { logError } from '$lib/utils';

// the pages re-read their saved state when another build or loadout opens
const revision = derived([buildRevision, currentBuild], ([r, b]) => [r, b === undefined]);

// CalcsTab.lua:208-216
const subsectionId = (label: string) => label.replace(/[^A-Za-z0-9]/g, '');
const sectionKeys = calcSections.map((s) => [s[1], subsectionId(s[4][0]?.label ?? '')] as [string, string]);

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
  },
  config: {
    table: () => Promise.resolve(poe1ConfigTable),
    revision,
    inputs: () => (get(currentBuild) ? syncWrap?.GetAllConfigOptions() : undefined),
    placeholders: () => syncWrap?.GetConfigPlaceholders(),
    relevance: derived(outputs, (o) => o?.ConfigRelevance ?? {}),
    output: derived(outputs, (o) => o?.Output),
    set: (key, value) => syncWrap?.SetConfigOption(key, value ?? (null as unknown as number)),
    reset: () => syncWrap?.ResetConfigOptions(),
    sets: {
      read: () => syncWrap?.GetConfigSets(),
      select: (id) => syncWrap?.SetActiveConfigSet(id),
      edit: (op) => syncWrap?.EditConfigSets(op)
    },
    custom: {
      read: () => (get(currentBuild) ? syncWrap?.GetCustomModifierBlocks() : undefined),
      write: (blocks) => syncWrap?.SetCustomModifierBlocks(blocks),
      parse: (lines) => syncWrap?.ParseModLines(lines),
      compare: (g) => syncWrap?.CustomModGroupCompare(g),
      browser: { list: (search) => syncWrap?.ModBrowserList(search), sources: (line) => syncWrap?.ModBrowserSources(line) }
    },
    compare: (key, value) => syncWrap?.ConfigCompare(key, value)
  },
  calcs: {
    // CalcsTab.lua:452-455, CalcsTab.lua:40-119
    tab: derived(outputs, (o): CalcsTabView | undefined => {
      const tab = o?.CalcsTab;
      if (!tab?.Breakdown) return undefined;
      return { sections: calcsView(calcSections, tab), controls: tab.MainSkill, lists: tab.OutputStrings, haveMinion: o?.SkillFlags?.haveMinion };
    }),
    setMainSkill: (field, value) => writeMainSkill(get(outputs)?.CalcsTab?.MainSkill, field, value, true),
    inputs: {
      read: () => syncWrap?.GetCalcsTabInputs(),
      buffMode: (mode) => void syncWrap?.SetCalcsBuffMode(mode),
      showMinion: (show) => void syncWrap?.SetCalcsShowMinion(show),
      skillNumber: (n) => void syncWrap?.SetCalcsSkillNumber(n)
    },
    // CalcsTab.lua:174-189, CalcSectionControl.lua:23-39
    collapse: {
      revision: buildRevision,
      read: () =>
        syncWrap
          ?.GetCalcsSectionsCollapsed(sectionKeys)
          .then((saved) => (saved ? Object.fromEntries(calcSections.map((s, i) => [s[1], saved[i] ?? !!s[4][0]?.defaultCollapsed])) : undefined)),
      set: (id, value) => {
        const key = sectionKeys.find(([k]) => k === id);
        if (key) void syncWrap?.SetCalcsSectionCollapsed(key[0], key[1], value).catch(logError);
      }
    },
    attach: () => {
      let detached = false;
      void backendLoaded.then(() => {
        if (!detached) void syncWrap?.setCalcTabElements(calcElements(calcSections));
      });
      return () => {
        detached = true;
        void syncWrap?.clearCalcTabElements();
      };
    },
    grouptip: (i) => syncWrap?.SocketGroupTooltip(i),
    modrows: (sd) => syncWrap?.CalcsModSection(sd as calculator.ModSectionData),
    listrows: (bd) => syncWrap?.CalcsBreakdownModList(bd),
    moddebug: () => syncWrap?.GetModDebugDump(),
    itemtip: (name) =>
      syncWrap
        ?.ItemSourceTooltip(name)
        .then((t) =>
          t
            ? { influence: [...(t.Influence ?? [])], title: [...(t.Title ?? [])], lines: (t.Lines ?? []).map((l) => ({ text: l.Text, sep: l.Separator })) }
            : undefined
        )
  }
};

// the app's own version and game data, which the PoE1 runtime reports whichever game is open
export const runtime = {
  buildInfo: () => syncWrap?.BuildInfo(),
  gameDataVersion: () => syncWrap?.GameDataVersion()
};
