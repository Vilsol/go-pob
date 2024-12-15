<script lang="ts">
  import { Layer, type Render } from 'svelte-canvas';
  import { toCanvasCoords } from '../../skill_tree';
  import type { Tree } from '$lib/skill_tree/types';

  interface Props {
    scaling: number;
    offsetX: number;
    offsetY: number;
    cdnBase: string;
    cullingPadding: number;
    currentClass?: string;
    drawScaling: number;
    skillTree: Tree;
  }

  let { scaling, offsetX, offsetY, cdnBase, currentClass, drawScaling, cullingPadding, skillTree }: Props = $props();

  const extraCache = $state<Record<string, HTMLImageElement>>({});

  const render: Render = ({ context, width, height }) => {
    if (currentClass) {
      const classIndex = skillTree.classes.findIndex((c) => c.name === currentClass);
      if (classIndex in skillTree.extraImages) {
        const img = skillTree.extraImages[classIndex];

        if (!(img.image in extraCache)) {
          extraCache[img.image] = new Image();
          extraCache[img.image].src = cdnBase + '/raw/' + img.image;
        }

        if (extraCache[img.image].complete) {
          const canvasPos = toCanvasCoords(img.x, img.y, offsetX, offsetY, scaling);

          const newWidth = (extraCache[img.image].width / scaling) * drawScaling * 0.5;
          const newHeight = (extraCache[img.image].height / scaling) * drawScaling * 0.5;

          const rightX = canvasPos.x + newWidth;
          const bottomY = canvasPos.y + newHeight;

          const topLeft =
            canvasPos.x < cullingPadding || canvasPos.x > width - cullingPadding || canvasPos.y < cullingPadding || canvasPos.y > height - cullingPadding;
          const topRight = rightX < cullingPadding || rightX > width - cullingPadding || canvasPos.y < cullingPadding || canvasPos.y > height - cullingPadding;
          const bottomLeft =
            canvasPos.x < cullingPadding || canvasPos.x > width - cullingPadding || bottomY < cullingPadding || bottomY > height - cullingPadding;
          const bottomRight = rightX < cullingPadding || rightX > width - cullingPadding || bottomY < cullingPadding || bottomY > height - cullingPadding;
          if (topLeft && topRight && bottomLeft && bottomRight) {
            return;
          }

          context.drawImage(
            extraCache[img.image],
            0,
            0,
            extraCache[img.image].width,
            extraCache[img.image].height,
            canvasPos.x,
            canvasPos.y,
            newWidth,
            newHeight
          );
        }
      }
    }
  };
</script>

<Layer {render} />
