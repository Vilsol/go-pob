import type { Tone } from '../tone';
import type { Impact } from '../items';

export type PoolItem = { name: string; rarity: Tone; base: string; slot: string; scope: string; mods: string[]; impact: Impact[] };

export const sharedItems: PoolItem[] = [
  {
    name: 'Leveling Tabula',
    rarity: 'unique',
    base: 'Simple Robe',
    slot: 'Body Armour',
    scope: 'all builds',
    mods: ['6 white sockets'],
    impact: [
      { label: 'Life', text: '▼ −33', dir: 'down' },
      { label: 'Energy Shield', text: '▼ −288', dir: 'down' }
    ]
  },
  {
    name: 'Goldrim',
    rarity: 'unique',
    base: 'Leather Cap',
    slot: 'Helmet',
    scope: 'all builds',
    mods: ['+30% to all Elemental Resistances'],
    impact: [
      { label: 'Fire Res', text: '▲ +30% (capped)', dir: 'up' },
      { label: 'Life', text: '▼ −410', dir: 'down' }
    ]
  },
  {
    name: 'Wanderlust',
    rarity: 'unique',
    base: 'Wool Shoes',
    slot: 'Boots',
    scope: 'all builds',
    mods: ['20% increased Movement Speed'],
    impact: [{ label: 'Movement Speed', text: '▼ −10%', dir: 'down' }]
  }
];

export type BrowseRow = { name: string; rarity: Tone; type: string; note: string; req: string; action: 'add' | 'craft'; impact?: Impact };
export type BrowseKind = {
  id: string;
  label: string;
  title: string;
  col: string;
  search: string;
  note: string;
  filters: { label: string; options: string[] }[];
  checks: { label: string; on: boolean }[];
  rows: BrowseRow[];
};

export const browseKinds: BrowseKind[] = [
  {
    id: 'unique',
    label: 'Uniques',
    title: 'Uniques',
    col: 'IF EQUIPPED',
    search: 'Search uniques',
    note: 'IF EQUIPPED is the equip impact in its slot, a what-if against the item you have there.',
    filters: [
      { label: 'Slot', options: ['Any slot', 'Helmet', 'Body Armour', 'Weapon', 'Boots'] },
      { label: 'Sort by', options: ['Build impact (Full DPS)', 'Build impact (EHP)', 'Name', 'Required level'] }
    ],
    checks: [
      { label: 'Only items this build can use', on: true },
      { label: 'Include legacy variants', on: false }
    ],
    rows: [
      {
        name: 'The Baron',
        rarity: 'unique',
        type: 'Helmet',
        note: 'Close Helmet',
        req: '26',
        action: 'add',
        impact: { label: 'Full DPS', text: '▲ +4.8%', dir: 'up' }
      },
      {
        name: 'Bones of Ullr',
        rarity: 'unique',
        type: 'Boots',
        note: 'Silk Slippers',
        req: '22',
        action: 'add',
        impact: { label: 'Full DPS', text: '▲ +3.2%', dir: 'up' }
      },
      {
        name: 'Victario’s Influence',
        rarity: 'unique',
        type: 'Body Armour',
        note: 'Lacquered Garb',
        req: '50',
        action: 'add',
        impact: { label: 'Full DPS', text: '▼ −14.0%', dir: 'down' }
      },
      {
        name: 'Midnight Bargain',
        rarity: 'unique',
        type: 'Weapon 1',
        note: 'Calling Wand',
        req: '30',
        action: 'add',
        impact: { label: 'Full DPS', text: '▲ +5.5%', dir: 'up' }
      },
      {
        name: 'Grip of the Council',
        rarity: 'unique',
        type: 'Gloves',
        note: 'Arcanist Gloves',
        req: '60',
        action: 'add',
        impact: { label: 'Full DPS', text: '▲ +2.7%', dir: 'up' }
      }
    ]
  },
  {
    id: 'base',
    label: 'Rare / magic bases',
    title: 'Bases',
    col: 'BASE STATS',
    search: 'Search bases',
    note: 'Pick a base, then craft: the item editor opens with the base’s implicit and defence ranges.',
    filters: [
      { label: 'Slot', options: ['Body Armour', 'Helmet', 'Gloves', 'Boots', 'Any slot'] },
      { label: 'Rarity to craft', options: ['Rare', 'Magic', 'Normal'] }
    ],
    checks: [{ label: 'Top-tier bases only', on: true }],
    rows: [
      { name: 'Vaal Regalia', rarity: 'normal', type: 'Body Armour', note: 'ES 171–197', req: '68', action: 'craft' },
      { name: 'Carnal Armour', rarity: 'normal', type: 'Body Armour', note: 'Evasion 353–388, ES 103+', req: '71', action: 'craft' },
      { name: 'Hubris Circlet', rarity: 'normal', type: 'Helmet', note: 'ES 80–92', req: '69', action: 'craft' },
      { name: 'Sorcerer Gloves', rarity: 'normal', type: 'Gloves', note: 'ES 49–57', req: '69', action: 'craft' }
    ]
  },
  {
    id: 'flask',
    label: 'Flasks',
    title: 'Flasks',
    col: 'EFFECT',
    search: 'Search flasks',
    note: 'Equip impact is shown against Flask 4 when every flask slot is full. A new flask starts active.',
    filters: [{ label: 'Kind', options: ['Any', 'Life', 'Mana', 'Utility'] }],
    checks: [{ label: 'Hide flasks already equipped', on: true }],
    rows: [
      {
        name: 'Divine Life Flask',
        rarity: 'normal',
        type: 'Flask 4',
        note: '2400 Life over 3.5s',
        req: '60',
        action: 'add',
        impact: { label: 'Life recovery', text: '▲ +2,400', dir: 'up' }
      },
      {
        name: 'Silver Flask',
        rarity: 'normal',
        type: 'Flask 4',
        note: 'Onslaught · 6s',
        req: '27',
        action: 'add',
        impact: { label: 'Cast Rate', text: '▲ +9.1%', dir: 'up' }
      },
      {
        name: 'Basalt Flask',
        rarity: 'normal',
        type: 'Flask 4',
        note: '20% more Armour',
        req: '27',
        action: 'add',
        impact: { label: 'Armour', text: '▲ +842', dir: 'up' }
      }
    ]
  },
  {
    id: 'jewel',
    label: 'Jewels',
    title: 'Jewels',
    col: 'NOTE',
    search: 'Search jewels',
    note: 'Jewels go into tree sockets; free sockets are listed first.',
    filters: [{ label: 'Kind', options: ['Any', 'Crimson', 'Viridian', 'Cobalt', 'Prismatic', 'Abyss', 'Timeless'] }],
    checks: [{ label: 'Only jewels that fit a free socket', on: true }],
    rows: [
      { name: 'Crimson Jewel', rarity: 'normal', type: 'Jewel', note: 'Strength-scaling rare jewel base', req: '—', action: 'craft' },
      { name: 'Viridian Jewel', rarity: 'normal', type: 'Jewel', note: 'Dexterity-scaling rare jewel base', req: '—', action: 'craft' },
      { name: 'Cobalt Jewel', rarity: 'normal', type: 'Jewel', note: 'Intelligence-scaling rare jewel base', req: '—', action: 'craft' }
    ]
  }
];

export type AffixRow = { tier: string; text: string; ilvl: number; source: string; weight: number; impact: string };

export const modPicker = {
  tiers: ['Any', 'T1 only'],
  sources: ['Any', 'Natural', 'Crafted', 'Influence'],
  sorts: ['Build impact', 'Tier', 'Weight'],
  rows: [
    { tier: 'T1', text: '+(110–119) to maximum Life', ilvl: 86, source: 'Natural', weight: 250, impact: '▲ Life +73' },
    { tier: 'T2', text: '+(100–109) to maximum Life', ilvl: 64, source: 'Natural', weight: 500, impact: '▲ Life +63' },
    { tier: 'T1', text: '+(46–51) to maximum Energy Shield', ilvl: 86, source: 'Natural', weight: 250, impact: '▲ ES +62' },
    { tier: 'T3', text: '+(80–99) to maximum Life', ilvl: 44, source: 'Natural', weight: 1000, impact: '▲ Life +53' },
    { tier: 'T1', text: '+(46–48)% to Cold Resistance', ilvl: 84, source: 'Natural', weight: 1000, impact: '▲ Cold Res +48%' },
    { tier: '—', text: '+(15–25) to maximum Life, +(15–25)% to Fire Resistance', ilvl: 1, source: 'Crafted', weight: 0, impact: '▲ Life +25' }
  ] satisfies AffixRow[]
};

export const gemDeltas: Record<string, number> = {
  'Increased Critical Damage Support': 11.8,
  'Elemental Focus Support': 9.4,
  'Controlled Destruction Support': 8.9,
  'Concentrated Effect Support': 7.1,
  'Faster Casting Support': 3.2
};
