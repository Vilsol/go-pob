<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import ItemTooltip from './ItemTooltip.svelte';
  import { fireballBuild, type ItemTip } from './fixtures';
  import devHelmet from './stories/item_mod_debug_sample.json';

  const live = (pick: (it: (typeof fireballBuild.items.items)[string]) => boolean) =>
    Object.values(fireballBuild.items.items).find((it) => it.tooltip && pick(it))!;
  const weapon = live((it) => it.fits.includes('Weapon 1'));
  const armour = live((it) => it.fits.includes('Body Armour'));
  const flask = live((it) => it.flask);
  // ItemsTab.lua:5160-5166
  const devTip: ItemTip = {
    influence: devHelmet.tooltip.Influence ?? [],
    title: devHelmet.tooltip.Title,
    lines: devHelmet.tooltip.Lines.map((l) => ({ text: l.Text, sep: l.Separator }))
  };

  const { Story } = defineMeta({
    title: 'Spec/ItemTooltip',
    component: ItemTooltip,
    argTypes: { rarity: { control: 'inline-radio', options: ['normal', 'magic', 'rare', 'unique'] } },
    parameters: {
      a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } }
    },
    args: {
      item: 'Dusk Shroud',
      base: 'Carnal Armour',
      rarity: 'rare',
      mods: ['+94 to maximum Life', '+42% to Cold Resistance'],
      impactTitle: 'EQUIP IMPACT (IN BODY ARMOUR)',
      impact: [
        { label: 'Life', text: '▲ +61', dir: 'up' },
        { label: 'Minion DPS', text: '▼ −10.7%', dir: 'down' }
      ]
    }
  });
</script>

<Story name="Equip candidate" />

<Story
  name="Equipped (unequip impact)"
  args={{
    item: 'Doom Visor',
    base: 'Hubris Circlet',
    mods: ['+72 to maximum Life', '+38% to Lightning Resistance'],
    impactTitle: 'UNEQUIP IMPACT',
    impact: [
      { label: 'Life', text: '▼ −410', dir: 'down' },
      { label: 'Lightning Res', text: '▼ −38%', dir: 'down' }
    ]
  }} />

<Story
  name="Inactive flask"
  args={{
    item: 'Quicksilver Flask',
    base: 'Quicksilver Flask',
    rarity: 'magic',
    mods: ['40% increased Movement Speed during Effect'],
    flask: { on: false, state: 'Inactive — click the tile to activate' },
    impactTitle: 'ACTIVATING THIS FLASK WILL GIVE YOU',
    impact: [{ label: 'Movement Speed', text: '▲ +40%', dir: 'up' }]
  }} />

<Story
  name="Base browser (no mods)"
  args={{
    item: 'Carnal Armour',
    base: 'Body Armour',
    baseTone: 'dim',
    rarity: 'normal',
    mods: [],
    impactTitle: 'EQUIP IMPACT (IN BODY ARMOUR)',
    impact: [{ label: 'Energy Shield', text: '▼ −288', dir: 'down' }]
  }} />

<Story
  name="Live tooltip: weapon"
  args={{ item: weapon.name, base: weapon.base, rarity: weapon.rarity, tooltip: weapon.tooltip, impactTitle: 'UNEQUIP IMPACT', impact: [] }} />

<Story
  name="Live tooltip: body armour"
  args={{ item: armour.name, base: armour.base, rarity: armour.rarity, tooltip: armour.tooltip, impactTitle: 'UNEQUIP IMPACT', impact: [] }} />

<Story
  name="Live tooltip: flask"
  args={{ item: flask.name, base: flask.base, rarity: flask.rarity, tooltip: flask.tooltip, impactTitle: 'UNEQUIP IMPACT', impact: [] }} />

<Story
  name="Live tooltip: dev mode mod list"
  args={{ item: devHelmet.name, base: devHelmet.base, rarity: 'rare', tooltip: devTip, impactTitle: 'UNEQUIP IMPACT', impact: [] }} />
