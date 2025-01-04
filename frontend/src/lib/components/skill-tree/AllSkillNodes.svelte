<script lang="ts">
  import { T } from '@threlte/core';
  import type { Node, Sprite } from '../../skill_tree/types';
  import { drawnNodes, inverseSpritesInactive, inverseSpritesActive, inverseSpritesOther, type Point } from '../../skill_tree';
  import { PlaneGeometry } from 'three';
  import { loadSpriteTexture, relativeScale } from './common';
  import { useCursor } from '@threlte/extras';

  interface Props {
    node?: Node;
    cdnBase: string;
    hoverPath: number[];
    visibleNodePos: Map<number, Point>;
    activeNodes: number[];
    hoveredNode: Node | undefined;
    clickNode: (node: Node) => void;
  }

  let { cdnBase, hoverPath, hoveredNode = $bindable(), visibleNodePos, activeNodes, clickNode }: Props = $props();

  interface PrecalculatedNode {
    node: Node;
    nNodeId: number;
    draw(
      active: boolean,
      highlighted: boolean
    ): {
      path: string;
      source: Record<string, Sprite>;
    }[];
  }

  const precalculatedNodes: Map<number, PrecalculatedNode> = $derived.by(() => {
    const result = new Map<number, PrecalculatedNode>();

    drawnNodes.keys().forEach((nNodeId) => {
      const node: Node = drawnNodes.get(nNodeId)!;

      if (node.isAscendancyStart) {
        result.set(nNodeId, {
          node,
          nNodeId,
          draw() {
            return [
              {
                path: 'AscendancyMiddle',
                source: inverseSpritesOther
              }
            ];
          }
        });
      } else if (node.isKeystone) {
        result.set(nNodeId, {
          node,
          nNodeId,
          draw(active: boolean, highlighted: boolean) {
            return [
              {
                path: node.icon!,
                source: active ? inverseSpritesActive : inverseSpritesInactive
              },
              {
                path: active || highlighted ? 'KeystoneFrameAllocated' : 'KeystoneFrameUnallocated',
                source: inverseSpritesOther
              }
            ];
          }
        });
      } else if (node.isNotable) {
        if (node.ascendancyName) {
          result.set(nNodeId, {
            node,
            nNodeId,
            draw(active: boolean, highlighted: boolean) {
              return [
                {
                  path: node.icon!,
                  source: active ? inverseSpritesActive : inverseSpritesInactive
                },
                {
                  path: active || highlighted ? 'AscendancyFrameLargeAllocated' : 'AscendancyFrameLargeNormal',
                  source: inverseSpritesOther
                }
              ];
            }
          });
        } else {
          result.set(nNodeId, {
            node,
            nNodeId,
            draw(active: boolean, highlighted: boolean) {
              return [
                {
                  path: node.icon!,
                  source: active ? inverseSpritesActive : inverseSpritesInactive
                },
                {
                  path: active || highlighted ? 'NotableFrameAllocated' : 'NotableFrameUnallocated',
                  source: inverseSpritesOther
                }
              ];
            }
          });
        }
      } else if (node.isJewelSocket) {
        if (node.expansionJewel) {
          result.set(nNodeId, {
            node,
            nNodeId,
            draw(active: boolean, highlighted: boolean) {
              return [
                {
                  path: active || highlighted ? 'JewelSocketAltActive' : 'JewelSocketAltNormal',
                  source: inverseSpritesOther
                }
              ];
            }
          });
        } else {
          result.set(nNodeId, {
            node,
            nNodeId,
            draw(active: boolean, highlighted: boolean) {
              return [
                {
                  path: active || highlighted ? 'JewelFrameAllocated' : 'JewelFrameUnallocated',
                  source: inverseSpritesOther
                }
              ];
            }
          });
        }
      } else if (node.isMastery) {
        result.set(nNodeId, {
          node,
          nNodeId,
          draw(active: boolean, highlighted: boolean) {
            return [
              {
                path: active || highlighted ? node.activeIcon! : node.inactiveIcon!,
                source: active || highlighted ? inverseSpritesActive : inverseSpritesInactive
              }
            ];
          }
        });
      } else {
        if (node.ascendancyName) {
          result.set(nNodeId, {
            node,
            nNodeId,
            draw(active: boolean, highlighted: boolean) {
              return [
                {
                  path: node.icon!,
                  source: active ? inverseSpritesActive : inverseSpritesInactive
                },
                {
                  path: active || highlighted ? 'AscendancyFrameSmallAllocated' : 'AscendancyFrameSmallNormal',
                  source: inverseSpritesOther
                }
              ];
            }
          });
        } else {
          result.set(nNodeId, {
            node,
            nNodeId,
            draw(active: boolean, highlighted: boolean) {
              return [
                {
                  path: node.icon!,
                  source: active ? inverseSpritesActive : inverseSpritesInactive
                },
                {
                  path: active || highlighted ? 'PSSkillFrameActive' : 'PSSkillFrame',
                  source: inverseSpritesOther
                }
              ];
            }
          });
        }
      }
    });

    return result;
  });

  const { onPointerEnter, onPointerLeave } = useCursor();

  const groupGeometry = new PlaneGeometry(1, 1);

  const onEnter = (node: Node) => {
    onPointerEnter();
    hoveredNode = node;
  };

  const onLeave = (node: Node) => {
    if (hoveredNode?.name === node.name) {
      onPointerLeave();
      hoveredNode = undefined;
    }
  };

  let hoverSet: Set<number> = $derived(new Set(hoverPath));
  let activeSet: Set<number> = $derived(new Set(activeNodes));
</script>

{#each Array.from(visibleNodePos) as [nodeId, canvasPos]}
  {@const node = precalculatedNodes.get(nodeId)}
  {@const active = activeSet.has(nodeId)}
  {@const highlighted = hoverSet.has(nodeId) || hoveredNode === node?.node}

  {#each node?.draw(active, highlighted) || [] as layer}
    {#await loadSpriteTexture(layer.path, cdnBase, layer.source) then t}
      <T.Mesh
        onpointerenter={() => onEnter(node.node)}
        onpointerleave={() => onLeave(node.node)}
        onpointerup={() => clickNode(node.node)}
        geometry={groupGeometry}
        position={[canvasPos.x, -canvasPos.y, 0.003]}
        scale={relativeScale(t.sprite, [1, 1, 1])}>
        <T.MeshBasicMaterial map={t.texture} transparent />
      </T.Mesh>
    {/await}
  {/each}
{/each}
