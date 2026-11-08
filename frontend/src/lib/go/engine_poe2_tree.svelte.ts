import { untrack } from 'svelte';
import { poe2Build } from '$lib/global';
import type { SpecOp } from '$lib/design/tree';
import type { TreeModel } from './engine';
import { poe2Error, poe2Run, poe2Worker } from './engine_poe2.svelte';
import type { Poe2Calc, Poe2TreeClass, Poe2TreeSpecRow } from './sync_worker2';
import { Poe2TreeCalc } from './engine_poe2_tree_calc.svelte';

const fail = (err: unknown) => poe2Error.set(err instanceof Error ? err.message : String(err));

// TreeTab.lua, PassiveSpecListControl.lua, Build.lua:246-299 (PoB-PoE2) over the open PoE2 build
class Poe2Tree implements TreeModel {
  version = '0_5';
  classes = $state<string[]>([]);
  ascendancies = $state<string[]>(['None']);
  className = $state('');
  ascendancy = $state('None');
  level = $state<number>();
  levelAuto = $state(false);
  specs = $state<string[]>([]);
  active = $state(0);
  specDetails = $state<string[]>([]);
  specTip = $state<TreeModel['specTip']>();
  classConfirm = $state<string>();
  // stat differences, pathing lines, Shift tracing, socket jewels and node power
  readonly calc = new Poe2TreeCalc();

  private classList: Poe2TreeClass[] = [];
  private rows: Promise<Poe2TreeSpecRow[]> | undefined;
  private specHover: number | null = null;

  constructor() {
    $effect(() =>
      poe2Build.subscribe((v) =>
        untrack(() => {
          const calc = v as Partial<Poe2Calc> | undefined;
          this.className = calc?.className ?? '';
          this.ascendancy = calc?.ascendClassName ?? 'None';
          this.level = calc?.level;
          this.levelAuto = !!calc?.tree?.levelAuto;
          this.specs = calc?.treeSpecs?.labels ?? [];
          this.active = (calc?.treeSpecs?.active ?? 1) - 1;
          this.ascendancies = this.ascendanciesOf(this.className);
          this.rows = undefined;
          if (this.specDetails.length) this.readSpecRows();
        })
      )
    );
    void poe2Worker()
      .then((w) => w.TreeClassList())
      .then((list) => {
        this.classList = list;
        this.classes = list.map((c) => c.label);
        this.ascendancies = this.ascendanciesOf(this.className);
      })
      .catch(fail);
  }

  private ascendanciesOf(className: string) {
    return this.classList.find((c) => c.label === className)?.ascendancies ?? ['None'];
  }

  setClass(className: string) {
    const c = this.classList.find((x) => x.label === className);
    if (!c) return;
    void poe2Run(
      (w) => w.ChangeTreeClass(c.classId, 0),
      (r) => r?.calc
    ).then((r) => {
      if (r?.ask) this.classConfirm = className;
    });
  }

  // Build.lua:255-269 (PoB-PoE2)
  answerClass(choice: 'continue' | 'connect' | 'cancel') {
    const c = this.classList.find((x) => x.label === this.classConfirm);
    this.classConfirm = undefined;
    if (!c || choice === 'cancel') return;
    void poe2Run(
      (w) => w.ChangeTreeClass(c.classId, choice === 'continue' ? 1 : 2),
      (r) => r?.calc
    );
  }

  setAscendancy(ascendancy: string) {
    const id = this.ascendancies.indexOf(ascendancy);
    if (id >= 0) void poe2Run((w) => w.ChangeTreeAscendancy(id));
  }

  // Build.lua:202-216 (PoB-PoE2)
  setLevel(level: number) {
    if (level === this.level) return;
    this.level = level;
    void poe2Run((w) => w.SetLevel(level));
  }

  setLevelAuto(auto: boolean) {
    void poe2Run((w) => w.SetLevelAutoMode(auto));
  }

  setSpec(index: number) {
    void poe2Run((w) => w.SetActiveTreeSpec(index + 1));
  }

  editSpecs(op: SpecOp) {
    void poe2Run((w) => w.EditTreeSpecs(op));
  }

  // PassiveSpecListControl.lua:77-85 (PoB-PoE2)
  readSpecRows() {
    void this.specRows().then((rows) => {
      this.specDetails = rows.map((r) => `${r.ascendClassName !== 'None' ? r.ascendClassName : r.className}, ${r.used} points`);
    });
  }

  private specRows(): Promise<Poe2TreeSpecRow[]> {
    this.rows ??= poe2Worker()
      .then((w) => w.TreeSpecRows())
      .catch((err: unknown) => {
        fail(err);
        return [];
      });
    return this.rows;
  }

  // TreeTab.lua:67-119 (PoB-PoE2), without the stat comparison and the respec cost
  hoverSpec(index: number | null) {
    this.specHover = index;
    if (index === null) {
      this.specTip = undefined;
      return;
    }
    void this.specRows().then((rows) => {
      const r = rows[index];
      if (this.specHover !== index || !r) return;
      const lines = [`Class: ${r.className}`, `Ascendancy: ${r.ascendClassName}`, `Points used: ${r.used}`];
      if (r.sockets > 0) lines.push(`Jewel sockets: ${r.sockets}`);
      this.specTip = { index, title: this.specs[index] ?? r.title, lines, footer: `Game Version: ${r.version}`, pending: false, blocks: [] };
    });
  }

  // TreeTab.lua:145-161 (PoB-PoE2)
  reset() {
    void poe2Run((w) => w.TreeReset());
  }
}

export const poe2TreeModel = () => new Poe2Tree();
