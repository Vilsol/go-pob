<script lang="ts">
  import type { Node } from '../../skill_tree/types';
  import { calculateNodePos, drawnNodes, toCanvasCoords, orbitAngleAt, drawnGroups, ascendancyGroupPositionOffsets, scaling } from '../../skill_tree';
  import { onMount } from 'svelte';
  import type { Tree } from '../../skill_tree/types';
  import { Container, Graphics } from 'pixi.js';

  interface Props {
    hoverPath: number[];
    skillTree: Tree;
    activeNodes: number[];
    parentContainer: Container;
  }

  let { hoverPath, skillTree, activeNodes, parentContainer }: Props = $props();

  const container = new Container();

  let hoverSet = $derived(new Set(hoverPath));
  let activeSet = $derived(new Set(activeNodes));

  onMount(() => {
    parentContainer.addChild(container);

    const connected: Record<string, boolean> = {};
    drawnNodes.keys().forEach((nNodeId) => {
      const node: Node = drawnNodes.get(nNodeId)!;

      // Do not draw connections out of class starting nodes
      if (node.classStartIndex !== undefined) {
        return;
      }

      const angle = orbitAngleAt(node.orbit!, node.orbitIndex!);

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

        const targetAngle = orbitAngleAt(targetNode.orbit!, targetNode.orbitIndex!);

        if (node.group != targetNode.group || node.orbit != targetNode.orbit) {
          const canvasPos = calculateNodePos(node);
          const targetCanvasPos = calculateNodePos(targetNode);

          const graphics = new Graphics();

          graphics.moveTo(canvasPos.x, canvasPos.y);
          graphics.lineTo(targetCanvasPos.x, targetCanvasPos.y);
          graphics.stroke({ width: 1.5, color: 0x524518 });

          container.addChild(graphics);

          $effect(() => {
            graphics.clear();

            graphics.moveTo(canvasPos.x, canvasPos.y);
            graphics.lineTo(targetCanvasPos.x, targetCanvasPos.y);

            if (activeSet.has(node.skill!) && activeSet.has(targetNode.skill!)) {
              graphics.stroke({ width: 3, color: 0xe9deb6 });
            } else if (hoverSet.has(node.skill!) && hoverSet.has(targetNode.skill!)) {
              graphics.stroke({ width: 1.5, color: 0xc89c01 });
            } else {
              graphics.stroke({ width: 1.5, color: 0x524518 });
            }
          });
        } else {
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

          const groupPos = toCanvasCoords(posX, posY);

          const graphics = new Graphics();
          graphics.arc(groupPos.x, groupPos.y, skillTree.constants.orbitRadii[node.orbit!] / scaling + 1, finalA, finalB);
          graphics.stroke({ width: 1.5, color: 0x524518 });
          container.addChild(graphics);

          $effect(() => {
            graphics.clear();

            graphics.arc(groupPos.x, groupPos.y, skillTree.constants.orbitRadii[node.orbit!] / scaling + 1, finalA, finalB);

            if (activeSet.has(node.skill!) && activeSet.has(targetNode.skill!)) {
              graphics.stroke({ width: 2.5, color: 0xe9deb6 });
            } else if (hoverSet.has(node.skill!) && hoverSet.has(targetNode.skill!)) {
              graphics.stroke({ width: 1.5, color: 0xc89c01 });
            } else {
              graphics.stroke({ width: 1.5, color: 0x524518 });
            }
          });
        }
      });
    });

    return () => {
      container.destroy({
        children: true
      });
    };
  });
</script>
