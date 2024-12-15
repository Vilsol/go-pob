<script lang="ts">
  import { Layer, type Render } from 'svelte-canvas';
  import type { Node } from '../../skill_tree/types';
  import { calculateNodePos, drawnNodes, toCanvasCoords, orbitAngleAt, drawnGroups, ascendancyGroupPositionOffsets, type Point } from '../../skill_tree';
  import { onMount } from 'svelte';
  import type { Tree } from '../../skill_tree/types';

  interface Props {
    scaling: number;
    offsetX: number;
    offsetY: number;
    hoverPath: number[];
    cullingPadding: number;
    skillTree: Tree;
    activeNodes: number[];
  }

  let { scaling, offsetX, offsetY, hoverPath, cullingPadding, skillTree, activeNodes }: Props = $props();

  interface PrecalculatedConnection {
    node: Node;
    targetNode: Node;
    draw(
      context: CanvasRenderingContext2D,
      canvasPos: Point,
      targetCanvasPos: Point,
      canvasOffsetX: number,
      canvasOffsetY: number,
      canvasScaling: number,
      canvasSkillTree: Tree
    ): void;
  }

  const connections: Array<PrecalculatedConnection> = [];

  onMount(() => {
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
          connections.push({
            node,
            targetNode,
            draw(context, canvasPos, targetCanvasPos) {
              context.moveTo(canvasPos.x, canvasPos.y);
              context.lineTo(targetCanvasPos.x, targetCanvasPos.y);
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

          connections.push({
            node,
            targetNode,
            draw(context, canvasPos, targetCanvasPos, canvasOffsetX, canvasOffsetY, canvasScaling, canvasSkillTree) {
              const groupPos = toCanvasCoords(posX, posY, canvasOffsetX, canvasOffsetY, canvasScaling);
              context.arc(groupPos.x, groupPos.y, canvasSkillTree.constants.orbitRadii[node.orbit!] / canvasScaling + 1, finalA, finalB);
            }
          });
        }
      });
    });
  });

  const render: Render = ({ context, width, height }) => {
    if (!connections) {
      return;
    }

    const hoverSet = new Set(hoverPath);
    const activeSet = new Set(activeNodes);

    connections.forEach((connection) => {
      const canvasPos = calculateNodePos(connection.node, offsetX, offsetY, scaling);
      const targetCanvasPos = calculateNodePos(connection.targetNode, offsetX, offsetY, scaling);

      if (
        (canvasPos.x < cullingPadding || canvasPos.x > width - cullingPadding || canvasPos.y < cullingPadding || canvasPos.y > height - cullingPadding) &&
        (targetCanvasPos.x < cullingPadding ||
          targetCanvasPos.x > width - cullingPadding ||
          targetCanvasPos.y < cullingPadding ||
          targetCanvasPos.y > height - cullingPadding)
      ) {
        return;
      }

      context.beginPath();

      connection.draw(context, canvasPos, targetCanvasPos, offsetX, offsetY, scaling, skillTree);

      let lineWidth = 6;
      if (activeSet.has(connection.node.skill!) && activeSet.has(connection.targetNode.skill!)) {
        context.strokeStyle = `#e9deb6`;
        lineWidth = 12;
      } else if (hoverSet.has(connection.node.skill!) && hoverSet.has(connection.targetNode.skill!)) {
        context.strokeStyle = `#c89c01`;
      } else {
        context.strokeStyle = `#524518`;
      }

      context.lineWidth = lineWidth / scaling;

      context.stroke();
    });
  };
</script>

<Layer {render} />
