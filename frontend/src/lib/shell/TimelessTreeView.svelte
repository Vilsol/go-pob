<script lang="ts">
  import SkillTree from '$lib/components/skill-tree/SkillTree.svelte';
  import type { TimelessTreeArgs } from '$lib/design/timeless';
  import { ascendancyOffsets, nodeCentre, toScreen, type Point } from '$lib/skill_tree/geometry';
  import type { Node, Tree } from '$lib/skill_tree/types';

  type Props = TimelessTreeArgs & { tree: Tree; version: string };
  let { tree, version, rings, picked, centre, hovered, sockets, enter, leave, pick, tip, legend }: Props = $props();

  // Data.lua:609-613
  const LARGE_RADIUS = 1800;
  const TIP_W = 262;

  let view = $state<{ offsetX: number; offsetY: number; scaling: number; width: number; height: number }>();
  let canvas = $state<{ focus: (id: number, scaling?: number) => void }>();

  const at = $derived.by(() => {
    const offsets = ascendancyOffsets(tree);
    const out = new Map<number, Point>();
    for (const n of Object.values(tree.nodes)) {
      const p = n.skill !== undefined && !n.ascendancyName && !n.isProxy ? nodeCentre(tree, offsets, n) : undefined;
      if (p) out.set(n.skill!, p);
    }
    return out;
  });
  const project = (id: number) => {
    const p = at.get(id);
    return p && view ? toScreen(tree, p, view.offsetX, view.offsetY, view.scaling) : undefined;
  };
  const hot = $derived(rings.find((r) => r.hot)?.id ?? centre);
  const conquered = $derived.by(() => {
    const c = hot !== null ? at.get(hot) : undefined;
    if (!c) return [];
    return [...at].filter(([id, p]) => {
      const n = tree.nodes[id];
      return (n.isNotable || n.isKeystone) && (p.x - c.x) ** 2 + (p.y - c.y) ** 2 <= LARGE_RADIUS ** 2;
    });
  });

  let centred: number | null | undefined;
  $effect(() => {
    const v = view;
    const id = centre;
    if (!v?.width || !v.height || !canvas || id === centred) return;
    centred = id;
    if (id !== null) canvas.focus(id, (2.8 * LARGE_RADIUS) / Math.min(v.width, v.height));
  });

  const isSocket = (n: Node | undefined) => n?.skill !== undefined && sockets.has(n.skill);
</script>

<section
  aria-label="Passive tree around the build"
  class="relative h-[560px] min-w-0 flex-[0_1_1000px] overflow-hidden rounded-md border border-line bg-tree-ground"
  data-live-tree>
  <SkillTree
    bind:this={canvas}
    skillTree={tree}
    skillTreeVersion={version}
    chrome={false}
    whatIf={false}
    bind:view
    onhover={(n: Node | undefined) => (isSocket(n) ? enter(n!.skill!) : leave())}
    onpick={(n: Node) => {
      if (isSocket(n)) pick(n.skill!);
    }} />
  {#if view}
    {@const v = view}
    <div class="pointer-events-none absolute inset-0">
      {#each rings as r (r.id)}
        {@const p = project(r.id)}
        {#if p}
          {@const d = LARGE_RADIUS / v.scaling}
          <div class="radius-ring" data-ring={r.id} data-hot={r.hot || undefined} style:--x="{p.x - d}px" style:--y="{p.y - d}px" style:--z="{2 * d}px"></div>
        {/if}
      {/each}
      {#each [...(picked ?? [])] as id (id)}
        {@const p = project(id)}
        {#if p}
          <div
            class="absolute size-7 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-accent"
            data-picked={id}
            style:left="{p.x}px"
            style:top="{p.y}px">
          </div>
        {/if}
      {/each}
      {#each conquered as [id] (id)}
        {@const p = project(id)}
        {#if p}
          <div
            class="absolute size-6 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-node-conq"
            data-conq={id}
            style:left="{p.x}px"
            style:top="{p.y}px">
          </div>
        {/if}
      {/each}
      {#if hovered !== null}
        {@const p = project(hovered)}
        {#if p}
          <div
            role="tooltip"
            class="tooltip absolute z-map flex w-65.5 flex-col gap-0.75 rounded-md px-2.5 py-2 text-12 shadow-pop"
            style:left="{p.x + 18 + TIP_W <= v.width ? p.x + 18 : Math.max(p.x - 18 - TIP_W, 0)}px"
            style:top="{Math.max(Math.min(p.y - 30, v.height - 120), 6)}px">
            {@render tip(hovered)}
          </div>
        {/if}
      {/if}
      <div class="map-note bottom-2.5 left-2.5 flex flex-wrap gap-3 text-text-2">
        {@render legend()}
        <span>{version.replace('_', '.')} tree · Large radius (1800) · drag to pan, wheel to zoom</span>
      </div>
    </div>
  {/if}
</section>
