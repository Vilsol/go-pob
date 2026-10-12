import type { Tone } from '../tone';

export type PageTab = { label: string; href: string; key: string };

export const pageTabs: PageTab[] = ['Tree', 'Skills', 'Items', 'Calcs', 'Config', 'Notes', 'Party', 'Compare'].map((label, i) => ({
  label,
  href: `#${label.toLowerCase()}`,
  key: String(i + 1)
}));

export type HistoryEntry = { action: string; page: string; when: string };

export const undoHistory: HistoryEntry[] = [
  { action: 'allocate Heart and Soul', page: 'Tree', when: 'just now' },
  { action: 'toggle Flask 2 (Quicksilver Flask)', page: 'Items', when: '1 min ago' },
  { action: 'main skill → Raise Spectre', page: 'Sidebar', when: '2 min ago' },
  { action: 'switch item set → Endgame', page: 'Items', when: '4 min ago' },
  { action: 'socket Glorious Vanity in Socket #2', page: 'Items', when: '6 min ago' },
  { action: 'set Config → Pinnacle boss', page: 'Config', when: '9 min ago' }
];

export const loadouts = ['Endgame', 'Leveling (act 5)', 'Budget'];

export type LogLine = { time: string; level: 'info' | 'warn' | 'error'; src: string; msg: string };

export const appLog: LogLine[] = [
  { time: '12:04:01', level: 'info', src: 'data', msg: 'Game data 3.29 loaded from cache (38.4 MB, 412 ms)' },
  { time: '12:04:01', level: 'info', src: 'worker', msg: 'Calculation worker started (wasm 21.3 MB)' },
  { time: '12:04:02', level: 'info', src: 'build', msg: 'Opened the build from the corpus' },
  { time: '12:06:41', level: 'warn', src: 'import', msg: 'Item “Doom Visor”: unknown mod line kept as custom' },
  { time: '12:11:15', level: 'error', src: 'save', msg: 'Save failed: browser storage quota exceeded (retrying after cleanup)' },
  { time: '12:11:16', level: 'info', src: 'save', msg: 'Saved after removing 2 old autosaves' }
];

export type Breakdown = { formula: string; groups: { title?: string; rows: { src: string; val: string; tone?: Tone }[] }[] };

export const breakdowns: Record<string, Breakdown> = {
  'Total Life': {
    formula: 'base + flat, × (1 + increased)',
    groups: [
      { title: 'ITEMS', rows: [{ src: 'Body armour', val: '+98' }] },
      { title: 'PASSIVES', rows: [{ src: 'Heart and Soul', val: '+8%' }] }
    ]
  },
  Strength: {
    formula: 'class base + flat, × (1 + increased)',
    groups: [
      {
        title: 'SOURCES',
        rows: [
          { src: 'Passive tree', val: '+60' },
          { src: 'Items', val: '+72' }
        ]
      }
    ]
  }
};
