<script lang="ts">
  import { Layer, type Render } from 'svelte-canvas';
  import type { Node } from '../../skill_tree/types';
  import { calculateNodePos, drawnNodes, inverseSpritesInactive, inverseSpritesActive, skillTree, inverseSpritesOther } from '../../skill_tree';
  import { drawSprite } from '$lib/components/skill-tree/common';

  interface Props {
    node?: Node;
    cdnBase: string;
    scaling: number;
    offsetX: number;
    offsetY: number;
    cullingPadding: number;
    hoverPath: number[];
    hoveredNode?: Node;
  }

  let { cdnBase, scaling, offsetX, offsetY, cullingPadding, hoverPath, hoveredNode }: Props = $props();

  const render: Render = ({ context, width, height }) => {
    if (!$skillTree) {
      return;
    }

    Object.keys(drawnNodes).forEach((nodeId) => {
      const nNodeId = parseInt(nodeId);

      const node: Node = drawnNodes[nNodeId];
      const canvasPos = calculateNodePos(node, offsetX, offsetY, scaling);

      if (canvasPos.x < cullingPadding || canvasPos.x > width - cullingPadding || canvasPos.y < cullingPadding || canvasPos.y > height - cullingPadding) {
        return;
      }

      const active = false; // TODO Actually check if node is active
      const highlighted = hoverPath.indexOf(node.skill!) >= 0 || hoveredNode === node;

      if (node.classStartIndex !== undefined) {
        // Do not draw class start index node
      } else if (node.isAscendancyStart) {
        drawSprite(context, 'AscendancyMiddle', canvasPos, inverseSpritesOther, scaling, cdnBase);
      } else if (node.isKeystone) {
        drawSprite(context, node.icon!, canvasPos, active ? inverseSpritesActive : inverseSpritesInactive, scaling, cdnBase);
        if (active || highlighted) {
          drawSprite(context, 'KeystoneFrameAllocated', canvasPos, inverseSpritesOther, scaling, cdnBase);
        } else {
          drawSprite(context, 'KeystoneFrameUnallocated', canvasPos, inverseSpritesOther, scaling, cdnBase);
        }
      } else if (node.isNotable) {
        drawSprite(context, node.icon!, canvasPos, active ? inverseSpritesActive : inverseSpritesInactive, scaling, cdnBase);

        if (node.ascendancyName) {
          if (active || highlighted) {
            drawSprite(context, 'AscendancyFrameLargeAllocated', canvasPos, inverseSpritesOther, scaling, cdnBase);
          } else {
            drawSprite(context, 'AscendancyFrameLargeNormal', canvasPos, inverseSpritesOther, scaling, cdnBase);
          }
        } else {
          if (active || highlighted) {
            drawSprite(context, 'NotableFrameAllocated', canvasPos, inverseSpritesOther, scaling, cdnBase);
          } else {
            drawSprite(context, 'NotableFrameUnallocated', canvasPos, inverseSpritesOther, scaling, cdnBase);
          }
        }
      } else if (node.isJewelSocket) {
        if (node.expansionJewel) {
          if (active || highlighted) {
            drawSprite(context, 'JewelSocketAltActive', canvasPos, inverseSpritesOther, scaling, cdnBase);
          } else {
            drawSprite(context, 'JewelSocketAltNormal', canvasPos, inverseSpritesOther, scaling, cdnBase);
          }
        } else {
          if (active || highlighted) {
            drawSprite(context, 'JewelFrameAllocated', canvasPos, inverseSpritesOther, scaling, cdnBase);
          } else {
            drawSprite(context, 'JewelFrameUnallocated', canvasPos, inverseSpritesOther, scaling, cdnBase);
          }
        }
      } else if (node.isMastery) {
        if (active || highlighted) {
          drawSprite(context, node.activeIcon!, canvasPos, inverseSpritesActive, scaling, cdnBase);
        } else {
          drawSprite(context, node.inactiveIcon!, canvasPos, inverseSpritesInactive, scaling, cdnBase);
        }
      } else {
        drawSprite(context, node.icon!, canvasPos, active ? inverseSpritesActive : inverseSpritesInactive, scaling, cdnBase);

        if (node.ascendancyName) {
          if (active || highlighted) {
            drawSprite(context, 'AscendancyFrameSmallAllocated', canvasPos, inverseSpritesOther, scaling, cdnBase);
          } else {
            drawSprite(context, 'AscendancyFrameSmallNormal', canvasPos, inverseSpritesOther, scaling, cdnBase);
          }
        } else {
          if (active || highlighted) {
            drawSprite(context, 'PSSkillFrameActive', canvasPos, inverseSpritesOther, scaling, cdnBase);
          } else {
            drawSprite(context, 'PSSkillFrame', canvasPos, inverseSpritesOther, scaling, cdnBase);
          }
        }
      }
    });
  };
</script>

<Layer {render} />
