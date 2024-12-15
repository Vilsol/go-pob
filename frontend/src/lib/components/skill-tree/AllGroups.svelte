<script lang="ts">
  import { Layer, type Render } from 'svelte-canvas';
  import type { Tree } from '../../skill_tree/types';
  import {
    toCanvasCoords,
    drawnGroups,
    ascendancyGroupPositionOffsets,
    ascendancyGroups,
    classStartGroups,
    inverseSpritesOther,
    ascendancyStartGroups
  } from '../../skill_tree';
  import { drawSprite } from '$lib/components/skill-tree/common';

  interface Props {
    scaling: number;
    offsetX: number;
    offsetY: number;
    cdnBase: string;
    cullingPadding: number;
    currentClass?: string;
    currentAscendancy?: string;
    skillTree: Tree;
  }

  let { scaling, offsetX, offsetY, cdnBase, currentClass, currentAscendancy, cullingPadding, skillTree }: Props = $props();

  const render: Render = ({ context, width, height }) => {
    drawnGroups.forEach((group, groupId) => {
      const posX = ((groupId in ascendancyGroups && ascendancyGroupPositionOffsets[ascendancyGroups[groupId]]?.x) || 0) + group.x;
      const posY = ((groupId in ascendancyGroups && ascendancyGroupPositionOffsets[ascendancyGroups[groupId]]?.y) || 0) + group.y;
      const canvasPos = toCanvasCoords(posX, posY, offsetX, offsetY, scaling);

      if (canvasPos.x < cullingPadding || canvasPos.x > width - cullingPadding || canvasPos.y < cullingPadding || canvasPos.y > height - cullingPadding) {
        return;
      }

      const maxOrbit = Math.max(...group.orbits);
      if (groupId in classStartGroups) {
        if (currentClass === skillTree.classes[classStartGroups[groupId]].name) {
          drawSprite(context, 'center' + skillTree.classes[classStartGroups[groupId]].name.toLowerCase(), canvasPos, inverseSpritesOther, scaling, cdnBase);
        } else {
          drawSprite(context, 'PSStartNodeBackgroundInactive', canvasPos, inverseSpritesOther, scaling, cdnBase, false, true);
        }
      } else if (groupId in ascendancyGroups) {
        if (ascendancyStartGroups.has(groupId)) {
          drawSprite(
            context,
            'Classes' + ascendancyGroups[groupId],
            canvasPos,
            inverseSpritesOther,
            scaling,
            cdnBase,
            false,
            true,
            currentAscendancy === ascendancyGroups[groupId]
          );
        }
      } else if (maxOrbit == 1) {
        drawSprite(context, 'PSGroupBackground1', canvasPos, inverseSpritesOther, scaling, cdnBase);
      } else if (maxOrbit == 2) {
        drawSprite(context, 'PSGroupBackground2', canvasPos, inverseSpritesOther, scaling, cdnBase);
      } else if (maxOrbit == 3 || group.orbits.length > 1) {
        drawSprite(context, 'PSGroupBackground3', canvasPos, inverseSpritesOther, scaling, cdnBase, true);
      }
    });
  };
</script>

<Layer {render} />
