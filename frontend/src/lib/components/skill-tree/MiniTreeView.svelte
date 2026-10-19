<script lang="ts" module>
  import type { Tree } from '$lib/skill_tree/types';
  import { buildScene, type Scene } from './scene';

  const scenes = new WeakMap<Tree, Scene>();
  const sceneOf = (t: Tree) => {
    let s = scenes.get(t);
    if (!s) scenes.set(t, (s = buildScene(t)));
    return s;
  };
</script>

<script lang="ts">
  import { onDestroy } from 'svelte';
  import { syncWrap } from '$lib/go/worker';
  import { ascendancyGroupPositionOffsets, loadSkillTree, skillTree, skillTreeVersion } from '$lib/skill_tree';
  import { nodeCentre } from '$lib/skill_tree/geometry';
  import { logError } from '$lib/utils';
  import { buildClusterLayer } from './scene';
  import { createRenderer, type TreeRenderer } from './renderer';

  // CalcBreakdownControl.lua:594-612
  // TimelessJewelSocketControl.lua:24-43

  let { nodeId, mark = 'ring' }: { nodeId: number; mark?: 'ring' | 'cross' } = $props();

  const SIZE = 300;
  let canvas = $state<HTMLCanvasElement>();
  let renderer: TreeRenderer | undefined;
  let found = $state(true);

  $effect(() => {
    syncWrap
      ?.GetTreeVersion()
      .then((v) => {
        if (v && v !== $skillTreeVersion) return loadSkillTree(v);
      })
      .catch(logError);
  });

  $effect(() => {
    const t = $skillTree;
    const el = canvas;
    const id = nodeId;
    if (!t || !el || !syncWrap) return;
    let live = true;
    Promise.all([syncWrap.GetPassiveNodes(), syncWrap.GetTreeClusterNodes()])
      .then(([active, cluster]) => {
        if (!live) return;
        const scene = sceneOf(t);
        const placed = new Map(scene.nodes.map((n) => [n.id, n]));
        const layer = buildClusterLayer(t, cluster, (i) => placed.get(i));
        const at =
          placed.get(id)?.at ?? layer.nodes.find((n) => n.id === id)?.at ?? (t.nodes[id] && nodeCentre(t, ascendancyGroupPositionOffsets, t.nodes[id]));
        found = !!at;
        if (!at) return;
        renderer?.destroy();
        const version = $skillTreeVersion ?? '';
        renderer = createRenderer(el, scene, `https://data.pob.gg/data/${version.replace('_', '.')}`, () => {}, 'canvas');
        const scaling = (t.max_x - t.min_x) / 1.2 ** 5 / SIZE;
        renderer.setView({
          offsetX: (SIZE / 2) * scaling - Math.abs(t.min_x) - at.x,
          offsetY: (SIZE / 2) * scaling - Math.abs(t.min_y) - at.y,
          scaling,
          width: SIZE,
          height: SIZE,
          dpr: devicePixelRatio
        });
        renderer.setState({
          active: new Set(active ?? []),
          hoverPath: new Set(),
          hovered: undefined,
          className: undefined,
          ascendancy: undefined,
          cluster: layer
        });
      })
      .catch(logError);
    return () => {
      live = false;
    };
  });

  onDestroy(() => renderer?.destroy());
</script>

<div role="img" aria-label="Node location" class="relative bg-tree-ground" style:width="{SIZE}px" style:height="{SIZE}px" data-node={nodeId}>
  <canvas bind:this={canvas} width={SIZE} height={SIZE} class="block size-full"></canvas>
  {#if found && mark === 'cross'}
    <span class="pointer-events-none absolute top-0 left-[149px] h-full w-[2px] bg-text/20"></span>
    <span class="pointer-events-none absolute top-[149px] left-0 h-[2px] w-full bg-text/20"></span>
  {:else if found}
    <span class="pointer-events-none absolute top-[135px] left-[135px] size-[30px] rounded-full border-2 border-code-red"></span>
  {:else}
    <span class="absolute inset-0 grid place-items-center text-11 text-text-3">This node has no place on the tree</span>
  {/if}
</div>
