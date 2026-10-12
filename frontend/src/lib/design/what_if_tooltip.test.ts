// @vitest-environment jsdom

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import WhatIfTooltip from './WhatIfTooltip.svelte';

afterEach(cleanup);

describe('the what-if tooltip', () => {
  // Tooltip.lua:95
  it('draws each line of a multi-line mod and info entry on its own', () => {
    const { container } = render(WhatIfTooltip, {
      title: 'Chaos Inoculation',
      action: '',
      sections: [],
      mods: ['Maximum Life becomes 1, Immune to Chaos Damage\nNo Life Recovery', '+1 to Level'],
      info: ['Witch\nOccultist']
    });
    expect([...container.querySelectorAll('.text-magic > span')].map((s) => s.textContent)).toEqual([
      'Maximum Life becomes 1, Immune to Chaos Damage',
      'No Life Recovery',
      '+1 to Level'
    ]);
    expect([...container.querySelectorAll('[aria-label="Details"] > span')].map((s) => s.textContent)).toEqual(['Witch', 'Occultist']);
  });

  it("colours a stat difference's label as the sidebar does", () => {
    const { container } = render(WhatIfTooltip, {
      title: 'Node',
      action: 'Allocating',
      sections: [
        {
          lines: [
            { label: 'Total Life', text: '+10', dir: 'up', color: '#E05030' },
            { label: 'X', text: '+1', dir: 'up' }
          ]
        }
      ]
    });
    const labels = [...container.querySelectorAll('[data-line] > span:first-child')] as HTMLElement[];
    expect(labels.map((l) => l.style.getPropertyValue('--tone'))).toEqual(['#E05030', '']);
  });
});
