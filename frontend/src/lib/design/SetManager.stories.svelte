<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import SetManager from './SetManager.svelte';
  import type { SetListOp } from './set_list';

  // ConfigSetListControl.lua:13-110
  const { Story } = defineMeta({ title: 'Shell/SetManager', component: SetManager, parameters: { layout: 'fullscreen' } });
</script>

<script lang="ts">
  let sets = $state(['Default', 'Mapping', 'Uber bosses']);
  let current = $state(0);
  const onop = (op: SetListOp) => {
    if (op.op === 'new' || op.op === 'copy') sets.push(op.title);
    else if (op.op === 'rename') sets[op.index] = op.title;
    else if (op.op === 'delete') sets.splice(op.index, 1);
    else if (op.op === 'move') sets.splice(op.to, 0, ...sets.splice(op.from, 1));
  };
</script>

<Story
  name="Manage config sets"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement.ownerDocument.body);
    const list = c.getByRole('listbox', { name: 'Config sets' });
    await expect(within(list).getAllByRole('option')[0]).toHaveTextContent('current');
    await userEvent.click(within(list).getByRole('option', { name: /Uber bosses/ }));
    await userEvent.click(c.getByRole('button', { name: 'Move up' }));
    await expect(within(list).getAllByRole('option')[1]).toHaveTextContent('Uber bosses');
    await userEvent.click(c.getByRole('button', { name: 'New' }));
    await userEvent.type(c.getByRole('textbox', { name: 'Name for this config set' }), 'Delve');
    await userEvent.click(c.getByRole('button', { name: 'Save' }));
    await expect(within(list).getAllByRole('option')).toHaveLength(4);
    await userEvent.dblClick(within(list).getByRole('option', { name: /Mapping/ }));
    await expect(within(list).getByRole('option', { name: /Mapping/ })).toHaveTextContent('current');
  }}>
  <div class="h-150">
    <SetManager open title="Manage Config Sets" noun="config set" {sets} {current} movable onactivate={(i: number) => (current = i)} {onop} />
  </div>
</Story>
