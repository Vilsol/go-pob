<script lang="ts">
  import { onMount } from 'svelte';
  import Button from '$lib/design/Button.svelte';
  import SearchField from '$lib/design/SearchField.svelte';
  import { isTypingTarget } from '$lib/design/keys';
  import { createTree2Renderer, type Tree2Renderer } from './gl_renderer2';
  import { park, unpark, type Parked } from './tree_cache';
  import { GL2_ATTRS, hardwareContext } from './webgl';
  import { buildAtlas2, sheetPath2 } from '$lib/skill_tree/tree2_art';
  import {
    attachArt2,
    buildScene2,
    clampView,
    nodeAt2,
    screenToTree,
    treeToScreen,
    zoomAbout,
    zoomOf,
    scaleOf,
    SPRITE2_FLOATS,
    type Tree2,
    type Scene2,
    type Tree2Node,
    type View2
  } from './scene2';
  import { searchResults } from './tree2_search';
  import { heatMixColour, heatSingleColour, type NodePowerTheme } from '$lib/skill_tree/heat';

  type TreeState = {
    alloc: number[];
    allocMode: Record<number, number>;
    text: Record<number, { name: string; stats: string[] }>;
    ascendClassBaseName: string;
    // the jewelRadiusIndex of each socketed jewel that has one
    jewelRadiusIndex?: Record<number, number>;
    // spec.curClassName; without it the class background follows the allocated class start
    className?: string;
    // the jewel in each socket that holds one
    jewels?: { socket: number; title: string; baseName: string; unique: boolean }[];
    // each node's author note
    notes?: Record<number, string>;
  };

  interface Props {
    data: Tree2;
    spec: TreeState;
    busy?: boolean;
    // the tree2.ClickResult of a click; allocMode is spec.allocMode, the weapon set the click allocates to; trace is
    // the Shift-traced path while tracing
    onclick: (id: number, right: boolean, hotkey: number, confirmClassChange: boolean, allocMode: number, trace?: number[]) => Promise<number | undefined>;
    onattribute: (id: number, attributeIndex: number, allocMode: number) => void;
    // trace is the Shift-traced path while tracing, when the path and dependents are not asked for
    onhover: (id: number, allocMode: number, trace?: number[]) => Promise<{ path: number[]; depends: number[] }>;
    // one hover while Shift is held (PassiveTreeView.lua:321-350): the trace path after it, and whether the node is
    // highlighted
    ontrace?: (trace: number[], id: number) => Promise<{ trace: number[]; hover: boolean }>;
    // the pointer left the hovered node
    onunhover?: () => void;
    // the Class Change popup's Connect Path (PassiveTreeView.lua:520-528); the tree2.ClickResult
    onconnect?: (id: number, hotkey: number, allocMode: number) => Promise<number | undefined>;
    // spec:Undo and spec:Redo
    onundo?: () => void;
    onredo?: () => void;
    // the hosting page's search matches; without them the tree has its own search box
    matches?: Set<number>;
    // the data version's root on the data host, which the tree's art loads from
    artBase?: string;
    // the compared spec's allocated nodes, drawn as PassiveTreeView.lua draws its compareSpec
    compared?: Set<number>;
    // while the heat map shows, node.power (CalcsTab.lua:551-709) and calcsTab.powerMax; single for a stat's own power,
    // else offence and defence; theme is main.nodePowerTheme
    power?: {
      nodes: Map<number, { single?: number; offence: number; defence: number }>;
      max: number[];
      single: boolean;
      theme: NodePowerTheme;
    };
  }

  let {
    data,
    spec,
    busy = false,
    onclick,
    onattribute,
    onhover,
    ontrace,
    onunhover,
    onconnect,
    onundo,
    onredo,
    matches,
    artBase = 'https://data.pob.gg/data/0.5',
    compared,
    power
  }: Props = $props();

  let canvas = $state<HTMLCanvasElement>();
  let host = $state<HTMLDivElement>();
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
  /* PassiveTreeView.lua:276-283: Shift traces a path; undefined while not tracing */
  let trace = $state.raw<number[]>();

  type Probe = {
    frame?: (f: unknown) => void;
    screenOf?: (id: number) => { x: number; y: number } | undefined;
    focus?: (id: number) => void;
    // the art a test probes the canvas against: a sprite's sheet and place, and the sprites drawn from each sheet
    sprite?: (name: string) => { url: string; x: number; y: number; w: number; h: number } | undefined;
    sheetCounts?: () => Record<string, number>;
    icon?: (id: number) => string | undefined;
    scale?: () => number;
    renderer?: string;
  };
  const probe = () => (globalThis as { __tree2Probe?: Probe }).__tree2Probe;
  // the worker hands each visit its own copy of the tree, so the last visit's renderer is matched by the tree's shape
  const treeKey = () => `poe2:${artBase}:${data.nodes.length}:${data.minX},${data.minY},${data.maxX},${data.maxY}:${probe()?.renderer ?? ''}`;
  const kept = unpark<Tree2Renderer, Scene2>(treeKey());
  // a tree without sprites keeps its vector shapes
  const scene = $derived.by(() => {
    if (kept?.key === treeKey()) return kept.scene;
    const s = buildScene2(data);
    if (data.art?.sprites && Object.keys(data.art.sprites).length) attachArt2(s, data.art, buildAtlas2(data.art.sprites), artBase);
    return s;
  });

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
  // a switched or attribute-chosen node draws its option's icon (PassiveTree.lua:592-613)
  const icons = $derived.by(() => {
    const out = new Map<number, string>();
    for (const [id, t] of Object.entries(spec.text)) {
      const icon = data.art?.nodes[id]?.options?.find((o) => o.name === t.name)?.icon;
      if (icon) out.set(Number(id), icon);
    }
    return out;
  });
  const jewels = $derived(new Map((spec.jewels ?? []).map((j) => [j.socket, j])));
  /* PassiveTreeView.lua:1003-1030 */
  const heat = $derived.by(() => {
    if (!power) return undefined;
    const out = new Map<number, [number, number, number]>();
    const [single, offence, , defence] = power.max;
    for (const [id, p] of power.nodes)
      out.set(id, power.single ? heatSingleColour(p.single ?? 0, single, power.theme) : heatMixColour(p.offence, p.defence, offence, defence, power.theme));
    return out;
  });
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
      jewelRadiusIndex: new Map(Object.entries(spec.jewelRadiusIndex ?? {}).map(([k, v]) => [Number(k), v])),
      className: spec.className,
      icons,
      jewels,
      compared,
      heat
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
      complete: renderer?.complete(),
      overlays: scene.counts.strokes - scene.counts.connectorStrokes,
      sprites: scene.counts.under + scene.counts.over,
      lines: scene.counts.lines,
      compared: compared?.size,
      heat: heat?.size
    });

  // a node's position on the canvas, in CSS pixels
  export const project = (id: number) => {
    const n = scene.byId.get(id);
    return n && treeToScreen(scene, view, n.x, n.y);
  };
  /* PassiveTreeView.lua:206-209, 1500-1514: a step in or out about the pointer, or about the centre with the pointer
     off the canvas; fit shows the whole tree */
  let pointerIn = false;
  export const zoom = (how: 'in' | 'out' | 'fit') => {
    if (how === 'fit') {
      view = { ...view, zoomLevel: 0, zoomX: 0, zoomY: 0 };
      return;
    }
    const at = pointerIn ? mouse : { x: view.width / 2, y: view.height / 2 };
    view = clampView(zoomAbout(view, how === 'in' ? 1 : -1, at.x, at.y));
  };
  /* PassiveTreeView.lua:1517-1526 */
  export const focus = (id: number) => {
    const n = scene.byId.get(id);
    if (!n) return;
    const v = { ...view, zoomLevel: 20 };
    const k = scaleOf(scene, v);
    view = { ...v, zoomX: -n.x * k, zoomY: -n.y * k };
  };

  const resize = () => {
    if (!box) return;
    const r = box.getBoundingClientRect();
    view = { ...view, width: Math.max(1, Math.round(r.width)), height: Math.max(1, Math.round(r.height)), dpr: window.devicePixelRatio || 1 };
  };

  $effect(() => {
    if (!canvas) return;
    canvas.style.width = `${view.width}px`;
    canvas.style.height = `${view.height}px`;
  });

  onMount(() => {
    let entry = kept;
    if (entry && entry.scene !== scene) {
      entry.renderer.destroy();
      entry = undefined;
    }
    const c = entry?.canvas ?? document.createElement('canvas');
    c.className = 'absolute inset-0 h-full w-full';
    host!.append(c);
    canvas = c;
    const attach = (e: Parked<Tree2Renderer, Scene2>) => {
      entry = e;
      e.onDraw = afterDraw;
      renderer = e.renderer;
      const p = probe();
      if (p) {
        p.renderer = e.renderer.kind;
        p.screenOf = project;
        p.focus = focus;
        p.sprite = (name) => {
          const sp = scene.art?.atlas.byName.get(name);
          return sp && { url: `${scene.art!.base}/${sheetPath2(scene.art!.atlas.sheets[sp.sheet])}`, x: sp.x, y: sp.y, w: sp.w, h: sp.h };
        };
        p.icon = (id) => scene.art?.data.nodes[String(id)]?.icon;
        p.scale = () => scaleOf(scene, view);
        p.sheetCounts = () => {
          const out: Record<string, number> = {};
          for (const layer of ['under', 'lines', 'over'] as const) {
            for (let i = 0; i < scene.counts[layer]; i++) {
              const name = sheetPath2(scene.art!.atlas.sheets[scene[layer][i * SPRITE2_FLOATS + 10]]);
              out[name] = (out[name] ?? 0) + 1;
            }
          }
          return out;
        };
      }
    };
    // a new renderer, with its graphics context, is built in a task of its own, apart from the page's mount
    let build = entry
      ? undefined
      : setTimeout(() => {
          // the graphics context in one task and the renderer's programs and buffers in the next
          const force = probe()?.renderer as 'webgl' | 'canvas' | undefined;
          const g = document.createElement('canvas');
          const gl = force === undefined ? hardwareContext(g, GL2_ATTRS) : null;
          build = setTimeout(() => {
            if (gl) {
              g.className = c.className;
              c.replaceWith(g);
              canvas = g;
            }
            const fresh: Parked<Tree2Renderer, Scene2> = { key: treeKey(), canvas: canvas!, scene, onDraw: afterDraw, renderer: undefined! };
            fresh.renderer = createTree2Renderer(canvas!, scene, (ms) => fresh.onDraw(ms), force, gl ?? undefined);
            attach(fresh);
          });
        });
    if (entry) attach(entry);
    const ro = new ResizeObserver(resize);
    ro.observe(box!);
    resize();
    return () => {
      clearTimeout(build);
      ro.disconnect();
      renderer = undefined;
      if (entry) park(entry);
    };
  });

  let down: { x: number; y: number; zoomX: number; zoomY: number } | undefined;
  // the node under the pointer, which a trace that cannot reach it leaves unhovered
  let pointed: Tree2Node | undefined;
  let dragging = false;
  let hoverSeq = 0;

  const local = (e: MouseEvent) => {
    const r = canvas!.getBoundingClientRect();
    return { x: e.clientX - r.left, y: e.clientY - r.top };
  };

  /* PassiveTreeView.lua:321-355: while tracing, the trace is the path and a node the trace cannot reach is not hovered */
  const traceHover = async (node: Tree2Node, seq: number) => {
    const traced = trace ?? [];
    // the host hears of the node at once, so a Shift+right-click note reaches it before the trace step answers
    void onhover(node.id, allocMode, traced);
    const t = await ontrace!(traced, node.id);
    if (seq !== hoverSeq || !trace) return;
    trace = t.trace;
    hoverPath = new Set(t.trace);
    if (!t.hover) {
      hovered = undefined;
      onunhover?.();
      return;
    }
    void onhover(node.id, allocMode, t.trace);
  };

  const setHover = (node: Tree2Node | undefined, refresh = false) => {
    if (!refresh && node?.id === pointed?.id) return;
    pointed = node;
    hovered = node;
    hoverPath = trace ? new Set(trace) : new Set();
    hoverDep = new Set();
    if (!node) {
      onunhover?.();
      return;
    }
    const seq = ++hoverSeq;
    if (trace && ontrace) {
      void traceHover(node, seq);
      return;
    }
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
    if (e.key === 'Shift' && !trace && ontrace) {
      trace = [];
      setHover(pointed, true);
    }
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

  const stopTrace = () => {
    if (!trace) return;
    trace = undefined;
    setHover(pointed, true);
  };
  const keyup = (e: KeyboardEvent) => {
    hotkeys.delete(e.key.toLowerCase());
    if (e.key === 'Shift') stopTrace();
  };
  const blur = () => {
    hotkeys.clear();
    stopTrace();
  };

  /* PassiveTreeView.lua:378-391: "2"/S Strength, "3"/D Dexterity, "1"/I Intelligence */
  const hotkey = () => (hotkeys.has('2') || hotkeys.has('s') ? 1 : hotkeys.has('3') || hotkeys.has('d') ? 2 : hotkeys.has('1') || hotkeys.has('i') ? 3 : 0);

  const click = async (node: Tree2Node, right: boolean, confirm = false, connect = false) => {
    const traced = trace?.length ? trace : undefined;
    const res =
      connect && onconnect
        ? await onconnect(node.id, hotkey(), allocMode)
        : traced
          ? await onclick(node.id, right, hotkey(), confirm, allocMode, traced)
          : await onclick(node.id, right, hotkey(), confirm, allocMode);
    if (res === 2) attributeFor = node;
    else if (res === 3) classChangeFor = node;
    const t = screenToTree(scene, view, mouse.x, mouse.y);
    setHover(nodeAt2(scene, t.x, t.y), true);
  };

  /* PassiveTreeView.lua:251-255: a Ctrl+click zooms in, a Ctrl+right-click out, and allocates nothing */
  const ctrlZoom = (e: MouseEvent, right: boolean) => {
    if (!e.ctrlKey) return false;
    const p = local(e);
    view = clampView(zoomAbout(view, right ? -2 : 2, p.x, p.y));
    return true;
  };

  const pointerUp = (e: PointerEvent) => {
    const wasClick = down && !dragging && e.button === 0;
    down = undefined;
    dragging = false;
    if (wasClick && ctrlZoom(e, false)) return;
    if (wasClick && hovered && !busy) void click(hovered, false);
  };

  const contextMenu = (e: MouseEvent) => {
    e.preventDefault();
    if (ctrlZoom(e, true)) return;
    if (hovered && !busy) void click(hovered, true);
  };

  /* PassiveTreeView.lua:190-195, 222-225, 1500-1514 */
  const wheel = (e: WheelEvent) => {
    e.preventDefault();
    // a browser can turn a Shift+wheel into a horizontal scroll
    const delta = e.deltaY || e.deltaX;
    if (e.altKey) {
      allocMode = Math.max(0, Math.min(2, allocMode + (delta < 0 ? 1 : -1)));
      setHover(pointed, true);
      return;
    }
    const p = local(e);
    view = clampView(zoomAbout(view, (delta < 0 ? 1 : -1) * (e.shiftKey ? 3 : 1), p.x, p.y));
  };

  const ATTRIBUTES = ['Strength', 'Dexterity', 'Intelligence'];
  /* PassiveTreeView.lua:478-488: the base class whose ascendancies hold the clicked node's */
  const classChangeName = $derived(
    data.art?.classes?.find((c) => c.ascendancies?.some((a) => (a.id ?? a.name) === classChangeFor?.ascendancy))?.name ?? classChangeFor?.ascendancy
  );
</script>

<svelte:window onkeydown={keydown} onkeyup={keyup} onblur={blur} onpointerup={pointerUp} />

<div class="relative h-full min-h-[320px] w-full overflow-hidden" bind:this={box}>
  <div
    role="img"
    aria-label="Passive skill tree"
    class="absolute inset-0"
    bind:this={host}
    style="touch-action: none; cursor: {hovered ? 'pointer' : 'grab'}"
    onpointerdown={pointerDown}
    onpointermove={pointerMove}
    onpointerenter={() => (pointerIn = true)}
    onpointerleave={() => {
      pointerIn = false;
      if (!down) setHover(undefined);
    }}
    onwheel={wheel}
    oncontextmenu={contextMenu}>
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
    <!-- PassiveTreeView.lua:511-531 (PoB-PoE2) -->
    <div
      role="dialog"
      aria-label="Class Change"
      class="absolute top-4 left-1/2 z-dock flex -translate-x-1/2 flex-col gap-2 rounded-md border border-line-strong bg-ground p-3 text-14">
      <span>Changing class to {classChangeName} will reset your passive tree.</span>
      <span>This can be avoided by connecting one of the {classChangeName} starting nodes to your tree.</span>
      <div class="flex gap-2">
        <Button
          onclick={() => {
            const n = classChangeFor!;
            classChangeFor = undefined;
            void click(n, false, true);
          }}>Continue</Button>
        {#if onconnect}
          <Button
            onclick={() => {
              const n = classChangeFor!;
              classChangeFor = undefined;
              void click(n, false, true, true);
            }}>Connect Path</Button>
        {/if}
        <Button onclick={() => (classChangeFor = undefined)}>Cancel</Button>
      </div>
    </div>
  {/if}
</div>
