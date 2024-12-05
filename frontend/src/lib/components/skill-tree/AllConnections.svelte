<script lang="ts">
  import { Layer, type Render } from 'svelte-canvas';
  import type { Node } from '../../skill_tree/types';
  import { calculateNodePos, drawnNodes, skillTree, toCanvasCoords, orbitAngleAt, drawnGroups, ascendancyGroupPositionOffsets } from '../../skill_tree';

  interface Props {
    scaling: number;
    offsetX: number;
    offsetY: number;
    hoverPath: number[];
    cullingPadding: number;
  }

  let { scaling, offsetX, offsetY, hoverPath, cullingPadding }: Props = $props();

  const render: Render = ({ context, width, height }) => {
    if (!$skillTree) {
      return;
    }

    const connected: Record<string, boolean> = {};
    Object.keys(drawnNodes).forEach((nodeId) => {
      const nNodeId = parseInt(nodeId);

      const node: Node = drawnNodes[nNodeId];

      // Do not draw connections out of class starting nodes
      if (node.classStartIndex !== undefined) {
        return;
      }

      const angle = orbitAngleAt(node.orbit!, node.orbitIndex!);
      const canvasPos = calculateNodePos(node, offsetX, offsetY, scaling);

      const sourceActive = hoverPath.indexOf(node.skill!) >= 0;

      node.out?.forEach((o) => {
        if (!drawnNodes[parseInt(o)]) {
          return;
        }

        const min = Math.min(parseInt(o), parseInt(nodeId));
        const max = Math.max(parseInt(o), parseInt(nodeId));
        const joined = min + ':' + max;

        if (joined in connected) {
          return;
        }
        connected[joined] = true;

        const targetNode = drawnNodes[parseInt(o)];

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
        const targetCanvasPos = calculateNodePos(targetNode, offsetX, offsetY, scaling);

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

        if (node.group != targetNode.group || node.orbit != targetNode.orbit) {
          context.moveTo(canvasPos.x, canvasPos.y);
          context.lineTo(targetCanvasPos.x, targetCanvasPos.y);
        } else {
          let a = Math.PI / 180 - (Math.PI / 180) * angle;
          let b = Math.PI / 180 - (Math.PI / 180) * targetAngle;

          a -= Math.PI / 2;
          b -= Math.PI / 2;

          const diff = Math.abs(Math.max(a, b) - Math.min(a, b));

          const finalA = diff > Math.PI ? Math.max(a, b) : Math.min(a, b);
          const finalB = diff > Math.PI ? Math.min(a, b) : Math.max(a, b);

          const group = drawnGroups[node.group!];
          const posX = ((node.ascendancyName && ascendancyGroupPositionOffsets[node.ascendancyName]?.x) || 0) + group.x;
          const posY = ((node.ascendancyName && ascendancyGroupPositionOffsets[node.ascendancyName]?.y) || 0) + group.y;
          const groupPos = toCanvasCoords(posX, posY, offsetX, offsetY, scaling);
          context.arc(groupPos.x, groupPos.y, $skillTree.constants.orbitRadii[node.orbit!] / scaling + 1, finalA, finalB);
        }

        if (sourceActive && hoverPath.indexOf(targetNode.skill!) >= 0) {
          context.strokeStyle = `#c89c01`;
        } else {
          context.strokeStyle = `#524518`;
        }

        context.lineWidth = 6 / scaling;
        context.stroke();
      });
    });
  };
</script>

<Layer {render} />
