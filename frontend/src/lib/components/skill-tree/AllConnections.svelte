<script lang="ts">
  import { T } from '@threlte/core';
  import type { Node } from '../../skill_tree/types';
  import { calculateNodePos, drawnNodes, toCanvasCoords, orbitAngleAt, drawnGroups, ascendancyGroupPositionOffsets, scaling } from '../../skill_tree';
  import type { Tree } from '../../skill_tree/types';
  import { EllipseCurve, Vector3 } from 'three';
  import { MeshLineGeometry, MeshLineMaterial } from '@threlte/extras';

  interface Props {
    hoverPath: number[];
    skillTree: Tree;
    activeNodes: number[];
  }

  let { hoverPath, skillTree, activeNodes }: Props = $props();

  interface PrecalculatedConnection {
    node: Node;
    targetNode: Node;
    points: Vector3[];
  }

  let connections: Array<PrecalculatedConnection> = $derived.by(() => {
    const result: Array<PrecalculatedConnection> = [];

    if (!skillTree) {
      return result;
    }

    const connected: Record<string, boolean> = {};
    drawnNodes.keys().forEach((nNodeId) => {
      const node: Node = drawnNodes.get(nNodeId)!;

      // Do not draw connections out of class starting nodes
      if (node.classStartIndex !== undefined) {
        return;
      }

      node.out?.forEach((o) => {
        if (!drawnNodes.get(parseInt(o))) {
          return;
        }

        const min = Math.min(parseInt(o), nNodeId);
        const max = Math.max(parseInt(o), nNodeId);
        const joined = min + ':' + max;

        if (joined in connected) {
          return;
        }
        connected[joined] = true;

        const targetNode = drawnNodes.get(parseInt(o))!;

        // Do not draw connections to mastery nodes
        if (targetNode.isMastery) {
          return;
        }

        // Do not draw connections to ascendancy trees from main tree
        if (node.ascendancyName !== targetNode.ascendancyName) {
          return;
        }

        // Do not draw connections to class starting nodes
        if (targetNode.classStartIndex !== undefined) {
          return;
        }

        if (node.group != targetNode.group || node.orbit != targetNode.orbit) {
          const canvasPos = calculateNodePos(node);
          const targetCanvasPos = calculateNodePos(targetNode);

          result.push({
            node,
            targetNode,
            points: [new Vector3(canvasPos.x, -canvasPos.y, 0.002), new Vector3(targetCanvasPos.x, -targetCanvasPos.y, 0.002)]
          });
        } else {
          const angle = orbitAngleAt(node.orbit!, node.orbitIndex!);
          const targetAngle = orbitAngleAt(targetNode.orbit!, targetNode.orbitIndex!);

          let a = Math.PI / 180 - (Math.PI / 180) * angle;
          let b = Math.PI / 180 - (Math.PI / 180) * targetAngle;

          a -= Math.PI / 2;
          b -= Math.PI / 2;

          const diff = Math.abs(Math.max(a, b) - Math.min(a, b));

          const finalA = diff > Math.PI ? Math.max(a, b) : Math.min(a, b);
          const finalB = diff > Math.PI ? Math.min(a, b) : Math.max(a, b);

          const group = drawnGroups.get(node.group!)!;
          const posX = ((node.ascendancyName && ascendancyGroupPositionOffsets[node.ascendancyName]?.x) || 0) + group.x;
          const posY = ((node.ascendancyName && ascendancyGroupPositionOffsets[node.ascendancyName]?.y) || 0) + group.y;

          const segments = Math.max(angle, targetAngle) - Math.min(angle, targetAngle);

          const groupPos = toCanvasCoords(posX, posY);
          const radius = skillTree.constants.orbitRadii[node.orbit!] / scaling;
          const curve = new EllipseCurve(groupPos.x, -groupPos.y, radius, radius, finalA, finalB, false, 0);
          const points = curve.getPoints(segments / 8).map((p) => new Vector3(p.x, -groupPos.y + (-groupPos.y - p.y), 0.002));

          result.push({
            node,
            targetNode,
            points
          });
        }
      });
    });

    return result;
  });

  let hoverSet: Set<number> = $derived(new Set(hoverPath));
  let activeSet: Set<number> = $derived(new Set(activeNodes));
</script>

{#each connections as connection}
  <T.Mesh>
    <MeshLineGeometry points={connection.points} />
    {#if activeSet.has(connection.node.skill) && activeSet.has(connection.targetNode.skill)}
      <MeshLineMaterial color="#e9deb6" width={0.02} />
    {:else if hoverSet.has(connection.node.skill) && hoverSet.has(connection.targetNode.skill)}
      <MeshLineMaterial color="#c89c01" width={0.01} />
    {:else}
      <MeshLineMaterial color="#524518" width={0.01} />
    {/if}
  </T.Mesh>
{/each}
