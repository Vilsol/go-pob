// SkillListControl.lua:9-27, 234-260

const SLOT_ICONS: Record<string, string> = {
  'Weapon 1': 'icon_weapon',
  'Weapon 2': 'icon_weapon_2',
  'Weapon 1 Swap': 'icon_weapon_swap',
  'Weapon 2 Swap': 'icon_weapon_2_swap',
  Bow: 'icon_bow',
  Quiver: 'icon_quiver',
  Shield: 'icon_shield',
  'Shield Swap': 'icon_shield_swap',
  Helmet: 'icon_helmet',
  'Body Armour': 'icon_body_armour',
  Gloves: 'icon_gloves',
  Boots: 'icon_boots',
  Amulet: 'icon_amulet',
  'Ring 1': 'icon_ring_left',
  'Ring 2': 'icon_ring_right',
  'Ring 3': 'icon_ring_right',
  Belt: 'icon_belt'
};

export function slotIcon(slot: string, types: Record<string, string | undefined>): string | undefined {
  const type = (s: string) => types[s] ?? 'None';
  let key = slot;
  if (slot === 'Weapon 1' && type('Weapon 1') === 'Bow') key = 'Bow';
  if (slot === 'Weapon 1 Swap' && type('Weapon 1 Swap') === 'Bow') key = 'Bow Swap';
  if (slot === 'Weapon 2' && ['Quiver', 'Shield'].includes(type('Weapon 2'))) key = type('Weapon 2');
  if (slot === 'Weapon 2 Swap' && ['Quiver', 'Shield'].includes(type('Weapon 2 Swap'))) key = `${type('Weapon 2 Swap')} Swap`;
  const file = SLOT_ICONS[key];
  return file && `${file}.png`;
}
