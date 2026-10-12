<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import Menu from './Menu.svelte';
  import Button from './Button.svelte';

  const { Story } = defineMeta({ title: 'Spec/Menu', component: Menu });
</script>

<script lang="ts">
  let last = $state('—');
  const items = [
    { label: 'Open', onselect: () => (last = 'Open') },
    { label: 'Rename…', onselect: () => (last = 'Rename…') },
    { label: 'Delete…', tone: 'warn' as const, separator: true, onselect: () => (last = 'Delete…') }
  ];
</script>

<Story name="Open" asChild>
  <div class="h-40">
    <Menu label="Actions for Spectre Necro – budget" {items} open class="top-full right-auto left-0 mt-1 w-57.5">
      {#snippet trigger(attrs)}<Button size="sm" {...attrs}>Actions</Button>{/snippet}
    </Menu>
  </div>
</Story>

<Story
  name="Interactive"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const trigger = c.getByRole('button', { name: 'Actions' });
    await expect(trigger).toHaveAttribute('aria-expanded', 'false');
    await userEvent.click(trigger);
    await expect(trigger).toHaveAttribute('aria-expanded', 'true');
    await expect(c.getByRole('menu', { name: 'Actions for Spectre Necro – budget' })).toBeVisible();
    await expect(c.getByRole('menuitem', { name: 'Open' })).toHaveFocus();
    await userEvent.keyboard('{ArrowDown}');
    await expect(c.getByRole('menuitem', { name: 'Rename…' })).toHaveFocus();
    await userEvent.keyboard('{End}');
    await expect(c.getByRole('menuitem', { name: 'Delete…' })).toHaveFocus();
    await userEvent.keyboard('{ArrowDown}');
    await expect(c.getByRole('menuitem', { name: 'Open' })).toHaveFocus();
    await userEvent.keyboard('{Escape}');
    await expect(c.queryByRole('menu')).toBeNull();
    await expect(trigger).toHaveFocus();
    await userEvent.keyboard('{ArrowUp}');
    await expect(c.getByRole('menuitem', { name: 'Delete…' })).toHaveFocus();
    await userEvent.click(c.getByRole('menuitem', { name: 'Rename…' }));
    await expect(c.queryByRole('menu')).toBeNull();
    await expect(c.getByText('Last action: Rename…')).toBeInTheDocument();
  }}>
  <div class="flex h-40 items-start gap-4">
    <Menu label="Actions for Spectre Necro – budget" {items} class="top-full right-auto left-0 mt-1 w-57.5">
      {#snippet trigger(attrs)}<Button size="sm" {...attrs}>Actions</Button>{/snippet}
    </Menu>
    <span class="text-13 text-text-2">Last action: {last}</span>
  </div>
</Story>
