// @vitest-environment jsdom

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import ItemTooltip from './ItemTooltip.svelte';
import type { ItemTip } from './fixtures';

afterEach(cleanup);

const wand: ItemTip = {
  influence: ['Shaper', 'Shaper'],
  title: ['Beast Chant', 'Pagan Wand'],
  lines: [
    { text: '', sep: true },
    { text: '^x7F7F7FWand', sep: false },
    { text: '^x7F7F7FQuality: ^x8888FF+20%', sep: false },
    { text: '^x7F7F7FPhysical Damage: ^x8888FF24-44 (49.3 DPS)', sep: false },
    { text: '^x7F7F7FCritical Strike Chance: ^78.00%', sep: false },
    { text: '', sep: true },
    { text: '^x7F7F7FRequires ^x7F7F7FLevel ^xDD002234^x7F7F7F, ^7118 ^x7F7F7FInt', sep: false },
    { text: '', sep: true },
    { text: '^xB8DAF110% increased Cast Speed', sep: false },
    { text: '', sep: true },
    { text: '^xDD0022Corrupted', sep: false },
    { text: '', sep: true }
  ]
};

const props = {
  item: 'Beast Chant',
  base: 'Pagan Wand',
  rarity: 'rare' as const,
  mods: ['10% increased Cast Speed'],
  impactTitle: 'UNEQUIP IMPACT',
  impact: []
};

describe('ItemTooltip', () => {
  it("draws Go's tooltip lines with their colours and separators", () => {
    const { getByRole } = render(ItemTooltip, { ...props, tooltip: wand });
    const body = getByRole('tooltip').querySelector('[data-item-lines]');
    expect(body, 'the tooltip body').not.toBeNull();
    const rows = [...body!.children].map((el) => (el.tagName === 'HR' ? '--' : el.textContent));
    expect(rows).toEqual([
      'Wand',
      'Quality: +20%',
      'Physical Damage: 24-44 (49.3 DPS)',
      'Critical Strike Chance: 8.00%',
      '--',
      'Requires Level 34, 118 Int',
      '--',
      '10% increased Cast Speed',
      '--',
      'Corrupted'
    ]);
    const level = [...body!.querySelectorAll('[data-pob-color]')].find((el) => el.textContent === '34') as HTMLElement;
    expect(level.style.getPropertyValue('--tone')).toBe('#DD0022');
  });

  it('shows both title lines and the influence header', () => {
    const { getByRole } = render(ItemTooltip, { ...props, tooltip: wand });
    const head = getByRole('tooltip').querySelector('.item-head')!;
    expect(head.textContent).toContain('Beast Chant');
    expect(head.textContent).toContain('Pagan Wand');
    expect(getByRole('tooltip').querySelector('[data-influence]')?.textContent).toBe('Shaper');
  });

  it('keeps the plain mod list when Go gave no tooltip', () => {
    const { getByRole } = render(ItemTooltip, props);
    expect(getByRole('tooltip').querySelector('[data-item-lines]')).toBeNull();
    expect(getByRole('tooltip').textContent).toContain('10% increased Cast Speed');
  });
});

describe('ItemTooltip flask what-if', () => {
  it('draws the "Effective flask stats:" lines above the toggle block', () => {
    const effective = ['^7Effective flask stats:', '^8Charge gain modifier: ^7+21%', '^8Flask uptime: ^70%^8 average, ^70%^8 minimum'];
    const { getByRole } = render(ItemTooltip, {
      ...props,
      impactTitle: 'ACTIVATING THIS FLASK WILL GIVE YOU',
      whatIf: { key: 'Flask 1', pending: false, blocks: [{ header: 'Activating this flask will give you:', lines: effective, sections: [] }] }
    });
    const block = getByRole('group', { name: 'Effective flask stats' });
    expect([...block.children].map((el) => el.textContent)).toEqual([
      'Effective flask stats:',
      'Charge gain modifier: +21%',
      'Flask uptime: 0% average, 0% minimum'
    ]);
    const header = getByRole('tooltip').querySelector('[data-block-header]')!;
    expect(block.compareDocumentPosition(header) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });
});
