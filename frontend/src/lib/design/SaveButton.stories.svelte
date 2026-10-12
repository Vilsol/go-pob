<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import SaveButton from './SaveButton.svelte';
  import Button from './Button.svelte';

  const { Story } = defineMeta({
    title: 'Spec/SaveButton',
    component: SaveButton,
    args: { dirty: true }
  });
</script>

<script lang="ts">
  let dirty = $state(true);
</script>

<Story name="Unsaved" args={{ dirty: true }} />

<Story name="Saved" args={{ dirty: false }} />

<Story
  name="Interactive"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const save = c.getByRole('button', { name: 'Save (unsaved changes)' });
    await userEvent.click(save);
    await expect(c.getByRole('button', { name: 'Save (no unsaved changes)' })).toBeInTheDocument();
    await userEvent.click(c.getByRole('button', { name: 'Undo' }));
    await expect(c.getByRole('button', { name: 'Save (unsaved changes)' })).toBeInTheDocument();
  }}>
  <div class="flex items-center gap-2">
    <Button onclick={() => (dirty = true)}>Undo</Button>
    <SaveButton {dirty} onsave={() => (dirty = false)} />
  </div>
</Story>
