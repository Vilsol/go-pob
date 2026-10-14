// @vitest-environment jsdom

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import type { StatLine } from '$lib/display/stat_list';
import type { CalcsFixture } from './fixtures';
import { showStatBars } from '$lib/display/appearance';
import Sidebar from './Sidebar.svelte';
import CalcsPage from './CalcsPage.svelte';

afterEach(() => {
  cleanup();
  showStatBars.set(true);
});

const gauge = { pct: 60, cap: 75 };
const line = { label: 'Fire Resistance', labelColor: '', value: '60%', valueColor: '', gauge } as StatLine;
const sidebar = () => render(Sidebar, { props: { sidebar: { sections: [{ groups: [[line]] }], warnings: [] }, socketGroups: ['Group'], mainSocketGroup: 0 } });

const calcs: CalcsFixture = {
  sections: [
    {
      id: 'resist',
      title: 'Resists',
      wide: false,
      col: 1,
      color: '',
      applicable: true,
      blocks: [{ label: 'Resists', rows: [{ label: 'Fire Resistance', cells: [{ text: '60%' }], gauge }] }]
    }
  ],
  breakdowns: {}
};
const calcsPage = () => render(CalcsPage, { props: { calcs, socketGroups: ['Group'], mainSocketGroup: 0, onmaingroup: () => {}, oncontrol: () => {} } });

describe('Show stat bars', () => {
  it('draws the bars by default', () => {
    expect(sidebar().container.querySelectorAll('[data-gauge]').length).toBe(1);
    cleanup();
    expect(calcsPage().container.querySelectorAll('[data-gauge]').length).toBe(1);
  });

  it('hides them in the sidebar and on the Calcs page when off', () => {
    showStatBars.set(false);
    const s = sidebar();
    expect(s.getByText('Fire Resistance')).toBeTruthy();
    expect(s.container.querySelectorAll('[data-gauge]').length).toBe(0);
    cleanup();
    const c = calcsPage();
    expect(c.getAllByText('Fire Resistance').length).toBeGreaterThan(0);
    expect(c.container.querySelectorAll('[data-gauge]').length).toBe(0);
  });
});
