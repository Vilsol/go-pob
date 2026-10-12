<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { TreeFixture, TreeNodeFx } from './fixtures';

  type Props = {
    tree: TreeFixture;
    label: string;
    alloc: Record<number, boolean>;
    hovered?: number | null;
    path?: number[];
    lose?: number[];
    matches?: Set<number>;
    conquered?: Set<number>;
    inRadius?: Set<number>;
    rings?: { id: number; hot?: boolean }[];
    picked?: Set<number>;
    active?: (n: TreeNodeFx) => boolean;
    describedBy?: string;
    width?: number;
    class?: string;
    onenter?: (n: TreeNodeFx) => void;
    onleave?: (n: TreeNodeFx) => void;
    onpick?: (n: TreeNodeFx) => void;
    onmenu?: (n: TreeNodeFx, at: { x: number; y: number }) => void;
    overlay?: Snippet;
  };
  let {
    tree,
    label,
    alloc,
    hovered = null,
    path = [],
    lose = [],
    matches,
    conquered,
    inRadius,
    rings = [],
    picked,
    active = () => false,
    describedBy,
    width = $bindable(0),
    class: extra = '',
    onenter,
    onleave,
    onpick,
    onmenu,
    overlay
  }: Props = $props();

  const SIZE = { s: 8, n: 14, k: 19, m: 12, c: 26, j: 14 };
  const height = $derived(width / tree.aspect);
  const byId = $derived(new Map(tree.nodes.map((n) => [n.id, n])));
  const onPath = $derived(new Set(path));
  const losing = $derived(new Set(lose));
  const px = (n: TreeNodeFx) => ({ x: n.x * width, y: n.y * height });
  const edges = $derived(
    tree.edges.map(([a, b]) => {
      const A = px(tree.nodes[a]);
      const B = px(tree.nodes[b]);
      const on = !!alloc[tree.nodes[a].id] && !!alloc[tree.nodes[b].id];
      const near = (id: number) => onPath.has(id) || !!alloc[id];
      const gone = losing.has(tree.nodes[a].id) || losing.has(tree.nodes[b].id);
      return {
        key: `${a}-${b}`,
        x: A.x,
        y: A.y - 1,
        w: Math.hypot(B.x - A.x, B.y - A.y),
        r: (Math.atan2(B.y - A.y, B.x - A.x) * 180) / Math.PI,
        state: on && gone ? 'lose' : on ? 'on' : near(tree.nodes[a].id) && near(tree.nodes[b].id) ? 'path' : undefined
      };
    })
  );
  const menuKey = (e: KeyboardEvent, n: TreeNodeFx) => {
    if (e.key !== 'ContextMenu' && !(e.shiftKey && e.key === 'F10')) return;
    e.preventDefault();
    const p = px(n);
    onmenu?.(n, p);
  };
</script>

<section aria-label={label} class={['relative min-w-0 overflow-hidden', extra]} bind:clientWidth={width} data-tree-view>
  <div class="relative h-(--h) w-full" style:--h="{height}px">
    {#each edges as e (e.key)}<div class="tree-edge" data-state={e.state} style:--x="{e.x}px" style:--y="{e.y}px" style:--w="{e.w}px" style:--r="{e.r}deg">
      </div>{/each}
    {#each rings as r (r.id)}
      {@const c = byId.get(r.id)}
      {#if c}
        {@const d = tree.radius * width}
        <div class="radius-ring" data-hot={r.hot || undefined} style:--x="{px(c).x - d}px" style:--y="{px(c).y - d}px" style:--z="{2 * d}px"></div>
      {/if}
    {/each}
    {#each tree.nodes as n (n.id)}
      {@const z = SIZE[n.k]}
      {@const a = !!alloc[n.id]}
      {@const state = losing.has(n.id) || (n.id === hovered && a && lose.length) ? 'lose' : !a && onPath.has(n.id) ? 'path' : undefined}
      {#if active(n)}
        <button
          class="tree-node"
          data-node={n.id}
          data-kind={n.k}
          data-alloc={a}
          data-state={state}
          data-hover={n.id === hovered}
          data-match={matches?.has(n.id) || undefined}
          data-conq={conquered?.has(n.id) || undefined}
          data-radius={inRadius?.has(n.id) || undefined}
          data-picked={picked?.has(n.id) || undefined}
          aria-label="{n.name}{a ? ' (allocated)' : ''}"
          aria-describedby={n.id === hovered ? describedBy : undefined}
          aria-haspopup={onmenu ? 'menu' : undefined}
          tabindex={n.k === 's' ? -1 : 0}
          style:--x="{px(n).x - z / 2}px"
          style:--y="{px(n).y - z / 2}px"
          style:--z="{z}px"
          onmouseenter={() => onenter?.(n)}
          onfocus={() => onenter?.(n)}
          onmouseleave={() => onleave?.(n)}
          onblur={() => onleave?.(n)}
          onclick={() => onpick?.(n)}
          oncontextmenu={(e) => {
            if (!onmenu) return;
            e.preventDefault();
            onmenu(n, px(n));
          }}
          onkeydown={(e) => menuKey(e, n)}></button>
      {:else}
        <span
          class="tree-node"
          data-node={n.id}
          data-kind={n.k}
          data-alloc={a}
          data-state={state}
          data-match={matches?.has(n.id) || undefined}
          data-conq={conquered?.has(n.id) || undefined}
          data-radius={inRadius?.has(n.id) || undefined}
          title={n.name}
          style:--x="{px(n).x - z / 2}px"
          style:--y="{px(n).y - z / 2}px"
          style:--z="{z}px"></span>
      {/if}
    {/each}
    {@render overlay?.()}
  </div>
</section>
