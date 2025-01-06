<script lang="ts">
  import type { Node } from '../../skill_tree/types';
  import { type Point } from '../../skill_tree';
  import { devMode } from '$lib/global';
  import { onMount } from 'svelte';
  import { TextStyle, CanvasTextMetrics, Graphics, Container, Text, type Application } from 'pixi.js';

  interface Props {
    hoveredNode?: Node;
    app: Application;
    pointerPosition: Point;
  }

  let { hoveredNode, app, pointerPosition }: Props = $props();

  let container: Container;
  let tooltip: Graphics;

  let titleStyle: TextStyle;
  let title: Text;

  let statsStyle: TextStyle;
  let stats: Text;

  onMount(() => {
    container = new Container({
      position: { x: app.screen.width / 2, y: app.screen.height / 2 },
      interactive: false,
      hitArea: undefined,
      eventMode: 'none'
    });

    app.stage.addChild(container);

    tooltip = new Graphics();
    container.addChild(tooltip);

    titleStyle = new TextStyle({ fontFamily: 'Arial', fontSize: 25, fill: 0xffffff, align: 'center' });
    title = new Text('', titleStyle);
    container.addChild(title);

    statsStyle = new TextStyle({ fontFamily: 'Arial', fontSize: 17, fill: 0xffffff, align: 'left' });
    stats = new Text('', statsStyle);
    container.addChild(stats);

    return () => {
      container.destroy({
        children: true
      });
    };
  });

  $effect(() => {
    let maxWidth = 600;

    tooltip.clear();

    if (!hoveredNode) {
      container.alpha = 0;
    } else {
      container.alpha = 1;

      let nodeName = hoveredNode.name || 'N/A';
      if ($devMode) {
        nodeName += ' (' + hoveredNode.skill + ')';
      }

      const statsText = hoveredNode?.stats?.join('\n').trim() || '';

      const titleMetrics = CanvasTextMetrics.measureText(nodeName, titleStyle);
      const statsMetrics = CanvasTextMetrics.measureText(statsText, statsStyle);

      maxWidth = Math.max(titleMetrics.width + 50, maxWidth);
      maxWidth = Math.max(statsMetrics.width + 30, maxWidth);

      const titleHeight = 55;

      tooltip.fillStyle = 'rgba(75,63,24,0.9)';
      tooltip.rect(0, 0, maxWidth, titleHeight);
      tooltip.fill();

      title.text = nodeName;
      title.position.set(maxWidth / 2, titleHeight / 2);
      title.anchor = 0.5;

      tooltip.fillStyle = 'rgba(0,0,0,0.8)';
      tooltip.rect(0, titleHeight, maxWidth, statsMetrics.height + 30);
      tooltip.fill();

      stats.text = statsText;
      stats.position.set(15, titleHeight + 15);
    }

    container.position.set(pointerPosition.x - maxWidth / 2, pointerPosition.y + 25);
  });
</script>
