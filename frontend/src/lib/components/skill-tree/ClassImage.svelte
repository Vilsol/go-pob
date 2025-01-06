<script lang="ts">
  import { allExtraImages, toCanvasCoords } from '../../skill_tree';
  import type { Tree } from '$lib/skill_tree/types';
  import { Container, Sprite } from 'pixi.js';
  import { onMount } from 'svelte';

  interface Props {
    currentClass?: string;
    skillTree: Tree;
    parentContainer: Container;
  }

  let { currentClass, skillTree, parentContainer }: Props = $props();

  const container = new Container();

  onMount(() => {
    parentContainer.addChild(container);

    const classIndex = skillTree.classes.findIndex((c) => c.name === currentClass);

    const g = new Sprite(allExtraImages[classIndex in skillTree.extraImages ? classIndex : 0]);

    if (classIndex in skillTree.extraImages) {
      const img = skillTree.extraImages[classIndex];
      g.position = toCanvasCoords(img.x, img.y);
    } else {
      g.position.set(0, 0);
    }

    g.scale = 0.5;
    g.alpha = 1;
    container.addChild(g);

    $effect(() => {
      const classIndex2 = skillTree.classes.findIndex((c) => c.name === currentClass);
      if (classIndex2 in skillTree.extraImages) {
        g.texture = allExtraImages[classIndex2];
        g.alpha = 1;

        const img = skillTree.extraImages[classIndex2];
        g.position = toCanvasCoords(img.x, img.y);
      } else {
        g.alpha = 0;
      }
    });

    return () => {
      container.destroy({
        children: true
      });
    };
  });
</script>
