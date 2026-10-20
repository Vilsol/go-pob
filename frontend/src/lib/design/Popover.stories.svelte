<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, waitFor, within } from 'storybook/test';
  import { expectFloating } from './stories/layout_checks';
  import Popover from './Popover.svelte';

  const { Story } = defineMeta({ title: 'Spec/Popover', component: Popover });
</script>

<script lang="ts">
  import Checkbox from './Checkbox.svelte';
  import Button from './Button.svelte';
  let open = $state(false);
  let root = $state<HTMLElement>();
</script>

<Story
  name="Dismissal"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const trigger = c.getByRole('button', { name: 'Columns' });
    await userEvent.click(trigger);
    const panel = await c.findByRole('group', { name: 'Visible columns' });
    await waitFor(() => expectFloating(trigger, panel));
    await userEvent.click(c.getByRole('checkbox', { name: 'Level' }));
    await expect(c.getByRole('group', { name: 'Visible columns' }), 'a press inside keeps it open').toBeVisible();
    await userEvent.keyboard('{Escape}');
    await waitFor(() => expect(c.queryByRole('group', { name: 'Visible columns' })).toBeNull());
    await expect(trigger, 'Escape returns focus to the trigger').toHaveFocus();
    await userEvent.click(trigger);
    await c.findByRole('group', { name: 'Visible columns' });
    await userEvent.click(trigger);
    await waitFor(() => expect(c.queryByRole('group', { name: 'Visible columns' }), 'the trigger toggles it closed').toBeNull());
    await userEvent.click(trigger);
    await c.findByRole('group', { name: 'Visible columns' });
    await userEvent.click(c.getByText('Elsewhere'));
    await waitFor(() => expect(c.queryByRole('group', { name: 'Visible columns' }), 'a press outside closes it').toBeNull());
  }}>
  <div class="flex h-80 flex-col items-start gap-4">
    <div class="relative" bind:this={root}>
      <Button size="sm" aria-expanded={open} onclick={() => (open = !open)}>Columns</Button>
      <Popover {open} anchor={root} role="group" label="Visible columns" onclose={() => (open = false)} class="flex w-55 flex-col p-1.5 text-13">
        <label class="menu-item"><Checkbox class="size-4" />Level</label>
        <label class="menu-item"><Checkbox class="size-4" />Life</label>
      </Popover>
    </div>
    <p class="note">Elsewhere</p>
  </div>
</Story>
