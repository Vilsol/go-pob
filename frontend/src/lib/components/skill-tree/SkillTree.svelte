<script lang="ts">
  import { Canvas, Layer, type Render } from 'svelte-canvas';
  import type { Node, Tree } from '$lib/skill_tree/types';
  import { calculateNodePos, distance, drawnNodes, classStartNodes, type Point } from '../../skill_tree';
  import { onMount } from 'svelte';
  import { currentBuild, zoomSensitivity } from '$lib/global';
  import { syncWrap } from '$lib/go/worker';
  import { get, writable } from 'svelte/store';
  import { logError } from '$lib/utils';
  import AllSkillNodes from '$lib/components/skill-tree/AllSkillNodes.svelte';
  import AllConnections from '$lib/components/skill-tree/AllConnections.svelte';
  import AllGroups from '$lib/components/skill-tree/AllGroups.svelte';
  import ClassImage from '$lib/components/skill-tree/ClassImage.svelte';
  import Tooltip from '$lib/components/skill-tree/Tooltip.svelte';
  import { calculateAllocationPath, type PrecalculatedAllocationPaths } from './paths';

  let currentClass: string | undefined = $state();
  $effect(() => {
    $currentBuild?.Build.ClassName.then((newClass) => (currentClass = newClass)).catch(logError);
  });

  let currentAscendancy: string | undefined = $state();
  $effect(() => {
    $currentBuild?.Build.AscendClassName.then((newAscendancy) => (currentAscendancy = newAscendancy)).catch(logError);
  });

  interface Props {
    skillTree: Tree;
    skillTreeVersion: string;
  }

  let { skillTree, skillTreeVersion }: Props = $props();

  const cullingPadding = -150;

  let scaling = $state(10);

  let offsetX = $state(0);
  let offsetY = $state(0);

  let activeNodes: number[] = $state([]);

  $effect(() => {
    $currentBuild?.Build?.PassiveNodes?.then((newNodes) => (activeNodes = newNodes ?? [])).catch(logError);
  });

  function precalculateAllocationPaths(active: number[]): Promise<PrecalculatedAllocationPaths> {
    const version = skillTreeVersion || '3_18';
    const rootNodes = classStartNodes[skillTree.classes.findIndex((c) => c.name === currentClass)];
    return syncWrap
      ?.CalculateAllocationPaths(
        version,
        // Need to clone; can't directly marshal a state-proxy to go
        [...active],
        rootNodes
      )
      .then((paths) => paths ?? {})
      .catch((e: Error) => {
        logError(e);
        return {};
      });
  }

  // Must be recalculated when the active nodes change or when the graph topology
  // changes (e.g. socketing a Thread of Hope).
  let allocationPaths: Promise<PrecalculatedAllocationPaths> = $derived(precalculateAllocationPaths(activeNodes));

  let clickNode = (node: Node) => {
    const nodeId = node.skill ?? -1;
    if (activeNodes?.includes(nodeId)) {
      const version = skillTreeVersion || '3_18';
      const rootNodes = classStartNodes[skillTree.classes.findIndex((c) => c.name === currentClass)];
      syncWrap
        ?.DeallocateNodes([nodeId])
        .then(() => {
          const nowActiveNodes = activeNodes.filter((n) => n !== nodeId);
          syncWrap
            ?.CalculatePrunableNodes(version, nowActiveNodes, rootNodes)
            .then((nodeIds) => {
              if (nodeIds) {
                void syncWrap?.DeallocateNodes(nodeIds);
              }
              currentBuild.set($currentBuild);
            })
            .catch(logError);
        })
        .catch(logError);
    } else {
      allocationPaths
        .then((paths) => {
          const path = calculateAllocationPath(paths, nodeId);
          if (!path) {
            return;
          }
          void syncWrap?.AllocateNodes(path);
          currentBuild.set($currentBuild);
        })
        .catch(logError);
    }
  };

  const drawScaling = 2.6;

  let cdnBase = $derived(`https://go-pob-data.pages.dev/data/${(skillTreeVersion || '3_18').replace('_', '.')}`);

  let mousePos = $state<Point>({
    x: Number.MIN_VALUE,
    y: Number.MIN_VALUE
  });

  let cursor = $state('unset');

  const hoverPath = writable<number[]>([]);

  let start: DOMHighResTimeStamp;

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
  let visibleNodePositions = $state<Map<number, Point>>(new Map<number, Point>());

  $effect(() => {
    let found = false;
    const visibleNodePos: Map<number, Point> = new Map<number, Point>();

    drawnNodes.forEach((node: Node, nNodeId: number) => {
      const canvasPos = calculateNodePos(node, offsetX, offsetY, scaling);

      if (!found) {
        let touchDistance = 0;

        if (node.classStartIndex !== undefined) {
          // No touch distance for class start
        } else if (node.isAscendancyStart) {
          // No touch distance for ascendancy start
        } else if (node.isKeystone) {
          touchDistance = 110;
        } else if (node.isNotable) {
          touchDistance = 70;
        } else if (node.isJewelSocket) {
          touchDistance = 70;
        } else if (node.isMastery) {
          touchDistance = 85;
        } else {
          touchDistance = 50;
        }

        if (distance(canvasPos, mousePos) < touchDistance / scaling) {
          hoveredNode.set(node);
          found = true;
        }
      }

      if (
        !(
          canvasPos.x < cullingPadding ||
          canvasPos.x > canvasWidth - cullingPadding ||
          canvasPos.y < cullingPadding ||
          canvasPos.y > canvasHeight - cullingPadding
        )
      ) {
        visibleNodePos.set(nNodeId, canvasPos);
      }
    });

    if (!found) {
      hoveredNode.set(undefined);
    }

    visibleNodePositions = visibleNodePos;
  });

  $effect(() => {
    if (!skillTree) {
      return;
    }

    if ($hoveredNode !== undefined && currentClass) {
      const target = $hoveredNode.skill!;
      allocationPaths
        .then((paths) => {
          const path = calculateAllocationPath(paths, target);
          if (path && get(hoveredNode)) {
            hoverPath.set(path);
          }
        })
        .catch(logError);
    } else {
      hoverPath.set([]);
    }
  });

  const renderStart: Render = ({ context, width, height }) => {
    start = window.performance.now();

    context.fillStyle = '#080c11';
    context.fillRect(0, 0, width, height);
  };

  const renderEnd: Render = ({ context, width, height }) => {
    if ($hoveredNode) {
      cursor = 'pointer';
    } else {
      cursor = 'unset';
    }

    context.fillStyle = '#ffffff';
    context.textAlign = 'right';
    context.font = '12px Roboto Mono';

    const end = window.performance.now();

    context.fillText(`${(end - start).toFixed(1)}ms`, width - 5, 17);

    context.strokeStyle = 'red';
    context.strokeRect(cullingPadding, cullingPadding, width - cullingPadding * 2, height - cullingPadding * 2);
  };

  let downX = 0;
  let downY = 0;

  let startX = 0;
  let startY = 0;

  let down = false;
  const mouseDown = (event: Event) => {
    if (event instanceof MouseEvent) {
      down = true;
      downX = event.offsetX;
      downY = event.offsetY;
      startX = offsetX;
      startY = offsetY;

      mousePos = {
        x: event.offsetX,
        y: event.offsetY
      };

      if ($hoveredNode) {
        clickNode($hoveredNode);
      }
    }
  };

  const mouseUp = (event: PointerEvent) => {
    if (event.type === 'pointerup') {
      down = false;
    }

    mousePos = {
      x: event.offsetX,
      y: event.offsetY
    };
  };

  const mouseMove = (event: MouseEvent) => {
    if (down) {
      offsetX = startX - (downX - event.offsetX) * scaling;
      offsetY = startY - (downY - event.offsetY) * scaling;
    }

    mousePos = {
      x: event.offsetX,
      y: event.offsetY
    };
  };

  const onScroll = (event: Event) => {
    if (event instanceof WheelEvent) {
      const scalingAdjustment = Math.pow(2, $zoomSensitivity - 5) * (event.deltaY / 100);
      const newScaling = Math.min(30, Math.max(3, scaling + scalingAdjustment));

      // Keep the rescaled mouse position constant
      offsetX += event.offsetX * (newScaling - scaling);
      offsetY += event.offsetY * (newScaling - scaling);
      scaling = newScaling;

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
  });
</script>

<svelte:window onpointerup={mouseUp} onpointermove={mouseMove} onresize={resize} />

<div class="w-full h-full max-w-full max-h-full overflow-hidden" bind:this={parentContainer}>
  {#if canvasWidth && canvasHeight}
    <div style="touch-action: none; cursor: {cursor}">
      <Canvas width={canvasWidth} height={canvasHeight} onpointerdown={mouseDown} onwheel={onScroll}>
        <Layer render={renderStart} />
        <ClassImage {scaling} {offsetX} {offsetY} {cullingPadding} {drawScaling} {currentClass} {cdnBase} {skillTree} />
        <AllGroups {scaling} {offsetX} {offsetY} {cullingPadding} {currentAscendancy} {currentClass} {cdnBase} {skillTree} />
        <AllConnections {scaling} {offsetX} {offsetY} {cullingPadding} hoverPath={$hoverPath} {skillTree} {activeNodes} />
        <AllSkillNodes
          hoveredNode={$hoveredNode}
          {cdnBase}
          {scaling}
          {offsetX}
          {offsetY}
          {cullingPadding}
          hoverPath={$hoverPath}
          visibleNodePos={visibleNodePositions}
          {activeNodes} />
        <Tooltip hoveredNode={$hoveredNode} {mousePos} />
        <Layer render={renderEnd} />
      </Canvas>
    </div>
  {/if}
</div>
