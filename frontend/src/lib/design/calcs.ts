import { colorCodes } from '$lib/display/colors';
import type { Tone } from './tone';
import type { CalcRow, CalcSectionFixture } from './fixtures';

// CalcSections.lua:105-1731, Global.lua:70-77
// CalcSectionControl.lua:193, 445-467
const SECTION_TONES: Record<string, Tone> = {
  [colorCodes.OFFENCE]: 'off',
  [colorCodes.DEFENCE]: 'def',
  [colorCodes.LIFE]: 'life',
  [colorCodes.MANA]: 'mana',
  [colorCodes.ES]: 'es',
  [colorCodes.WARD]: 'ward',
  [colorCodes.EVASION]: 'evasion'
};

export const sectionTone = (s: Pick<CalcSectionFixture, 'applicable' | 'color'>): Tone => (!s.applicable ? 'na' : (SECTION_TONES[s.color] ?? 'phys'));

// CalcSections.lua:112-115, 516, Global.lua:77
const DAMAGE_TONES: Record<string, Tone> = { Physical: 'phys', Fire: 'fire', Cold: 'cold', Lightning: 'light', Chaos: 'chaos' };

export const damageTone = (head: string): Tone | undefined => DAMAGE_TONES[head.replace(/:$/, '')];

export const rowKey = (section: string, block: number, row: number) => `${section}:${block}:${row}`;

export const isHeaderRow = (r: CalcRow) => r.label === '' && r.cells.every((c) => !c.bd && !c.mods);
