// @vitest-environment jsdom

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';
import ComparePower from './ComparePower.svelte';
import type { PowerRow } from './fixtures';

afterEach(cleanup);

// ComparePowerReportListControl.lua:112-143
describe('the power report row tooltips', () => {
  it('asks for a Tree or Item row, not a Config one', async () => {
    const rows: PowerRow[] = [
      { cat: 'tree', what: 'Take Heart of Ice', detail: '', tone: 'text', impact: [10, 0, 0] },
      { cat: 'config', what: 'Is the enemy Shocked?', detail: '', tone: 'text', impact: [5, 0, 0] }
    ];
    const asked: string[] = [];
    const { getByText, container } = render(ComparePower, {
      power: rows,
      metric: 'dps',
      a: 100,
      b: 110,
      onmetric: () => {},
      onapply: () => {},
      rowtip: (r: PowerRow) => (
        asked.push(r.what), Promise.resolve({ influence: [], title: ['Heart of Ice'], lines: [{ text: '25% increased Cold Damage', sep: false }] })
      )
    });
    await fireEvent.mouseEnter(getByText('Is the enemy Shocked?'));
    await fireEvent.mouseEnter(getByText('Take Heart of Ice'));
    await waitFor(() => expect(container.ownerDocument.body.textContent).toContain('25% increased Cold Damage'));
    expect(asked).toEqual(['Take Heart of Ice']);
  });
});
