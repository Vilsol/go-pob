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
      'Introduction',
      'Contents',
      'Getting started',
      'Tree tab',
      'Skills tab',
      'Items tab',
      'Calcs tab',
      'Config tab',
      'Party tab',
      'Notes tab',
      'Import tab',
      'Compare tab',
      'Loadouts',
      'Hotkeys',
      'Mouse and touch',
      'Desktop app',
      'Offline and updates',
      'FAQ'
    ]);
    const hotkeys = sections.find((s) => s.title === 'Hotkeys')!;
    expect(hotkeys.lines).toContainEqual({ key: 'Ctrl+M', text: 'Manage trees' });
    expect(hotkeys.lines.some((l) => l.dev)).toBe(false);
    const dev = parseHelp(text, true).find((s) => s.title === 'Hotkeys')!;
    expect(dev.lines).toContainEqual({ text: 'Dev mode (Settings > General) shows node ids in passive tooltips. Ctrl+U does nothing in Dev mode.', dev: true });
    expect(sections.find((s) => s.title === 'Loadouts')!.lines.some((l) => l.text.startsWith('The Loadout selector'))).toBe(true);
  });

  // Main.lua:1398-1411
  it('lists the sections, numbered, under Contents', () => {
    const contents = parseHelp(text, false).find((s) => s.title === 'Contents')!;
    expect(contents.lines.slice(0, 3)).toEqual([{ text: ' ' }, { text: '1. Introduction' }, { text: '2. Contents' }]);
    expect(contents.lines[18]).toEqual({ text: '18. FAQ' });
  });
});

// Main.lua:1338-1358
describe('parseChangelog', () => {
  const log = readFileSync(resolve(__dirname, '../../../static/pob-changelog.txt'), 'utf8');
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
