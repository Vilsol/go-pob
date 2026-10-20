<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import SearchField from './SearchField.svelte';

  const { Story } = defineMeta({ title: 'Spec/SearchField', component: SearchField });
</script>

<script lang="ts">
  let q = $state('');
</script>

<Story
  name="Sizes"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const box = c.getByRole('searchbox', { name: 'Search builds' });
    await userEvent.type(box, 'witch');
    await expect(c.getByTestId('echo')).toHaveTextContent('witch');
  }}>
  <div class="flex w-80 flex-col gap-2">
    <SearchField size="xs" label="Filter skills" placeholder="Type to filter…" />
    <SearchField size="sm" label="Search notables" placeholder="Search" />
    <SearchField label="Search builds" placeholder="Search builds" bind:value={q} />
    <SearchField size="md" label="Search the tree" placeholder="Search nodes and stats" />
    <SearchField size="lg" label="Search all builds" placeholder="Search builds" />
    <span class="note" data-testid="echo">{q}</span>
  </div>
</Story>
