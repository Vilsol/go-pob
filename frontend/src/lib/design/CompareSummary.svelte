<script lang="ts">
  import EmptyState from './EmptyState.svelte';
  import ButterflyRow from './ButterflyRow.svelte';
  import Ruler from './Ruler.svelte';
  import Button from './Button.svelte';
  import MetricSelect from './MetricSelect.svelte';
  import PobText from './PobText.svelte';
  import { pobPlain } from './pob_text';
  import { tone as toneClass } from './tone';
  import { METRICS, type Metric, type MetricId, type SummaryRow, arrowed, dirTone, metricIndex, signed } from './compare';
  import type { PowerRow } from './fixtures';

  type Props = {
    groups: { title: string; rows: SummaryRow[] }[];
    power: PowerRow[];
    metric: MetricId;
    metrics?: Metric[];
    better: boolean;
    onmetric: (m: MetricId) => void;
    onpower: () => void;
  };
  let { groups, power, metric, metrics = METRICS, better, onmetric, onpower }: Props = $props();

  const mi = $derived(metricIndex(metric, metrics));
  const nonzero = $derived(power.filter((p) => p.impact[mi]));
  const top = $derived([...nonzero].sort((x, y) => Math.abs(y.impact[mi]) - Math.abs(x.impact[mi])).slice(0, 5));
  const CATS = { items: 'Item', tree: 'Tree', config: 'Config', gems: 'Gem', supports: 'Support' };
</script>

<div class="flex min-w-0 shrink-0 items-start gap-3 max-phone:flex-col max-phone:items-stretch">
  <section aria-label="Stat comparison" class="panel min-w-0 flex-[1_1_0]">
    {#each groups as g (g.title + g.rows[0].key)}
      <div role="group" aria-label={g.title}>
        <Ruler title={g.title} />
        {#each g.rows as r (r.key)}
          <div role="group" aria-label="{r.label}: A {r.a}, B {r.b}{r.winner ? `, ${r.winner.toUpperCase()} better` : ', equal'}">
            <ButterflyRow label={r.label} a={r.a} b={r.b} lead={r.lead} winner={r.winner} pA={r.pA} pB={r.pB} diff={r.diff} diffBetter={r.diffBetter} />
          </div>
        {/each}
      </div>
    {/each}
    <div class="note-sm px-4 py-2.5">
      Track = the larger of the two values. The better build’s bar is solid to its value; on the other side, its own bar is followed by the shortfall, striped
      in the better build’s colour. Pill = winner and its lead over the other build: % of the other’s value, or percentage points for percent stats; “=” when
      equal. The winner is the better build, not the larger number: for costs, lower wins.
    </div>
  </section>
  <aside
    aria-label="Biggest differences"
    class="panel flex max-h-full min-w-65 flex-[0_1_25rem] flex-col max-phone:min-w-0 max-phone:flex-none gap-1.5 px-3 py-2.5 text-12">
    <div class="flex items-center justify-between gap-2">
      <h3 class="subhead">WHY B IS {better ? 'BETTER' : 'WORSE'}: TOP 5</h3>
      <MetricSelect value={metric} onchange={onmetric} {metrics} />
    </div>
    {#each top as p, i (i)}
      <div class="grid grid-cols-[3.25rem_minmax(0,1fr)_6.875rem] items-baseline gap-2 border-b border-surface-2 py-0.75">
        <span class="text-text-3">{CATS[p.cat]}</span><span class="truncate" title={pobPlain(p.what)}><PobText text={p.what} /></span><span
          class={['toned text-right font-mono', toneClass(dirTone(p.impact[mi]))]}>{arrowed(p.impact[mi], signed(p.impact[mi]))}</span>
      </div>
    {:else}
      <EmptyState text="No single difference moves this metric." />
    {/each}
    <Button size="sm" class="h-7 self-start" onclick={onpower}>Full power report ({nonzero.length} differences) →</Button>
  </aside>
</div>
