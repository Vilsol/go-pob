import { untrack } from 'svelte';
import { goto } from '$app/navigation';
import { poe2Build } from '$lib/global';
import type { StatCompareSection } from '$lib/skill_tree/stat_compare';
import { poe2Error, poe2Worker } from './engine_poe2.svelte';
import type { Poe2Item, Poe2NodePower, Poe2PowerProgress, Poe2PowerRow } from './sync_worker2';

const fail = (err: unknown) => poe2Error.set(err instanceof Error ? err.message : String(err));

/* TreePage's `diff`: the hovered node's stat difference blocks, `empty` when none has a difference */
export type Poe2NodeDiff = {
  node: number;
  pending: boolean;
  empty?: string;
  blocks: { header: string; sections: StatCompareSection[]; calcMs?: number }[];
};

/* the part of PoB-PoE2's node tooltip below the stat differences, and an allocated socket's jewel */
export type Poe2NodeTipInfo = {
  node: number;
  pathLines: string[];
  distanceToStart: number;
  granted: boolean;
  jewel?: { item: Poe2Item; slot: string; enabled: boolean };
};

// a power run steps for this long per worker call, so hovers and clicks interleave with it
const POWER_STEP_MS = 30;
// and sends the node powers at most this often while running
const POWER_PARTIAL_MS = 500;

// PassiveTreeView.lua:321-350, 1704-1727, 1967-2053, TreeTab.lua:197-325, 924-1048 (PoB-PoE2): the tree's
// calculation-backed parts over the open PoE2 build. The heat map is drawn by the renderer from `power`.
export class Poe2TreeCalc {
  diff = $state<Poe2NodeDiff>();
  tip = $state<Poe2NodeTipInfo>();

  /* TreeTab.lua:197-265 */
  powerStats = $state<{ Stat: string; Label: string }[]>([]);
  powerOn = $state(false);
  powerStat = $state('');
  powerDepth = $state(0);
  power = $state.raw<Poe2NodePower>();
  powerProgress = $state<Poe2PowerProgress>();
  /* TreeTab.lua:932-1048: the report's rows once the run finishes; undefined while it runs */
  powerRows = $state.raw<Poe2PowerRow[]>();

  private hoverSeq = 0;
  private powerSeq = 0;

  constructor() {
    $effect(() =>
      poe2Build.subscribe(() =>
        untrack(() => {
          // CalcsTab.lua:488: every recalculation rebuilds the power
          if (this.powerOn) void this.runPower();
          this.diff = undefined;
          this.tip = undefined;
        })
      )
    );
    $effect(() => () => {
      this.powerSeq++;
    });
  }

  // TreeTab.lua:260-265: the stat list, read when the power controls first show
  async readPowerStats() {
    if (this.powerStats.length) return;
    const s = await (await poe2Worker()).TreePowerStats();
    if (this.powerStats.length) return;
    this.powerStats = s;
    this.powerStat = s[0]?.Stat ?? '';
  }

  // PassiveTreeView.lua:1967-2008: null leaves the node; trace is the Shift-traced path while tracing
  hoverNode(id: number | null, trace?: number[]) {
    const seq = ++this.hoverSeq;
    if (id === null) {
      this.diff = undefined;
      this.tip = undefined;
      return;
    }
    this.diff = { node: id, pending: true, blocks: [] };
    void poe2Worker()
      .then(async (w) => {
        const [cmp, tip] = await Promise.all([w.TreeNodeCompare(id, trace), w.TreeNodeTip(id, trace)]);
        if (seq !== this.hoverSeq) return;
        const blocks = cmp.blocks.filter((b) => b.sections.length);
        this.diff = { node: id, pending: false, blocks, empty: blocks.length ? undefined : cmp.noChange };
        this.tip = { node: id, ...tip };
        const jewel = await this.socketJewel(id);
        if (seq === this.hoverSeq && jewel && this.tip) this.tip = { ...this.tip, jewel };
      })
      .catch((err: unknown) => {
        if (seq === this.hoverSeq) this.diff = undefined;
        fail(err);
      });
  }

  // PassiveTreeView.lua:321-350: one Shift hover; the new trace path and whether the node is highlighted
  async traceNode(trace: number[], id: number) {
    return (await poe2Worker()).TreeTraceNode(trace, id);
  }

  // PassiveTreeView.lua:1704-1722, ItemsTab.lua:1763-1765: an allocated socket's jewel, its slot, and whether the
  // slot lists a jewel to choose (ItemSlotControl.lua:32-34)
  private async socketJewel(id: number): Promise<Poe2NodeTipInfo['jewel']> {
    const page = await (await poe2Worker()).ItemsPage();
    const slot = page.slots.find((s) => s.nodeId === id);
    if (!slot) return undefined;
    const item = page.items.find((it) => it.id === slot.itemId);
    return item ? { item, slot: slot.name, enabled: slot.choices.length > 0 } : undefined;
  }

  // PassiveTreeView.lua:565-572: right-clicking an allocated socket opens the Items page on its slot
  openSocket(id: number) {
    void goto(`/items?socket=${id}`);
  }

  // TreeTab.lua:198-205
  setPowerOn(on: boolean) {
    this.powerOn = on;
    if (on) void this.runPower();
    else this.stopPower();
  }

  // TreeTab.lua:208-254, 924-930
  setPowerStat(stat: string) {
    this.powerStat = stat;
    if (this.powerOn) void this.runPower();
  }

  setPowerDepth(depth: number) {
    const old = this.powerDepth;
    this.powerDepth = depth;
    // TreeTab.lua:230-236: only a deeper limit rebuilds
    if (this.powerOn && old !== 0 && (depth === 0 || depth > old)) void this.runPower();
  }

  stopPower() {
    this.powerSeq++;
    this.power = undefined;
    this.powerProgress = undefined;
    this.powerRows = undefined;
  }

  private async runPower() {
    const seq = ++this.powerSeq;
    this.powerRows = undefined;
    try {
      const w = await poe2Worker();
      this.powerProgress = await w.TreePowerStart(this.powerStat, this.powerDepth);
      let sent = performance.now();
      while (seq === this.powerSeq) {
        const partial = performance.now() - sent >= POWER_PARTIAL_MS;
        const r = await w.TreePowerStep(POWER_STEP_MS, partial);
        if (seq !== this.powerSeq) return;
        this.powerProgress = r.progress;
        if (r.power) {
          this.power = r.power;
          sent = performance.now();
        }
        if (r.progress.finished) {
          this.powerRows = await w.TreePowerReport();
          return;
        }
      }
    } catch (err) {
      // a recalculation started the next run, which owns the builder now
      if (seq === this.powerSeq) fail(err);
    }
  }
}
