import { describe, expect, it } from 'vitest';
import { searchResults } from './tree2_search';

// the expected ids are PoB-PoE2's: prepSearch (PassiveTreeView.lua:862-876), DoesNodeMatchSearchParams
// (1528-1618) and string:matchOrPattern (Common.lua:1009-1050) run under LuaJIT over these nodes
const nodes = [
  { id: 1, type: 'Notable', name: 'Heart of Flame', stats: ['20% increased Fire Damage', '10% increased Area of Effect'] },
  { id: 2, type: 'Normal', name: 'Life', stats: ['+10 to maximum Life'] },
  { id: 3, type: 'Keystone', name: 'Chaos Inoculation', stats: ['Maximum Life becomes 1'] },
  { id: 4, type: 'ClassStart', name: 'Witch', stats: [] },
  { id: 5, type: 'Socket', name: 'Jewel Socket', stats: [] }
];

const cases: [string, number[]][] = [
  ['life', [2, 3]],
  ['LIFE', [2, 3]],
  ['fire area', [1]],
  ['"maximum life"', [2, 3]],
  ['keystone', [3]],
  ['socket', [5]],
  ['witch', []],
  ['(fire|cold).damage', [1]],
  ['increased.(fire|area)', [1]],
  ['[', []],
  ['%d+ to', [2, 3]],
  ['flame inoculation', []],
  ['', []],
  ['  ', []],
  ['^life', [2]],
  ['life$', [2]],
  ['chaos 1', [3]]
];

describe('the PoE2 tree search (PassiveTreeView.lua:858-882, 1528-1618)', () => {
  it.each(cases)('%j matches %j', (query, want) => {
    expect([...searchResults(query, nodes)]).toEqual(want);
  });
});
