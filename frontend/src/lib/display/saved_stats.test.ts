// Build.lua:1014-1061
import { describe, expect, it } from 'vitest';
import { savedStats, luaToString } from './saved_stats';

describe('savedStats', () => {
  it('writes the shown stats that have a value, the Full DPS skills, then the extra save stats', () => {
    const s = savedStats({
      Output: { Life: 1234, EnergyShield: 56.5, PowerCharges: 3, PowerChargesMax: 4, ActiveMinionLimit: 2 },
      SkillFlags: {},
      SkillDPS: [{ Name: 'Fireball', DPS: 10.5, Count: 2, Trigger: '', SkillPart: '', Source: 'Item' } as never]
    });
    expect(s.player).toContainEqual({ stat: 'Life', value: '1234' });
    expect(s.player).toContainEqual({ stat: 'EnergyShield', value: '56.5' });
    expect(s.fullDPS).toEqual([{ stat: '2x Fireball', value: '21', skillPart: '', source: 'Item' }]);
    // Build.lua:1046-1051
    const tail = s.player.slice(-3);
    expect(tail).toEqual([
      { stat: 'PowerCharges', value: '3' },
      { stat: 'PowerChargesMax', value: '4' },
      { stat: 'ActiveMinionLimit', value: '2' }
    ]);
    expect(s.minion).toEqual([]);
  });

  it('writes the minion stats when there is a minion', () => {
    const s = savedStats({ Output: {}, SkillFlags: {}, Minion: { Output: { Life: 500 } } });
    expect(s.minion).toContainEqual({ stat: 'Life', value: '500' });
  });
});

describe('luaToString', () => {
  it('formats numbers as %.14g', () => {
    expect(luaToString(1395701.3538564)).toBe('1395701.3538564');
    expect(luaToString(0.1 + 0.2)).toBe('0.3');
    expect(luaToString(100)).toBe('100');
    expect(luaToString(1e21)).toBe('1e+21');
    expect(luaToString(Infinity)).toBe('inf');
    expect(luaToString(-Infinity)).toBe('-inf');
  });
});
