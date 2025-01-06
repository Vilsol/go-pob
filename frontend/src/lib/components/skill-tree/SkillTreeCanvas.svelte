<script lang="ts">
  import type { Tree } from '$lib/skill_tree/types';
  import { onMount } from 'svelte';
  import { type WebGLRenderer } from 'pixi.js';
  import { Stats } from 'pixi-stats';
  import SkillTree from '$lib/components/skill-tree/SkillTree.svelte';
  import { app } from '$lib/components/skill-tree/common';
  import { initializeSpritesheets } from '$lib/skill_tree';
  import { initDevtools } from '@pixi/devtools';
  import { devMode } from '$lib/global';

  interface Props {
    skillTree: Tree;
    skillTreeVersion: string;
  }

  let { skillTree, skillTreeVersion }: Props = $props();

  let cursor = $state('unset');

  let parentContainer = $state<HTMLElement>();
  let innerContainer = $state<HTMLElement>();

  let canvasWidth = $state(0);
  let canvasHeight = $state(0);
  const resize = () => {
    if (parentContainer) {
      canvasWidth = parentContainer.offsetWidth;
      canvasHeight = parentContainer.offsetHeight;
    }
  };

  let cdnBase = $derived(`https://go-pob-data.pages.dev/data/${(skillTreeVersion || '3_18').replace('_', '.')}`);

  let initialized = $state(false);

  let stats: Stats;

  onMount(() => {
    resize();

    if (!innerContainer) {
      return;
    }

    if (app && app.renderer) {
      app.resizeTo = parentContainer!;

      innerContainer?.appendChild(app.canvas);

      if (stats && stats.domElement) {
        stats.domElement.remove();
      }

      stats = new Stats(app.renderer as WebGLRenderer, innerContainer);
      stats.domElement.id = 'pixi-stats';

      initialized = true;

      return;
    }

    void app
      .init({
        background: '#080c11',
        autoStart: true,
        sharedTicker: false,
        width: canvasWidth,
        height: canvasHeight,
        resizeTo: parentContainer,
        antialias: true
      })
      .then(async () => {
        // Append the application canvas to the document body
        innerContainer?.appendChild(app.canvas);

        await initDevtools({ app });

        stats = new Stats(app.renderer as WebGLRenderer, innerContainer);
        stats.domElement.id = 'pixi-stats';

        await initializeSpritesheets(cdnBase);

        initialized = true;
      });
  });
</script>

<svelte:window onresize={resize} />

<div class="w-full h-full max-w-full max-h-full overflow-hidden" bind:this={parentContainer} onresize={resize}>
  <div
    class:hide-stats={!$devMode}
    class="relative w-full h-full max-w-full max-h-full"
    style="touch-action: none; cursor: {cursor}"
    bind:this={innerContainer}>
    {#if initialized && app}
      <SkillTree {skillTree} {skillTreeVersion} {app} />
    {/if}
  </div>
</div>
