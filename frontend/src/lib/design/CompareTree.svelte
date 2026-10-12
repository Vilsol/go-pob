<script lang="ts">
  import Button from './Button.svelte';
  import type { CompareFixture } from './fixtures';

  type Props = {
    tree: CompareFixture['tree'];
    aPoints: number;
    bPoints: number;
    canApply?: boolean;
    onnode: (id: number, action: 'Take' | 'Drop') => void;
    /* CompareTab.lua:1331-1368 */
    oncopytree?: (andUse: boolean) => void;
  };
  let { tree, aPoints, bPoints, canApply = true, onnode, oncopytree }: Props = $props();

  const at = (v: number) => 2 + v * 96;
  const SIZE = { s: 6, n: 11, k: 15, m: 11, j: 11, c: 18, a: 6 } as Record<string, number>;
  const OWNER = { a: 'tone-build-a', b: 'tone-build-b', both: 'tone-text-3' };
  const KIND = { n: 'Notable', k: 'Keystone', m: 'Mastery', j: 'Jewel socket', a: 'Ascendancy' } as Record<string, string>;

  const placed = $derived(tree.nodes.map((n) => ({ ...n, px: n.x === null ? null : at(n.x), py: n.y === null ? null : at(n.y) })));
  const edges = $derived(
    tree.edges.map(([i, j]) => {
      const [p, q] = [placed[i], placed[j]];
      const dx = q.px! - p.px!;
      const dy = q.py! - p.py!;
      return { x: p.px!, y: p.py!, w: Math.hypot(dx, dy), r: (Math.atan2(dy, dx) * 180) / Math.PI, on: p.owner === q.owner ? p.owner : 'both' };
    })
  );
  const shared = $derived(tree.nodes.filter((n) => n.owner === 'both').length);
  const diffs = $derived(
    tree.nodes.filter((n) => n.owner !== 'both' && KIND[n.kind]).sort((x, y) => x.owner.localeCompare(y.owner) || x.name.localeCompare(y.name))
  );
  const small = $derived(tree.nodes.filter((n) => n.owner !== 'both' && !KIND[n.kind]));
  // CompareTab.lua:746-764, 3374-3458
  let overlay = $state(true);
  let q = $state({ overlay: '', a: '', b: '' });
  const hit = (name: string, s: string) => s.trim().length >= 2 && name.toLowerCase().includes(s.trim().toLowerCase());
  const sides = $derived(
    overlay
      ? [{ key: 'overlay' as const, label: 'Tree overlay', owners: ['a', 'b', 'both'] }]
      : [
          { key: 'a' as const, label: 'Tree A', owners: ['a', 'both'] },
          { key: 'b' as const, label: 'Tree B', owners: ['b', 'both'] }
        ]
  );
</script>

<div class="flex min-w-0 gap-3">
  <div class="flex shrink-0 flex-col gap-1.5">
    <label class="field field-inline gap-1 text-12"><input class="check" type="checkbox" bind:checked={overlay} /><span>Overlay comparison</span></label>
    <div class="flex gap-2">
      {#each sides as side (side.key)}
        <div class="flex flex-col gap-1">
          <input class="input h-7 text-12" type="search" aria-label="Search {side.label}" placeholder="Search" bind:value={q[side.key]} />
          <section
            aria-label={side.label}
            class={[
              'relative shrink-0 overflow-hidden rounded-md border border-line bg-tree-ground',
              overlay ? 'size-140 max-[1500px]:size-90' : 'size-90 max-[1500px]:size-70'
            ]}>
            {#each edges as e, i (i)}
              {#if side.owners.includes(e.on)}
                <div
                  class={['tree-edge bg-(--tone) opacity-60', OWNER[e.on]]}
                  style:--x="{e.x}%"
                  style:--y="calc({e.y}% - 1px)"
                  style:--w="{e.w.toFixed(2)}%"
                  style:--r="{e.r.toFixed(1)}deg">
                </div>
              {/if}
            {/each}
            {#each placed as n (n.id)}
              {#if n.px !== null && n.py !== null && side.owners.includes(n.owner)}
                {@const z = SIZE[n.kind] ?? 6}
                <span
                  class={['tree-node bg-(--tone)', OWNER[n.owner], hit(n.name, q[side.key]) && 'outline-2 outline-offset-2 outline-[#f00]']}
                  data-kind={['n', 'k', 'm', 'j', 'c'].includes(n.kind) ? n.kind : 's'}
                  data-match={hit(n.name, q[side.key]) || undefined}
                  title="{n.name} ({n.owner === 'both' ? 'both' : `only ${n.owner.toUpperCase()}`})"
                  style:--x="calc({n.px}% - {z / 2}px)"
                  style:--y="calc({n.py}% - {z / 2}px)"
                  style:--z="{z}px"></span>
              {/if}
            {/each}
            {#if side.key === 'overlay'}
              <div class="absolute bottom-2.5 left-2.5 flex gap-3 rounded border border-line bg-surface-1/92 px-2 py-1.25 text-11 text-text-2">
                <span class="text-build-a">● only A</span><span class="text-build-b">● only B</span><span class="text-text-3">● both</span>
              </div>
            {/if}
          </section>
        </div>
      {/each}
    </div>
  </div>
  <section aria-label="Node differences" class="panel flex min-w-0 grow flex-col gap-1.5 px-3 py-2.5 text-12">
    <div class="flex flex-wrap justify-between gap-2">
      <h3 class="subhead text-12">NODE DIFFERENCES</h3>
      <span class="text-text-3">A {aPoints} pts · B {bPoints} pts · {shared} shared · {small.length} small nodes differ</span>
    </div>
    {#each diffs as d (d.id)}
      <div class="grid grid-cols-[28px_minmax(0,1fr)_120px_64px] items-center gap-1.5 border-b border-surface-2 py-1">
        <span class={['toned font-semibold', OWNER[d.owner]]}>{d.owner.toUpperCase()}</span>
        <span class="truncate" title={d.name}>{d.name}</span>
        <span class="text-text-3">{KIND[d.kind]}</span>
        {#if canApply}<Button size="2xs" class="px-0" onclick={() => onnode(d.id, d.owner === 'a' ? 'Drop' : 'Take')}
            >{d.owner === 'a' ? 'Drop' : 'Take'}</Button
          >{:else}<span></span>{/if}
      </div>
    {/each}
    <span class="text-text-3">“Take” copies B’s node into A (with its path); “Drop” removes A’s node. Each is one undoable step on this build.</span>
    {#if canApply && oncopytree}
      <div class="flex flex-wrap items-center gap-1.5">
        <Button size="xs" title="B's tree joins this build's trees (without its jewels)" onclick={() => oncopytree(false)}>Copy tree</Button>
        <Button size="xs" title="B's tree joins this build's trees and becomes the active one" onclick={() => oncopytree(true)}>Copy and use</Button>
      </div>
    {/if}
  </section>
</div>
