<script lang="ts">
  import { Canvas, Layer, type Render } from 'svelte-canvas';
  import type { Coord, Node, Sprite, Tree } from '../skill_tree/types';
  import {
    calculateNodePos,
    distance,
    drawnGroups,
    drawnNodes,
    inverseSpritesInactive,
    inverseSpritesActive,
    orbitAngleAt,
    toCanvasCoords,
    inverseSpritesOther,
    ascendancyGroups,
    ascendancyStartGroups,
    classStartGroups,
    ascendancyGroupPositionOffsets,
    classStartNodes
  } from '../skill_tree';
  import type { Point } from '../skill_tree';
  import { onMount } from 'svelte';
  import { currentBuild } from '../global';
  import { syncWrap } from '../go/worker';
  import { get, writable } from 'svelte/store';
  import { logError } from '$lib/utils';
  import { openOverlay } from '$lib/overlay';
  import NodeSelectionOptions from './overlays/NodeSelectionOptions.svelte';

  interface Props {
    skillTree: Tree;
    skillTreeVersion: string;
    children?: import('svelte').Snippet;
  }

  let { skillTree, skillTreeVersion, children }: Props = $props();

  let currentClass: string | undefined = $state();
  $effect(() => {
    $currentBuild?.Build.ClassName.then((newClass) => (currentClass = newClass)).catch(logError);
  });

  let currentAscendancy: string | undefined = $state();
  $effect(() => {
    $currentBuild?.Build.AscendClassName.then((newAscendancy) => (currentAscendancy = newAscendancy)).catch(logError);
  });

  let activeNodes: number[] | undefined;
  $effect(() => {
    $currentBuild?.Build?.PassiveNodes?.then((newNodes) => (activeNodes = newNodes)).catch(logError);
  });

  let allocatePathToTarget = (nodeId: number) => {
    const rootNodes = classStartNodes[skillTree.classes.findIndex((c) => c.name === currentClass)];
    void syncWrap?.CalculateTreePath(skillTreeVersion || '3_18', [...rootNodes, ...(activeNodes ?? [])], nodeId).then((pathData) => {
      if (!pathData) {
        return;
      }

      // The first in the path is always an already allocated node
      const isRootInPath = rootNodes.includes(pathData[0]);
      void syncWrap?.AllocateNodes(isRootInPath ? pathData : pathData.slice(1));
      currentBuild.set($currentBuild);
    });
  };

  let selectOption = (node: Node, optionIndex: number) => {
    // Todo: allocate mastery option to currentBuild
    console.debug(`Selected mastery option ${optionIndex} for mastery node ${node.name} - nodeId ${node.skill}`);
    allocatePathToTarget(node.skill ?? -1);
  };

  const openNodeOptions = (node: Node) => {
    openOverlay({
      component: NodeSelectionOptions,
      props: { node: node, onSelectOption: selectOption },
      backdropClose: true
    });
  };

  let clickNode = (node: Node) => {
    const nodeId = node.skill ?? -1;
    if (activeNodes?.includes(nodeId)) {
      void syncWrap?.DeallocateNodes(nodeId);
      currentBuild.set($currentBuild);
    } else {
      // TODO: Needs support for ascendancies or any other disconnect groups
      if (node.isMastery) {
        openNodeOptions(node);
        return;
      }
      allocatePathToTarget(nodeId);
    }
  };

  const titleFont = '25px Roboto Flex';
  const statsFont = '17px Roboto Flex';

  let scaling = $state(10);

  let offsetX = $state(0);
  let offsetY = $state(0);

  const drawScaling = 2.6;

  let cdnBase = $derived(`https://go-pob-data.pages.dev/data/${(skillTreeVersion || '3_18').replace('_', '.')}`);
  let cdnTreeBase = $derived(cdnBase + `/tree/assets/`);

  const spriteCache: Record<string, HTMLImageElement> = {};
  const cropCache: Record<string, HTMLCanvasElement> = {};
  const drawSprite = (
    context: CanvasRenderingContext2D,
    path: string | undefined,
    pos: Point,
    source: Record<string, Sprite>,
    mirror = false,
    cropCircle = false,
    active = false
  ) => {
    if (!path) {
      return;
    }

    const sprite = source[path];
    if (!sprite) {
      return;
    }

    const spriteSheetUrl = sprite.filename;
    if (!(spriteSheetUrl in spriteCache)) {
      const urlPath = new URL(spriteSheetUrl).pathname;
      const base = urlPath.substring(urlPath.lastIndexOf('/') + 1);
      const finalUrl = cdnTreeBase + base;

      spriteCache[spriteSheetUrl] = new Image();
      spriteCache[spriteSheetUrl].src = finalUrl;
    }

    const self: Coord = sprite.coords[path];

    const newWidth = (self.w / scaling) * drawScaling;
    const newHeight = (self.h / scaling) * drawScaling;

    const topLeftX = pos.x - newWidth / 2;
    const topLeftY = pos.y - newHeight / 2;

    let finalY = topLeftY;
    if (mirror) {
      finalY = topLeftY - newHeight / 2;
    }

    if (cropCircle && spriteCache[spriteSheetUrl].complete) {
      const cacheKey = spriteSheetUrl + ':' + path + '--' + (active ? 'active' : 'inactive');

      if (!(cacheKey in cropCache)) {
        const tempCanvas = document.createElement('canvas');
        const tempCtx = tempCanvas.getContext('2d')!;
        tempCanvas.width = self.w;
        tempCanvas.height = self.h;

        tempCtx.save();

        tempCtx.beginPath();
        tempCtx.arc(self.w / 2, self.h / 2, self.w / 2, 0, Math.PI * 2, true);
        tempCtx.closePath();
        tempCtx.clip();

        if (!active) {
          tempCtx.filter = 'brightness(50%) opacity(50%)';
        }

        tempCtx.drawImage(spriteCache[spriteSheetUrl], self.x, self.y, self.w, self.h, 0, 0, self.w, self.h);

        cropCache[cacheKey] = tempCanvas;
      }

      context.drawImage(cropCache[cacheKey], 0, 0, self.w, self.h, topLeftX, finalY, newWidth, newHeight);
    } else {
      context.drawImage(spriteCache[spriteSheetUrl], self.x, self.y, self.w, self.h, topLeftX, finalY, newWidth, newHeight);
    }

    if (mirror) {
      context.save();

      context.translate(topLeftX, topLeftY);
      context.rotate(Math.PI);

      context.drawImage(spriteCache[spriteSheetUrl], self.x, self.y, self.w, self.h, -newWidth, -(newHeight / 2), newWidth, -newHeight);

      context.restore();
    }
  };

  const wrapText = (text: string, context: CanvasRenderingContext2D, width: number): string[] => {
    const result = [];

    let currentWord = '';
    text.split(' ').forEach((word) => {
      if (context.measureText(currentWord + word).width < width) {
        currentWord += ' ' + word;
      } else {
        result.push(currentWord.trim());
        currentWord = word;
      }
    });

    if (currentWord.length > 0) {
      result.push(currentWord.trim());
    }

    return result;
  };

  let mousePos = $state<Point>({
    x: Number.MIN_VALUE,
    y: Number.MIN_VALUE
  });

  let cursor = $state('unset');

  const hoverPath = writable<number[]>([]);
  const extraCache = $state<Record<string, HTMLImageElement>>({});

  const hoveredNode = writable<Node | undefined>();
  const render: Render = ({ context, width, height }) => {
    const start = window.performance.now();

    if (!skillTree) {
      return;
    }

    context.clearRect(0, 0, width, height);

    context.fillStyle = '#080c11';
    context.fillRect(0, 0, width, height);

    if (currentClass) {
      const classIndex = skillTree.classes.findIndex((c) => c.name === currentClass);
      if (classIndex in skillTree.extraImages) {
        const img = skillTree.extraImages[classIndex];

        if (!(img.image in extraCache)) {
          extraCache[img.image] = new Image();
          extraCache[img.image].src = cdnBase + '/raw/' + img.image;
        }

        if (extraCache[img.image].complete) {
          const newWidth = (extraCache[img.image].width / scaling) * drawScaling * 0.5;
          const newHeight = (extraCache[img.image].height / scaling) * drawScaling * 0.5;

          const imgPos = toCanvasCoords(img.x, img.y, offsetX, offsetY, scaling);

          context.drawImage(extraCache[img.image], 0, 0, extraCache[img.image].width, extraCache[img.image].height, imgPos.x, imgPos.y, newWidth, newHeight);
        }
      }
    }

    const connected: Record<string, boolean> = {};
    for (const [groupId, group] of drawnGroups) {
      const posX = ((groupId in ascendancyGroups && ascendancyGroupPositionOffsets[ascendancyGroups[groupId]]?.x) || 0) + group.x;
      const posY = ((groupId in ascendancyGroups && ascendancyGroupPositionOffsets[ascendancyGroups[groupId]]?.y) || 0) + group.y;

      const groupPos = toCanvasCoords(posX, posY, offsetX, offsetY, scaling);

      const maxOrbit = Math.max(...group.orbits);
      if (groupId in classStartGroups) {
        if (currentClass === skillTree.classes[classStartGroups[groupId]].name) {
          drawSprite(context, 'center' + skillTree.classes[classStartGroups[groupId]].name.toLowerCase(), groupPos, inverseSpritesOther);
        } else {
          drawSprite(context, 'PSStartNodeBackgroundInactive', groupPos, inverseSpritesOther, false, true);
        }
      } else if (groupId in ascendancyGroups) {
        if (ascendancyStartGroups.has(groupId)) {
          drawSprite(
            context,
            'Classes' + ascendancyGroups[groupId],
            groupPos,
            inverseSpritesOther,
            false,
            true,
            currentAscendancy === ascendancyGroups[groupId]
          );
        }
      } else if (maxOrbit == 1) {
        drawSprite(context, 'PSGroupBackground1', groupPos, inverseSpritesOther);
      } else if (maxOrbit == 2) {
        drawSprite(context, 'PSGroupBackground2', groupPos, inverseSpritesOther);
      } else if (maxOrbit == 3 || group.orbits.length > 1) {
        drawSprite(context, 'PSGroupBackground3', groupPos, inverseSpritesOther, true);
      }
    }

    // Render connections between nodes
    for (const [nodeId, node] of drawnNodes) {
      // Do not draw connections out of class starting nodes
      if (node.classStartIndex !== undefined || !node.out || node.group === undefined || node.orbit === undefined) {
        continue;
      }

      for (const o of node.out) {
        const otherNodeId = parseInt(o);
        if (!drawnNodes.has(otherNodeId)) {
          continue;
        }

        const min = Math.min(otherNodeId, nodeId);
        const max = Math.max(otherNodeId, nodeId);
        const joined = min + ':' + max;

        if (joined in connected) {
          continue;
        }
        connected[joined] = true;

        const targetNode = drawnNodes.get(otherNodeId);
        // Do not draw connections to mastery nodes, ascendancy trees from main tree, or class starting nodes.
        if (!targetNode || targetNode.isMastery || node.ascendancyName !== targetNode.ascendancyName || targetNode.classStartIndex !== undefined) {
          continue;
        }

        context.beginPath();

        if (node.group != targetNode.group || node.orbit != targetNode.orbit) {
          const rotatedPos = calculateNodePos(node, offsetX, offsetY, scaling);
          const targetRotatedPos = calculateNodePos(targetNode, offsetX, offsetY, scaling);
          context.moveTo(rotatedPos.x, rotatedPos.y);
          context.lineTo(targetRotatedPos.x, targetRotatedPos.y);
        } else {
          let a = Math.PI / 180 - (Math.PI / 180) * orbitAngleAt(node.orbit, node.orbitIndex!);
          let b = Math.PI / 180 - (Math.PI / 180) * orbitAngleAt(targetNode.orbit, targetNode.orbitIndex!);

          a -= Math.PI / 2;
          b -= Math.PI / 2;

          const diff = Math.abs(Math.max(a, b) - Math.min(a, b));

          const finalA = diff > Math.PI ? Math.max(a, b) : Math.min(a, b);
          const finalB = diff > Math.PI ? Math.min(a, b) : Math.max(a, b);

          const group = drawnGroups.get(node.group);
          if (group === undefined) {
            continue;
          }

          const posX = ((node.ascendancyName && ascendancyGroupPositionOffsets[node.ascendancyName]?.x) || 0) + group.x;
          const posY = ((node.ascendancyName && ascendancyGroupPositionOffsets[node.ascendancyName]?.y) || 0) + group.y;
          const groupPos = toCanvasCoords(posX, posY, offsetX, offsetY, scaling);
          context.arc(groupPos.x, groupPos.y, skillTree.constants.orbitRadii[node.orbit] / scaling + 1, finalA, finalB);
        }

        let lineWidth = 6;
        if (activeNodes?.includes(nodeId) && activeNodes?.includes(otherNodeId)) {
          context.strokeStyle = `#e9deb6`;
          lineWidth = 12;
        } else if ($hoverPath.includes(nodeId) && $hoverPath.includes(otherNodeId)) {
          context.strokeStyle = `#c89c01`;
        } else {
          context.strokeStyle = `#524518`;
        }

        context.lineWidth = lineWidth / scaling;
        context.stroke();
      }
    }

    // let hoveredNodeActive = false;
    let newHoverNode: Node | undefined;
    for (const [nodeId, node] of drawnNodes) {
      const rotatedPos = calculateNodePos(node, offsetX, offsetY, scaling);
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

      if (distance(rotatedPos, mousePos) < touchDistance / scaling) {
        newHoverNode = node;
        // hoveredNodeActive = active;
      }

      const active = activeNodes?.includes(nodeId); // TODO Actually check if node is active
      const highlighted = $hoverPath.indexOf(node.skill!) >= 0 || newHoverNode === node;

      if (node.classStartIndex !== undefined) {
        // Do not draw class start index node
      } else if (node.isAscendancyStart) {
        drawSprite(context, 'AscendancyMiddle', rotatedPos, inverseSpritesOther);
      } else if (node.isKeystone) {
        drawSprite(context, node.icon, rotatedPos, active ? inverseSpritesActive : inverseSpritesInactive);
        if (active || highlighted) {
          drawSprite(context, 'KeystoneFrameAllocated', rotatedPos, inverseSpritesOther);
        } else {
          drawSprite(context, 'KeystoneFrameUnallocated', rotatedPos, inverseSpritesOther);
        }
      } else if (node.isNotable) {
        drawSprite(context, node.icon, rotatedPos, active ? inverseSpritesActive : inverseSpritesInactive);

        if (node.ascendancyName) {
          if (active || highlighted) {
            drawSprite(context, 'AscendancyFrameLargeAllocated', rotatedPos, inverseSpritesOther);
          } else {
            drawSprite(context, 'AscendancyFrameLargeNormal', rotatedPos, inverseSpritesOther);
          }
        } else {
          if (active || highlighted) {
            drawSprite(context, 'NotableFrameAllocated', rotatedPos, inverseSpritesOther);
          } else {
            drawSprite(context, 'NotableFrameUnallocated', rotatedPos, inverseSpritesOther);
          }
        }
      } else if (node.isJewelSocket) {
        if (node.expansionJewel) {
          if (active || highlighted) {
            drawSprite(context, 'JewelSocketAltActive', rotatedPos, inverseSpritesOther);
          } else {
            drawSprite(context, 'JewelSocketAltNormal', rotatedPos, inverseSpritesOther);
          }
        } else {
          if (active || highlighted) {
            drawSprite(context, 'JewelFrameAllocated', rotatedPos, inverseSpritesOther);
          } else {
            drawSprite(context, 'JewelFrameUnallocated', rotatedPos, inverseSpritesOther);
          }
        }
      } else if (node.isMastery) {
        if (active || highlighted) {
          drawSprite(context, node.activeIcon, rotatedPos, inverseSpritesActive);
        } else {
          drawSprite(context, node.inactiveIcon, rotatedPos, inverseSpritesInactive);
        }
      } else {
        drawSprite(context, node.icon, rotatedPos, active ? inverseSpritesActive : inverseSpritesInactive);

        if (node.ascendancyName) {
          if (active || highlighted) {
            drawSprite(context, 'AscendancyFrameSmallAllocated', rotatedPos, inverseSpritesOther);
          } else {
            drawSprite(context, 'AscendancyFrameSmallNormal', rotatedPos, inverseSpritesOther);
          }
        } else {
          if (active || highlighted) {
            drawSprite(context, 'PSSkillFrameActive', rotatedPos, inverseSpritesOther);
          } else {
            drawSprite(context, 'PSSkillFrame', rotatedPos, inverseSpritesOther);
          }
        }
      }
    }

    if (get(hoveredNode) != newHoverNode) {
      hoveredNode.set(newHoverNode);
      if (newHoverNode !== undefined && currentClass) {
        const rootNodes = classStartNodes[skillTree.classes.findIndex((c) => c.name === currentClass)];
        const target = newHoverNode.skill!;
        syncWrap
          .CalculateTreePath(skillTreeVersion || '3_18', [...rootNodes, ...(activeNodes ?? [])], target)
          .then((data) => {
            if (data) {
              hoverPath.set(data);
            }
          })
          .catch(logError);
      } else {
        hoverPath.set([]);
      }
    }

    const hNode = get(hoveredNode);
    if (hNode) {
      const nodeName = hNode.name || 'N/A';
      const nodeStats: { text: string; special: boolean }[] = (hNode.stats || []).map((s) => ({
        text: s,
        special: false
      }));

      context.font = titleFont;
      const textMetrics = context.measureText(nodeName ?? '');

      const maxWidth = Math.max(textMetrics.width + 50, 600);

      context.font = statsFont;

      const allLines: {
        text: string;
        offset: number;
        special: boolean;
      }[] = [];

      const padding = 30;

      let offset = 85;

      if (nodeStats && nodeStats.length > 0) {
        nodeStats.forEach((stat) => {
          if (allLines.length > 0) {
            offset += 5;
          }

          stat.text.split('\n').forEach((line) => {
            if (allLines.length > 0) {
              offset += 10;
            }

            const lines = wrapText(line, context, maxWidth - padding);
            lines.forEach((l) => {
              allLines.push({
                text: l,
                offset,
                special: stat.special
              });
              offset += 20;
            });
          });
        });
      } else if (hNode.isJewelSocket) {
        allLines.push({
          text: 'Click to select this socket',
          offset,
          special: true
        });

        offset += 20;
      } else if (hNode.isMastery) {
        allLines.push({
          text: 'Available mastery options are:',
          offset,
          special: false
        });
        offset += 20;
        hNode.masteryEffects?.forEach((effect) => {
          if (allLines.length > 0) {
            offset += 10;
          }
          effect.stats.forEach((stat) => {
            stat.split('\n').forEach(() => {
              if (allLines.length > 0) {
                offset += 10;
              }
              allLines.push({
                text: stat ?? 'N/A',
                offset,
                special: true
              });
              offset += 20;
            });
          });
        });
      }

      const titleHeight = 55;

      context.fillStyle = 'rgba(75,63,24,0.9)';
      context.fillRect(mousePos.x, mousePos.y, maxWidth, titleHeight);

      context.fillStyle = '#ffffff';
      context.font = titleFont;
      context.textAlign = 'center';
      context.fillText(nodeName ?? '', mousePos.x + maxWidth / 2, mousePos.y + 35);

      context.fillStyle = 'rgba(0,0,0,0.8)';
      context.fillRect(mousePos.x, mousePos.y + titleHeight, maxWidth, offset - titleHeight);

      context.font = statsFont;
      context.textAlign = 'left';
      allLines.forEach((l) => {
        if (l.special) {
          context.fillStyle = '#8cf34c';
        } else {
          context.fillStyle = '#ffffff';
        }

        context.fillText(l.text, mousePos.x + padding / 2, mousePos.y + l.offset);
      });
    }

    if (hNode) {
      cursor = 'pointer';
    } else {
      cursor = 'unset';
    }

    context.fillStyle = '#ffffff';
    context.textAlign = 'right';
    context.font = '12px Roboto Mono';

    const end = window.performance.now();

    context.fillText(`${(end - start).toFixed(1)}ms`, width - 5, 17);
  };

  let downX = 0;
  let downY = 0;

  let startX = 0;
  let startY = 0;

  let down = false;
  const mouseDown = (event: MouseEvent) => {
    down = true;
    downX = event.offsetX;
    downY = event.offsetY;
    startX = offsetX;
    startY = offsetY;

    mousePos = {
      x: event.offsetX,
      y: event.offsetY
    };

    const hNode = get(hoveredNode);
    if (hNode) {
      clickNode(hNode);
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

  const onScroll = (event: WheelEvent) => {
    if (event.deltaY > 0) {
      if (scaling < 30) {
        offsetX += event.offsetX;
        offsetY += event.offsetY;
      }
    } else {
      if (scaling > 3) {
        offsetX -= event.offsetX;
        offsetY -= event.offsetY;
      }
    }

    scaling = Math.min(30, Math.max(3, scaling + event.deltaY / 100));

    event.preventDefault();
    event.stopPropagation();
    event.stopImmediatePropagation();
  };

  let parentContainer = $state<HTMLElement>();

  let width = $state(0);
  let height = $state(0);
  const resize = () => {
    if (parentContainer) {
      width = parentContainer.offsetWidth;
      height = parentContainer.offsetHeight;
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
  {#if width && height}
    <div style="touch-action: none; cursor: {cursor}">
      <Canvas {width} {height} onpointerdown={mouseDown} onwheel={onScroll}>
        <Layer {render} />
      </Canvas>
      {@render children?.()}
    </div>
  {/if}
</div>
