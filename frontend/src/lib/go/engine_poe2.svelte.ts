import { derived, get, writable } from 'svelte/store';
import type { Remote } from 'comlink';
import { poe2Build, poe2Edited, poe2Session, type Poe2View } from '$lib/global';
import type { BuildWarning, Sidebar, SidebarSection } from '$lib/display/sidebar';
import { buildStatList2, displayStats2, minionDisplayStats2 } from '$lib/display/stats2';
import { savePoe2Build } from '$lib/shell/poe2_save';
import type { calculator } from '$lib/types';
import type { BuildEngine, MainGroupView } from './engine';
import type { Poe2Calc, Poe2SocketGroup, Worker2Type } from './sync_worker2';

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

export const poe2Engine: BuildEngine = {
  game: 'poe2',
  build: poe2Session,
  calculated: poe2Build,
  sidebar: derived(poe2Build, (v) => (v ? sidebar2(v as Calc) : undefined)),
  mainSkill: derived(poe2Build, (v) => controls2(v as Calc | undefined)),
  mainGroup: () => new Poe2MainGroup(),
  // Build.lua:376-381 (PoB-PoE2)
  setMainSkill: (field, value) => {
    if (field === 'MainActiveSkill' && value !== undefined) void poe2Run((w) => w.SetMainActiveSkill(value));
  },
  classInfo: () => {
    const v = get(poe2Build);
    return Promise.resolve(v && { className: v.className, ascendClassName: v.ascendClassName });
  },
  // Build.lua:2773-2787 (PoB-PoE2)
  save: (path) => savePoe2Build(path),
  clear: () => {
    poe2Build.set(undefined);
    poe2Error.set(undefined);
    return Promise.resolve();
  }
};
