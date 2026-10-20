<script lang="ts">
  import type { ComponentProps } from 'svelte';
  import Tabs from './Tabs.svelte';
  import { panelId, tabId } from './tab_ids';
  import CompareSummary from './CompareSummary.svelte';
  import ComparePower from './ComparePower.svelte';
  import CompareTree from './CompareTree.svelte';
  import CompareItems from './CompareItems.svelte';
  import CompareSkills from './CompareSkills.svelte';
  import CompareConfig from './CompareConfig.svelte';
  import CompareCalcs from './CompareCalcs.svelte';
  import CompareCalcsHeader from './CompareCalcsHeader.svelte';
  import {
    METRICS,
    pairSkills,
    sidesOf,
    summaryOf,
    type CalcsHeaderField,
    type CalcsHeaderSide,
    type CompareView,
    type Metric,
    type MetricId
  } from './compare';
  import type { CompareFixture, ItemTip, PowerRow, SkillsFixture } from './fixtures';
  import type { CompareCalcSection } from '$lib/display/compare_calcs';
  import type { calculator } from '$lib/types';

  type Props = {
    compare: CompareFixture;
    aSkills: SkillsFixture;
    swapped?: boolean;
    view?: CompareView;
    metric?: MetricId;
    metrics?: Metric[];
    calcs?: CompareCalcSection[];
    calcsOnlyDifferences?: boolean;
    calcsNames?: [string, string];
    calcsModRows?: (side: 0 | 1, sd: calculator.ModSectionData) => Promise<calculator.CompareModRow[]>;
    /* CompareTab.lua:4480-4608 */
    calcsHeader?: {
      sides: [CalcsHeaderSide, CalcsHeaderSide];
      oncontrol: (side: 0 | 1, field: CalcsHeaderField, value: number | string | boolean | undefined) => void;
      breakdown?: (side: 0 | 1, name: string) => Promise<calculator.CompareSkillBreakdownRow[]>;
    };
    itemtip?: (side: 'a' | 'b', name: string) => Promise<ItemTip | undefined>;
    itemcards?: (slot: string) => Promise<[string[], string[]]>;
    socketview?: ComponentProps<typeof CompareItems>['socketview'];
    onbuy?: (slot: string) => void;
    powertip?: (row: PowerRow) => Promise<ItemTip | undefined>;
    supportpairs?: (side: 'a' | 'b', group: number) => Promise<[number, number][]>;
    powerBusy?: boolean;
    onswap: () => void;
    onapply: (row: PowerRow) => void;
    onnode: (id: number, action: 'Take' | 'Drop') => void;
    onuseconfig: (key: string) => void;
    oncopyconfig: () => void;
    oncopytree?: (andUse: boolean) => void;
    onitem?: (slot: string, equip: boolean) => void;
    onconfigset?: (side: 'a' | 'b', key: string, value: boolean | number | string | null) => void;
    /* CompareTab.lua:923-933 */
    configShowAll?: boolean;
    onconfigshowall?: () => void;
  };
  let {
    compare,
    aSkills,
    swapped = false,
    view = $bindable('summary'),
    metric = $bindable('dps'),
    metrics = METRICS,
    calcs,
    calcsOnlyDifferences = $bindable(true),
    calcsNames = ['A', 'B'],
    calcsModRows,
    itemtip,
    itemcards,
    socketview,
    onbuy,
    powertip,
    supportpairs,
    calcsHeader,
    powerBusy = false,
    onswap,
    onapply,
    onnode,
    onuseconfig,
    oncopyconfig,
    oncopytree,
    onitem,
    configShowAll = false,
    onconfigshowall,
    onconfigset
  }: Props = $props();

  const viewsId = $props.id();
  const VIEWS: { id: CompareView; label: string }[] = [
    { id: 'summary', label: 'Summary' },
    { id: 'power', label: 'Power report' },
    { id: 'tree', label: 'Tree' },
    { id: 'items', label: 'Items' },
    { id: 'skills', label: 'Skills' },
    { id: 'calcs', label: 'Calcs' },
    { id: 'config', label: 'Config' }
  ];
  const s = $derived(sidesOf(compare, aSkills, swapped));
  const summary = $derived(summaryOf(s.A, s.B));
  const skills = $derived(pairSkills(s.A.skills, s.B.skills));
  const mA = $derived(s.A.metrics[metric] ?? 0);
  const mB = $derived(s.B.metrics[metric] ?? 0);
  const line = (x: typeof s.A) => `${x.className} · ${x.ascendClassName} · level ${x.level} · ${x.points} points · config ${x.configSet}`;
</script>

<section aria-label="A versus B" class="grid shrink-0 grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-stretch overflow-hidden rounded-lg border border-line">
  <div
    class="flex min-w-0 flex-col gap-0.5 px-3.5 py-2.5 [background:linear-gradient(90deg,_color-mix(in_srgb,_var(--color-build-a)_22%,_transparent),_color-mix(in_srgb,_var(--color-build-a)_4%,_transparent))]">
    <span class="text-11 font-semibold tracking-caps text-build-a">A · {swapped ? 'COMPARISON' : 'THIS BUILD'}</span>
    <span class="max-w-full min-w-0 truncate text-18 leading-6 font-semibold max-phone:text-14" title={s.A.name}>{s.A.name}</span>
    <span class="max-w-full text-12 leading-[1.0625rem] text-text-2 [overflow-wrap:anywhere] max-phone:text-11">{line(s.A)}</span>
  </div>
  <div class="flex flex-col items-center justify-center gap-1 bg-surface-1 px-3.5 max-phone:px-1.5">
    <span class="text-20 font-bold tracking-caps text-text">VS</span>
    <button class="btn btn-2xs" aria-label="Swap A and B" title="Swap A and B" onclick={onswap}>⇄ Swap</button>
  </div>
  <div
    class="flex min-w-0 flex-col items-end gap-0.5 px-3.5 py-2.5 text-right [background:linear-gradient(270deg,_color-mix(in_srgb,_var(--color-build-b)_22%,_transparent),_color-mix(in_srgb,_var(--color-build-b)_4%,_transparent))]">
    <span class="text-11 font-semibold tracking-caps text-build-b">B · {swapped ? 'THIS BUILD' : 'COMPARISON'}</span>
    <span class="max-w-full min-w-0 truncate text-18 leading-6 font-semibold max-phone:text-14" title={s.B.name}>{s.B.name}</span>
    <span class="max-w-full text-12 leading-[1.0625rem] text-text-2 [overflow-wrap:anywhere] max-phone:text-11">{line(s.B)}</span>
  </div>
</section>
<div class="flex min-w-0 shrink-0 flex-wrap items-center gap-2.5">
  <div class="max-phone:w-full max-phone:overflow-x-auto">
    <Tabs id={viewsId} variant="segmented" label="Compare views" tabs={VIEWS} selected={view} onchange={(id) => (view = id as CompareView)} />
  </div>
  <div class="grow"></div>
  <span class="note min-w-0 truncate" title={summary.tally}>{summary.tally}</span>
</div>
<div
  role="tabpanel"
  tabindex="0"
  id={panelId(viewsId)}
  aria-labelledby={tabId(viewsId, view)}
  class="scroll flex flex-[1_1_0] flex-col gap-2.5 max-phone:overflow-x-auto">
  {#if view === 'summary'}
    <CompareSummary
      groups={summary.groups}
      power={s.A.power}
      {metric}
      {metrics}
      better={mB > mA}
      onmetric={(m: MetricId) => (metric = m)}
      onpower={() => (view = 'power')} />
  {:else if view === 'power'}
    <ComparePower
      power={s.A.power}
      {metric}
      {metrics}
      a={mA}
      b={mB}
      busy={powerBusy}
      canApply={!swapped}
      onmetric={(m: MetricId) => (metric = m)}
      {onapply}
      rowtip={powertip} />
  {:else if view === 'tree'}
    <CompareTree tree={s.tree} aPoints={s.A.points} bPoints={s.B.points} canApply={!swapped} {onnode} {oncopytree} />
  {:else if view === 'items'}
    <CompareItems
      items={s.items}
      jewels={s.jewels}
      onitem={swapped ? undefined : onitem}
      {itemtip}
      cards={itemcards}
      socketview={swapped ? undefined : socketview}
      onbuy={swapped ? undefined : onbuy} />
  {:else if view === 'skills'}
    <CompareSkills both={skills.both} onlyA={skills.onlyA} onlyB={skills.onlyB} {supportpairs} />
  {:else if view === 'calcs'}
    {#if calcsHeader}<CompareCalcsHeader {...calcsHeader} />{/if}
    <CompareCalcs sections={calcs} bind:onlyDifferences={calcsOnlyDifferences} nameA={calcsNames[0]} nameB={calcsNames[1]} modrows={calcsModRows} />
  {:else}
    <CompareConfig
      config={s.config}
      setA={s.A.configSet}
      setB={s.B.configSet}
      {metric}
      {metrics}
      baseA={mA}
      baseB={mB}
      canApply={!swapped}
      onmetric={(m: MetricId) => (metric = m)}
      onuse={onuseconfig}
      oncopy={oncopyconfig}
      showAll={configShowAll}
      onshowall={onconfigshowall}
      onset={onconfigset} />
  {/if}
</div>
