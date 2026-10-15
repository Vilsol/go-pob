import { describe, expect, it } from 'vitest';
import { calcSections } from '$lib/calcs/calc_sections';
import { sidebarBreakdown } from './sidebar_breakdown';

const shape = (key: string, modNames?: string[], ignored?: string[]) =>
  sidebarBreakdown(calcSections, key, modNames, ignored).map((c) =>
    c.breakdown !== undefined ? `bd:${c.breakdown}${c.gearOnly ? ' gear' : ''}` : `${c.label ?? ''}|${[c.modName].flat().join(',')}|${c.modType ?? ''}`
  );

// Build.lua:1764-1869
describe('the sidebar breakdown as Build.lua assembles it', () => {
  it('Total Life: the Total box, then its modifier rows less the ignored ones', () => {
    expect(shape('Life', ['Life'], ['Base from Gear', 'Inc. from Tree'])).toEqual([
      'bd:Life',
      'Total Base|Life|BASE',
      'Total Increased|Life|INC',
      'Total More|Life|MORE'
    ]);
  });

  it('Energy Shield skips the gear-only box, and a row named twice shows once', () => {
    expect(shape('EnergyShield', ['EnergyShield', 'Defences'], ['Base from Gear', 'Inc. from Tree'])).toEqual([
      'bd:EnergyShield',
      'Global Base|EnergyShield|BASE',
      'Total Increased|EnergyShield,Defences|INC',
      'Total More|EnergyShield,Defences|MORE'
    ]);
  });

  it('a key no Calcs box names: the breakdown, then the modNames as one table', () => {
    expect(shape('', ['SpellSuppressionChance'])).toEqual(['bd:', '|SpellSuppressionChance|']);
    expect(shape('NoSuchBreakdown')).toEqual(['bd:NoSuchBreakdown']);
  });
});
