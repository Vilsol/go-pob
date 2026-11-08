// @vitest-environment jsdom
// ItemsTab.lua:501-503

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import DisplayItemControls from './DisplayItemControls.svelte';
import type { DisplayEdit, DisplayInfo } from './items';

afterEach(cleanup);

const info = (sockets: string): DisplayInfo => ({
  quality: 0,
  hasQuality: false,
  variants: [],
  variant: 0,
  altVariants: [],
  sockets,
  socketLimit: 4,
  defaultColor: 'R',
  canInfluence: false,
  influence: [],
  showsCatalyst: false,
  catalyst: 0,
  catalystQuality: 0,
  mods: []
});

describe('DisplayItemControls sockets', () => {
  it('builds a socket edit on one Go has not answered yet, and on Go once it has', async () => {
    const edits: DisplayEdit[] = [];
    const { getByRole, rerender } = render(DisplayItemControls, { info: info('B-B-B-G'), onedit: (e: DisplayEdit) => edits.push(e) });
    const colour = getByRole('combobox', { name: 'Socket 4 colour' }) as HTMLSelectElement;
    colour.value = 'R';
    await fireEvent.change(colour);
    await fireEvent.click(getByRole('checkbox', { name: 'Link sockets 3 and 4' }));
    expect(edits).toEqual([
      { kind: 'sockets', text: 'B-B-B-R' },
      { kind: 'sockets', text: 'B-B-B R' }
    ]);

    await rerender({ info: info('B-B-B R') });
    await fireEvent.click(getByRole('checkbox', { name: 'Link sockets 1 and 2' }));
    expect(edits.at(-1)).toEqual({ kind: 'sockets', text: 'B B-B R' });
  });
});
