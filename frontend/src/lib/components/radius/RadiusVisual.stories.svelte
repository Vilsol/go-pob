<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import RadiusVisual from './RadiusVisual.svelte';
  import { expect } from 'storybook/test';

  const { Story } = defineMeta({ title: 'Calcs/RadiusVisual', component: RadiusVisual, args: { radius: 24 } });
</script>

<Story
  name="A skill's area"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const svg = canvasElement.querySelector('svg')!;
    await expect(svg.querySelectorAll('text').length, 'the 20-80 labels').toBe(4);
    await expect(svg.querySelector('[data-radius-edge]')?.getAttribute('d')?.length ?? 0).toBeGreaterThan(100);
  }}>
  <div class="w-80 bg-surface-1 p-3">
    <RadiusVisual radius={24} />
  </div>
</Story>

<Story name="Small, large and wide" asChild>
  <div class="flex w-[1000px] gap-3 bg-surface-1 p-3">
    {#each [8, 44, 100] as r (r)}
      <div class="w-80"><RadiusVisual radius={r} /></div>
    {/each}
  </div>
</Story>
