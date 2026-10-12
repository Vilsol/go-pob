import { describe, expect, it } from 'vitest';
import { statGauge } from './stat_gauge';

describe('statGauge', () => {
  it('caps a resist at its max (value + missing), a block at <stat>Max, evade at the Misc cap', () => {
    expect(statGauge('ColdResist', { ColdResist: 70, MissingColdResist: 6 })).toEqual({ pct: 70, cap: 76, tone: 'cold' });
    expect(statGauge('ChaosResist', { ChaosResist: -30, MissingChaosResist: 105 })).toEqual({ pct: 0, cap: 75, tone: 'chaos' });
    expect(statGauge('EffectiveBlockChance', { EffectiveBlockChance: 40, BlockChanceMax: 78 })?.cap).toBe(78);
    expect(statGauge('MeleeEvadeChance', { MeleeEvadeChance: 40 })?.cap).toBe(95);
  });

  it('caps a chance with no max output only when over it', () => {
    expect(statGauge('SpellDodgeChance', { SpellDodgeChance: 55, SpellDodgeChanceOverCap: 0 })?.cap).toBeUndefined();
    expect(statGauge('SpellDodgeChance', { SpellDodgeChance: 75, SpellDodgeChanceOverCap: 4 })?.cap).toBe(75);
    expect(statGauge('SpellDodgeChance', { SpellDodgeChance: 55 }, ['Base: 0%', 'Max: 75%'])?.cap).toBe(75);
  });

  it('shares a pool with its reserved band, and measures a cost against its pool', () => {
    expect(statGauge('ManaUnreserved', { ManaUnreserved: 38, ManaUnreservedPercent: 4, ManaReservedPercent: 96 })).toEqual({ pct: 4, res: 96, tone: 'mana' });
    expect(statGauge('ManaCost', { ManaCost: 42, ManaUnreserved: 38 })).toEqual({ pct: (42 / 38) * 100, warn: true, tone: 'mana' });
    expect(statGauge('ESCost', { ESCost: 10, EnergyShield: 0 })?.pct).toBe(Infinity);
    expect(statGauge('ManaPerSecondCost', { ManaPerSecondCost: 90, ManaUnreserved: 38 })).toBeUndefined();
    expect(statGauge('SoulCost', { SoulCost: 0 })).toBeUndefined();
  });

  it('leaves unbounded stats alone', () => {
    expect(statGauge('Life', { Life: 5000 })).toBeUndefined();
    expect(statGauge('FireResist', {})).toBeUndefined();
  });
});
