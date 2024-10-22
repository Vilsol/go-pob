<script lang="ts">
  import { Canvas, Layer } from 'svelte-canvas';
  import type { Coord, Group, Node, Sprite } from '../skill_tree/types';
  import {
    calculateNodePos,
    distance,
    drawnGroups,
    drawnNodes,
    inverseSpritesInactive,
    inverseSpritesActive,
    orbitAngleAt,
    skillTree,
    toCanvasCoords,
    inverseSpritesOther,
    skillTreeVersion,
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
  import { writable } from 'svelte/store';
  import { logError } from '$lib/utils';

  let currentClass: string | undefined = $state();
  $effect(() => {
    $currentBuild?.Build.ClassName.then((newClass) => (currentClass = newClass)).catch(logError);
  });

  let currentAscendancy: string | undefined = $state();
  $effect(() => {
    $currentBuild?.Build.AscendClassName.then((newAscendancy) => (currentAscendancy = newAscendancy)).catch(logError);
  });

  interface RenderParams {
    context: CanvasRenderingContext2D;
    width: number;
    height: number;
  }

  type RenderFunc = (params: RenderParams) => void;

  interface Props {
    clickNode?: (node: Node) => void;
    children?: import('svelte').Snippet;
  }

  let { clickNode = console.log, children }: Props = $props();

  const titleFont = '25px Roboto Flex';
  const statsFont = '17px Roboto Flex';

  let scaling = $state(10);

  let offsetX = $state(0);
  let offsetY = $state(0);

  const drawScaling = 2.6;

  let cdnBase = $derived(`https://go-pob-data.pages.dev/data/${($skillTreeVersion || '3_18').replace('_', '.')}`);
  let cdnTreeBase = $derived(cdnBase + `/tree/assets/`);

  const spriteCache: Record<string, HTMLImageElement> = {};
  const cropCache: Record<string, HTMLCanvasElement> = {};
  const drawSprite = (
    context: CanvasRenderingContext2D,
    path: string,
    pos: Point,
    source: Record<string, Sprite>,
    mirror = false,
    cropCircle = false,
    active = false
  ) => {
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
      const cacheKey = spriteSheetUrl + ':' + path + ';' + (active ? 'active' : '');
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

  let hoveredNode: Node | undefined = $state();
  let render = $derived((({ context, width, height }) => {
    const start = window.performance.now();

    if (!$skillTree) {
      return;
    }

    context.clearRect(0, 0, width, height);

    context.fillStyle = '#080c11';
    context.fillRect(0, 0, width, height);

    if (currentClass) {
      const classIndex = $skillTree.classes.findIndex((c) => c.name === currentClass);
      if (classIndex in $skillTree.extraImages) {
        const img = $skillTree.extraImages[classIndex];

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
    Object.keys(drawnGroups).forEach((groupId) => {
      const nGroupId = parseInt(groupId);

      const group: Group = drawnGroups[nGroupId];
      const posX = ((nGroupId in ascendancyGroups && ascendancyGroupPositionOffsets[ascendancyGroups[nGroupId]]?.x) || 0) + group.x;
      const posY = ((nGroupId in ascendancyGroups && ascendancyGroupPositionOffsets[ascendancyGroups[nGroupId]]?.y) || 0) + group.y;
      const groupPos = toCanvasCoords(posX, posY, offsetX, offsetY, scaling);

      const maxOrbit = Math.max(...group.orbits);
      if (nGroupId in classStartGroups) {
        if (currentClass === $skillTree.classes[classStartGroups[nGroupId]].name) {
          drawSprite(context, 'center' + $skillTree.classes[classStartGroups[nGroupId]].name.toLowerCase(), groupPos, inverseSpritesOther);
        } else {
          drawSprite(context, 'PSStartNodeBackgroundInactive', groupPos, inverseSpritesOther, false, true);
        }
      } else if (nGroupId in ascendancyGroups) {
        if (ascendancyStartGroups.has(nGroupId)) {
          drawSprite(
            context,
            'Classes' + ascendancyGroups[nGroupId],
            groupPos,
            inverseSpritesOther,
            false,
            true,
            currentAscendancy === ascendancyGroups[nGroupId]
          );
        }
      } else if (maxOrbit == 1) {
        drawSprite(context, 'PSGroupBackground1', groupPos, inverseSpritesOther);
      } else if (maxOrbit == 2) {
        drawSprite(context, 'PSGroupBackground2', groupPos, inverseSpritesOther);
      } else if (maxOrbit == 3 || group.orbits.length > 1) {
        drawSprite(context, 'PSGroupBackground3', groupPos, inverseSpritesOther, true);
      }
    });

    Object.keys(drawnNodes).forEach((nodeId) => {
      const nNodeId = parseInt(nodeId);

      const node: Node = drawnNodes[nNodeId];
      const angle = orbitAngleAt(node.orbit!, node.orbitIndex!);
      const rotatedPos = calculateNodePos(node, offsetX, offsetY, scaling);

      // Do not draw connections out of class starting nodes
      if (node.classStartIndex !== undefined) {
        return;
      }

      const sourceActive = $hoverPath.indexOf(node.skill!) >= 0;

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
        const targetRotatedPos = calculateNodePos(targetNode, offsetX, offsetY, scaling);

        context.beginPath();

        if (node.group != targetNode.group || node.orbit != targetNode.orbit) {
          context.moveTo(rotatedPos.x, rotatedPos.y);
          context.lineTo(targetRotatedPos.x, targetRotatedPos.y);
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

        if (sourceActive && $hoverPath.indexOf(targetNode.skill!) >= 0) {
          context.strokeStyle = `#c89c01`;
        } else {
          context.strokeStyle = `#524518`;
        }

        context.lineWidth = 6 / scaling;
        context.stroke();
      });
    });

    // let hoveredNodeActive = false;
    let newHoverNode: Node | undefined;
    Object.keys(drawnNodes).forEach((nodeId) => {
      const nNodeId = parseInt(nodeId);

      const node: Node = drawnNodes[nNodeId];
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

      const active = false; // TODO Actually check if node is active
      const highlighted = $hoverPath.indexOf(node.skill!) >= 0 || newHoverNode === node;

      if (node.classStartIndex !== undefined) {
        // Do not draw class start index node
      } else if (node.isAscendancyStart) {
        drawSprite(context, 'AscendancyMiddle', rotatedPos, inverseSpritesOther);
      } else if (node.isKeystone) {
        drawSprite(context, node.icon!, rotatedPos, active ? inverseSpritesActive : inverseSpritesInactive);
        if (active || highlighted) {
          drawSprite(context, 'KeystoneFrameAllocated', rotatedPos, inverseSpritesOther);
        } else {
          drawSprite(context, 'KeystoneFrameUnallocated', rotatedPos, inverseSpritesOther);
        }
      } else if (node.isNotable) {
        drawSprite(context, node.icon!, rotatedPos, active ? inverseSpritesActive : inverseSpritesInactive);

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
          drawSprite(context, node.activeIcon!, rotatedPos, inverseSpritesActive);
        } else {
          drawSprite(context, node.inactiveIcon!, rotatedPos, inverseSpritesInactive);
        }
      } else {
        drawSprite(context, node.icon!, rotatedPos, active ? inverseSpritesActive : inverseSpritesInactive);

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
    });

    if (hoveredNode != newHoverNode) {
      hoveredNode = newHoverNode;
      if (hoveredNode !== undefined && currentClass) {
        const rootNodes = classStartNodes[$skillTree.classes.findIndex((c) => c.name === currentClass)];
        const target = hoveredNode.skill!;
        syncWrap
          .CalculateTreePath($skillTreeVersion || '3_18', rootNodes, target)
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

    if (hoveredNode) {
      const nodeName = hoveredNode.name || 'N/A';
      const nodeStats: { text: string; special: boolean }[] = (hoveredNode.stats || []).map((s) => ({
        text: s,
        special: false
      }));

      context.font = titleFont;
      const textMetrics = context.measureText(nodeName);

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
      } else if (hoveredNode.isJewelSocket) {
        allLines.push({
          text: 'Click to select this socket',
          offset,
          special: true
        });

        offset += 20;
      }

      const titleHeight = 55;

      context.fillStyle = 'rgba(75,63,24,0.9)';
      context.fillRect(mousePos.x, mousePos.y, maxWidth, titleHeight);

      context.fillStyle = '#ffffff';
      context.font = titleFont;
      context.textAlign = 'center';
      context.fillText(nodeName, mousePos.x + maxWidth / 2, mousePos.y + 35);

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

    if (hoveredNode) {
      cursor = 'pointer';
    } else {
      cursor = 'unset';
    }

    context.fillStyle = '#ffffff';
    context.textAlign = 'right';
    context.font = '12px Roboto Mono';

    const end = window.performance.now();

    context.fillText(`${(end - start).toFixed(1)}ms`, width - 5, 17);
  }) as RenderFunc);

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

    if (hoveredNode) {
      clickNode(hoveredNode);
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
    if (!initialized && $skillTree) {
      initialized = true;
      offsetX = $skillTree.min_x + (window.innerWidth / 2) * scaling;
      offsetY = $skillTree.min_y + (window.innerHeight / 2) * scaling;
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
      <Canvas {width} {height} on:pointerdown={mouseDown} on:wheel={onScroll}>
        <Layer {render} />
      </Canvas>
      {@render children?.()}
    </div>
  {/if}
</div>
