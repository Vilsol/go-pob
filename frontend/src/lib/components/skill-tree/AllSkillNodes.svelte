<script lang="ts">
  import { Layer, type Render } from 'svelte-canvas';
  import type { Node } from '../../skill_tree/types';
  import { drawnNodes, inverseSpritesInactive, inverseSpritesActive, inverseSpritesOther, type Point } from '../../skill_tree';
  import { drawSprite } from '$lib/components/skill-tree/common';
  import { onMount } from 'svelte';

  interface Props {
    node?: Node;
    cdnBase: string;
    scaling: number;
    offsetX: number;
    offsetY: number;
    cullingPadding: number;
    hoverPath: number[];
    hoveredNode?: Node;
    visibleNodePos: Map<number, Point>;
    activeNodes: number[];
  }

  let { cdnBase, scaling, hoverPath, hoveredNode, visibleNodePos, activeNodes }: Props = $props();

  interface PrecalculatedNode {
    node: Node;
    nNodeId: number;
    draw(context: CanvasRenderingContext2D, canvasPos: Point, canvasScaling: number, active: boolean, highlighted: boolean): void;
  }

  const precalculatedNodes: Map<number, PrecalculatedNode> = new Map<number, PrecalculatedNode>();

  onMount(() => {
    drawnNodes.keys().forEach((nNodeId) => {
      const node: Node = drawnNodes.get(nNodeId)!;

      if (node.isAscendancyStart) {
        precalculatedNodes.set(nNodeId, {
          node,
          nNodeId,
          draw(context: CanvasRenderingContext2D, canvasPos: Point, canvasScaling: number) {
            drawSprite(context, 'AscendancyMiddle', canvasPos, inverseSpritesOther, canvasScaling, cdnBase);
          }
        });
      } else if (node.isKeystone) {
        precalculatedNodes.set(nNodeId, {
          node,
          nNodeId,
          draw(context: CanvasRenderingContext2D, canvasPos: Point, canvasScaling: number, active: boolean, highlighted: boolean) {
            drawSprite(context, node.icon!, canvasPos, active ? inverseSpritesActive : inverseSpritesInactive, canvasScaling, cdnBase);
            if (active || highlighted) {
              drawSprite(context, 'KeystoneFrameAllocated', canvasPos, inverseSpritesOther, canvasScaling, cdnBase);
            } else {
              drawSprite(context, 'KeystoneFrameUnallocated', canvasPos, inverseSpritesOther, canvasScaling, cdnBase);
            }
          }
        });
      } else if (node.isNotable) {
        if (node.ascendancyName) {
          precalculatedNodes.set(nNodeId, {
            node,
            nNodeId,
            draw(context: CanvasRenderingContext2D, canvasPos: Point, canvasScaling: number, active: boolean, highlighted: boolean) {
              drawSprite(context, node.icon!, canvasPos, active ? inverseSpritesActive : inverseSpritesInactive, canvasScaling, cdnBase);
              if (active || highlighted) {
                drawSprite(context, 'AscendancyFrameLargeAllocated', canvasPos, inverseSpritesOther, canvasScaling, cdnBase);
              } else {
                drawSprite(context, 'AscendancyFrameLargeNormal', canvasPos, inverseSpritesOther, canvasScaling, cdnBase);
              }
            }
          });
        } else {
          precalculatedNodes.set(nNodeId, {
            node,
            nNodeId,
            draw(context: CanvasRenderingContext2D, canvasPos: Point, canvasScaling: number, active: boolean, highlighted: boolean) {
              drawSprite(context, node.icon!, canvasPos, active ? inverseSpritesActive : inverseSpritesInactive, canvasScaling, cdnBase);
              if (active || highlighted) {
                drawSprite(context, 'NotableFrameAllocated', canvasPos, inverseSpritesOther, canvasScaling, cdnBase);
              } else {
                drawSprite(context, 'NotableFrameUnallocated', canvasPos, inverseSpritesOther, canvasScaling, cdnBase);
              }
            }
          });
        }
      } else if (node.isJewelSocket) {
        if (node.expansionJewel) {
          precalculatedNodes.set(nNodeId, {
            node,
            nNodeId,
            draw(context: CanvasRenderingContext2D, canvasPos: Point, canvasScaling: number, active: boolean, highlighted: boolean) {
              if (active || highlighted) {
                drawSprite(context, 'JewelSocketAltActive', canvasPos, inverseSpritesOther, canvasScaling, cdnBase);
              } else {
                drawSprite(context, 'JewelSocketAltNormal', canvasPos, inverseSpritesOther, canvasScaling, cdnBase);
              }
            }
          });
        } else {
          precalculatedNodes.set(nNodeId, {
            node,
            nNodeId,
            draw(context: CanvasRenderingContext2D, canvasPos: Point, canvasScaling: number, active: boolean, highlighted: boolean) {
              if (active || highlighted) {
                drawSprite(context, 'JewelFrameAllocated', canvasPos, inverseSpritesOther, canvasScaling, cdnBase);
              } else {
                drawSprite(context, 'JewelFrameUnallocated', canvasPos, inverseSpritesOther, canvasScaling, cdnBase);
              }
            }
          });
        }
      } else if (node.isMastery) {
        precalculatedNodes.set(nNodeId, {
          node,
          nNodeId,
          draw(context: CanvasRenderingContext2D, canvasPos: Point, canvasScaling: number, active: boolean, highlighted: boolean) {
            if (active || highlighted) {
              drawSprite(context, node.activeIcon!, canvasPos, inverseSpritesActive, canvasScaling, cdnBase);
            } else {
              drawSprite(context, node.inactiveIcon!, canvasPos, inverseSpritesInactive, canvasScaling, cdnBase);
            }
          }
        });
      } else {
        if (node.ascendancyName) {
          precalculatedNodes.set(nNodeId, {
            node,
            nNodeId,
            draw(context: CanvasRenderingContext2D, canvasPos: Point, canvasScaling: number, active: boolean, highlighted: boolean) {
              drawSprite(context, node.icon!, canvasPos, active ? inverseSpritesActive : inverseSpritesInactive, canvasScaling, cdnBase);
              if (active || highlighted) {
                drawSprite(context, 'AscendancyFrameSmallAllocated', canvasPos, inverseSpritesOther, canvasScaling, cdnBase);
              } else {
                drawSprite(context, 'AscendancyFrameSmallNormal', canvasPos, inverseSpritesOther, canvasScaling, cdnBase);
              }
            }
          });
        } else {
          precalculatedNodes.set(nNodeId, {
            node,
            nNodeId,
            draw(context: CanvasRenderingContext2D, canvasPos: Point, canvasScaling: number, active: boolean, highlighted: boolean) {
              drawSprite(context, node.icon!, canvasPos, active ? inverseSpritesActive : inverseSpritesInactive, canvasScaling, cdnBase);
              if (active || highlighted) {
                drawSprite(context, 'PSSkillFrameActive', canvasPos, inverseSpritesOther, canvasScaling, cdnBase);
              } else {
                drawSprite(context, 'PSSkillFrame', canvasPos, inverseSpritesOther, canvasScaling, cdnBase);
              }
            }
          });
        }
      }
    });
  });

  const render: Render = ({ context }) => {
    const hoverSet = new Set(hoverPath);
    const activeSet = new Set(activeNodes);

    visibleNodePos.forEach((canvasPos, nodeId) => {
      const node = precalculatedNodes.get(nodeId)!;

      const active = activeSet.has(nodeId); // TODO Actually check if node is active
      const highlighted = hoverSet.has(nodeId) || hoveredNode === node;

      node.draw(context, canvasPos, scaling, active, highlighted);
    });
  };
</script>

<Layer {render} />
