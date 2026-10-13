<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import BuildPicker from './BuildPicker.svelte';
  import { buildList } from './fixtures/app';
  import type { SavedBuild } from './fixtures/app';

  // CompareTab.lua:1529-1640
  const { Story } = defineMeta({ title: 'Shell/BuildPicker', component: BuildPicker, parameters: { layout: 'fullscreen' } });
</script>

<script lang="ts">
  let picked = $state('');
</script>

<Story
  name="Pick a saved build"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement.ownerDocument.body);
    const list = c.getByRole('listbox', { name: 'Builds in all folders' });
    await userEvent.click(within(list).getByText('Imported'));
    await expect(c.getByRole('listbox', { name: 'Builds in Imported' })).toBeTruthy();
    await userEvent.type(c.getByRole('searchbox', { name: 'Search builds' }), 'totems');
    await userEvent.keyboard('{Enter}');
    await expect(c.getByTestId('picked')).toHaveTextContent('Imported/pobb.in/Ice Spear Totems');
  }}>
  <div class="h-150">
    <BuildPicker open action="Compare" builds={buildList.builds} folders={buildList.folders} onpick={(b: SavedBuild) => (picked = `${b.folder}/${b.name}`)} />
    <output data-testid="picked">{picked}</output>
  </div>
</Story>

<Story name="Loading" asChild>
  <div class="h-150"><BuildPicker open builds={undefined} folders={[]} onpick={() => {}} /></div>
</Story>
