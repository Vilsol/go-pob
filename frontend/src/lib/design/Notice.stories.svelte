<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, within } from 'storybook/test';
  import InlineMessage from './InlineMessage.svelte';
  import Notice from './Notice.svelte';

  const { Story } = defineMeta({ title: 'Spec/Notice', component: Notice });
</script>

<Story
  name="Kinds"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    await expect(c.getAllByRole('alert')).toHaveLength(1);
    await expect(c.getByRole('button', { name: 'Restore rich version' })).toBeTruthy();
    const float = c.getAllByRole('status').find((e) => e.textContent?.includes('Loading game data'))!;
    await expect(float.textContent, 'no stray whitespace before the text (e2e matches /^Offline/)').toMatch(/^Loading game data/);
  }}>
  <div class="flex w-180 flex-col gap-2">
    <Notice>Recalculating 3 / 40 in the background · Build name…</Notice>
    <Notice kind="info" title="IMPORTED FROM POB"
      >Colour codes converted: 4 ^xRRGGBB / ^0–^9 codes became text colours.{#snippet actions()}<button class="btn btn-sm btn-ghost">Dismiss</button
        >{/snippet}</Notice>
    <Notice kind="warn" title="CHANGED OUTSIDE GO-POB"
      >These notes were edited as PoB text, so that text is shown.{#snippet actions()}<button class="btn btn-sm btn-primary">Restore rich version</button
        ><button class="btn btn-sm">Keep PoB text</button>{/snippet}</Notice>
    <Notice kind="warn" size="sm" class="text-worse">⚠ You have 2 unallocated passive points</Notice>
    <Notice kind="error">The build file could not be read.</Notice>
    <Notice float>Loading game data: items</Notice>
  </div>
</Story>

<Story name="Inline messages" asChild>
  <div class="flex flex-col gap-1.5">
    <InlineMessage>Character import status: idle</InlineMessage>
    <InlineMessage kind="success">✓ Copied</InlineMessage>
    <InlineMessage kind="warn">A buffer changed since the effects were applied</InlineMessage>
    <InlineMessage kind="error">Import failed: invalid build code</InlineMessage>
    <InlineMessage kind="error" size="md" tag="p">The item is invalid.</InlineMessage>
  </div>
</Story>
