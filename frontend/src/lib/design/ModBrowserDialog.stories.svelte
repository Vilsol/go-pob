<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within, waitFor, screen } from 'storybook/test';
  import ModBrowserDialog from './ModBrowserDialog.svelte';

  const LINES = ['+10 to maximum Life', '10% increased Attack Speed while Chilled', '1% increased Area of Effect', 'Quux frobnicates'];
  const list = (search: string) => {
    const q = search.trim().toLowerCase();
    const shown = q ? LINES.filter((l) => l.toLowerCase().includes(q)) : LINES;
    return Promise.resolve(shown.length ? shown : ['No matching modifiers found']);
  };
  const supported = (line: string) => Promise.resolve(!line.startsWith('Quux'));

  const { Story } = defineMeta({
    title: 'Pages/Config/Mod Browser',
    component: ModBrowserDialog,
    args: { open: true, list, supported, onadd: () => {} }
  });
</script>

<Story
  name="Search and hover"
  play={async () => {
    const d = within(await screen.findByRole('dialog', { name: 'Mod Browser' }));
    await waitFor(() => expect(d.getAllByRole('option')).toHaveLength(4));
    await userEvent.type(d.getByRole('searchbox', { name: 'Search modifiers' }), 'quux');
    await waitFor(() => expect(d.getAllByRole('option')).toHaveLength(1));
    await userEvent.hover(d.getByRole('option', { name: 'Quux frobnicates' }));
    await waitFor(() => expect(d.getByRole('status')).toHaveTextContent('Supported: No'));
    await userEvent.clear(d.getByRole('searchbox', { name: 'Search modifiers' }));
    await userEvent.type(d.getByRole('searchbox', { name: 'Search modifiers' }), 'zzz');
    await waitFor(() => expect(d.getByRole('button', { name: 'Add' })).toBeDisabled());
  }} />
