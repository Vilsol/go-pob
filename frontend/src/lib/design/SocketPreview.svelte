<script lang="ts">
  import { ascendancyOffsets, groupCentre, nodeCentre, type Point } from '$lib/skill_tree/geometry';
  import type { Node, Tree } from '$lib/skill_tree/types';
  import { float, raise } from './floating';

  // ItemSlotControl.lua:132-149
  // ItemSlotControl.lua:139-146
  type Radius = { inner: number; outer: number; col: string; nodes: number[] };
  type Props = { tree: Tree; node: number; radii: Radius[]; alloc: Set<number>; label: string; anchor?: HTMLElement };
  let { tree, node, radii, alloc, label, anchor }: Props = $props();

  const W = 308;
  const H = 280;
  const colour = (code: string) => (/^\^x[0-9A-Fa-f]{6}$/.test(code) ? `#${code.slice(2)}` : 'var(--color-accent)');
  const view = $derived.by(() => {
    const offsets = ascendancyOffsets(tree);
    const nodes = Object.values(tree.nodes);
    const centre = nodes.find((n) => n.skill === node);
    const c = centre && nodeCentre(tree, offsets, centre);
    if (!c) return undefined;
    const reach = Math.max(1200, ...radii.map((r) => r.outer)) * 1.1;
    const scale = Math.min(W, H) / 2 / reach;
    const at = (p: Point) => ({ x: W / 2 + (p.x - c.x) * scale, y: H / 2 + (p.y - c.y) * scale });
    const inRadius = new Map<number, string>();
    for (const r of radii) for (const id of r.nodes) if (!inRadius.has(id)) inRadius.set(id, colour(r.col));
    const drawn = new Map<number, { n: Node; s: Point }>();
    const dots = nodes.flatMap((n) => {
      if (n.skill === undefined || n.isProxy || n.ascendancyName || n.classStartIndex !== undefined) return [];
      const p = nodeCentre(tree, offsets, n);
      if (!p || Math.hypot(p.x - c.x, p.y - c.y) > reach) return [];
      const s = at(p);
      drawn.set(n.skill, { n, s });
      const r = n.isKeystone ? 7 : n.isNotable || n.isJewelSocket ? 5 : n.isMastery ? 4 : 3;
      return [{ id: n.skill, ...s, r, on: alloc.has(n.skill), ring: inRadius.get(n.skill) }];
    });
    const seen = new Set<string>();
    const edges = [...drawn].flatMap(([id, { n, s }]) =>
      (n.out ?? []).flatMap((o) => {
        const to = drawn.get(Number(o));
        const joined = `${Math.min(id, Number(o))}:${Math.max(id, Number(o))}`;
        if (!to || to.n.isMastery || n.isMastery || seen.has(joined)) return [];
        seen.add(joined);
        let d = `M${s.x.toFixed(1)},${s.y.toFixed(1)} `;
        if (n.group === to.n.group && n.orbit === to.n.orbit && n.orbit) {
          const g = at(groupCentre(tree, offsets, n.group!, n.ascendancyName));
          const turn = Math.atan2(to.s.y - g.y, to.s.x - g.x) - Math.atan2(s.y - g.y, s.x - g.x);
          const sweep = (turn + 2 * Math.PI) % (2 * Math.PI) < Math.PI ? 1 : 0;
          const r = (tree.constants.orbitRadii[n.orbit] * scale).toFixed(1);
          d += `A${r},${r} 0 0 ${sweep} `;
        } else d += 'L';
        d += `${to.s.x.toFixed(1)},${to.s.y.toFixed(1)}`;
        return [{ key: `${id}-${o}`, d, on: alloc.has(id) && alloc.has(Number(o)) }];
      })
    );
    return { dots, edges, rings: radii.map((r) => ({ outer: r.outer * scale, inner: r.inner * scale, col: colour(r.col) })) };
  });

  function overAnchor(panel: HTMLElement, to: HTMLElement | undefined) {
    raise(panel);
    let stop: (() => void) | undefined;
    const place = (a: HTMLElement | undefined) => {
      if (!a) return;
      stop?.();
      stop = float(panel, a, { placement: 'top-start', fallback: ['bottom-start'], gap: 4, vars: true });
    };
    place(to);
    return { update: place, destroy: () => stop?.() };
  }
</script>

{#if view}
  <div data-socket-preview class="tooltip pointer-events-none fixed top-(--y) left-(--x) z-float rounded-lg p-1 shadow-tip" use:overAnchor={anchor}>
    <svg role="img" aria-label={label} width={W} height={H} viewBox="0 0 {W} {H}" class="block rounded-md bg-well">
      {#each view.rings as r, i (i)}
        <circle cx={W / 2} cy={H / 2} r={r.outer} fill="none" stroke={r.col} stroke-opacity="0.7" />
        {#if r.inner > 0}<circle cx={W / 2} cy={H / 2} r={r.inner} fill="none" stroke={r.col} stroke-opacity="0.4" stroke-dasharray="3 3" />{/if}
      {/each}
      {#each view.edges as e (e.key)}
        <path
          data-edge={e.key}
          data-alloc={e.on ? 'true' : undefined}
          d={e.d}
          fill="none"
          stroke={e.on ? 'var(--color-accent)' : 'var(--color-line-strong)'}
          stroke-width={e.on ? 2 : 1.5} />
      {/each}
      {#each view.dots as d (d.id)}
        <circle
          data-node={d.id}
          data-alloc={d.on ? 'true' : undefined}
          cx={d.x}
          cy={d.y}
          r={d.r}
          fill={d.on ? 'var(--color-accent)' : 'var(--color-surface-3)'}
          stroke={d.ring ?? 'var(--color-line-strong)'}
          stroke-width={d.ring ? 2 : 1} />
      {/each}
      <circle cx={W / 2} cy={H / 2} r="8" fill="none" stroke="var(--color-text)" stroke-width="2" />
    </svg>
  </div>
{/if}
