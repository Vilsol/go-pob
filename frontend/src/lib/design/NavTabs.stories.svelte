<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import NavTabs from './NavTabs.svelte';

  const tabs = [
    { label: 'Tree', href: '#tree', key: '1' },
    { label: 'Skills', href: '#skills', key: '2' },
    { label: 'Items', href: '#items', key: '3' },
    { label: 'Calcs', href: '#calcs', key: '4' }
  ];

  const { Story } = defineMeta({ title: 'Spec/NavTabs', component: NavTabs, args: { tabs, current: '#tree', label: 'Build pages' } });
</script>

<script lang="ts">
  let current = $state('#tree');
  const go = (t: { href: string }, e: Event) => {
    e.preventDefault();
    current = t.href;
  };
</script>

<Story name="Static" />

<Story
  name="Interactive"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    await userEvent.click(c.getByRole('link', { name: 'Items 3' }));
    await expect(c.getByRole('link', { name: 'Items 3' })).toHaveAttribute('aria-current', 'page');
    await expect(c.getByRole('link', { name: 'Tree 1' })).not.toHaveAttribute('aria-current');
    await userEvent.keyboard('4');
    await expect(c.getByRole('link', { name: 'Calcs 4' })).toHaveAttribute('aria-current', 'page');
    c.getByRole('link', { name: 'Tree 1' }).focus();
    await userEvent.keyboard('{ArrowRight}');
    await expect(c.getByRole('link', { name: 'Skills 2' })).toHaveFocus();
    await userEvent.keyboard('{End}');
    await expect(c.getByRole('link', { name: 'Calcs 4' })).toHaveFocus();
    await userEvent.keyboard('{ArrowRight}');
    await expect(c.getByRole('link', { name: 'Tree 1' })).toHaveFocus();
    await userEvent.keyboard('{ArrowLeft}');
    await expect(c.getByRole('link', { name: 'Calcs 4' })).toHaveFocus();
    const search = c.getByRole('textbox', { name: 'Search' });
    await userEvent.click(search);
    await userEvent.keyboard('2');
    await expect(c.getByRole('link', { name: 'Calcs 4' })).toHaveAttribute('aria-current', 'page');
    await userEvent.keyboard('{Alt>}2{/Alt}');
    await expect(c.getByRole('link', { name: 'Skills 2' })).toHaveAttribute('aria-current', 'page');
  }}>
  <div class="flex flex-col gap-3">
    <NavTabs {tabs} {current} label="Build pages" onnavigate={go} />
    <input class="input w-60" aria-label="Search" placeholder="Search (digits type here; Alt+digit switches)" />
  </div>
</Story>
