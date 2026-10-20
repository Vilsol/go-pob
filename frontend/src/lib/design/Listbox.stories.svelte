<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import Listbox from './Listbox.svelte';
  import SearchList from './SearchList.svelte';

  const { Story } = defineMeta({ title: 'Spec/Listbox', component: Listbox });
  const NOTABLES = ['Constitution', 'Heart of Oak', 'Thick Skin', 'Sanctity', 'Tireless'];
</script>

<script lang="ts">
  let sel = $state(-1);
  let used = $state('');
  let q = $state('');
  let pick = $state(-1);
  const found = $derived(NOTABLES.filter((n) => n.toLowerCase().includes(q.toLowerCase())));
</script>

<Story
  name="Keyboard"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const list = c.getByRole('listbox', { name: 'Notables' });
    list.focus();
    await userEvent.keyboard('{ArrowDown}');
    await expect(c.getByRole('option', { name: 'Constitution' })).toHaveAttribute('aria-selected', 'true');
    await expect(list.getAttribute('aria-activedescendant')).toBe(c.getByRole('option', { name: 'Constitution' }).id);
    await userEvent.keyboard('{ArrowDown}{ArrowDown}');
    await expect(c.getByRole('option', { name: 'Thick Skin' })).toHaveAttribute('aria-selected', 'true');
    await userEvent.keyboard('{End}{ArrowDown}');
    await expect(c.getByRole('option', { name: 'Tireless' }), 'stops at the end').toHaveAttribute('aria-selected', 'true');
    await userEvent.keyboard('{Enter}');
    await expect(c.getByTestId('used')).toHaveTextContent('Tireless');
    await userEvent.dblClick(c.getByRole('option', { name: 'Sanctity' }));
    await expect(c.getByTestId('used')).toHaveTextContent('Sanctity');
  }}>
  <div class="flex w-70 flex-col gap-2 text-13">
    <Listbox
      items={NOTABLES}
      key={(n: string) => n}
      label="Notables"
      selected={sel}
      class="h-40"
      onselect={(_: string, i: number) => (sel = i)}
      onactivate={(n: string) => (used = n)}>
      {#snippet row(n: string)}{n}{/snippet}
    </Listbox>
    <span class="note" data-testid="used">{used}</span>
  </div>
</Story>

<Story
  name="Search list"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    await userEvent.type(c.getByRole('searchbox', { name: 'Search notables' }), 'zzz');
    await expect(c.getByText('No notables found that match those filters.')).toBeVisible();
  }}>
  <div class="flex w-70 flex-col gap-1.5 text-13">
    <SearchList
      bind:query={q}
      searchLabel="Search notables"
      items={found}
      key={(n: string) => n}
      label="Notables"
      selected={pick}
      emptyText="No notables found that match those filters."
      listClass="h-40"
      onselect={(_: string, i: number) => (pick = i)}>
      {#snippet row(n: string)}{n}{/snippet}
    </SearchList>
  </div>
</Story>
