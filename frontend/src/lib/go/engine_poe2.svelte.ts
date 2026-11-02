import { derived, get, readable, writable } from 'svelte/store';
import type { Remote } from 'comlink';
import { poe2Build, poe2Edited, poe2Session, type Poe2View } from '$lib/global';
import type { BuildWarning, Sidebar, SidebarSection } from '$lib/display/sidebar';
import { buildStatList2, displayStats2, minionDisplayStats2 } from '$lib/display/stats2';
import { savePoe2Build, savedStats2 } from '$lib/shell/poe2_save';
import { EXPORT_SITES2 } from '$lib/design/share';
import type { calculator } from '$lib/types';
import type { calculator2 } from '$lib/types2';
import type { MainSkillField } from '$lib/shell/main_skill';
import type { BuildEngine, CalcsTabView, MainGroupView } from './engine';
import type { Poe2Calc, Poe2CalcsPage, Poe2SocketGroup, Worker2Type } from './sync_worker2';

// the PoE2 worker hands back a Poe2Calc; a view without its fields shows what it has
type Calc = Partial<Poe2Calc> & Poe2View;

let worker2: Promise<typeof import('$lib/go/worker2')> | undefined;
export const poe2Worker = async () => (await (worker2 ??= import('$lib/go/worker2'))).poe2Worker();

// an edit in flight, and the last one's error
export const poe2Busy = writable(false);
export const poe2Error = writable<string | undefined>();

// runs op in the PoE2 worker and shows the build it hands back, if it hands one back
export const poe2Run = async <T>(op: (w: Remote<Worker2Type>) => Promise<T>, view: (r: T) => Poe2View | undefined = (r) => r as Poe2View | undefined) => {
  poe2Busy.set(true);
  poe2Error.set(undefined);
  try {
    const r = await op(await poe2Worker());
    const next = view(r);
    if (next) poe2Edited(next);
    return r;
  } catch (err) {
    poe2Error.set(err instanceof Error ? err.message : String(err));
    return undefined;
  } finally {
    poe2Busy.set(false);
  }
};

const fail = (err: unknown) => poe2Error.set(err instanceof Error ? err.message : String(err));

// Build.lua:376-381 (PoB-PoE2)
const setMainSkill2 = (field: MainSkillField, value: number | undefined) => {
  if (field === 'MainActiveSkill' && value !== undefined) void poe2Run((w) => w.SetMainActiveSkill(value));
};

// the config page reads the options again after every calculation
let calcCount = 0;
const revision2 = derived(poe2Build, (v) => (v ? ++calcCount : 0));

// Build.lua:2375-2481 (PoB-PoE2): the minion's list over the player's, each stat warning once
const sidebar2 = (calc: Calc): Sidebar => {
  const lines: string[] = [];
  const sections: SidebarSection[] = [];
  if (calc.minion)
    sections.push({ header: 'Minion', groups: buildStatList2({ Output: calc.minion, SkillFlags: calc.minionSkillFlags }, minionDisplayStats2, lines) });
  sections.push({
    header: calc.minion ? 'Player' : undefined,
    info: calc.skillFlags?.disable ? ['^7Skill disabled:'] : undefined,
    groups: buildStatList2({ Output: calc.output, SkillFlags: calc.skillFlags }, displayStats2, lines)
  });
  const warnings: BuildWarning[] = [...new Set(lines)].map((text) => ({ kind: 'stat', text }));
  return { sections, warnings };
};

const mainGroupOf = (calc: Calc | undefined): Poe2SocketGroup | undefined => calc?.socketGroups?.[(calc.mainSocketGroup ?? 1) - 1];

// Build.lua:363-381 (PoB-PoE2): the main group's active skills; PoE2 has none of PoE1's gem parts, stages or minions here
const controls2 = (calc: Calc | undefined): calculator.MainSkillControls | undefined => {
  const group = mainGroupOf(calc);
  if (!group) return undefined;
  return {
    SkillSet: 0,
    SocketGroup: calc!.mainSocketGroup ?? 1,
    GemIndex: -1,
    Skills: group.skills,
    SkillEnabled: group.skills.length > 1,
    MainActiveSkill: calc!.mainActiveSkill || 1,
    PartShown: false,
    SkillPart: 0,
    StageShown: false,
    StageCount: '',
    MineShown: false,
    MineCount: '',
    MinionShown: false,
    MinionEnabled: false,
    MinionLibraryShown: false,
    Minion: 0,
    MinionSkillShown: false,
    MinionSkillEnabled: false,
    MinionSkill: 0,
    SkillDisabled: false,
    DisableReason: '',
    InfoMessage: '',
    InfoMessage2: '',
    MinionInfoMessage: '',
    MinionInfoMessage2: ''
  } as unknown as calculator.MainSkillControls;
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

// CalcsTab.lua (PoB-PoE2) over the main calculation: calculator2 runs no CALCS pass, so the tab's skill number and
// buff mode are the sidebar's. Its mod values are fetched again for every calculation while the page is open, and
// the last shows meanwhile
const calcsTab2 = readable<CalcsTabView | undefined>(undefined, (set) => {
  const code = Promise.all([import('$lib/display/calcs2'), import('$lib/calcs/calc_sections2')]);
  let calc: Calc | undefined;
  let page: Poe2CalcsPage | undefined;
  // a view kept from the last time the page was open belongs to that build
  set(undefined);
  const show = async () => {
    const at = calc;
    if (!at || !page) return;
    const [{ calcsView2 }, { calcSections2 }] = await code;
    if (at !== calc) return;
    const output: Record<string, unknown> = { ...at.output, ...page.flags, ...page.strings };
    if (page.mainHand) output.MainHand = page.mainHand;
    if (page.offHand) output.OffHand = page.offHand;
    const group = mainGroupOf(at);
    set({
      sections: calcsView2(calcSections2, {
        output: output as never,
        skillFlags: at.skillFlags ?? {},
        skillData: at.skillData ?? {},
        mods: page.mods,
        breakdowns: page.breakdowns
      }),
      controls:
        group && group.skills.length > 0 ? ({ Skills: group.skills, MainActiveSkill: at.mainActiveSkill || 1 } as calculator.MainSkillControls) : undefined
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
  // ConfigTab.lua (PoB-PoE2): no config sets, custom modifiers or comparisons yet
  config: {
    table: () => import('$lib/design/config_table2').then((m) => m.poe2ConfigTable),
    revision: revision2,
    inputs: () => {
      const v = get(poe2Build) as Calc | undefined;
      return v && Promise.resolve({ ...v.configInputs });
    },
    placeholders: () => (get(poe2Build) ? Promise.resolve({}) : undefined),
    relevance: derived(poe2Build, (v) => (v as Calc | undefined)?.relevance ?? ({} as calculator2.ConfigRelevance)),
    output: derived(poe2Build, (v) => v?.output),
    set: (key, value) => poe2Run((w) => w.SetConfigOption(key, value))
  },
  // SkillsTab.lua (PoB-PoE2)
  skills: () => import('./engine_poe2_skills.svelte').then((m) => m.poe2SkillsPage),
  calcs: {
    tab: calcsTab2,
    setMainSkill: setMainSkill2
  }
};
