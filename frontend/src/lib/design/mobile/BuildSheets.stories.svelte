<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, fn, userEvent, waitFor, within } from 'storybook/test';
  import BuildSheets from './BuildSheets.svelte';
  import { undoHistory, loadouts } from '../fixtures/shell';
  import { phone } from '../stories/mobile_checks';

  const jump = fn();

  const { Story } = defineMeta({
    title: 'Mobile/BuildSheets',
    component: BuildSheets,
    parameters: { layout: 'fullscreen' },
    globals: phone
  });
</script>

<Story
  name="Undo history"
  asChild
  play={async ({ canvasElement }) => {
    const sheet = within(document.body).getByRole('dialog', { name: 'Undo history' });
    const list = within(sheet).getByRole('list', { name: 'Undo history' });
    const steps = within(list).getAllByRole('listitem');
    await expect(steps).toHaveLength(undoHistory.length);
    await expect(steps[0].querySelector('svg')).not.toBeNull();

    const stats = within(steps[0]).getByRole('button', { name: /^Stat change of / });
    await expect(stats).toHaveAttribute('aria-expanded', 'false');
    await userEvent.click(stats);
    await expect(stats).toHaveAttribute('aria-expanded', 'true');
    await expect(within(steps[0]).getByRole('list', { name: 'Stat change' })).toHaveTextContent('+212 (+4.1%)');
    await expect(within(steps[4]).queryByRole('button', { name: /^Stat change of / }), 'a step without a recorded change offers none').toBeNull();

    jump.mockClear();
    await userEvent.click(within(steps[3]).getByRole('button', { name: /^Switched item set/ }));
    await expect(jump).toHaveBeenCalledWith(3);
    await waitFor(() => expect(within(canvasElement.ownerDocument.body).queryByRole('dialog', { name: 'Undo history' })).toBeNull());
  }}>
  <BuildSheets open="history" history={undoHistory} undone={1} {loadouts} issues={[]} onhistory={jump} />
</Story>
