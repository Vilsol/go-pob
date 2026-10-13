export type SavedBuild = {
  name: string;
  className: string;
  ascendancy: string;
  level: number;
  skill: string;
  edited: number;
  folder: string;
  dps: number;
  ehp: number;
  life: number;
  es: number;
  version: string;
};

export const buildList = {
  gameVersion: '3.29',
  classes: ['All', 'Witch', 'Shadow', 'Marauder', 'Ranger', 'Duelist', 'Templar', 'Scion'],
  folders: [
    'League starters',
    'League starters/Necromancer',
    'League starters/Necromancer/Budget',
    'Bossers',
    'Bossers/Uber',
    'Bossers/Uber/Pinnacle',
    'Imported',
    'Imported/pobb.in',
    'Imported/pobb.in/Friends'
  ],
  builds: [
    ['Spectre Necro – mapping', 'Witch', 'Necromancer', 94, 'Raise Spectre', 2, 'League starters/Necromancer', 2.06e6, 42.1e3, 5120, 1284, '3.29'],
    ['Spectre Necro – budget', 'Witch', 'Necromancer', 88, 'Raise Spectre', 1440, 'League starters/Necromancer/Budget', 1.51e6, 35.9e3, 4410, 1720, '3.29'],
    ['Winter Orb Elementalist', 'Witch', 'Elementalist', 100, 'Winter Orb', 43200, 'League starters', 1.18e6, 38.4e3, 4880, 2210, '3.28'],
    ['Blade Vortex Occultist', 'Witch', 'Occultist', 92, 'Blade Vortex', 4320, 'Bossers/Uber', 2.66e6, 31.0e3, 4212, 3890, '3.29'],
    ['Stormblast Mine Saboteur', 'Shadow', 'Saboteur', 90, 'Stormblast Mine', 10080, 'Bossers', 124.0e6, 178.9e3, 1, 11840, '3.29'],
    ['Lightning Strike Juggernaut', 'Marauder', 'Juggernaut', 96, 'Lightning Strike', 10080, 'Imported', 19.9e6, 14.6e3, 2843, 0, '3.27'],
    ['Ice Spear Totems', 'Templar', 'Hierophant', 91, 'Ice Spear', 20160, 'Imported/pobb.in', 605.6e3, 48.9e3, 3462, 540, '3.28'],
    ['Scourge Arrow Pathfinder', 'Ranger', 'Pathfinder', 93, 'Scourge Arrow', 43200, 'Imported/pobb.in/Friends', 2.92e6, 96.9e3, 4953, 0, '3.29']
  ].map(
    ([name, className, ascendancy, level, skill, edited, folder, dps, ehp, life, es, version]) =>
      ({ name, className, ascendancy, level, skill, edited, folder, dps, ehp, life, es, version }) as SavedBuild
  )
};

export type SettingKind = 'range' | 'check' | 'select' | 'number' | 'color' | 'info' | 'button';
export type Setting = {
  id: string;
  label: string;
  kind: SettingKind;
  value: string | number | boolean;
  hint?: string;
  min?: number;
  max?: number;
  step?: number;
  unit?: string;
  opts?: string[];
  danger?: boolean;
  onRelease?: boolean;
  confirm?: string;
};
export type SettingsSection = { id: string; label: string; rows: Setting[] };

export const settingsSections: SettingsSection[] = [
  {
    id: 'general',
    label: 'General',
    rows: [
      { id: 'fontScaling', label: 'Font scaling', kind: 'range', value: 12, hint: 'Base text size for the whole app', min: 6, max: 18, step: 0.5, unit: 'pt' },
      {
        id: 'theme',
        label: 'Theme',
        kind: 'select',
        value: 'Dark',
        hint: 'Light and high-contrast variants',
        opts: ['Dark', 'Light', 'High contrast', 'Follow system']
      },
      { id: 'animations', label: 'Show animations', kind: 'check', value: true, hint: 'Off also honours the system “reduce motion” setting' },
      { id: 'titleBuild', label: 'Show build name in window title', kind: 'check', value: true, hint: 'Browser tab title becomes “<build> · go-pob”' },
      { id: 'warnings', label: 'Show build issues', kind: 'check', value: true, hint: 'The ⚠ count in the top bar' },
      { id: 'devMode', label: 'Dev mode', kind: 'check', value: false, hint: 'Extra diagnostics in Calcs and the logs' }
    ]
  },
  {
    id: 'numbers',
    label: 'Numbers',
    rows: [
      { id: 'thousands', label: 'Show thousands separators', kind: 'check', value: true },
      { id: 'thousandsSep', label: 'Thousands separator', kind: 'select', value: ',', opts: [',', '.', 'space', '’'] },
      { id: 'decimalSep', label: 'Decimal separator', kind: 'select', value: '.', opts: ['.', ','] },
      {
        id: 'compact',
        label: 'Compact large numbers (e.g. 12.3K)',
        kind: 'check',
        value: false,
        hint: 'Sidebar and tables; tooltips always show the full value'
      },
      { id: 'posColor', label: 'Colour for positive changes', kind: 'color', value: '#6cb4ff' },
      { id: 'negColor', label: 'Colour for negative changes', kind: 'color', value: '#ff9b52' }
    ]
  },
  {
    id: 'tree',
    label: 'Passive tree',
    rows: [
      { id: 'zoomSensitivity', label: 'Zoom sensitivity', kind: 'range', value: 5, hint: 'Mouse wheel and pinch', min: 1, max: 10, step: 0.1, unit: '' },
      {
        id: 'nodePower',
        label: 'Node power colours',
        kind: 'select',
        value: 'Red & Blue',
        hint: 'Heat-map colours when node power is shown',
        opts: ['Red & Blue', 'Red & Green', 'Green & Blue']
      },
      { id: 'highlight', label: 'Highlight colour for searched nodes', kind: 'color', value: '#e0a84a' },
      { id: 'edgeCircles', label: 'Show search circles at viewport edge', kind: 'check', value: true, hint: 'Points toward matches that are off screen' }
    ]
  },
  {
    id: 'items',
    label: 'Items and new builds',
    rows: [
      { id: 'defLevel', label: 'Default character level', kind: 'number', value: 1, hint: 'Level for new builds and imports without one', min: 1, max: 100 },
      { id: 'gemQuality', label: 'Default gem quality', kind: 'number', value: 0, hint: 'Quality given to newly added gems', min: 0, max: 23 },
      {
        id: 'affixQuality',
        label: 'Default item affix quality',
        kind: 'range',
        value: 0.5,
        hint: 'Where in its range a new affix rolls (0 = min, 1 = max)',
        min: 0,
        max: 1,
        step: 0.05,
        unit: ''
      },
      { id: 'affixSliders', label: 'Show all item affix sliders', kind: 'check', value: false, hint: 'Item editor shows a roll slider on every affix' },
      {
        id: 'slotTooltips',
        label: 'Tooltips only for affected slots',
        kind: 'check',
        value: false,
        hint: 'Equip-impact tooltips compare against the target slot only'
      },
      { id: 'unsupported', label: 'Show tooltip for unsupported mods', kind: 'check', value: true, hint: 'Marks lines the calculator ignores' }
    ]
  },
  {
    id: 'data',
    label: 'Data and storage',
    rows: [
      { id: 'checkData', label: 'Check for newer game data', kind: 'button', value: 'Check now', hint: 'Downloads a newer data module if one exists' },
      { id: 'cache', label: 'Cached data', kind: 'info', value: '38.4 MB', hint: 'Game data kept in browser storage' },
      {
        id: 'clearCache',
        label: 'Clear cached data',
        kind: 'button',
        value: 'Clear…',
        hint: 'Builds are not affected; data downloads again',
        danger: true,
        confirm: 'Clear the cached game data? Builds are kept.'
      },
      { id: 'exportAll', label: 'Export all builds', kind: 'button', value: 'Export .zip', hint: 'Builds live only in this browser' }
    ]
  }
];

export const settingsSample = 2060418;

export const partySample = {
  member: { id: 'curse', name: 'Curse support (imported)', className: 'Witch', ascendancy: 'Occultist', level: 92 },
  buffers: {
    Aura: [
      'Wrath',
      '74',
      '26|Skill:Wrath|PhysicalDamageGainAsLightning|BASE|0|0|',
      '---',
      'Discipline',
      '74',
      '338|Skill:Discipline|EnergyShield|BASE|0|0|',
      '---'
    ].join('\n'),
    Curse: [
      '1',
      'Elemental Weakness',
      '21',
      'false',
      '-46|Skill:Elemental Weakness|ElementalResist|BASE|0|0|',
      '---',
      'Temporal Chains',
      '20',
      'false',
      '-17|Skill:Temporal Chains|ActionSpeed|INC|0|0|',
      '---'
    ].join('\n'),
    Warcry: '',
    Link: ['Flame Link', '0', '10|Skill:Flame Link|LifeAsFireDamage|BASE|0|0|', '---'].join('\n'),
    EnemyConditions: ['Condition:Cursed', 'Condition:Hindered'].join('\n'),
    EnemyMods: ['LightningExposure', '-10|Skill:Wave of Conviction|LightningResist|BASE|0|0|'].join('\n')
  } as Record<string, string>,
  from: {
    '@PartyMemberStats': 'b',
    Wrath: 'b',
    Discipline: 'b',
    'Flame Link': 'b',
    'Elemental Weakness': 'curse',
    'Temporal Chains': 'curse',
    Cursed: 'curse',
    Hindered: 'curse',
    LightningExposure: 'curse'
  } as Record<string, string>,
  impact: [
    ['Full DPS', '▲ +38.2%', 'up'],
    ['Fire / Cold / Lightning Res', '▲ capped (+43%)', 'up'],
    ['Energy Shield', '▲ +338', 'up']
  ] as [string, string, 'up' | 'down'][]
};

export const timelessSample = {
  jewels: [
    ['Glorious Vanity', 'CLASSIC', '100–8000', ['Doryani (Corrupted Soul)', 'Xibaqua (Divine Flesh)', 'Ahuana (Immortal Ambition)']],
    ['Lethal Pride', 'CLASSIC', '10000–18000', ['Kaom (Strength of Blood)', 'Rakiata (Tempered by War)', 'Akoya (Chainbreaker)']],
    ['Brutal Restraint', 'CLASSIC', '500–8000', ['Asenath (Dance with Death)', 'Nasima (Second Sight)', 'Balbala (The Traitor)']],
    ['Militant Faith', 'CLASSIC', '2000–10000', ['Avarius (Power of Purpose)', 'Dominus (Inner Conviction)', 'Maxarius (Transcendence)']],
    ['Elegant Hubris', 'CLASSIC', '2000–160000', ['Cadiro (Supreme Decadence)', 'Victario (Supreme Grandstanding)', 'Caspiro (Supreme Ostentation)']],
    ['Heroic Tragedy', 'CLASSIC', '100–8000', ['Vorana (Black Scythe Training)', 'Uhtred (Celestial Mathematics)', 'Medved (The Unbreaking Circle)']]
  ].map(([name, kind, seeds, conquerors]) => ({ name, kind, seeds, conquerors })) as { name: string; kind: string; seeds: string; conquerors: string[] }[],
  devotion: [
    'Any',
    'Minion Attack and Cast Speed',
    'Totem Damage',
    'Brand Damage',
    'Channelling Damage',
    'Area Damage',
    'Elemental Damage',
    'Elemental Resistances'
  ],
  requirements: [
    ['#% increased maximum Life', 10, 1],
    ['#% increased Damage', 6, 0],
    ['#% increased maximum Energy Shield', 3, 0],
    ['+# to Devotion (small passives)', 1, 0]
  ] as [string, number, number][],
  stats: [
    '#% increased maximum Life',
    '#% increased Damage',
    '#% increased maximum Energy Shield',
    '+# to Devotion (small passives)',
    '#% increased Armour',
    '#% increased Evasion Rating',
    '#% increased Attack Speed',
    '#% increased Cast Speed',
    '#% increased Critical Strike Chance'
  ],
  ranks: [
    ['weight', 'Total weight', 'required × weight + secondary weight'],
    ['whatif', 'What-if on a sidebar stat', 'recalculates the build with each candidate socketed'],
    ['cost', 'Weight pp', 'total weight divided by points to reach the socket']
  ] as [string, string, string][],
  rankStats: ['Full DPS', 'Combined DPS', 'Effective Hit Pool', 'Total Life'],
  results: [
    [7342, 0, 44, '+9.8%'],
    [3016, 0, 41, '+9.1%'],
    [8811, 1, 39, '+7.4%'],
    [5129, 2, 37, '+6.9%'],
    [2270, 1, 36, '+6.2%'],
    [9403, 2, 33, '+4.8%']
  ] as [number, number, number, string][]
};

export type Character = { name: string; className: string; ascendancy: string; level: number; league: string };

export const characterImport = {
  realms: ['PC', 'Xbox', 'Sony'],
  account: 'Exile#1234',
  characters: [
    { name: 'Spectrewitch', className: 'Witch', ascendancy: 'Necromancer', level: 94, league: 'Standard' },
    { name: 'VortexHag', className: 'Witch', ascendancy: 'Occultist', level: 92, league: 'Standard' },
    { name: 'MineShadow', className: 'Shadow', ascendancy: 'Saboteur', level: 90, league: 'Standard' },
    { name: 'LSJugg', className: 'Marauder', ascendancy: 'Juggernaut', level: 96, league: 'Hardcore' }
  ] as Character[]
};
