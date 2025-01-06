<script lang="ts">
  import type { Node, Tree } from '$lib/skill_tree/types';
  import { classStartNodes, type Point } from '../../skill_tree';
  import { onMount, untrack } from 'svelte';
  import { currentBuild, zoomSensitivity } from '$lib/global';
  import { syncWrap } from '$lib/go/worker';
  import { writable } from 'svelte/store';
  import { logError } from '$lib/utils';
  import AllSkillNodes from '$lib/components/skill-tree/AllSkillNodes.svelte';
  import AllConnections from '$lib/components/skill-tree/AllConnections.svelte';
  import AllGroups from '$lib/components/skill-tree/AllGroups.svelte';
  import ClassImage from '$lib/components/skill-tree/ClassImage.svelte';
  import Tooltip from '$lib/components/skill-tree/Tooltip.svelte';
  import { calculateAllocationPath, type PrecalculatedAllocationPaths } from './paths';
  import { type Application, Container } from 'pixi.js';
  import * as PIXI_VIEWPORT from 'pixi-viewport';

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
    app: Application;
  }

  let { skillTree, skillTreeVersion, app }: Props = $props();

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
        $state.snapshot(active),
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
      void syncWrap?.DeallocateNodes(nodeId);
      currentBuild.set($currentBuild);
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

  const hoverPath = writable<number[]>([]);

  let hoveredNode: Node | undefined = $state();

  $effect(() => {
    if (!skillTree) {
      return;
    }

    if (hoveredNode !== undefined && currentClass) {
      const target = hoveredNode.skill!;
      allocationPaths
        .then((paths) => {
          const path = calculateAllocationPath(paths, target);
          if (path && untrack(() => hoveredNode)) {
            hoverPath.set(path);
          }
        })
        .catch(logError);
    } else {
      hoverPath.set([]);
    }
  });

  const everythingContainer = new Container();

  let pointerPosition: Point = $state({ x: 0, y: 0 });

  let initialized = $state(false);
  onMount(() => {
    app.stage.children = [];

    const viewport = new PIXI_VIEWPORT.Viewport({
      screenWidth: window.innerWidth,
      screenHeight: window.innerHeight,
      events: app.renderer.events,
      passiveWheel: false,
      worldWidth: skillTree.max_x - skillTree.min_x,
      worldHeight: skillTree.max_y - skillTree.min_y
    });

    viewport.scale = 0.2;
    viewport.position.set(app.screen.width / 2, app.screen.height / 2);

    viewport.drag().pinch().wheel({
      percent: $zoomSensitivity
    });

    viewport.addChild(everythingContainer);

    app.stage.addChild(viewport);

    app.stage.eventMode = 'static';
    app.stage.hitArea = app.screen;
    app.stage.addEventListener('pointermove', (e) => (pointerPosition = { ...e.global }));

    initialized = true;

    return () => {
      everythingContainer.destroy({
        children: true
      });
    };
  });
</script>

{#if initialized}
  <ClassImage {currentClass} {skillTree} parentContainer={everythingContainer} />
  <AllGroups {currentAscendancy} {currentClass} {skillTree} parentContainer={everythingContainer} />
  <AllConnections hoverPath={$hoverPath} {skillTree} {activeNodes} parentContainer={everythingContainer} />
  <AllSkillNodes bind:hoveredNode hoverPath={$hoverPath} {activeNodes} parentContainer={everythingContainer} onClick={clickNode} />
  <Tooltip {hoveredNode} {app} {pointerPosition} />
{/if}
