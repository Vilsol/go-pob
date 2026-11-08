<script lang="ts">
  import { onMount } from 'svelte';
  import Button from '$lib/design/Button.svelte';
  import SearchField from '$lib/design/SearchField.svelte';
  import { isTypingTarget } from '$lib/design/keys';
  import { createTree2Renderer, type Tree2Renderer } from './gl_renderer2';
  import { buildScene2, clampView, nodeAt2, screenToTree, treeToScreen, zoomAbout, zoomOf, type Tree2, type Tree2Node, type View2 } from './scene2';
  import { searchResults } from './tree2_search';

  type TreeState = {
    alloc: number[];
    allocMode: Record<number, number>;
    text: Record<number, { name: string; stats: string[] }>;
    ascendClassBaseName: string;
    // the jewelRadiusIndex of each socketed jewel that has one
    jewelRadiusIndex?: Record<number, number>;
  };

  interface Props {
    data: Tree2;
    spec: TreeState;
    busy?: boolean;
    // the tree2.ClickResult of a click; allocMode is spec.allocMode, the weapon set the click allocates to
    onclick: (id: number, right: boolean, hotkey: number, confirmClassChange: boolean, allocMode: number) => Promise<number | undefined>;
    onattribute: (id: number, attributeIndex: number, allocMode: number) => void;
    onhover: (id: number, allocMode: number) => Promise<{ path: number[]; depends: number[] }>;
    // spec:Undo and spec:Redo
    onundo?: () => void;
    onredo?: () => void;
    // the hosting page's search matches; without them the tree has its own search box
    matches?: Set<number>;
  }

  let { data, spec, busy = false, onclick, onattribute, onhover, onundo, onredo, matches }: Props = $props();

  const scene = $derived(buildScene2(data));
  let canvas = $state<HTMLCanvasElement>();
  let box = $state<HTMLDivElement>();
  let renderer = $state.raw<Tree2Renderer | undefined>();
  /* PassiveTreeView.lua:45-46 */
  let view = $state<View2>({ zoomLevel: 3, zoomX: 0, zoomY: 0, width: 0, height: 0, dpr: 1 });
  let hovered = $state<Tree2Node | undefined>();
  let hoverPath = $state(new Set<number>());
  let hoverDep = $state(new Set<number>());
  let mouse = $state({ x: 0, y: 0 });
  let attributeFor = $state<Tree2Node | undefined>();
  let classChangeFor = $state<Tree2Node | undefined>();
  /* PassiveSpec.lua:52 */
  let allocMode = $state(0);

  type Probe = { frame?: (f: unknown) => void; screenOf?: (id: number) => { x: number; y: number } | undefined; renderer?: string };
  const probe = () => (globalThis as { __tree2Probe?: Probe }).__tree2Probe;

  const alloc = $derived(new Set(spec.alloc));
  let searchStr = $state('');
  const SEARCH_TIP = 'Uses Lua pattern matching for complex searches.\nTo search for multiple terms: (increased.fire.damage|increased.area.of.effect|etc)';
  /* PassiveTreeView.lua:859-882: the spec's nodes, with their switched or conquered text */
  const search = $derived(
    matches ??
      searchResults(
        searchStr,
        scene.nodes.map(({ node: n }) => ({ id: n.id, type: n.type, name: spec.text[n.id]?.name ?? n.name, stats: spec.text[n.id]?.stats ?? n.stats }))
      )
  );
  /* PassiveTreeView.lua:1186 */
  const searchR = $derived(140 / zoomOf(view) ** 0.2);
  $effect(() => {
    renderer?.setState({
      alloc,
      allocMode: new Map(Object.entries(spec.allocMode).map(([k, v]) => [Number(k), v])),
      hovered: hovered?.id,
      hoverPath,
      hoverDep,
      ascendClassBaseName: spec.ascendClassBaseName,
      search,
      searchR,
      specAllocMode: allocMode,
      jewelRadiusIndex: new Map(Object.entries(spec.jewelRadiusIndex ?? {}).map(([k, v]) => [Number(k), v]))
    });
  });
  $effect(() => {
    renderer?.setView(view);
  });

  // overlays counts the strokes drawn over the connectors: search highlights and jewel radius rings
  const afterDraw = (ms: number) =>
    probe()?.frame?.({
      view: { ...view },
      ms,
      hovered: hovered?.id,
      alloc: spec.alloc.length,
      renderer: renderer?.kind,
      overlays: scene.counts.strokes - scene.connectors.length
    });

  const resize = () => {
    if (!box) return;
    const r = box.getBoundingClientRect();
    view = { ...view, width: Math.max(1, Math.round(r.width)), height: Math.max(1, Math.round(r.height)), dpr: window.devicePixelRatio || 1 };
  };

  onMount(() => {
    renderer = createTree2Renderer(canvas!, scene, afterDraw, probe()?.renderer as 'webgl' | 'canvas' | undefined);
    const p = probe();
    if (p) {
      p.renderer = renderer.kind;
      p.screenOf = (id) => {
        const n = scene.byId.get(id);
        return n && treeToScreen(scene, view, n.x, n.y);
      };
    }
    const ro = new ResizeObserver(resize);
    ro.observe(box!);
    resize();
    return () => {
      ro.disconnect();
      renderer?.destroy();
      renderer = undefined;
    };
  });

  let down: { x: number; y: number; zoomX: number; zoomY: number } | undefined;
  let dragging = false;
  let hoverSeq = 0;

  const local = (e: MouseEvent) => {
    const r = canvas!.getBoundingClientRect();
    return { x: e.clientX - r.left, y: e.clientY - r.top };
  };

  const setHover = (node: Tree2Node | undefined, refresh = false) => {
    if (!refresh && node?.id === hovered?.id) return;
    hovered = node;
    hoverPath = new Set();
    hoverDep = new Set();
    if (!node) return;
    const seq = ++hoverSeq;
    void onhover(node.id, allocMode).then((h) => {
      if (seq !== hoverSeq) return;
      hoverPath = new Set(h.path);
      hoverDep = new Set(h.depends);
    });
  };

  const pointerDown = (e: PointerEvent) => {
    if (e.button !== 0) return;
    const p = local(e);
    down = { ...p, zoomX: view.zoomX, zoomY: view.zoomY };
    dragging = false;
  };

  /* PassiveTreeView.lua:244-251 */
  const pointerMove = (e: PointerEvent) => {
    const p = local(e);
    mouse = p;
    if (down && !dragging && (Math.abs(p.x - down.x) > 5 || Math.abs(p.y - down.y) > 5)) dragging = true;
    if (down && dragging) {
      view = clampView({ ...view, zoomX: down.zoomX + p.x - down.x, zoomY: down.zoomY + p.y - down.y });
      return;
    }
    const t = screenToTree(scene, view, p.x, p.y);
    setHover(nodeAt2(scene, t.x, t.y));
  };

  const hotkeys = new Set<string>();
  /* TreeTab.lua:355-364 */
  const keydown = (e: KeyboardEvent) => {
    if (isTypingTarget(e.target)) return;
    hotkeys.add(e.key.toLowerCase());
    if (!(e.ctrlKey || e.metaKey) || busy) return;
    const k = e.key.toLowerCase();
    if (k === 'z' && onundo) {
      e.preventDefault();
      onundo();
    } else if (k === 'y' && onredo) {
      e.preventDefault();
      onredo();
    }
  };

  /* PassiveTreeView.lua:378-391: "2"/S Strength, "3"/D Dexterity, "1"/I Intelligence */
  const hotkey = () => (hotkeys.has('2') || hotkeys.has('s') ? 1 : hotkeys.has('3') || hotkeys.has('d') ? 2 : hotkeys.has('1') || hotkeys.has('i') ? 3 : 0);

  const click = async (node: Tree2Node, right: boolean, confirm = false) => {
    const res = await onclick(node.id, right, hotkey(), confirm, allocMode);
    if (res === 2) attributeFor = node;
    else if (res === 3) classChangeFor = node;
    hovered = undefined;
    const t = screenToTree(scene, view, mouse.x, mouse.y);
    setHover(nodeAt2(scene, t.x, t.y));
  };

  const pointerUp = (e: PointerEvent) => {
    const wasClick = down && !dragging && e.button === 0;
    down = undefined;
    dragging = false;
    if (wasClick && hovered && !busy) void click(hovered, false);
  };

  const contextMenu = (e: MouseEvent) => {
    e.preventDefault();
    if (hovered && !busy) void click(hovered, true);
  };

  /* PassiveTreeView.lua:190-195, 222-225, 1500-1514 */
  const wheel = (e: WheelEvent) => {
    e.preventDefault();
    // a browser can turn a Shift+wheel into a horizontal scroll
    const delta = e.deltaY || e.deltaX;
    if (e.altKey) {
      allocMode = Math.max(0, Math.min(2, allocMode + (delta < 0 ? 1 : -1)));
      setHover(hovered, true);
      return;
    }
    const p = local(e);
    view = clampView(zoomAbout(view, (delta < 0 ? 1 : -1) * (e.shiftKey ? 3 : 1), p.x, p.y));
  };

  const ATTRIBUTES = ['Strength', 'Dexterity', 'Intelligence'];
  const shownName = $derived(hovered ? (spec.text[hovered.id]?.name ?? hovered.name) : '');
  const shownStats = $derived(hovered ? (spec.text[hovered.id]?.stats ?? hovered.stats ?? []) : []);
</script>

<svelte:window onkeydown={keydown} onkeyup={(e: KeyboardEvent) => hotkeys.delete(e.key.toLowerCase())} onblur={() => hotkeys.clear()} onpointerup={pointerUp} />

<div class="relative h-full min-h-[320px] w-full overflow-hidden" bind:this={box}>
  <div role="img" aria-label="Passive skill tree" class="absolute inset-0">
    <canvas
      bind:this={canvas}
      class="absolute inset-0 h-full w-full"
      style="touch-action: none; cursor: {hovered ? 'pointer' : 'grab'}"
      style:width="{view.width}px"
      style:height="{view.height}px"
      onpointerdown={pointerDown}
      onpointermove={pointerMove}
      onpointerleave={() => !down && setHover(undefined)}
      onwheel={wheel}
      oncontextmenu={contextMenu}></canvas>
  </div>

  {#if !matches}
    <!-- TreeTab.lua:183-188 (PoB-PoE2) -->
    <div class="absolute top-2 left-2 z-sticky w-72 max-w-[calc(100%-1rem)]">
      <SearchField label="Search passive nodes" placeholder="Search" title={SEARCH_TIP} bind:value={searchStr} />
    </div>
  {/if}

  {#if allocMode > 0}
    <!-- PassiveTreeView.lua:2156-2176 -->
    <div
      role="status"
      aria-label="Weapon set allocation mode"
      class="pointer-events-none absolute inset-x-0 bottom-0 h-5 px-0.5 text-14 text-text"
      style:background={allocMode === 1 ? 'rgb(221 0 34 / 0.4)' : 'rgb(51 255 119 / 0.4)'}>
      Allocating Weapon set {allocMode} Mode
    </div>
  {/if}

  {#if hovered && !attributeFor && !classChangeFor}
    <div
      role="tooltip"
      aria-label="Node tooltip"
      class="pointer-events-none absolute z-dock max-w-[32rem] min-w-[16rem] rounded-md border border-line-strong bg-ground/90 text-13 text-text shadow-tip"
      style="left: {mouse.x}px; top: {mouse.y}px; transform: translate({mouse.x > view.width / 2 ? 'calc(-100% - 12px)' : '12px'}, {mouse.y > view.height / 2
        ? 'calc(-100% - 12px)'
        : '12px'})">
      <h2 class="bg-warn-bg px-3 py-2 text-center text-15 font-semibold">{shownName || 'N/A'}</h2>
      {#if hovered.ascendancy}<p class="m-0 px-3 pt-2 text-text-2">{hovered.ascendancy}</p>{/if}
      {#if shownStats.length}
        <ul class="m-0 list-none space-y-1 px-3 py-2">
          {#each shownStats as stat, i (i)}<li class="whitespace-pre-line">{stat}</li>{/each}
        </ul>
      {/if}
    </div>
  {/if}

  {#if attributeFor}
    <!-- TreeTab.lua:851-876 (PoB-PoE2) -->
    <div
      role="dialog"
      aria-label="Choose Attribute"
      class="absolute top-4 left-1/2 z-dock flex -translate-x-1/2 flex-col gap-2 rounded-md border border-line-strong bg-ground p-3 text-14">
      <span>Choose an attribute for {attributeFor.name}:</span>
      <div class="flex gap-2">
        {#each ATTRIBUTES as a, i (a)}
          <Button
            onclick={() => {
              const n = attributeFor!;
              attributeFor = undefined;
              onattribute(n.id, i + 1, allocMode);
            }}>{a}</Button>
        {/each}
        <Button onclick={() => (attributeFor = undefined)}>Cancel</Button>
      </div>
    </div>
  {/if}

  {#if classChangeFor}
    <!-- PassiveTreeView.lua:511-531 (PoB-PoE2), without Connect Path -->
    <div
      role="dialog"
      aria-label="Class Change"
      class="absolute top-4 left-1/2 z-dock flex -translate-x-1/2 flex-col gap-2 rounded-md border border-line-strong bg-ground p-3 text-14">
      <span>Changing class to {classChangeFor.ascendancy} will reset your passive tree.</span>
      <div class="flex gap-2">
        <Button
          onclick={() => {
            const n = classChangeFor!;
            classChangeFor = undefined;
            void click(n, false, true);
          }}>Continue</Button>
        <Button onclick={() => (classChangeFor = undefined)}>Cancel</Button>
      </div>
    </div>
  {/if}
</div>
