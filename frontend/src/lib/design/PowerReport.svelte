<script lang="ts">
  import Icon from './Icon.svelte';
  import SortHeader from './SortHeader.svelte';
  import type { PowerFilter, PowerSort, ShownPowerRow } from './tree';

  // TreeTab.lua:260-265, PowerReportListControl.lua:58-96
  type Stat = { Stat: string; Label: string };
  type Props = {
    stats: Promise<Stat[]> | Stat[];
    stat: string;
    depth: number;
    custom: string;
    filter: PowerFilter;
    masteries: boolean;
    sort: PowerSort;
    /** the stat's label, the power column's header */
    statLabel: string;
    heat: boolean;
    started: boolean;
    /** undefined while the report is being built */
    rows: ShownPowerRow[] | undefined;
    progress: number;
    ms?: number;
    focus?: number;
    onheat: (on: boolean) => void;
    onstart: () => void;
    onstop: () => void;
    onclose: () => void;
    onhover: (id: number | undefined) => void;
    onpick: (row: ShownPowerRow) => void;
  };
  let {
    stats,
    stat = $bindable(),
    depth = $bindable(),
    custom = $bindable(),
    filter = $bindable(),
    masteries = $bindable(),
    sort = $bindable(),
    statLabel,
    heat,
    started,
    rows,
    progress,
    ms,
    focus,
    onheat,
    onstart,
    onstop,
    onclose,
    onhover,
    onpick
  }: Props = $props();
</script>

<section aria-label="Power report" class="flex max-h-[45%] min-h-0 flex-col gap-1.5 border-b border-line px-4 py-3">
  <div class="flex items-baseline justify-between">
    <h2 class="text-14 font-bold">Power report</h2>
    <button class="btn-close" aria-label="Close power report" onclick={onclose}><Icon name="close" size={14} /></button>
  </div>
  <div class="flex items-center gap-1.5">
    {#await stats then list}
      <select class="select input-sm grow text-12" aria-label="Power stat" bind:value={stat}
        >{#each list as s (s.Stat)}<option value={s.Stat}>{s.Label}</option>{/each}</select>
    {/await}
    <select class="select input-sm text-12" aria-label="Power report depth" title="Limit of node distance to search (lower = faster)" bind:value={depth}
      >{#each [0, 5, 10, 15, -1] as d (d)}<option value={d}>{d === 0 ? 'All' : d === -1 ? 'Custom' : d}</option>{/each}</select>
    <select class="select input-sm text-12" aria-label="Power report filter" bind:value={filter} onchange={() => (sort = 'power')}
      ><option value="unallocated">Show Unallocated</option><option value="clusters">Show Unallocated & Clusters</option><option value="allocated"
        >Show Allocated</option
      ></select>
    <label class="field field-inline gap-1 text-11"
      ><input class="check" type="checkbox" bind:checked={masteries} onchange={() => (sort = 'power')} /><span>Show Masteries</span></label>
  </div>
  {#if depth === -1}
    <input
      class="input input-sm w-17.5 font-mono text-12"
      inputmode="numeric"
      aria-label="Custom node power depth"
      value={custom}
      onchange={(e) => (custom = e.currentTarget.value.replace(/\D/g, ''))} />
  {/if}
  <div class="flex items-center gap-2">
    <label class="field field-inline gap-1 text-12" title="An estimate of each unallocated passive's power for the chosen stat"
      ><input class="check" type="checkbox" checked={heat} onchange={(e) => onheat(e.currentTarget.checked)} /><span>Show node power</span></label>
    {#if !started}
      <button class="btn btn-sm ml-auto px-3" onclick={onstart}>Start</button>
    {:else if !rows}
      <button class="btn btn-sm ml-auto px-3" onclick={onstop}>Stop</button>
    {/if}
  </div>
  {#if !started}
    <span class="note">Choose the stat and depth, then Start: the report calculates every node for it.</span>
  {:else if !rows}
    <div
      role="progressbar"
      aria-label="Building the power report"
      aria-valuemin="0"
      aria-valuemax="100"
      aria-valuenow={Math.round(progress * 100)}
      class="note">
      Building Power Report... ({Math.round(progress * 100)}%)
    </div>
  {:else}
    <span class="note" data-power-ms={ms !== undefined ? Math.round(ms) : undefined}>{rows.length} nodes · Click to focus node on tree</span>
    <div
      role="table"
      aria-label="Node power"
      class="scroll grid min-h-0 grid-cols-[minmax(0,auto)_minmax(0,1fr)_auto_auto_auto] content-start gap-x-3 text-12"
      onmouseleave={() => onhover(undefined)}>
      <div role="row" class="sticky top-0 z-raised col-span-5 grid grid-cols-subgrid border-b border-line bg-surface-1 px-1.5 pb-0.5 text-11 text-text-3">
        {#each [['type', 'Type'], ['name', 'Node Name'], ['power', statLabel], ['points', 'Points'], ['perPoint', 'pp']] as [col, label] (col)}
          {#if col === 'name'}<span role="columnheader">{label}</span>{:else}<SortHeader
              {label}
              active={col === sort}
              dir={col === 'type' || col === 'points' || filter === 'allocated' ? 'asc' : 'desc'}
              align={col === 'type' ? 'start' : 'end'}
              buttonClass="hover:text-text"
              onsort={() => (sort = col as PowerSort)} />{/if}
        {/each}
      </div>
      {#each rows as r (`${r.ID}:${r.Name}`)}
        <div
          role="row"
          tabindex="0"
          class={[
            'col-span-5 grid cursor-pointer grid-cols-subgrid px-1.5 py-1 text-left font-sans text-12 text-text even:bg-bar hover:bg-surface-3 focus-visible:-outline-offset-2',
            r.ID === focus && 'bg-surface-3'
          ]}
          data-power-node={r.ID}
          title={r.Stats?.join('\n')}
          onmouseenter={() => onhover(r.ID)}
          onfocus={() => onhover(r.ID)}
          onclick={() => onpick(r)}
          onkeydown={(e) => {
            if (e.key !== 'Enter' && e.key !== ' ') return;
            e.preventDefault();
            onpick(r);
          }}>
          <span role="cell" class="truncate text-text-2">{r.Type}</span><span role="cell" class="truncate">{r.Name}</span><span
            role="cell"
            class={['text-right font-mono', r.dir === 'up' ? 'text-better' : r.dir === 'down' ? 'text-worse' : '']}>{r.power}</span
          ><span role="cell" class="text-right font-mono">{r.points}</span><span role="cell" class="text-right font-mono text-text-2">{r.perPoint}</span>
        </div>
      {/each}
    </div>
  {/if}
</section>
