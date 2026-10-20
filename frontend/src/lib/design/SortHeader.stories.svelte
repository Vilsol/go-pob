<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import SortHeader from './SortHeader.svelte';

  const { Story } = defineMeta({ title: 'Spec/SortHeader', component: SortHeader });
</script>

<script lang="ts">
  let sort = $state<{ key: string; dir: 'asc' | 'desc' }>({ key: 'name', dir: 'asc' });
  const flip = (key: string) => (sort = { key, dir: sort.key === key && sort.dir === 'asc' ? 'desc' : 'asc' });
</script>

<Story
  name="Columns"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    await expect(c.getByRole('columnheader', { name: 'NAME' })).toHaveAttribute('aria-sort', 'ascending');
    await userEvent.click(c.getByRole('button', { name: 'LEVEL' }));
    await expect(c.getByRole('columnheader', { name: 'LEVEL' })).toHaveAttribute('aria-sort', 'ascending');
    await expect(c.getByRole('columnheader', { name: 'NAME' })).toHaveAttribute('aria-sort', 'none');
    await userEvent.click(c.getByRole('button', { name: 'LEVEL' }));
    await expect(c.getByRole('columnheader', { name: 'LEVEL' })).toHaveAttribute('aria-sort', 'descending');
  }}>
  <div role="table" aria-label="Builds" class="panel w-120">
    <div role="row" class="data-row data-head grid-cols-[minmax(0,1fr)_6rem_6rem] tracking-caps">
      <SortHeader label="NAME" active={sort.key === 'name'} dir={sort.dir} onsort={() => flip('name')} />
      <SortHeader label="LEVEL" active={sort.key === 'level'} dir={sort.dir} align="end" onsort={() => flip('level')} />
      <SortHeader label="LIFE" active={sort.key === 'life'} dir={sort.dir} align="end" onsort={() => flip('life')} />
    </div>
  </div>
</Story>
