<script lang="ts">
  import MetricSelect from './MetricSelect.svelte';
  import Button from './Button.svelte';
  import { tone as toneClass, type Tone } from './tone';
  import { METRICS, type Metric, type MetricId, arrowed, dirTone, metricIndex, num, pctOf, signed } from './compare';
  import type { ItemTip, PowerRow } from './fixtures';
  import FloatingItemTip from './FloatingItemTip.svelte';
  import PobText from './PobText.svelte';

  type Props = {
    power: PowerRow[];
    metric: MetricId;
    metrics?: Metric[];
    a: number;
    b: number;
    busy?: boolean;
    canApply?: boolean;
    onmetric: (m: MetricId) => void;
    onapply: (row: PowerRow) => void;
    /* ComparePowerReportListControl.lua:112-143 */
    rowtip?: (row: PowerRow) => Promise<ItemTip | undefined>;
  };
  let { power, metric, metrics = METRICS, a, b, busy = false, canApply = true, onmetric, onapply, rowtip }: Props = $props();

  let hover = $state.raw<{ row: PowerRow; anchor: HTMLElement; tip?: ItemTip }>();
  const showTip = async (row: PowerRow, anchor: HTMLElement) => {
    if (!rowtip || (row.cat !== 'tree' && row.cat !== 'items')) return;
    hover = { row, anchor };
    const tip = await rowtip(row);
    if (hover?.row === row && tip) hover = { row, anchor, tip };
  };

  const CATS: { id: PowerRow['cat']; label: string }[] = [
    { id: 'tree', label: 'Tree' },
    { id: 'items', label: 'Items' },
    { id: 'gems', label: 'Skill gems' },
    { id: 'supports', label: 'Support gems' },
    { id: 'config', label: 'Config' }
  ];
  const ACTION: Record<PowerRow['cat'], (r: PowerRow) => string> = {
    tree: (r) => r.what.split(' ')[0],
    items: () => 'Equip',
    config: () => 'Use',
    gems: () => '',
    supports: () => ''
  };
  let on = $state<Record<string, boolean>>({ tree: true, items: true, gems: true, supports: true, config: true });
  let sort = $state<'impact' | 'per'>('impact');

  const mi = $derived(metricIndex(metric, metrics));
  const label = $derived(metrics[mi].label);
  const nonzero = $derived(power.filter((p) => p.impact[mi]));
  const per = (p: PowerRow) => (p.points ? p.impact[mi] / p.points : 0);
  const rows = $derived(
    nonzero.filter((p) => on[p.cat]).sort((x, y) => (sort === 'per' ? Math.abs(per(y)) - Math.abs(per(x)) : Math.abs(y.impact[mi]) - Math.abs(x.impact[mi])))
  );
  const max = $derived(Math.max(1, ...rows.map((p) => Math.abs(p.impact[mi]))));
  const total = $derived(rows.reduce((s, p) => s + p.impact[mi], 0));
  const diff = $derived(b - a);
  const GRID = 'grid-cols-[4.375rem_minmax(0,1fr)_7.5rem_11.875rem_3.5rem_6.25rem_4.375rem]';
</script>

<section aria-label="Compare power report" class="panel text-13">
  <div class="flex flex-col gap-2 border-b border-line px-3.5 py-3">
    <div class="flex items-center justify-between gap-2.5">
      <h2 class="text-16 font-semibold">Why is B {diff >= 0 ? 'better' : 'worse'} at {label}?</h2>
      <MetricSelect value={metric} onchange={onmetric} {metrics} />
    </div>
    <div class="flex flex-wrap items-baseline gap-x-4 gap-y-1 text-text-2">
      <span>A <span class="font-mono text-text">{num(a)}</span></span>
      <span>B <span class="font-mono text-text">{num(b)}</span></span>
      <span class={['toned font-mono text-12 font-semibold', toneClass(dirTone(diff))]}>{diff >= 0 ? '▲' : '▼'} B {signed(diff)} ({pctOf(diff, a)})</span>
      <span class="text-text-3"
        >Each row: take B’s version of one thing into A, recalculate A, report the change. Rows with no effect on {label} are hidden ({power.length -
          nonzero.length} hidden).</span>
    </div>
    <div role="group" aria-label="Categories" class="flex flex-wrap items-center gap-1.5">
      <span class="note">Include</span>
      {#each CATS as c (c.id)}
        <label
          class="flex h-7 cursor-pointer items-center gap-1.5 rounded-[14px] border border-line px-2.5 text-12 text-text-2 has-checked:border-accent has-checked:bg-surface-3 has-checked:text-text"
          ><input type="checkbox" class="check size-3.5" bind:checked={on[c.id]} />{c.label}
          <span class="font-mono text-text-3">{nonzero.filter((p) => p.cat === c.id).length}</span></label>
      {/each}
      {#if busy}<span role="status" class="note">Calculating…</span>{/if}
    </div>
  </div>
  <div role="table" aria-label="Power report rows">
    <div role="row" class={['data-row data-head gap-2.5 px-3.5 py-1.5 tracking-caps-tight', GRID]}>
      <span role="columnheader">CATEGORY</span><span role="columnheader">B’S VERSION → INTO A</span><span role="columnheader"
        ><span class="sr-only">Bar</span></span>
      <span role="columnheader" aria-sort={sort === 'impact' ? 'descending' : 'none'} class="flex justify-end"
        ><button class="head-btn" onclick={() => (sort = 'impact')}>{label.toUpperCase()} ON A {sort === 'impact' ? '▼' : ''}</button></span>
      <span role="columnheader" class="text-right">POINTS</span>
      <span role="columnheader" aria-sort={sort === 'per' ? 'descending' : 'none'} class="flex justify-end"
        ><button class="head-btn" onclick={() => (sort = 'per')}>PP {sort === 'per' ? '▼' : ''}</button></span>
      <span role="columnheader"><span class="sr-only">Action</span></span>
    </div>
    {#each rows as p, i (p.cat + p.what + i)}
      {@const v = p.impact[mi]}
      {@const w = (Math.abs(v) / max) * 50}
      <div role="row" class={['data-row gap-2.5 px-3.5 py-1.25 hover:bg-hover-soft', GRID]}>
        <span role="cell" class="text-12 text-text-2">{CATS.find((c) => c.id === p.cat)?.label}</span>
        <span role="cell" class="flex min-w-0 flex-col"
          ><span
            data-pob-color={p.cat === 'items' ? '' : undefined}
            role="button"
            tabindex="0"
            class={['toned truncate', toneClass(p.tone as Tone)]}
            aria-describedby={hover?.row === p && hover.tip ? 'compare-power-tip' : undefined}
            onmouseenter={(e) => void showTip(p, e.currentTarget)}
            onfocus={(e) => void showTip(p, e.currentTarget)}
            onmouseleave={() => (hover = undefined)}
            onblur={() => (hover = undefined)}><PobText text={p.what} /></span
          ><span class="note-sm truncate">{p.detail}</span></span>
        <span
          role="cell"
          aria-hidden="true"
          class="relative h-4 before:absolute before:top-0 before:bottom-0 before:left-1/2 before:w-px before:bg-line-strong before:content-['']"
          ><span
            class={['absolute top-1 left-(--x) h-2 w-(--w) rounded-[1px] bg-(--tone)', toneClass(dirTone(v))]}
            style:--x={v >= 0 ? '50%' : `${(50 - w).toFixed(1)}%`}
            style:--w={`${w.toFixed(1)}%`}></span
          ></span>
        <span role="cell" class={['toned text-right font-mono', toneClass(dirTone(v))]}>{arrowed(v, `${signed(v)} (${pctOf(v, a)})`)}</span>
        <span role="cell" class="text-right font-mono text-text-2">{p.points ?? ''}</span>
        <span role="cell" class="text-right font-mono text-text-2">{p.points ? signed(per(p)) : ''}</span>
        <span role="cell" class="flex justify-end"
          >{#if canApply && ACTION[p.cat](p)}<Button size="2xs" title="Apply B’s version to A (undoable)" onclick={() => onapply(p)}>{ACTION[p.cat](p)}</Button
            >{/if}</span>
      </div>
    {/each}
    <div role="row" class={['data-row gap-2.5 border-b-0 bg-bar px-3.5 py-2 font-semibold', GRID]}>
      <span role="cell" class="text-12 text-text-2">TOTAL</span>
      <span role="cell" class="flex flex-col"
        ><span>Sum of the {rows.length} rows shown</span><span class="note-sm font-normal"
          >Not additive: each row is measured alone. Adopting all of B gives exactly B’s {label}.</span
        ></span>
      <span role="cell"></span>
      <span role="cell" class={['toned text-right font-mono', toneClass(dirTone(total))]}>{arrowed(total, signed(total))}</span>
      <span role="cell"></span><span role="cell"></span><span role="cell"></span>
    </div>
  </div>
</section>
{#if hover?.tip}
  <FloatingItemTip
    anchor={hover.anchor}
    id="compare-power-tip"
    item={hover.row.what}
    base=""
    rarity="normal"
    mods={[]}
    impactTitle=""
    impact={[]}
    tooltip={hover.tip} />
{/if}
