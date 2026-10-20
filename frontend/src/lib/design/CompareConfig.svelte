<script lang="ts">
  import Select from './Select.svelte';
  import Input from './Input.svelte';
  import Checkbox from './Checkbox.svelte';
  import SortHeader from './SortHeader.svelte';
  import SearchField from './SearchField.svelte';
  import MetricSelect from './MetricSelect.svelte';
  import Button from './Button.svelte';
  import PobText from './PobText.svelte';
  import { tone as toneClass } from './tone';
  import { pobPlain } from './pob_text';
  import { METRICS, type Metric, type MetricId, arrowed, dirTone, metricIndex, pctOf, signed } from './compare';
  import type { ConfigKind, ConfigValue } from './fixtures';

  // CompareTab.lua:2021-2197, 4907-4997
  type Row = {
    key: string;
    section: string;
    label: string;
    a: string;
    b: string;
    diff?: boolean;
    impactA: number[];
    impactB: number[];
    kind?: ConfigKind;
    options?: { value: string | number; label: string }[];
    aValue?: ConfigValue;
    bValue?: ConfigValue;
  };
  type Props = {
    config: Row[];
    setA: string;
    setB: string;
    metric: MetricId;
    metrics?: Metric[];
    baseA: number;
    baseB: number;
    canApply?: boolean;
    onmetric: (m: MetricId) => void;
    onuse: (key: string) => void;
    oncopy: () => void;
    /* CompareTab.lua:923-933 */
    showAll?: boolean;
    onshowall?: () => void;
    /* CompareTab.lua:1128-1160 */
    onset?: (side: 'a' | 'b', key: string, value: boolean | number | string | null) => void;
  };
  let {
    config,
    setA,
    setB,
    metric,
    metrics = METRICS,
    baseA,
    baseB,
    canApply = true,
    onmetric,
    onuse,
    oncopy,
    showAll = false,
    onshowall,
    onset
  }: Props = $props();

  let sort = $state<'a' | 'b'>('a');
  let q = $state('');
  const mi = $derived(metricIndex(metric, metrics));
  const sections = $derived.by(() => {
    const out: { name: string; rows: Row[]; diffs: number }[] = [];
    const want = q.trim().toLowerCase();
    for (const c of config) {
      if (want && !pobPlain(c.label).toLowerCase().includes(want)) continue;
      let s = out.find((x) => x.name === c.section);
      if (!s) out.push((s = { name: c.section, rows: [], diffs: 0 }));
      s.rows.push(c);
      if (c.diff !== false) s.diffs++;
    }
    for (const s of out) {
      s.rows.sort(
        (x, y) =>
          Number(y.diff !== false) - Number(x.diff !== false) ||
          Math.abs((sort === 'a' ? y.impactA : y.impactB)[mi]) - Math.abs((sort === 'a' ? x.impactA : x.impactB)[mi])
      );
    }
    return out;
  });
  // CompareTab.lua:2138-2168
  const SECTION_MIN = 760;
  let width = $state(0);
  const columns = $derived.by(() => {
    const n = Math.max(1, Math.floor(width / SECTION_MIN));
    const cols = Array.from({ length: n }, () => ({ h: 0, sections: [] as typeof sections }));
    for (const s of sections) {
      const col = cols.reduce((best, c) => (c.h < best.h ? c : best), cols[0]);
      col.sections.push(s);
      col.h += s.rows.length + 2;
    }
    return cols.filter((c, i) => i === 0 || c.sections.length);
  });
  const GRID = $derived(
    (columns.length > 1 ? 'grid-cols-[minmax(0,1fr)_88px_88px_160px_160px_88px]' : 'grid-cols-[minmax(0,1fr)_110px_110px_170px_170px_110px]') +
      ' max-phone:grid-cols-2 max-phone:gap-y-1'
  );
  const numberOf = (text: string) => (text.trim() === '' || Number.isNaN(Number(text)) ? null : Number(text));
</script>

{#snippet control(c: Row, side: 'a' | 'b')}
  {@const value = side === 'a' ? c.aValue : c.bValue}
  {@const name = `${pobPlain(c.label)} (${side === 'a' ? 'A' : 'B'})`}
  {#if !onset || !c.kind}
    <span class="truncate font-mono" title={side === 'a' ? c.a : c.b}>{side === 'a' ? c.a : c.b}</span>
  {:else if c.kind === 'check'}
    <Checkbox aria-label={name} checked={!!value} onchange={(e: Event & { currentTarget: HTMLInputElement }) => onset(side, c.key, e.currentTarget.checked)} />
  {:else if c.kind === 'number'}
    <Input
      size="xs"
      class="h-6 w-full px-1.5 font-mono"
      inputmode="decimal"
      aria-label={name}
      value={(value ?? '') as string | number}
      onchange={(e: Event & { currentTarget: HTMLInputElement }) => onset(side, c.key, numberOf(e.currentTarget.value))} />
  {:else}
    <Select
      size="xs"
      class="h-6 w-full"
      aria-label={name}
      value={String(value ?? c.options?.[0]?.value ?? '')}
      onchange={(e: Event & { currentTarget: HTMLSelectElement }) => {
        const o = c.options?.find((x) => String(x.value) === e.currentTarget.value);
        if (o) onset(side, c.key, o.value);
      }}
      >{#each c.options ?? [] as o (o.value)}<option value={String(o.value)}>{pobPlain(o.label)}</option>{/each}</Select>
  {/if}
{/snippet}

<div class="flex flex-col gap-2.5">
  <div class="flex flex-wrap items-center gap-2">
    {#if canApply}<Button size="sm" onclick={oncopy}>Copy Config from Compare Build</Button>{/if}
    {#if onshowall}<Button size="sm" onclick={onshowall}>{showAll ? 'Hide Ineligible Configurations' : 'Show All Configurations'}</Button>{/if}
    <MetricSelect value={metric} onchange={onmetric} {metrics} />
    <SearchField size="sm" class="w-55 px-2 text-12" placeholder="Search options" label="Search config options" bind:value={q} />
    <span class="note">A config set: {setA} · B config set: {setB} · differences first, then the options either build uses</span>
  </div>
  <section aria-label="Config differences" class="panel text-12">
    <div role="table" aria-label="Config options">
      <div class="flex items-start" bind:clientWidth={width}>
        {#each columns as col, k (k)}
          <div class={['min-w-0 flex-1', k > 0 && 'border-l border-line']}>
            {#if k === 0}
              <div role="row" class={['data-row data-head gap-2.5 py-1.5 text-10 tracking-caps', GRID]}>
                <span role="columnheader" class="max-phone:hidden">OPTION</span><span role="columnheader" class="text-build-a">A’S VALUE</span><span
                  role="columnheader"
                  class="text-build-b">B’S VALUE</span>
                <SortHeader label="A WITH B’S VALUE" active={sort === 'a'} align="end" onsort={() => (sort = 'a')} />
                <SortHeader label="B WITH A’S VALUE" active={sort === 'b'} align="end" onsort={() => (sort = 'b')} />
                <span role="columnheader" class="max-phone:hidden"><span class="sr-only">Action</span></span>
              </div>
            {:else}
              <div aria-hidden="true" class={['data-row data-head gap-2.5 py-1.5 text-10 tracking-caps', GRID]}>
                <span>OPTION</span><span class="text-build-a">A’S VALUE</span><span class="text-build-b">B’S VALUE</span>
                <span class="text-right">A WITH B’S VALUE {sort === 'a' ? '▼' : ''}</span><span class="text-right"
                  >B WITH A’S VALUE {sort === 'b' ? '▼' : ''}</span>
                <span></span>
              </div>
            {/if}
            {#each col.sections as s (s.name)}
              <div role="rowgroup" aria-label={s.name} class="mx-1.5 my-2.5 rounded-sm border border-line-strong">
                <!-- CompareTab.lua:4953-4964 -->
                <div aria-hidden="true" class="-mt-2 ml-1.5 w-fit bg-surface-1 px-1 text-13 text-text">{s.name}{s.diffs ? `  (${s.diffs} diff)` : ''}</div>
                {#each s.rows as c (c.key)}
                  {@const va = c.impactA[mi]}
                  {@const vb = c.impactB[mi]}
                  <div
                    role="row"
                    data-diff={c.diff !== false}
                    class={['data-row gap-2.5 py-1.25', GRID, c.diff !== false ? 'border-l-2 border-warn-line bg-alt' : 'border-l-2 border-transparent']}>
                    <span role="cell" class="min-w-0 max-phone:col-start-1 max-phone:row-start-1"
                      ><span class="sr-only">{c.section} ·</span> <PobText text={c.label} /></span>
                    <span role="cell" class="min-w-0">{@render control(c, 'a')}</span>
                    <span role="cell" class="min-w-0">{@render control(c, 'b')}</span>
                    {#if c.diff === false}
                      <span role="cell" class="text-right text-text-3 max-phone:hidden">same</span><span
                        role="cell"
                        class="text-right text-text-3 max-phone:hidden">same</span
                      ><span role="cell" class="max-phone:hidden"></span>
                    {:else}
                      <span role="cell" class={['toned text-right font-mono', toneClass(dirTone(va))]}
                        >{arrowed(va, `${signed(va)} (${pctOf(va, baseA)})`)}</span>
                      <span role="cell" class={['toned text-right font-mono', toneClass(dirTone(vb))]}
                        >{arrowed(vb, `${signed(vb)} (${pctOf(vb, baseB)})`)}</span>
                      <span role="cell" class="flex justify-end max-phone:col-start-2 max-phone:row-start-1"
                        >{#if canApply}<Button size="2xs" onclick={() => onuse(c.key)}>Use B’s value</Button>{/if}</span>
                    {/if}
                  </div>
                {/each}
              </div>
            {/each}
          </div>
        {/each}
      </div>
      {#if !sections.length}
        <!-- CompareTab.lua:4991-4994 -->
        <p class="px-3 py-2"><PobText text="^x33FF77No configuration options to display." /></p>
      {/if}
    </div>
    <div class="note-sm px-3 py-1.5">
      Each impact is measured alone on the selected metric ({metrics[mi].label}). Blue ▲ = the receiving build gets better, orange ▼ = worse.
    </div>
  </section>
</div>
