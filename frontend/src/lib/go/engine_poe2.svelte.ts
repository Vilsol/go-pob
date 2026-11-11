import { derived, get, readable, writable } from 'svelte/store';
import type { Remote } from 'comlink';
import { poe2Build, poe2Edited, poe2Session, type Poe2View } from '$lib/global';
import type { BuildWarning, Sidebar, SidebarSection } from '$lib/display/sidebar';
import { buildStatList2, displayStats2, minionDisplayStats2 } from '$lib/display/stats2';
import { savePoe2Build, savedStats2 } from '$lib/shell/poe2_save';
import { EXPORT_SITES2 } from '$lib/design/share';
import type { calculator, exposition } from '$lib/types';
import type { calculator2 } from '$lib/types2';
import type { CalcSection2 } from '$lib/calcs/calc_section_types2';
import type { MainSkillField } from '$lib/shell/main_skill';
import type { BuildEngine, CalcsTabView, MainGroupView, StatSetControls } from './engine';
import { startJob } from '$lib/activity/jobs';
import type { Poe2Calc, Poe2CalcsPage, Poe2Restore, Poe2SkillSelect, Poe2SocketGroup, Worker2Type } from './sync_worker2';

// the PoE2 worker hands back a Poe2Calc; a view without its fields shows what it has
type Calc = Partial<Poe2Calc> & Poe2View;

let worker2: Promise<typeof import('$lib/go/worker2')> | undefined;
// the first boot is a job; once booted, a call is not
let booting: Promise<Remote<Worker2Type>> | undefined;
let booted = false;
export const poe2Worker = async () => {
  if (booted) return (await (worker2 ??= import('$lib/go/worker2'))).poe2Worker();
  return (booting ??= (() => {
    const j = startJob('poe2-boot', 'Starting the PoE2 engine', { long: true }).phase('Loading the engine code');
    return (worker2 ??= import('$lib/go/worker2'))
      .then((m) => m.poe2Worker(j))
      .then(
        (w) => {
          booted = true;
          j.end();
          return w;
        },
        (err: unknown) => {
          booting = undefined;
          j.end('error', err);
          throw err;
        }
      );
  })());
};

// an edit in flight, and the last one's error
export const poe2Busy = writable(false);
export const poe2Error = writable<string | undefined>();

// runs op in the PoE2 worker and shows the build it hands back, if it hands one back
export const poe2Run = async <T>(op: (w: Remote<Worker2Type>) => Promise<T>, view: (r: T) => Poe2View | undefined = (r) => r as Poe2View | undefined) => {
  poe2Busy.set(true);
  poe2Error.set(undefined);
  const job = startJob('calc2', 'Recalculating');
  try {
    const w = await poe2Worker();
    job.phase('Calculating in the PoE2 engine');
    const r = await op(w);
    const next = view(r);
    if (next) poe2Edited(next);
    job.end();
    return r;
  } catch (err) {
    job.end('error', err);
    poe2Error.set(err instanceof Error ? err.message : String(err));
    return undefined;
  } finally {
    poe2Busy.set(false);
  }
};

const fail = (err: unknown) => poe2Error.set(err instanceof Error ? err.message : String(err));

// Build.lua:376-483 (PoB-PoE2): the sidebar's controls, and with calcs the Calcs tab's (CalcsTab.lua:58-162)
const writeSkillSelect2 = (calcs: boolean, field: MainSkillField, value: number | undefined) => {
  if (field === 'MainActiveSkill' && value !== undefined) void poe2Run((w) => (calcs ? w.SetMainActiveSkillCalcs(value) : w.SetMainActiveSkill(value)));
  else if (field === 'StatSet' && value !== undefined) void poe2Run((w) => w.SetSkillStatSet(calcs, value));
  else if (field === 'StageCount') void poe2Run((w) => w.SetSkillStageCount(calcs, value));
  else if (field === 'MineCount') void poe2Run((w) => w.SetSkillMineCount(calcs, value));
  else if (field === 'Minion' && value !== undefined) void poe2Run((w) => w.SetSkillMinion(calcs, value));
  else if (field === 'MinionSkill' && value !== undefined) void poe2Run((w) => w.SetSkillMinionSkill(calcs, value));
  else if (field === 'MinionSkillStatSet' && value !== undefined) void poe2Run((w) => w.SetSkillMinionSkillStatSet(calcs, value));
};

// Build.lua:2132-2191 (PoB-PoE2): the minion dropdowns and library buttons as the shared controls carry them
const minionControls2 = (c: Poe2SkillSelect | undefined) => ({
  MinionShown: c?.MinionShown ?? false,
  MinionEnabled: c?.MinionEnabled ?? false,
  MinionLibraryShown: c?.MinionLibraryShown ?? false,
  BeastLibraryShown: c?.BeastLibraryShown ?? false,
  Minions: c?.Minions?.map((Label) => ({ Label })),
  Minion: c?.Minion ?? 0,
  MinionSkillShown: c?.MinionSkillShown ?? false,
  MinionSkillEnabled: c?.MinionSkillEnabled ?? false,
  MinionSkills: c?.MinionSkills,
  MinionSkill: c?.MinionSkill ?? 0,
  MinionSkillStatSetShown: c?.MinionSkillStatSetShown ?? false,
  MinionSkillStatSetEnabled: c?.MinionSkillStatSetEnabled ?? false,
  MinionSkillStatSets: c?.MinionSkillStatSets,
  MinionSkillStatSet: c?.MinionSkillStatSet ?? 0
});
const setMainSkill2 = (field: MainSkillField, value: number | undefined) => writeSkillSelect2(false, field, value);

// Build.lua:2059-2195 (PoB-PoE2): RefreshSkillSelectControls' state as the shared controls carry it; PoE2's skill data
// has no multipart skill
export const skillSelect2 = (c: Poe2SkillSelect): calculator.MainSkillControls & StatSetControls =>
  ({
    Skills: c.Skills,
    SkillEnabled: c.SkillEnabled,
    MainActiveSkill: c.MainActiveSkill || 1,
    StatSets: c.StatSets,
    StatSet: c.StatSet,
    StatSetShown: c.StatSetShown,
    StatSetEnabled: c.StatSetEnabled,
    PartShown: false,
    StageShown: c.StageShown,
    StageCount: c.StageCount,
    MineShown: c.MineShown,
    MineCount: c.MineCount,
    ...minionControls2(c)
  }) as calculator.MainSkillControls & StatSetControls;

// the config page reads the options again after every calculation
let calcCount = 0;
const revision2 = derived(poe2Build, (v) => (v ? ++calcCount : 0));

// CalcSectionControl.lua:34 (PoB-PoE2): a subsection's id is its label without non-word characters
const sectionKeys2 = (sections: readonly CalcSection2[]) =>
  sections.map((s) => [s[1], (s[4][0]?.label ?? '').replace(/[^A-Za-z0-9]/g, '')] as [string, string]);

// Build.lua:2375-2481 (PoB-PoE2): the minion's list over the player's, each stat warning once
const sidebar2 = (calc: Calc): Sidebar => {
  const lines: string[] = [];
  const sections: SidebarSection[] = [];
  if (calc.minion)
    sections.push({
      header: 'Minion',
      groups: buildStatList2({ Output: calc.minion, SkillFlags: calc.minionSkillFlags }, minionDisplayStats2, lines).map((g) =>
        g.map((l) => ({ ...l, minion: true }))
      )
    });
  sections.push({
    header: calc.minion ? 'Player' : undefined,
    info: calc.skillFlags?.disable ? ['^7Skill disabled:'] : undefined,
    groups: buildStatList2({ Output: calc.output, SkillFlags: calc.skillFlags, SkillDPS: calc.skillDPS }, displayStats2, lines)
  });
  const warnings: BuildWarning[] = [...new Set(lines)].map((text) => ({ kind: 'stat', text }));
  // Build.lua:1053-1083 (PoB-PoE2)
  for (const text of calc.tree?.progress?.warnings ?? []) if (!warnings.some((w) => w.text === text)) warnings.push({ kind: 'points', text });
  return { sections, warnings };
};

const mainGroupOf = (calc: Calc | undefined): Poe2SocketGroup | undefined => calc?.socketGroups?.[(calc.mainSocketGroup ?? 1) - 1];

// Build.lua:363-483, 2059-2195 (PoB-PoE2): the main group's active skill, stat set, stages, mines and minion controls;
// PoE2 has none of PoE1's gem parts
const controls2 = (calc: Calc | undefined): (calculator.MainSkillControls & StatSetControls) | undefined => {
  const group = mainGroupOf(calc);
  if (!group) return undefined;
  const select = calc!.mainSkill;
  return {
    SkillSet: 0,
    SocketGroup: calc!.mainSocketGroup ?? 1,
    // the sidebar writes the stage and mine boxes only with a gem; the PoE2 worker finds it itself
    GemIndex: select ? 0 : -1,
    Skills: group.skills,
    SkillEnabled: group.skills.length > 1,
    MainActiveSkill: calc!.mainActiveSkill || 1,
    PartShown: false,
    SkillPart: 0,
    StageShown: select?.StageShown ?? false,
    StageCount: select?.StageCount ?? '',
    MineShown: select?.MineShown ?? false,
    MineCount: select?.MineCount ?? '',
    StatSets: select?.StatSets,
    StatSet: select?.StatSet,
    StatSetShown: select?.StatSetShown,
    StatSetEnabled: select?.StatSetEnabled,
    ...minionControls2(select),
    SkillDisabled: false,
    DisableReason: '',
    InfoMessage: '',
    InfoMessage2: '',
    MinionInfoMessage: '',
    MinionInfoMessage2: ''
  } as unknown as calculator.MainSkillControls & StatSetControls;
};

// Build.lua:364-368 (PoB-PoE2)
class Poe2MainGroup implements MainGroupView {
  groups = $state<string[]>([]);
  index = $state(-1);

  constructor() {
    $effect(() =>
      poe2Build.subscribe((v) => {
        const calc = v as Calc | undefined;
        this.groups = (calc?.socketGroups ?? []).map((g) => g.label);
        this.index = (calc?.mainSocketGroup ?? 1) - 1;
      })
    );
  }

  select(i: number) {
    this.index = i;
    void poe2Run((w) => w.SetMainSocketGroup(i + 1));
  }
}

// CalcsTab.lua (PoB-PoE2) over the tab's own calculation (CalcsTab.lua:512-513), fetched again for every calculation
// while the page is open; the last shows meanwhile
const calcsTab2 = readable<CalcsTabView | undefined>(undefined, (set) => {
  const code = Promise.all([import('$lib/display/calcs2'), import('$lib/calcs/calc_sections2'), import('$lib/display/format_stat')]);
  let calc: Calc | undefined;
  let page: Poe2CalcsPage | undefined;
  // a view kept from the last time the page was open belongs to that build
  set(undefined);
  const show = async () => {
    const at = calc;
    if (!at || !page) return;
    const [{ calcsView2 }, { calcSections2 }, { formatNumSep }] = await code;
    if (at !== calc) return;
    const output: Record<string, unknown> = { ...page.output, ...page.flags, ...page.strings };
    if (page.mainHand) output.MainHand = page.mainHand;
    if (page.offHand) output.OffHand = page.offHand;
    // CalcOffence.lua:4851-4857 (PoB-PoE2): formatNumSep follows the display settings
    if (page.strings.DisplayDamage) output.DisplayDamage = formatNumSep(page.strings.DisplayDamage);
    set({
      sections: calcsView2(calcSections2, {
        output: output as never,
        skillFlags: page.skillFlags,
        skillData: page.skillData,
        mods: page.mods,
        breakdowns: page.breakdowns
      }),
      // CalcsTab.lua:177-180 (PoB-PoE2)
      lists: { BuffList: page.strings.BuffList, CombatList: page.strings.CombatList, CurseList: page.strings.CurseList },
      // Build.lua:2059-2195 (PoB-PoE2): RefreshSkillSelectControls with the "Calcs" suffix
      controls: page.skills.length > 0 ? skillSelect2(page.controls) : undefined,
      // CalcsTab.lua:101 (PoB-PoE2)
      haveMinion: page.haveMinion
    });
  };
  return poe2Build.subscribe((v) => {
    calc = v as Calc | undefined;
    if (!calc) return set(undefined);
    void show();
    void poe2Worker()
      .then((w) => w.CalcsPage())
      .then((p) => {
        page = p;
        return show();
      })
      .catch(fail);
  });
});

// a number per opened PoE2 build
let opened = 0;
let openedBuild: object | undefined;
const notesRevision2 = derived(poe2Session, (s) => {
  if (s !== openedBuild) {
    openedBuild = s;
    if (s) opened++;
  }
  return s ? `poe2:${opened}` : 'poe2:none';
});

// NotesTab.lua:75-88 (PoB-PoE2): a write marks the build edited, as the tab's modFlag does, and is dropped once
// another build has opened
const readNotes2 = () => {
  const session = get(poe2Session);
  if (!session) return undefined;
  return poe2Worker()
    .then((w) => w.Notes())
    .then((n) => ({
      ...n,
      write: async (plain: string, doc: string) => {
        if (get(poe2Session) !== session) return;
        await (await poe2Worker()).SetNotes(plain, doc);
        const view = get(poe2Build);
        if (view && get(poe2Session) === session) poe2Edited(view);
      }
    }));
};

// the whole-build history: a step that moves it shows the build it restored, as an edit does
const moveHistory = (op: (w: Remote<Worker2Type>) => Promise<Poe2Restore>) =>
  get(poe2Build) ? poe2Run(op, (r) => r.calc).then((r) => r?.history ?? poe2Worker().then((w) => w.History())) : undefined;

export const poe2Engine: BuildEngine = {
  game: 'poe2',
  build: poe2Session,
  calculated: poe2Build,
  sidebar: derived(poe2Build, (v) => (v ? sidebar2(v as Calc) : undefined)),
  mainSkill: derived(poe2Build, (v) => controls2(v as Calc | undefined)),
  mainGroup: () => new Poe2MainGroup(),
  setMainSkill: setMainSkill2,
  classInfo: () => {
    const v = get(poe2Build);
    return Promise.resolve(v && { className: v.className, ascendClassName: v.ascendClassName });
  },
  // Build.lua:2773-2787 (PoB-PoE2)
  save: (path) => savePoe2Build(path),
  notes: { revision: notesRevision2, read: readNotes2 },
  // ImportTab.lua:135-137, 154-164 (PoB-PoE2)
  share: {
    code: () => {
      const v = get(poe2Build) as Calc | undefined;
      return v?.skillFlags ? poe2Worker().then((w) => w.ExportCode(savedStats2(v as Poe2Calc))) : undefined;
    },
    sites: EXPORT_SITES2
  },
  clear: () => {
    poe2Build.set(undefined);
    poe2Error.set(undefined);
    return Promise.resolve();
  },
  busy: poe2Busy,
  error: poe2Error,
  undo: {
    history: () => (get(poe2Build) ? poe2Worker().then((w) => w.History()) : undefined),
    undo: () => moveHistory((w) => w.Undo()),
    redo: () => moveHistory((w) => w.Redo()),
    undoTo: (i) => moveHistory((w) => w.UndoTo(i))
  },
  // ConfigTab.lua (PoB-PoE2): no mod browser yet
  config: {
    table: () => import('$lib/design/config_table2').then((m) => m.poe2ConfigTable),
    revision: revision2,
    inputs: () => {
      const v = get(poe2Build) as Calc | undefined;
      return v && Promise.resolve({ ...v.configInputs });
    },
    // ConfigTab.lua:1051-1052 (PoB-PoE2)
    placeholders: () => {
      const v = get(poe2Build) as Calc | undefined;
      return v && Promise.resolve({ ...v.configPlaceholders });
    },
    relevance: derived(poe2Build, (v) => (v as Calc | undefined)?.relevance ?? ({} as calculator2.ConfigRelevance)),
    output: derived(poe2Build, (v) => v?.output),
    set: (key, value) => poe2Run((w) => w.SetConfigOption(key, value)),
    // ConfigTab.lua:152-178, ConfigSetListControl.lua:13-129 (PoB-PoE2)
    sets: {
      read: () => {
        const v = get(poe2Build) as Calc | undefined;
        return v?.configSets && Promise.resolve(v.configSets);
      },
      select: (id) => poe2Run((w) => w.SelectConfigSet(id)),
      edit: (op) => poe2Run((w) => w.EditConfigSet(op))
    },
    // ConfigTab.lua:24-98, 1200-1223 (PoB-PoE2)
    custom: {
      read: () => {
        const v = get(poe2Build) as Calc | undefined;
        return v?.customBlocks && Promise.resolve(v.customBlocks);
      },
      write: (blocks) => poe2Run((w) => w.SetCustomModifierBlocks(blocks)),
      parse: (lines) =>
        poe2Worker()
          .then((w) => w.ParseModLines(lines))
          .then((l) => l as exposition.ModLine[]),
      // ConfigTab.lua:62-79 (PoB-PoE2)
      compare: (g) => (get(poe2Build) ? poe2Worker().then((w) => w.CustomModGroupCompare(g)) : undefined)
    },
    // ConfigTab.lua:774-838 (PoB-PoE2)
    compare: (key, value) => (get(poe2Build) ? poe2Worker().then((w) => w.ConfigCompare(key, value)) : undefined)
  },
  // Build.lua:1479-1511, 2226-2331 (PoB-PoE2): the main calculation's breakdown and mod tables a sidebar line names
  breakdownSteps: async (line) => {
    const calc = get(poe2Build) as Calc | undefined;
    if (!line.bd || line.minion || !calc) return undefined;
    const w = await poe2Worker();
    const ref = await w.SidebarBreakdownKey(line.bd);
    if (!ref) return undefined;
    const [{ checkFlag2, sidebarBreakdown2 }, { calcSections2 }] = await Promise.all([import('$lib/display/calcs2'), import('$lib/calcs/calc_sections2')]);
    const actor = { output: calc.output as never, skillFlags: calc.skillFlags ?? {}, skillData: calc.skillData ?? {}, mods: {} };
    const cells = sidebarBreakdown2(calcSections2, ref.key, ref.source, line.bd.modNames, line.bd.ignoredSections).filter((c) => checkFlag2(c as never, actor));
    return w.SidebarBreakdownSections(cells);
  },
  // SkillsTab.lua (PoB-PoE2)
  skills: () => import('./engine_poe2_skills.svelte').then((m) => m.poe2SkillsPage),
  // TreeTab.lua, Build.lua:246-299 (PoB-PoE2)
  tree: () => import('./engine_poe2_tree.svelte').then((m) => m.poe2TreeModel),
  calcs: {
    tab: calcsTab2,
    // CalcsTab.lua:58-100 (PoB-PoE2)
    setMainSkill: (field, value) => writeSkillSelect2(true, field, value),
    // CalcsTab.lua:31-33, 43-46, 101-105, 163-168 (PoB-PoE2)
    inputs: {
      read: () =>
        get(poe2Build)
          ? poe2Worker()
              .then((w) => w.CalcsInputs())
              .then((i) => i ?? { skillNumber: 1, buffMode: 'EFFECTIVE', showMinion: false })
          : undefined,
      buffMode: (mode) => void poe2Run((w) => w.SetCalcsBuffMode(mode)),
      showMinion: (show) => void poe2Run((w) => w.SetCalcsShowMinion(show)),
      skillNumber: (n) => void poe2Run((w) => w.SetCalcsSkillNumber(n))
    },
    // CalcsTab.lua:218-233, CalcSectionControl.lua:34, 49-55 (PoB-PoE2): the first subsection's state, as PoE1's page keeps
    collapse: {
      revision: poe2Session,
      read: () =>
        get(poe2Build)
          ? Promise.all([poe2Worker(), import('$lib/calcs/calc_sections2')]).then(async ([w, { calcSections2 }]) => {
              const saved = await w.GetCalcsSectionsCollapsed(sectionKeys2(calcSections2));
              return Object.fromEntries(calcSections2.map((s, i) => [s[1], saved[i] ?? !!s[4][0]?.defaultCollapsed]));
            })
          : undefined,
      set: (id, value) => {
        void Promise.all([poe2Worker(), import('$lib/calcs/calc_sections2')])
          .then(async ([w, { calcSections2 }]) => {
            const key = sectionKeys2(calcSections2).find(([k]) => k === id);
            if (!key) return;
            await w.SetCalcsSectionCollapsed(key[0], key[1], value);
            const view = get(poe2Build);
            if (view) poe2Edited(view);
          })
          .catch(fail);
      }
    },
    // CalcBreakdownControl.lua:300-548 (PoB-PoE2)
    modrows: (sd) =>
      get(poe2Build) ? poe2Worker().then((w) => w.CalcsModSection(sd as calculator2.ModSectionData) as unknown as calculator.ModSectionRow[]) : undefined,
    // CalcsTab.lua:178-180, CalcBreakdownControl.lua:293-296 (PoB-PoE2)
    listrows: (bd) => (get(poe2Build) ? poe2Worker().then((w) => w.CalcsBreakdownModList(bd) as unknown as calculator.ModSectionRow[]) : undefined)
  }
};
