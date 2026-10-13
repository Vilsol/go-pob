<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, fn, userEvent, within } from 'storybook/test';
  import CharacterImport from './CharacterImport.svelte';
  import { REALM_LIST, type AccountCharacter } from './account';

  // ImportTab.lua:321-472
  const { Story } = defineMeta({
    title: 'Pages/Character import',
    component: CharacterImport,
    parameters: { a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } } }
  });

  const characters: AccountCharacter[] = [
    { name: 'Spectrewitch', realm: 'pc', class: 'Necromancer', league: 'Standard', level: 94 },
    { name: 'mineShadow', realm: 'pc', class: 'Saboteur', league: 'Standard', level: 90 },
    { name: 'LSJugg', realm: 'pc', class: 'Juggernaut', league: 'Hardcore Mercenaries', level: 96 }
  ];
  const onstart = fn();
  const onimporttree = fn();
  const onimportitems = fn();
</script>

<Story
  name="Account name"
  args={{ realms: REALM_LIST, history: ['Exile#1234', 'another#0001'], onstart }}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const c = within(canvasElement);
    const box = c.getByRole('textbox', { name: /Account name/ });
    await userEvent.type(box, 'Exile');
    await expect(c.getByRole('button', { name: 'Start' })).toBeDisabled();
    await expect(c.getByText('Missing discriminator e.g. #1234')).toBeVisible();
    await userEvent.type(box, '#1234');
    await expect(c.queryByText('Missing discriminator e.g. #1234')).toBeNull();
    await userEvent.click(c.getByRole('button', { name: 'Start' }));
    await expect(onstart).toHaveBeenCalledWith(REALM_LIST[0], 'Exile#1234');
  }} />

<Story name="Private profile" args={{ realms: REALM_LIST, account: 'Exile#1234', status: '^xDD0022Account profile is private or does not exist.' }} />

<Story
  name="Choose a character"
  args={{ realms: REALM_LIST, stage: 'SELECTCHAR', status: 'Character list successfully retrieved.', characters, onimporttree, onimportitems }}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const c = within(canvasElement);
    const pick = c.getByRole<HTMLSelectElement>('combobox', { name: 'Character' });
    await expect([...pick.options].map((o) => o.textContent)).toEqual(['LSJugg — Juggernaut lvl 96']);
    await userEvent.selectOptions(c.getByRole('combobox', { name: 'League' }), 'All');
    await expect([...pick.options].map((o) => o.textContent)).toEqual([
      'LSJugg — Juggernaut lvl 96 in Hardcore Mercenaries',
      'mineShadow — Saboteur lvl 90 in Standard',
      'Spectrewitch — Necromancer lvl 94 in Standard'
    ]);
    await userEvent.selectOptions(pick, 'Spectrewitch');
    await userEvent.click(c.getByRole('checkbox', { name: 'Delete jewels' }));
    await userEvent.click(c.getByRole('button', { name: 'Passive Tree and Jewels' }));
    await expect(onimporttree).toHaveBeenCalledWith(characters[0], false);
    await userEvent.click(c.getByRole('button', { name: 'Items and Skills' }));
    await expect(onimportitems).toHaveBeenCalledWith(characters[0], { clearSkills: true, clearItems: true, ignoreWeaponSwap: false });
  }} />
