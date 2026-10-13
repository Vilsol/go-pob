<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import TattooPopup from './TattooPopup.svelte';
  import type { TattooChoiceFx } from '$lib/skill_tree/tattoo';

  // TreeTab.lua:841-990
  const { Story } = defineMeta({ title: 'Tree/TattooPopup', component: TattooPopup, parameters: { layout: 'fullscreen' } });

  const ROWS: TattooChoiceFx[] = [
    { dn: 'Tattoo of the Ngamahu Firewalker', label: '', descriptions: ['6% increased Fire Damage', 'Limited to 1 Ngamahu Tattoo'], legacy: false },
    { dn: 'Tattoo of the Rongokurai Warrior', label: '', descriptions: ['+2% to Physical Damage over Time Multiplier'], legacy: false },
    {
      dn: 'Honoured Tattoo of the Makanga',
      label: '',
      descriptions: ['+10 to Strength', 'Requires at least 8 adjacent Passive Skills to be allocated'],
      legacy: true
    }
  ];
</script>

<script lang="ts">
  let showLegacy = $state(false);
  let selected = $state(0);
  let added = $state('');
  const choices = $derived(ROWS.filter((r) => showLegacy || !r.legacy));
  // TreeTab.lua:871
  const KEYSTONE: TattooChoiceFx[] = [
    {
      dn: '',
      label: '                                                Limited to 1 Ascendancy Tattoo',
      descriptions: ['Limited to 1 Ascendancy Tattoo'],
      legacy: false
    },
    { dn: 'Acrobatics', label: 'Acrobatics', descriptions: ['Limited to 1 Keystone Tattoo'], legacy: false }
  ];
  let keystoneSelected = $state(0);
  let keystoneAdded = $state<string>();
</script>

<Story
  name="Tattoo a Strength node"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement.ownerDocument.body);
    const dialog = c.getByRole('dialog', { name: 'Replace modifier of Strength' });
    const select = within(dialog).getByRole('combobox', { name: 'Modifier' });
    await expect(within(select).getAllByRole('option')).toHaveLength(2);
    await userEvent.click(within(dialog).getByRole('checkbox', { name: 'Show Legacy Tattoos' }));
    await expect(within(select).getAllByRole('option')).toHaveLength(3);
    await userEvent.selectOptions(select, 'Honoured Tattoo of the Makanga');
    await expect(within(dialog).getByLabelText('Modifier lines')).toHaveTextContent('Requires at least 8 adjacent');
    await userEvent.click(within(dialog).getByRole('button', { name: 'Add' }));
    await expect(c.getByLabelText('Added')).toHaveTextContent('Honoured Tattoo of the Makanga');
  }}>
  <div class="h-150">
    <TattooPopup
      open
      nodeName="Strength"
      {choices}
      bind:selected
      bind:showLegacy
      count={51}
      countLines={[{ line: ' 6% increased Fire Damage', count: 51 }]}
      onadd={(dn: string) => (added = dn)}
      onreset={() => (added = '')}
      onclose={() => {}} />
    <output aria-label="Added">{added}</output>
  </div>
</Story>

<Story
  name="The nameless keystone row"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement.ownerDocument.body);
    const dialog = c.getByRole('dialog', { name: 'Replace modifier of Point Blank' });
    const options = within(within(dialog).getByRole('combobox', { name: 'Modifier' })).getAllByRole('option');
    await expect(options.map((o) => o.textContent?.trim())).toEqual(['Limited to 1 Ascendancy Tattoo', 'Acrobatics']);
    await userEvent.click(within(dialog).getByRole('button', { name: 'Add' }));
    await expect(c.getByLabelText('Added')).toHaveTextContent('(nameless)');
  }}>
  <div class="h-150">
    <TattooPopup
      open
      nodeName="Point Blank"
      choices={KEYSTONE}
      bind:selected={keystoneSelected}
      count={0}
      countLines={[]}
      onadd={(dn: string) => (keystoneAdded = dn)}
      onreset={() => {}}
      onclose={() => {}} />
    <output aria-label="Added">{keystoneAdded === '' ? '(nameless)' : (keystoneAdded ?? '')}</output>
  </div>
</Story>
