<script lang="ts">
  import EmptyState from './EmptyState.svelte';
  import PobText from './PobText.svelte';
  import Checkbox from './Checkbox.svelte';
  import { sectionTone } from './calcs';
  import { tone as toneClass } from './tone';
  import { pobPlain } from './pob_text';
  import TooltipLines from './TooltipLines.svelte';
  import { besideAnchor } from './items';
  import { modTipLines, type CompareCalcRow, type CompareCalcSection } from '$lib/display/compare_calcs';
  import type { calculator } from '$lib/types';

  // CompareTab.lua:4655-4902
  // CompareTab.lua:720-725
  type Props = {
    sections?: CompareCalcSection[];
    onlyDifferences?: boolean;
    nameA?: string;
    nameB?: string;
    modrows?: (side: 0 | 1, sd: calculator.ModSectionData) => Promise<calculator.CompareModRow[]>;
  };
  let { sections, onlyDifferences = $bindable(true), nameA = 'A', nameB = 'B', modrows }: Props = $props();

  // CompareCalcsHelpers.lua:167-313
  type Line = { Text: string; Separator: boolean };
  let hover = $state.raw<{ row: CompareCalcRow; anchor: HTMLElement; lines: Line[] }>();
  const text = (Text: string): Line => ({ Text, Separator: false });
  const SEP: Line = { Text: '', Separator: true };
  const show = async (row: CompareCalcRow, anchor: HTMLElement) => {
    const [label, ...bd] = row.tip ?? [row.label];
    const head = [text(`^7${label}`), SEP, ...bd.map(text), ...(bd.length ? [SEP] : [])];
    hover = { row, anchor, lines: head };
    if (!row.mods?.length || !modrows) return;
    const lines = [...head];
    for (const m of row.mods) {
      const [p, c] = await Promise.all([modrows(0, m.sd), modrows(1, m.sd)]);
      const sub = modTipLines(m.label, p, c, nameA, nameB);
      if (sub.length) lines.push(...sub.map(text), SEP);
    }
    if (hover?.row === row) hover = { row, anchor, lines };
  };
  const tipped = (row: CompareCalcRow) => !!row.tip || !!row.mods?.length;
  const GRID = 'grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)_minmax(0,1fr)]';
</script>

<section aria-label="Calcs" class="flex flex-col gap-2.5">
  <div class="flex flex-wrap items-center gap-3">
    <h2 class="text-16 font-semibold">Calculations: A against B</h2>
    <Checkbox bind:checked={onlyDifferences}>Show only differences</Checkbox>
  </div>
  {#if !sections}
    <EmptyState text="Calculating…" />
  {:else if !sections.length}
    <EmptyState text="No section differs." />
  {:else}
    <div class="columns-[21.25rem] gap-2.5">
      {#each sections as s, i (i)}
        <div role="table" aria-label={pobPlain(s.blocks[0]?.label ?? s.id)} class="mb-2.5 break-inside-avoid rounded border border-line bg-surface-1 text-11">
          <div role="row" class={['grid gap-2 border-b border-line px-2 py-0.5 text-10 tracking-caps-tight', GRID]}>
            <span role="columnheader" class={['toned font-bold uppercase', toneClass(sectionTone({ color: s.color, applicable: true }))]}>{s.id}</span><span
              role="columnheader"
              class="text-right text-build-a">A</span
            ><span role="columnheader" class="text-right text-build-b">B</span>
          </div>
          {#each s.blocks as b, j (j)}
            <div role="row" class={['grid gap-2 border-t border-line-faint px-2 pt-1 pb-0.5 font-semibold text-text-3 first:border-t-0', GRID]}>
              <span role="rowheader"><PobText text={b.label} />:</span>
              {#if b.extra}
                <span role="cell" class="text-right font-mono font-normal"><PobText text={b.extra.a} /></span><span
                  role="cell"
                  class="text-right font-mono font-normal"><PobText text={b.extra.b} /></span>
              {/if}
            </div>
            {#each b.rows as r, k (k)}
              <div
                role="row"
                tabindex={tipped(r) ? 0 : undefined}
                class={['grid gap-2 px-2 py-px', tipped(r) && 'hover:bg-surface-2 focus-visible:bg-surface-2', GRID]}
                aria-describedby={hover?.row === r ? 'compare-calcs-tip' : undefined}
                onmouseenter={(e) => tipped(r) && show(r, e.currentTarget)}
                onfocus={(e) => tipped(r) && show(r, e.currentTarget)}
                onmouseleave={() => hover?.row === r && (hover = undefined)}
                onblur={() => hover?.row === r && (hover = undefined)}>
                <span role="rowheader" class="truncate text-right text-text-2" title={pobPlain(r.label)}><PobText text={r.label} /></span>
                <span role="cell" class="truncate text-right font-mono"><PobText text={r.a} /></span>
                <span role="cell" class={['truncate text-right font-mono', r.a !== r.b && 'text-build-b']}><PobText text={r.b} /></span>
              </div>
            {/each}
          {/each}
        </div>
      {/each}
    </div>
  {/if}
</section>
{#if hover}
  <div
    id="compare-calcs-tip"
    role="tooltip"
    class="tooltip pointer-events-none fixed top-(--y) left-(--x) z-float max-w-[640px] rounded-md px-3 py-1.5 whitespace-pre shadow-tip"
    use:besideAnchor={hover.anchor}>
    <TooltipLines lines={hover.lines} />
  </div>
{/if}
