<script lang="ts">
  import Tag from './Tag.svelte';
  import type { GemDiff, GroupPair } from './compare';

  type Props = {
    both: GroupPair[];
    onlyA: GroupPair[];
    onlyB: GroupPair[];
    supportpairs?: (side: 'a' | 'b', group: number) => Promise<[number, number][]>;
  };
  let { both, onlyA, onlyB, supportpairs }: Props = $props();

  // CompareTab.lua:4139-4157, 4411-4427
  let lit = $state.raw<{ group: GroupPair; rows: Set<number> }>();
  let hovering: { g: GroupPair; k: number } | undefined;
  const supporting = (pairs: [number, number][], x: GemDiff, y: GemDiff, at: 'ia' | 'ib') => {
    const [i, j] = [x[at], y[at]];
    if (i === undefined || j === undefined) return false;
    const imbued = (u: GemDiff, v: GemDiff) => !!u.imbued && !v.support;
    return imbued(x, y) || imbued(y, x) || pairs.some(([p, q]) => (p === i && q === j) || (p === j && q === i));
  };
  const light = async (g: GroupPair, k: number) => {
    const sides = [
      { at: 'ia' as const, group: g.ga, side: 'a' as const },
      { at: 'ib' as const, group: g.gb, side: 'b' as const }
    ];
    const rows = new Set([k]);
    const token = (hovering = { g, k });
    lit = undefined;
    for (const s of sides) {
      if (s.group === undefined || g.gems[k][s.at] === undefined) continue;
      const pairs = (await supportpairs?.(s.side, s.group)) ?? [];
      g.gems.forEach((m, j) => {
        if (j !== k && supporting(pairs, g.gems[k], m, s.at)) rows.add(j);
      });
    }
    if (hovering === token) lit = rows.size > 1 ? { group: g, rows } : undefined;
  };

  const sections = $derived([
    { title: `IN BOTH BUILDS · ${both.length}`, tc: 'text-text-2', groups: both },
    { title: `ONLY IN A · ${onlyA.length}`, tc: 'text-build-a', groups: onlyA },
    { title: `ONLY IN B · ${onlyB.length}`, tc: 'text-build-b', groups: onlyB }
  ]);
  const TAG = { same: 'same', level: 'lvl/q', a: 'A only', b: 'B only' };
  const GRID = 'grid-cols-[minmax(0,1fr)_64px_64px]';
</script>

<section aria-label="Skill setups" class="flex flex-col gap-2.5">
  <div class="flex flex-col gap-0.5">
    <h2 class="text-16 font-semibold">Skill setups: what differs</h2>
    <span class="note">Socket groups are paired by the gems they share (best match first); levels are level / quality.</span>
  </div>
  {#each sections as sec (sec.title)}
    {#if sec.groups.length}
      <div class="flex flex-col gap-1.5">
        <h3 class={['subhead mt-1', sec.tc]}>{sec.title}</h3>
        <div class="grid grid-cols-2 gap-2.5">
          {#each sec.groups as g, i (i)}
            <div role="table" aria-label={g.name} class="panel min-w-0 text-12">
              <div role="row" class="flex items-baseline justify-between gap-2 border-b border-line px-3 py-1.75">
                <span role="rowheader" class="truncate text-13 font-semibold">{g.name}</span><span role="cell" class="shrink-0 text-text-3">{g.where}</span>
              </div>
              <div role="row" class={['grid gap-2 px-3 py-1 text-10 tracking-caps-tight text-text-3', GRID]}>
                <span role="columnheader">GEM</span><span role="columnheader" class="text-build-a">A LVL/Q</span><span role="columnheader" class="text-build-b"
                  >B LVL/Q</span>
              </div>
              {#each g.gems as m, k (k)}
                <div
                  role="row"
                  class={['grid items-center gap-2 border-t border-surface-2 px-3 py-0.75', lit?.group === g && lit.rows.has(k) && 'bg-surface-3', GRID]}
                  data-lit={(lit?.group === g && lit.rows.has(k)) || undefined}
                  tabindex="0"
                  onmouseenter={() => void light(g, k)}
                  onfocus={() => void light(g, k)}
                  onmouseleave={() => ((hovering = undefined), (lit = undefined))}
                  onblur={() => ((hovering = undefined), (lit = undefined))}>
                  <span role="cell" class="flex min-w-0 items-center gap-1.5"
                    ><Tag kind={m.status}>{TAG[m.status]}</Tag><span
                      data-pob-color
                      class={['truncate', m.status === 'same' ? 'text-text-3' : 'text-gem']}
                      title={m.gem}>{m.gem}</span
                    ></span>
                  <span role="cell" class={['font-mono', m.status === 'b' ? 'text-text-4' : m.status === 'level' ? 'text-build-a' : 'text-text-2']}>{m.a}</span>
                  <span role="cell" class={['font-mono', m.status === 'a' ? 'text-text-4' : m.status === 'level' ? 'text-build-b' : 'text-text-2']}>{m.b}</span>
                </div>
              {/each}
            </div>
          {/each}
        </div>
      </div>
    {/if}
  {/each}
</section>
