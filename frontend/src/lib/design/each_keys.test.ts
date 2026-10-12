// @vitest-environment jsdom

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import CompareSkills from './CompareSkills.svelte';
import WhatIfTooltip from './WhatIfTooltip.svelte';

afterEach(cleanup);

describe('keyed lists with repeated entries', () => {
  it('Compare › Skills draws a gem that appears twice in a group', () => {
    const gem = { gem: 'EnemyExplode', status: 'same' as const, a: '1/0', b: '1/0' };
    const { getAllByRole } = render(CompareSkills, {
      both: [{ name: 'Main', where: 'Body Armour', gems: [gem, { ...gem }] }],
      onlyA: [],
      onlyB: []
    });
    expect(getAllByRole('cell').filter((c) => c.textContent?.includes('EnemyExplode'))).toHaveLength(2);
  });

  it('a what-if tooltip draws a repeated mod and info line', () => {
    const { container } = render(WhatIfTooltip, {
      title: 'Node',
      action: '',
      sections: [],
      mods: ['+10 to Strength', '+10 to Strength'],
      info: ['Level 90', 'Level 90']
    });
    expect(container.textContent?.match(/\+10 to Strength/g)).toHaveLength(2);
    expect(container.textContent?.match(/Level 90/g)).toHaveLength(2);
  });
});
