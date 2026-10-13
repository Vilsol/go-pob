import { describe, expect, it } from 'vitest';
import { breakdownStep } from './breakdown_lines';

describe('a breakdown line as a step', () => {
  it('splits the operator, the value and its note; keeps anything else as prose', () => {
    expect(breakdownStep('1180 (base)')).toEqual({ value: '1180', note: '(base)' });
    expect(breakdownStep('x 1.95 ^8(increased/reduced)')).toEqual({ op: '×', value: '1.95', note: '^8(increased/reduced)' });
    expect(breakdownStep('= 2301')).toEqual({ op: '=', value: '2301' });
    expect(breakdownStep('3.8m (base radius)')).toEqual({ value: '3.8m', note: '(base radius)' });
    expect(breakdownStep('+ 12 to 40 (added)')).toEqual({ op: '+', value: '12 to 40', note: '(added)' });
    expect(breakdownStep('- 5% (penalty)')).toEqual({ op: '−', value: '5%', note: '(penalty)' });
    expect(breakdownStep('Next 0.1m breakpoint: 6% increased AoE')).toEqual({ prose: 'Next 0.1m breakpoint: 6% increased AoE' });
    expect(breakdownStep('^8Physical:')).toEqual({ prose: '^8Physical:' });
  });
});
