// Main.lua:1359-1397
import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { changesSince, parseChangelog, parseHelp } from './help';

const text = readFileSync(resolve(__dirname, '../../../static/help.txt'), 'utf8');

describe('parseHelp', () => {
  it("reads PoB's sections, key / text lines and dev lines", () => {
    const sections = parseHelp(text, false);
    expect(sections.map((s) => s.title)).toEqual([
      'Help File Introduction',
      'Contents',
      'FAQ',
      'Hotkeys',
      'Other Notable Things',
      'Timeless Jewels',
      'Loadouts',
      'Party Tab',
      'Skills Tab',
      'Items Tab',
      'Similar Builds Popup'
    ]);
    const hotkeys = sections.find((s) => s.title === 'Hotkeys')!;
    expect(hotkeys.lines).toContainEqual({ key: 'Ctrl + 1', text: 'Jump to tree tab' });
    expect(hotkeys.lines.some((l) => l.dev)).toBe(false);
    const dev = parseHelp(text, true).find((s) => s.title === 'Hotkeys')!;
    expect(dev.lines).toContainEqual({ text: 'Developer Mode Shortcuts:', dev: true });
    expect(sections.find((s) => s.title === 'Loadouts')!.lines.some((l) => l.text.startsWith('Loadouts can be selected'))).toBe(true);
  });

  // Main.lua:1398-1411
  it('lists the sections, numbered, under Contents', () => {
    const contents = parseHelp(text, false).find((s) => s.title === 'Contents')!;
    expect(contents.lines.slice(0, 3)).toEqual([{ text: ' ' }, { text: '1. Help File Introduction' }, { text: '2. Contents' }]);
    expect(contents.lines[11]).toEqual({ text: '11. Similar Builds Popup' });
  });
});

// Main.lua:1338-1358
describe('parseChangelog', () => {
  const log = readFileSync(resolve(__dirname, '../../../static/changelog.txt'), 'utf8');
  it('reads a version per VERSION[ver][date] line, its "---" sub-titles and every other line', () => {
    const versions = parseChangelog(log);
    const heads = log.split('\n').filter((l) => /^VERSION\[.+\]\[.+\]\r?$/.test(l));
    expect(versions).toHaveLength(heads.length);
    expect(versions[0]).toMatchObject({ version: '2.67.2', date: '2026/08/07' });
    expect(versions[0].lines.slice(0, 2)).toEqual([{ text: '' }, { text: '--- Fixed Calculations ---', sub: true }]);
    expect(versions[0].lines[2].text).toMatch(/^\* Fix Scornful Herald/);
    expect(versions.reduce((n, v) => n + 1 + v.lines.length, 0)).toBe(log.replace(/\r/g, '').replace(/\n$/, '').split('\n').length);
  });
});

// Main.lua:1300-1320
describe('changesSince', () => {
  const log = 'VERSION[2.68.0][2026/09/01]\n\n--- New ---\n* B\n\nVERSION[2.67.3][2026/08/20]\n* A\n\nVERSION[2.67.2][2026/08/07]\n* old\n';
  it('lists the versions above the running one', () => {
    expect(changesSince(log, '2.67.2').map((v) => [v.version, v.lines.map((l) => l.text).filter(Boolean)])).toEqual([
      ['2.68.0', ['--- New ---', '* B']],
      ['2.67.3', ['* A']]
    ]);
  });
  it('lists nothing when the update carries the running version first, everything when it lacks it', () => {
    expect(changesSince(log, '2.68.0')).toEqual([]);
    expect(changesSince(log, '1.0.0').map((v) => v.version)).toEqual(['2.68.0', '2.67.3', '2.67.2']);
  });
});
