<script lang="ts">
  import { T } from '@threlte/core';
  import { OrbitControls, PerfMonitor, Billboard, interactivity } from '@threlte/extras';
  import type { Node, Tree } from '$lib/skill_tree/types';
  import { calculateNodePos, drawnNodes, classStartNodes, type Point } from '../../skill_tree';
  import { currentBuild, zoomSensitivity } from '$lib/global';
  import { syncWrap } from '$lib/go/worker';
  import { writable } from 'svelte/store';
  import { logError } from '$lib/utils';
  import AllSkillNodes from '$lib/components/skill-tree/AllSkillNodes.svelte';
  import AllConnections from '$lib/components/skill-tree/AllConnections.svelte';
  import AllGroups from '$lib/components/skill-tree/AllGroups.svelte';
  import ClassImage from '$lib/components/skill-tree/ClassImage.svelte';
  // import Tooltip from '$lib/components/skill-tree/Tooltip.svelte';
  import { calculateAllocationPath, type PrecalculatedAllocationPaths } from './paths';
  import * as THREE from 'three';
  import { untrack } from 'svelte';

  interactivity();

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
    parentContainer: HTMLElement;
    groupsEnabled: boolean;
    connectionsEnabled: boolean;
    nodesEnabled: boolean;
  }

  let { skillTree, skillTreeVersion, parentContainer, groupsEnabled, connectionsEnabled, nodesEnabled }: Props = $props();

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

  let cdnBase = $derived(`https://go-pob-data.pages.dev/data/${(skillTreeVersion || '3_18').replace('_', '.')}`);

  const hoverPath = writable<number[]>([]);

  let hoveredNode: Node | undefined = $state();
  let visibleNodePositions = $state<Map<number, Point>>(new Map<number, Point>());

  $effect(() => {
    const visibleNodePos: Map<number, Point> = new Map<number, Point>();

    drawnNodes.forEach((node: Node, nNodeId: number) => {
      visibleNodePos.set(nNodeId, calculateNodePos(node));
    });

    visibleNodePositions = visibleNodePos;
  });

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
</script>

<PerfMonitor domElement={parentContainer} logsPerSecond={90} />

<T.PerspectiveCamera makeDefault fov={90} position={[12, 0, 12]}>
  <OrbitControls
    minPolarAngle={0}
    maxPolarAngle={0}
    minAzimuthAngle={0}
    maxAzimuthAngle={0}
    zoomToCursor={true}
    zoomSpeed={$zoomSensitivity}
    mouseButtons={{ LEFT: THREE.MOUSE.PAN, MIDDLE: THREE.MOUSE.DOLLY }} />
</T.PerspectiveCamera>

<T.AmbientLight intensity={1} />

<Billboard>
  <ClassImage {currentClass} {cdnBase} {skillTree} />

  {#if groupsEnabled}
    <AllGroups {currentAscendancy} {currentClass} {cdnBase} {skillTree} />
  {/if}

  {#if connectionsEnabled}
    <AllConnections hoverPath={$hoverPath} {skillTree} {activeNodes} />
  {/if}

  {#if nodesEnabled}
    <AllSkillNodes {cdnBase} hoverPath={$hoverPath} visibleNodePos={visibleNodePositions} {activeNodes} bind:hoveredNode {clickNode} />
  {/if}
</Billboard>
