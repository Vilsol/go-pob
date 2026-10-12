import { colorCodes } from '$lib/display/colors';
import type { Tone } from './tone';
import type { CalcRow, CalcSectionFixture } from './fixtures';

export const sectionTone = (s: CalcSectionFixture): Tone =>
  !s.applicable ? 'na' : s.color === colorCodes.OFFENCE ? 'off' : s.color === colorCodes.DEFENCE ? 'def' : 'phys';

export const rowKey = (section: string, block: number, row: number) => `${section}:${block}:${row}`;

export const isHeaderRow = (r: CalcRow) => r.label === '' && r.cells.every((c) => !c.bd && !c.mods);
