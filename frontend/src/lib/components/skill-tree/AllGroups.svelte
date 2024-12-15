<script lang="ts">
  import { Layer, type Render } from 'svelte-canvas';
  import type { Group } from '../../skill_tree/types';
  import {
    skillTree,
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
  }

  let { scaling, offsetX, offsetY, cdnBase, currentClass, currentAscendancy, cullingPadding }: Props = $props();

  const render: Render = ({ context, width, height }) => {
    if (!$skillTree) {
      return;
    }

    drawnGroups.keys().forEach((nGroupId) => {
      const group: Group = drawnGroups.get(nGroupId)!;
      const posX = ((nGroupId in ascendancyGroups && ascendancyGroupPositionOffsets[ascendancyGroups[nGroupId]]?.x) || 0) + group.x;
      const posY = ((nGroupId in ascendancyGroups && ascendancyGroupPositionOffsets[ascendancyGroups[nGroupId]]?.y) || 0) + group.y;
      const canvasPos = toCanvasCoords(posX, posY, offsetX, offsetY, scaling);

      if (canvasPos.x < cullingPadding || canvasPos.x > width - cullingPadding || canvasPos.y < cullingPadding || canvasPos.y > height - cullingPadding) {
        return;
      }

      const maxOrbit = Math.max(...group.orbits);
      if (nGroupId in classStartGroups) {
        if (currentClass === $skillTree.classes[classStartGroups[nGroupId]].name) {
          drawSprite(context, 'center' + $skillTree.classes[classStartGroups[nGroupId]].name.toLowerCase(), canvasPos, inverseSpritesOther, scaling, cdnBase);
        } else {
          drawSprite(context, 'PSStartNodeBackgroundInactive', canvasPos, inverseSpritesOther, scaling, cdnBase, false, true);
        }
      } else if (nGroupId in ascendancyGroups) {
        if (ascendancyStartGroups.has(nGroupId)) {
          drawSprite(
            context,
            'Classes' + ascendancyGroups[nGroupId],
            canvasPos,
            inverseSpritesOther,
            scaling,
            cdnBase,
            false,
            true,
            currentAscendancy === ascendancyGroups[nGroupId]
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
