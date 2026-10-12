// @vitest-environment jsdom

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import WhatIfTooltip from './WhatIfTooltip.svelte';

afterEach(cleanup);

const line = (label: string) => ({ label, text: '▲ +1', dir: 'up' as const });

describe('WhatIfTooltip rows', () => {
  it('shades every second line of each section', () => {
    const { container } = render(WhatIfTooltip, {
      title: 'Node',
      action: 'Allocating (1 point)',
      sections: [
        { title: 'Minion', lines: [line('a'), line('b'), line('c')] },
        { title: 'Player', lines: [line('d'), line('e')] }
      ]
    });
    const shaded = [...container.querySelectorAll('[data-line]')].map((el) => el.classList.contains('bg-bar'));
    expect(shaded).toEqual([false, true, false, false, true]);
  });
});
