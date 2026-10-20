<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, waitFor, within } from 'storybook/test';
  import { expectFloating } from './stories/layout_checks';
  import Tooltip from './Tooltip.svelte';

  const { Story } = defineMeta({ title: 'Spec/Tooltip', component: Tooltip });
</script>

<script lang="ts">
  let anchor = $state<HTMLElement>();
  let shown = $state(false);
  let at = $state<{ x: number; y: number }>();
</script>

<Story
  name="Placements"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const trigger = c.getByRole('button', { name: 'Life' });
    await userEvent.hover(trigger);
    const tip = await c.findByRole('tooltip', { name: 'Life breakdown' });
    await waitFor(() => expectFloating(trigger, tip));
    await userEvent.unhover(trigger);
    await waitFor(() => expect(c.queryByRole('tooltip', { name: 'Life breakdown' })).toBeNull());
    const map = c.getByTestId('map');
    await userEvent.hover(map);
    const near = await c.findByRole('tooltip', { name: 'Socket' });
    await waitFor(() => expect(near.getBoundingClientRect().left).toBeGreaterThan(map.getBoundingClientRect().left + 100));
  }}>
  <div class="flex flex-col items-start gap-4">
    <button class="btn btn-sm" bind:this={anchor} onmouseenter={() => (shown = true)} onmouseleave={() => (shown = false)}>Life</button>
    {#if shown}<Tooltip placement="beside" {anchor} label="Life breakdown">Life: 4,512 · 128% increased</Tooltip>{/if}
    <div
      role="presentation"
      data-testid="map"
      class="relative h-40 w-120 rounded-md bg-well"
      onmouseenter={() => (at = { x: 100, y: 60 })}
      onmouseleave={() => (at = undefined)}>
      {#if at}<Tooltip placement="point" {at} label="Socket" class="w-60">Jewel socket · 12 notables in radius</Tooltip>{/if}
    </div>
    <Tooltip size="sm" flat>An in-flow tooltip, sm padding, no shadow</Tooltip>
    <Tooltip size="lg" textSize={11}>An in-flow tooltip, lg padding, 11px text</Tooltip>
  </div>
</Story>
