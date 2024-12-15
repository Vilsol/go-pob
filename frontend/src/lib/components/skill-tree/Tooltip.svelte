<script lang="ts">
  import { Layer, type Render } from 'svelte-canvas';
  import type { Node } from '../../skill_tree/types';
  import { type Point } from '../../skill_tree';
  import { wrapText } from '$lib/components/skill-tree/common';

  interface Props {
    hoveredNode?: Node;
    mousePos: Point;
  }

  let { hoveredNode, mousePos }: Props = $props();

  const titleFont = '25px Roboto Flex';
  const statsFont = '17px Roboto Flex';

  const render: Render = ({ context }) => {
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
  };
</script>

<Layer {render} />
