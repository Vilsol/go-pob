<script lang="ts">
  import { ascendancyOffsets, groupCentre, nodeCentre, type Point } from '$lib/skill_tree/geometry';
  import type { Node, Tree } from '$lib/skill_tree/types';
  import { buildScene2, type Scene2, type Tree2 } from '$lib/components/skill-tree/scene2';
  import { float, raise } from './floating';

  // ItemSlotControl.lua:132-149
  // ItemSlotControl.lua:139-146
  type Radius = { inner: number; outer: number; col: string; nodes: number[] };
  // a PoE1 tree, or a PoE2 one (ItemSlotHelper.lua:10-41, PoB-PoE2)
  type Props = { tree: Tree | Tree2; node: number; radii: Radius[]; alloc: Set<number>; label: string; anchor?: HTMLElement };
  let { tree, node, radii, alloc, label, anchor }: Props = $props();

  const W = 308;
  const H = 280;
  const colour = (code: string) => (/^\^x[0-9A-Fa-f]{6}$/.test(code) ? `#${code.slice(2)}` : 'var(--color-accent)');
  const isTree2 = (t: Tree | Tree2): t is Tree2 => Array.isArray(t.nodes);
  const scenes = new WeakMap<Tree2, Scene2>();
  // ItemSlotHelper.lua:19-25 (PoB-PoE2): the viewer shows tree.size / 17 across, centred on the socket
  const view2 = (tree: Tree2) => {
    let scene = scenes.get(tree);
    if (!scene) scenes.set(tree, (scene = buildScene2(tree)));
    const centre = scene.byId.get(node);
    if (!centre) return undefined;
    const reach = Math.max(scene.size / 17 / 2, ...radii.map((r) => r.outer)) * 1.1;
    const scale = Math.min(W, H) / 2 / reach;
    const at = (x: number, y: number) => ({ x: W / 2 + (x - centre.x) * scale, y: H / 2 + (y - centre.y) * scale });
    const inRadius = new Map<number, string>();
    for (const r of radii) for (const id of r.nodes) if (!inRadius.has(id)) inRadius.set(id, colour(r.col));
    const drawn = new Set<number>();
    const dots = scene.nodes.flatMap(({ node: n }) => {
      if (n.ascendancy || n.classesStart || Math.hypot(n.x - centre.x, n.y - centre.y) > reach) return [];
      drawn.add(n.id);
      const r = n.type === 'Keystone' ? 7 : n.type === 'Notable' || n.type === 'Socket' || n.containJewelSocket ? 5 : 3;
      return [{ id: n.id, ...at(n.x, n.y), r, on: alloc.has(n.id), ring: inRadius.get(n.id) }];
    });
    const seen = new Set<string>();
    const edges = scene.connectors.flatMap((c) => {
      const joined = `${Math.min(c.a, c.b)}:${Math.max(c.a, c.b)}`;
      if (!drawn.has(c.a) || !drawn.has(c.b) || seen.has(joined)) return [];
      seen.add(joined);
      const [g0, g1, g2, g3, g4] = c.geom;
      let d: string;
      if (c.kind === 0) {
        const [p, q] = [at(g0, g1), at(g2, g3)];
        d = `M${p.x.toFixed(1)},${p.y.toFixed(1)} L${q.x.toFixed(1)},${q.y.toFixed(1)}`;
      } else {
        const [p, q] = [at(g0 + g2 * Math.cos(g3), g1 + g2 * Math.sin(g3)), at(g0 + g2 * Math.cos(g3 + g4), g1 + g2 * Math.sin(g3 + g4))];
        const r = (g2 * scale).toFixed(1);
        d = `M${p.x.toFixed(1)},${p.y.toFixed(1)} A${r},${r} 0 0 1 ${q.x.toFixed(1)},${q.y.toFixed(1)}`;
      }
      return [{ key: `${c.a}-${c.b}`, d, on: alloc.has(c.a) && alloc.has(c.b) }];
    });
    return { dots, edges, rings: radii.map((r) => ({ outer: r.outer * scale, inner: r.inner * scale, col: colour(r.col) })) };
  };
  const view = $derived.by(() => {
    if (isTree2(tree)) return view2(tree);
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
