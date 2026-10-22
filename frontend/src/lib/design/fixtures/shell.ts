import type { Tone } from '../tone';
import type { StepDelta, StepKind } from '$lib/go/step_labels';

export type PageTab = { label: string; href: string; key: string };

export const pageTabs: PageTab[] = ['Tree', 'Skills', 'Items', 'Calcs', 'Config', 'Notes', 'Party', 'Compare'].map((label, i) => ({
  label,
  href: `#${label.toLowerCase()}`,
  key: String(i + 1)
}));

export type HistoryEntry = { action: string; page: string; when: string; kind?: StepKind; delta?: StepDelta[] };

export const undoHistory: HistoryEntry[] = [
  {
    action: 'Allocated Heart and Soul (+2 path nodes)',
    page: 'Tree',
    when: 'just now',
    kind: 'tree',
    delta: [
      { label: 'Life', value: '+212 (+4.1%)', better: true },
      { label: 'Life Regen', value: '+18.4', better: true },
      { label: 'Mana', value: '-35', better: false }
    ]
  },
  {
    action: 'Activated Flask 2 (Quicksilver Flask)',
    page: 'Items',
    when: '1 min ago',
    kind: 'item',
    delta: [{ label: 'Movement Speed Modifier', value: '+40%', better: true }]
  },
  { action: 'Set the main active skill to Raise Spectre', page: 'Sidebar', when: '2 min ago', kind: 'gem', delta: [] },
  {
    action: 'Switched item set → Endgame',
    page: 'Items',
    when: '4 min ago',
    kind: 'set',
    delta: [
      { label: 'Total DPS', value: '+48,210 (+31.0%)', better: true },
      { label: 'Spectre DPS', value: '+12,004', better: true, section: 'Minion' }
    ]
  },
  { action: 'Socketed Glorious Vanity in Jewel Socket', page: 'Tree', when: '6 min ago', kind: 'item' },
  {
    action: 'Set Is the enemy a Boss to Pinnacle Boss',
    page: 'Config',
    when: '9 min ago',
    kind: 'config',
    delta: [{ label: 'Total DPS', value: '-9,120', better: false }]
  }
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
