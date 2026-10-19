<script lang="ts">
  import type { Node, Tree, TreeClusterNode } from '$lib/skill_tree/types';
  import { classStartNodes } from '../../skill_tree';
  import { MAX_SCALING, MIN_SCALING, secondaryAscendancy } from '$lib/skill_tree/geometry';
  import type { Point } from '$lib/skill_tree/geometry';
  import { onMount } from 'svelte';
  import { currentBuild, devMode, nodePowerTheme, outputs, zoomSensitivity } from '$lib/global.js';
  import { syncWrap } from '$lib/go/worker';
  import { get, writable } from 'svelte/store';
  import { logError } from '$lib/utils';
  import NodeSearch from '$lib/components/skill-tree/NodeSearch.svelte';
  import { calculateAllocationPath, extendTrace, withClusterPaths, type PrecalculatedAllocationPaths } from './paths';
  import { treeProbe } from './probe';
  import { buildClusterLayer, buildScene, sceneDrawList } from './scene';
  import { createRenderer, type TreeRenderer } from './renderer';
  import NodeTooltip from './NodeTooltip.svelte';
  import { keepForOffline } from '$lib/pwa';
  import type { NodeStatDiff, StatDiffBlock } from '$lib/skill_tree/stat_compare';
  import type { ClassChangeResult } from '$lib/skill_tree/class_change';
  import { heatMixColour, heatSingleColour } from '$lib/skill_tree/heat';
  import { gestureDriver } from '$lib/design/mobile/tree_gestures';

  let currentClass: string | undefined = $state();
  let currentAscendancy: string | undefined = $state();
  let currentSecondaryAscendancy: string | undefined = $state();
  let recordedTreeVersion: string | undefined = $state();

  interface Props {
    skillTree: Tree;
    skillTreeVersion: string;
    chrome?: boolean;
    onhover?: (node: Node | undefined) => void;
    onpick?: (node: Node) => void;
    onmenu?: (node: Node, at: Point) => void;
    onpaths?: (active: number[], paths: PrecalculatedAllocationPaths) => void;
    view?: { offsetX: number; offsetY: number; scaling: number; width: number; height: number };
    /* PassiveTreeView.lua:1658-1706 */
    statDiff?: NodeStatDiff;
    dependents?: { node: number; ids: number[] };
    /* PassiveTreeView.lua:840-846 */
    socketArt?: Map<number, string>;
    /* PassiveTreeView.lua:847-877 */
    tattoos?: Map<number, { icon: string; effect: string }>;
    /* PassiveTreeView.lua:906-919 */
    heat?: { values: Map<number, number>; max: number; mix?: { mix: Map<number, [number, number]>; maxOffence: number; maxDefence: number } };
    /* PassiveTreeView.lua:166-185, 814-816 */
    baseTint?: Map<number, [number, number, number]>;
    /* PassiveTreeView.lua:686-699, 811 */
    compared?: number[];
    /* PassiveTreeView.lua:679-681 */
    split?: number[];
    /* PassiveTreeView.lua:283-363 */
    ontrace?: (trace: number[]) => void;
    /* PassiveSpec.lua:1885-2271 */
    cluster?: TreeClusterNode[];
    /* PassiveTreeView.lua:1102-1156 */
    compareCluster?: TreeClusterNode[];
    /* PassiveTreeView.lua:474-490 */
    onclassconfirm?: (className: string, answer: (decision: 1 | 2) => Promise<ClassChangeResult> | undefined) => void;
    whatIf?: boolean;
    // phone: tap = hover, double tap = pick, hold = menu, one finger pans, two pinch
    gestures?: boolean;
  }

  let {
    skillTree,
    skillTreeVersion,
    chrome = true,
    onhover,
    onpick,
    onmenu,
    onpaths,
    onclassconfirm,
    heat,
    baseTint,
    compared,
    split,
    ontrace,
    socketArt,
    tattoos,
    cluster,
    compareCluster,
    whatIf = true,
    gestures = false,
    view = $bindable(),
    statDiff = $bindable(),
    dependents = $bindable()
  }: Props = $props();

  let scaling = $state(10);

  let offsetX = $state(0);
  let offsetY = $state(0);

  let activeNodes: number[] = $state([]);
  // PassiveSpec.lua:776-784
  let leapHeld = $state.raw(new Set<number>());

  $effect(() => {
    void $outputs;
    if (!$currentBuild) {
      return;
    }
    syncWrap
      ?.GetLeapHeldNodes()
      .then((held) => (leapHeld = new Set(held ?? [])))
      .catch(logError);
    Promise.all([syncWrap?.GetBuildInfo(), syncWrap?.GetPassiveNodes()])
      .then(([info, nodes]) => {
        currentClass = info?.className;
        currentAscendancy = info?.ascendClassName;
        currentSecondaryAscendancy = secondaryAscendancy(skillTree, info?.secondaryAscendClassId ?? 0);
        recordedTreeVersion = info?.recordedTreeVersion;
        activeNodes = nodes ?? [];
      })
      .catch(logError);
  });

  const classRoots = () => classStartNodes[skillTree.classes.findIndex((c) => c.name === currentClass)] ?? [];
  const subgraphNodes = () => (cluster ?? []).map((c) => ({ ID: c.id, Type: c.type, Linked: [...c.linked] }));

  // PassiveSpec.lua:856-857

  function precalculateAllocationPaths(active: number[]): Promise<PrecalculatedAllocationPaths> {
    const rootNodes = classRoots();
    const subgraphs = cluster ?? [];
    return syncWrap
      ?.CalculateAllocationPaths(skillTreeVersion, [...active], rootNodes)
      .then((paths) => withClusterPaths(paths ?? {}, subgraphs))
      .catch((e: Error) => {
        logError(e);
        return {};
      });
  }

  let allocationPaths: Promise<PrecalculatedAllocationPaths> = $derived(precalculateAllocationPaths(activeNodes));

  $effect(() => {
    const active = activeNodes;
    allocationPaths.then((paths) => onpaths?.(active, paths)).catch(logError);
  });

  // PassiveTreeView.lua:283-290
  let shiftDown = $state(false);
  let trace = $state.raw<number[]>([]);
  $effect(() => ontrace?.(trace));
  const linkedNodes = (a: number, b: number) => {
    const n = skillTree.nodes[a];
    return !!n && ((n.out ?? []).includes(String(b)) || (n.in ?? []).includes(String(b)));
  };
  const onShift = (e: KeyboardEvent) => {
    if (e.key !== 'Shift') return;
    shiftDown = e.type === 'keydown';
    if (!shiftDown) trace = [];
  };

  const allocate = (nodeId: number) => {
    if (shiftDown && trace.length && trace[trace.length - 1] === nodeId) {
      void syncWrap?.AllocateNodes([...trace].filter((id) => !activeNodes.includes(id)));
      currentBuild.set($currentBuild);
      trace = [];
      return;
    }
    allocationPaths
      .then((paths) => {
        const path = calculateAllocationPath(paths, nodeId);
        if (!path) {
          return;
        }
        void syncWrap?.AllocateNodes(leapHeld.has(nodeId) ? [nodeId] : path);
        currentBuild.set($currentBuild);
      })
      .catch(logError);
  };

  let clickNode = (node: Node) => {
    const nodeId = node.skill ?? -1;
    if (activeNodes?.includes(nodeId)) {
      const cut = syncWrap?.CalculateDependentNodesWith(skillTreeVersion, [...activeNodes], nodeId, [...classRoots()], subgraphNodes());
      syncWrap
        ?.DeallocateNodes([nodeId])
        .then(() => {
          cut
            ?.then((nodeIds) => {
              if (nodeIds) {
                void syncWrap?.DeallocateNodes(nodeIds);
              }
              currentBuild.set($currentBuild);
            })
            .catch(logError);
        })
        .catch(logError);
    } else if (node.ascendancyName) {
      // PassiveTreeView.lua:390-497
      syncWrap
        ?.ClickTreeAscendancyNode(nodeId)
        .then((r) => {
          if (!r?.handled) allocate(nodeId);
          else if (r.needsConfirm) onclassconfirm?.(r.className, (decision) => syncWrap?.ClickTreeAscendancyNode(nodeId, decision));
          else if (r.changed) currentBuild.set($currentBuild);
        })
        .catch(logError);
    } else {
      allocate(nodeId);
    }
  };

  export const toggle = (node: Node) => clickNode(node);

  let cdnBase = $derived(`https://data.pob.gg/data/${skillTreeVersion.replace('_', '.')}`);

  let mousePos = $state<Point>({
    x: Number.MIN_VALUE,
    y: Number.MIN_VALUE
  });

  let cursor = $state('unset');

  const hoverPath = writable<number[]>([]);

  let parentContainer = $state<HTMLElement>();

  let canvasWidth = $state(0);
  let canvasHeight = $state(0);
  const resize = () => {
    if (parentContainer) {
      canvasWidth = parentContainer.offsetWidth;
      canvasHeight = parentContainer.offsetHeight;
    }
  };

  const hoveredNode = writable<Node | undefined>();

  const scene = buildScene(skillTree);
  const placedById = new Map(scene.nodes.map((n) => [n.id, n]));
  const clusterLayer = $derived(
    buildClusterLayer(skillTree, [...(cluster ?? []), ...(compareCluster ?? [])], (id) => placedById.get(id), new Set((compareCluster ?? []).map((c) => c.id)))
  );
  $effect(() =>
    keepForOffline(
      scene.sheets.map((s) => `${cdnBase}/${s}`),
      'tree'
    )
  );

  const CELL = 256;
  const MAX_TOUCH = 110;
  const touchDistance = (node: Node) => {
    if (node.classStartIndex !== undefined || node.isAscendancyStart) return 0;
    if (node.isKeystone) return 110;
    if (node.isNotable || node.isJewelSocket) return 70;
    if (node.isMastery) return 85;
    return 50;
  };
  const cellKey = (cx: number, cy: number) => cx * 4096 + cy;
  const hoverGrid = new Map<number, number[]>();
  scene.nodes.forEach((n, i) => {
    if (touchDistance(n.node) === 0) return;
    const k = cellKey(Math.floor(n.at.x / CELL), Math.floor(n.at.y / CELL));
    hoverGrid.set(k, [...(hoverGrid.get(k) ?? []), i]);
  });
  const nodeAt = (x: number, y: number): Node | undefined => {
    let best = -1;
    for (let cx = Math.floor((x - MAX_TOUCH) / CELL); cx <= Math.floor((x + MAX_TOUCH) / CELL); cx++) {
      for (let cy = Math.floor((y - MAX_TOUCH) / CELL); cy <= Math.floor((y + MAX_TOUCH) / CELL); cy++) {
        for (const i of hoverGrid.get(cellKey(cx, cy)) ?? []) {
          if ((best < 0 || i < best) && Math.hypot(scene.nodes[i].at.x - x, scene.nodes[i].at.y - y) < touchDistance(scene.nodes[i].node)) best = i;
        }
      }
    }
    if (best >= 0) return scene.nodes[best].node;
    // PassiveSpec.lua:2043-2064
    return clusterLayer.nodes.find((n) => !n.node.isMastery && Math.hypot(n.at.x - x, n.at.y - y) < touchDistance(n.node))?.node;
  };

  $effect(() => {
    const node = nodeAt(mousePos.x * scaling - Math.abs(skillTree.min_x) - offsetX, mousePos.y * scaling - Math.abs(skillTree.min_y) - offsetY);
    if (node !== get(hoveredNode)) {
      hoveredNode.set(node);
      onhover?.(node);
    }
  });

  $effect(() => {
    if (!skillTree) {
      return;
    }

    if ($hoveredNode !== undefined && currentClass) {
      const target = $hoveredNode.skill!;
      const tracing = shiftDown;
      allocationPaths
        .then((paths) => {
          const path = calculateAllocationPath(paths, target);
          if (tracing) {
            // PassiveTreeView.lua:283-363
            if (!get(hoveredNode)) return;
            const own = activeNodes.includes(target) ? undefined : path;
            trace = extendTrace(trace, target, own, linkedNodes, (id) => !!skillTree.nodes[id]?.isMastery);
            hoverPath.set(trace);
            return;
          }
          if (path && get(hoveredNode) && leapHeld.has(target)) {
            // PassiveTreeView.lua:367
            hoverPath.set([]);
          } else if (path && get(hoveredNode)) {
            const anchor = paths[path[path.length - 1]];
            hoverPath.set(anchor !== undefined && anchor !== -1 ? [...path, anchor] : path);
          }
        })
        .catch(logError);
    } else {
      hoverPath.set([]);
    }
  });

  let down = $state(false);
  let dragging = $state(false);

  // PassiveTreeView.lua:372-375
  let hoverDepends = $state.raw(new Set<number>());
  $effect(() => {
    const node = $hoveredNode;
    const id = node?.skill;
    if (id === undefined || dragging || !activeNodes.includes(id)) {
      hoverDepends = new Set();
      return;
    }
    syncWrap
      ?.CalculateDependentNodesWith(skillTreeVersion, [...activeNodes], id, [...classRoots()], subgraphNodes())
      .then((ids) => {
        if (get(hoveredNode) === node) hoverDepends = new Set([id, ...(ids ?? [])]);
      })
      .catch(logError);
  });

  // PassiveTreeView.lua:1658-1706
  let grantedPassives: number[] = $state.raw([]);
  let wantedDiff: Node | undefined;
  let diffRunning = false;

  $effect(() => {
    void $outputs;
    void activeNodes;
    statDiff = undefined;
    if (!whatIf) return;
    syncWrap
      ?.PrepareStatCompare()
      .then(() => syncWrap?.GetGrantedPassives())
      .then((g) => (grantedPassives = g ?? []))
      .catch(logError);
  });

  const statDiffFor = async (node: Node) => {
    const id = node.skill!;
    const current = () => get(hoveredNode) === node && wantedDiff === undefined;
    const active = activeNodes;
    const alloc = active.includes(id);
    const isGranted = !alloc && grantedPassives.includes(id);

    let path: number[];
    if (alloc) {
      const ids = (await syncWrap?.CalculateDependentNodesWith(skillTreeVersion, [...active], id, [...classRoots()], subgraphNodes())) ?? [];
      if (get(hoveredNode) === node) dependents = { node: id, ids };
      path = [id, ...ids];
    } else {
      path = calculateAllocationPath(await allocationPaths, id) ?? [];
    }
    const pathLength = path.length;
    const showPath = pathLength > 1 && !isGranted;

    const blocks: StatDiffBlock[] = [];
    statDiff = { node: id, blocks, pending: true };

    const compare = async (add: number[], remove: number[], header: string, nodeCount?: number) => {
      if (!current()) return false;
      const result = await syncWrap?.StatCompareNodes(add, remove, nodeCount);
      if (!result || !current()) return false;
      if (result.sections.length) {
        blocks.push({ header, sections: result.sections, readyAt: performance.now(), calcMs: result.calcMs });
      }
      return true;
    };

    let pathNodes = path;
    let ok: boolean;
    if (alloc) {
      ok = await compare([], [id], 'Unallocating this node will give you:');
    } else if (isGranted) {
      ok = await compare([], [id], 'This node is granted by an item. Removing it will give you:');
    } else if (node.isMastery) {
      pathNodes = path.filter((n) => n !== id);
      ok = current();
    } else {
      ok = await compare([id], [], 'Allocating this node will give you:');
    }
    if (!ok) return;
    statDiff = { node: id, blocks: [...blocks], pending: showPath };

    if (showPath) {
      ok = alloc
        ? await compare([], pathNodes, 'Unallocating this node and all nodes depending on it will give you:', pathLength)
        : await compare(pathNodes, [], 'Allocating this node and all nodes leading to it will give you:', pathLength);
      if (!ok) return;
    }

    let empty: string | undefined;
    if (blocks.length === 0) {
      empty = isGranted
        ? 'This node is granted by an item. Removing it will cause no changes'
        : // PassiveTreeView.lua:1703
          `No changes from ${alloc ? 'unallocating' : 'allocating'} this node.`;
    }
    statDiff = { node: id, blocks: [...blocks], pending: false, empty };
  };

  const runStatDiffs = async () => {
    while (wantedDiff) {
      const node = wantedDiff;
      wantedDiff = undefined;
      await statDiffFor(node).catch(logError);
    }
  };

  const requestStatDiff = (node: Node | undefined) => {
    wantedDiff = node;
    if (!node) {
      statDiff = undefined;
      return;
    }
    if (!diffRunning) {
      diffRunning = true;
      void runStatDiffs().finally(() => (diffRunning = false));
    }
  };

  $effect(() => {
    void activeNodes;
    // PassiveTreeView.lua:1146
    const node = $hoveredNode;
    requestStatDiff(!whatIf || dragging || compareCluster?.some((c) => c.id === node?.skill) ? undefined : node);
  });

  let devicePixelRatio = $state(1);
  let glCanvas = $state<HTMLCanvasElement>();
  let overlay = $state<HTMLCanvasElement>();
  let renderer = $state.raw<TreeRenderer>();
  let lastDrawMs = $state(0);
  let overlayFrame = 0;

  const drawOverlay = () => {
    overlayFrame = 0;
    const context = overlay?.getContext('2d');
    if (!context) {
      return;
    }
    context.setTransform(devicePixelRatio, 0, 0, devicePixelRatio, 0, 0);
    context.clearRect(0, 0, canvasWidth, canvasHeight);
  };
  const requestOverlay = () => {
    if (overlayFrame === 0) {
      overlayFrame = requestAnimationFrame(drawOverlay);
    }
  };

  // PassiveTreeView.lua:906-949
  const nodeTint = $derived.by(() => {
    const out = new Map(baseTint ?? []);
    if (!heat) return out;
    const active = new Set(activeNodes);
    for (const { node } of scene.nodes) {
      const id = node.skill;
      if (id === undefined || active.has(id) || node.classStartIndex !== undefined || node.isAscendancyStart) continue;
      const [off, def] = heat.mix?.mix.get(id) ?? [0, 0];
      out.set(
        id,
        heat.mix
          ? heatMixColour(off, def, heat.mix.maxOffence, heat.mix.maxDefence, $nodePowerTheme)
          : heatSingleColour(heat.values.get(id) ?? 0, heat.max, $nodePowerTheme)
      );
    }
    return out;
  });

  const afterDraw = (ms: number) => {
    if ($devMode) lastDrawMs = ms;
    drawOverlay();
    treeProbe()?.frame({
      view: { offsetX, offsetY, scaling, width: canvasWidth, height: canvasHeight },
      ms,
      at: performance.now(),
      hovered: get(hoveredNode)?.skill,
      hoverPath: get(hoverPath).length,
      active: activeNodes.length,
      className: currentClass,
      version: skillTreeVersion,
      renderer: renderer?.kind,
      complete: renderer?.complete()
    });
  };

  $effect(() => {
    if (!glCanvas || renderer) {
      return;
    }
    const probe = treeProbe();
    renderer = createRenderer(glCanvas, scene, cdnBase, afterDraw, probe?.renderer);
    if (probe && renderer.kind === 'webgl') {
      probe.drawList = () => ({ complete: renderer!.complete(), items: sceneDrawList(scene, scaling, renderer!.drawListArt()) });
      probe.selfCheck = () => renderer!.selfCheck?.() ?? ['no self-check'];
    }
  });

  $effect(() => {
    renderer?.setView({ offsetX, offsetY, scaling, width: canvasWidth, height: canvasHeight, dpr: devicePixelRatio });
    view = { offsetX, offsetY, scaling, width: canvasWidth, height: canvasHeight };
  });

  const scaleAbout = (x: number, y: number, newScaling: number) => {
    offsetX += x * (newScaling - scaling);
    offsetY += y * (newScaling - scaling);
    scaling = newScaling;
  };

  export const zoom = (how: 'in' | 'out' | 'fit') => {
    if (how === 'fit') {
      const w = skillTree.max_x - skillTree.min_x;
      const h = skillTree.max_y - skillTree.min_y;
      scaling = Math.min(MAX_SCALING, Math.max(MIN_SCALING, Math.max(w / canvasWidth, h / canvasHeight)));
      offsetX = (canvasWidth / 2) * scaling - Math.abs(skillTree.min_x) - (skillTree.min_x + w / 2);
      offsetY = (canvasHeight / 2) * scaling - Math.abs(skillTree.min_y) - (skillTree.min_y + h / 2);
      return;
    }
    const step = 2 * Math.pow(2, $zoomSensitivity - 5) * 1.2;
    scaleAbout(canvasWidth / 2, canvasHeight / 2, Math.min(MAX_SCALING, Math.max(MIN_SCALING, scaling + (how === 'in' ? -step : step))));
  };

  // PassiveTreeView.lua:83-111
  const baseScaling = () => (skillTree.max_x - skillTree.min_x) / Math.max(1, Math.min(canvasWidth, canvasHeight));
  export const pobView = () => ({
    zoomLevel: Math.round(Math.log(baseScaling() / scaling) / Math.log(1.2)),
    zoomX: (Math.abs(skillTree.min_x) + offsetX) / scaling - canvasWidth / 2,
    zoomY: (Math.abs(skillTree.min_y) + offsetY) / scaling - canvasHeight / 2
  });
  export const setPobView = (v: { zoomLevel: number; zoomX: number; zoomY: number }) => {
    if (!canvasWidth || !canvasHeight) return;
    scaling = Math.min(MAX_SCALING, Math.max(MIN_SCALING, baseScaling() / 1.2 ** v.zoomLevel));
    offsetX = (v.zoomX + canvasWidth / 2) * scaling - Math.abs(skillTree.min_x);
    offsetY = (v.zoomY + canvasHeight / 2) * scaling - Math.abs(skillTree.min_y);
  };

  // phone: a view saved on a desktop-sized window points off a phone's canvas, so it opens on the allocated nodes
  let fitPending = $state(false);
  export const fitActive = () => (fitPending = true);
  $effect(() => {
    if (!fitPending || !canvasWidth || !canvasHeight || !activeNodes.length) return;
    fitPending = false;
    const ids = new Set(activeNodes);
    const at = scene.nodes.filter((n) => ids.has(n.id) && !n.node.ascendancyName).map((n) => n.at);
    if (!at.length) return;
    const [x0, x1] = [Math.min(...at.map((p) => p.x)), Math.max(...at.map((p) => p.x))];
    const [y0, y1] = [Math.min(...at.map((p) => p.y)), Math.max(...at.map((p) => p.y))];
    scaling = Math.min(MAX_SCALING, Math.max(MIN_SCALING, Math.max((x1 - x0) / (canvasWidth * 0.9), (y1 - y0) / (canvasHeight * 0.9))));
    offsetX = (canvasWidth / 2) * scaling - Math.abs(skillTree.min_x) - (x0 + x1) / 2;
    offsetY = (canvasHeight / 2) * scaling - Math.abs(skillTree.min_y) - (y0 + y1) / 2;
  });

  /* PassiveTreeView.lua:1312-1321 */
  export const focus = (id: number, to = MIN_SCALING) => {
    const n = scene.nodes.find((s) => s.node.skill === id) ?? clusterLayer.nodes.find((s) => s.id === id);
    if (!n) return;
    scaling = Math.min(MAX_SCALING, Math.max(MIN_SCALING, to));
    offsetX = (canvasWidth / 2) * scaling - Math.abs(skillTree.min_x) - n.at.x;
    offsetY = (canvasHeight / 2) * scaling - Math.abs(skillTree.min_y) - n.at.y;
  };

  $effect(() => {
    renderer?.setState({
      active: new Set(activeNodes),
      hoverPath: new Set($hoverPath),
      hovered: $hoveredNode?.skill,
      className: currentClass,
      ascendancy: currentAscendancy,
      secondaryAscendancy: currentSecondaryAscendancy,
      socketArt,
      tattoos,
      cluster: clusterLayer,
      tint: nodeTint,
      granted: new Set(grantedPassives),
      compared: compared && new Set(compared),
      split: split && new Set(split),
      heatMap: !!heat,
      depends: hoverDepends
    });
  });

  $effect(() => {
    cursor = $hoveredNode ? 'pointer' : 'unset';
    void mousePos;
    requestOverlay();
  });

  let downX = 0;
  let downY = 0;

  let startX = 0;
  let startY = 0;

  const canvasPoint = (event: MouseEvent): Point | undefined => {
    if (!glCanvas) return undefined;
    const r = glCanvas.getBoundingClientRect();
    return { x: event.clientX - r.left, y: event.clientY - r.top };
  };
  const NOWHERE = { x: -1e6, y: -1e6 };

  // PassiveTreeView.lua:259-263
  const ctrlZoom = (event: MouseEvent, how: 'in' | 'out') => {
    const step = 2 * Math.pow(2, $zoomSensitivity - 5) * 1.2;
    scaleAbout(event.offsetX, event.offsetY, Math.min(MAX_SCALING, Math.max(MIN_SCALING, scaling + (how === 'in' ? -step : step))));
  };

  const mouseDown = (event: Event) => {
    if (event instanceof MouseEvent) {
      if (event.ctrlKey && event.button === 0) {
        ctrlZoom(event, 'in');
        return;
      }
      if (event.button === 2 && (onmenu || event.ctrlKey)) {
        return;
      }
      down = true;
      dragging = false;
      downX = event.offsetX;
      downY = event.offsetY;
      startX = offsetX;
      startY = offsetY;

      mousePos = {
        x: event.offsetX,
        y: event.offsetY
      };
    }
  };

  const contextMenu = (event: MouseEvent) => {
    if (event.ctrlKey) {
      event.preventDefault();
      ctrlZoom(event, 'out');
      return;
    }
    if (!onmenu) return;
    event.preventDefault();
    const node = $hoveredNode;
    if (node) onmenu(node, { x: event.offsetX, y: event.offsetY });
  };

  const mouseUp = (event: PointerEvent) => {
    // PassiveTreeView.lua:221-226
    const click = event.type === 'pointerup' && down && !dragging && event.button === 0;
    if (event.type === 'pointerup') {
      down = false;
      dragging = false;
    }

    mousePos = event.target === glCanvas ? (canvasPoint(event) ?? NOWHERE) : NOWHERE;
    if (click && $hoveredNode) {
      if (onpick) onpick($hoveredNode);
      else clickNode($hoveredNode);
    }
  };

  const mouseMove = (event: MouseEvent) => {
    const p = canvasPoint(event);
    // PassiveTreeView.lua:244-251
    if (down && p && !dragging && (Math.abs(p.x - downX) > 5 || Math.abs(p.y - downY) > 5)) dragging = true;
    if (dragging && p) {
      offsetX = startX - (downX - p.x) * scaling;
      offsetY = startY - (downY - p.y) * scaling;
    }

    mousePos = event.target === glCanvas || down ? (p ?? NOWHERE) : NOWHERE;
  };

  // the node under each press, by the id the gesture classifier carries
  const pressed = new Map<number, Node>();
  const canvasAt = (p: Point): Point => {
    const r = glCanvas?.getBoundingClientRect();
    return r ? { x: p.x - r.left, y: p.y - r.top } : NOWHERE;
  };
  const touch = gestureDriver({
    gesture: (g, at) => {
      const node = g.node !== undefined ? pressed.get(g.node) : undefined;
      if (g.kind === 'tap') mousePos = canvasAt(at);
      else if (g.kind === 'double' && node) {
        if (onpick) onpick(node);
        else clickNode(node);
      } else if (g.kind === 'hold' && node && onmenu) {
        navigator.vibrate?.(15);
        mousePos = NOWHERE;
        onmenu(node, canvasAt(at));
      }
    },
    pan: (dx, dy) => {
      offsetX += dx * scaling;
      offsetY += dy * scaling;
    },
    pinch: (f, mid) => {
      const p = canvasAt(mid);
      scaleAbout(p.x, p.y, Math.min(MAX_SCALING, Math.max(MIN_SCALING, scaling / f)));
    }
  });
  const touchDown = (event: PointerEvent) => {
    const p = canvasPoint(event);
    const node = p && nodeAt(p.x * scaling - Math.abs(skillTree.min_x) - offsetX, p.y * scaling - Math.abs(skillTree.min_y) - offsetY);
    if (node?.skill !== undefined) pressed.set(node.skill, node);
    touch.down(event, node?.skill);
  };

  const onScroll = (event: Event) => {
    if (event instanceof WheelEvent) {
      const scalingAdjustment = Math.pow(2, $zoomSensitivity - 5) * (event.deltaY / 100);
      const newScaling = Math.min(MAX_SCALING, Math.max(MIN_SCALING, scaling + scalingAdjustment));

      scaleAbout(event.offsetX, event.offsetY, newScaling);

      event.preventDefault();
      event.stopPropagation();
      event.stopImmediatePropagation();
    }
  };

  let initialized = $state(false);
  $effect(() => {
    if (!initialized && skillTree) {
      initialized = true;
      offsetX = skillTree.min_x + (window.innerWidth / 2) * scaling;
      offsetY = skillTree.min_y + (window.innerHeight / 2) * scaling;
    }
    resize();
  });

  onMount(() => {
    new ResizeObserver(resize).observe(parentContainer!);
    resize();
    return () => {
      renderer?.destroy();
      renderer = undefined;
      cancelAnimationFrame(overlayFrame);
    };
  });
</script>

<svelte:window
  onpointerup={gestures ? touch.up : mouseUp}
  onpointercancel={gestures ? touch.up : undefined}
  onpointermove={gestures ? touch.move : mouseMove}
  onresize={resize}
  onkeydown={onShift}
  onkeyup={onShift}
  onblur={() => ((shiftDown = false), (trace = []))}
  bind:devicePixelRatio />

<div class="relative w-full h-full max-w-full max-h-full overflow-hidden" bind:this={parentContainer}>
  {#if chrome}
    <NodeSearch {activeNodes} onselect={clickNode} />
    <p class="pointer-events-none absolute right-2 bottom-1 z-sticky m-0 text-11 text-text-2" aria-label="Tree version">
      Tree {skillTreeVersion.replace('_', '.')}{recordedTreeVersion && recordedTreeVersion !== skillTreeVersion
        ? ` (build records ${recordedTreeVersion.replace('_', '.')})`
        : ''}
    </p>
  {/if}
  {#if canvasWidth && canvasHeight}
    <div
      role="img"
      aria-label="Passive skill tree"
      data-heat={heat ? heat.values.size : undefined}
      class="relative"
      style="touch-action: none; cursor: {cursor}; width: {canvasWidth}px; height: {canvasHeight}px">
      <canvas
        bind:this={glCanvas}
        class="absolute inset-0 w-full h-full"
        onpointerdown={gestures ? touchDown : mouseDown}
        onwheel={onScroll}
        oncontextmenu={gestures ? (e) => e.preventDefault() : contextMenu}></canvas>
      <canvas
        bind:this={overlay}
        class="absolute inset-0 w-full h-full pointer-events-none"
        width={canvasWidth * devicePixelRatio}
        height={canvasHeight * devicePixelRatio}></canvas>
      {#if $devMode}
        <output aria-label="Render time" class="pointer-events-none absolute right-1.5 bottom-1 font-mono text-12 text-text">{lastDrawMs.toFixed(1)}ms</output>
      {/if}
      {#if chrome && $hoveredNode}
        <NodeTooltip node={$hoveredNode} at={mousePos} width={canvasWidth} height={canvasHeight} devMode={$devMode} diff={statDiff} />
      {/if}
    </div>
  {/if}
</div>
