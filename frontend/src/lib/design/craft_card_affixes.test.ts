// @vitest-environment jsdom

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup, fireEvent, within, waitFor } from '@testing-library/svelte';
import DisplayItemControls from './DisplayItemControls.svelte';
import { craftCardSample as sample } from './craft-card/fixture';
import { affixLines, statTemplate, type AffixControls } from './crafting';
import type { DisplayEdit, DisplayInfo } from './items';

afterEach(cleanup);

const described = (s: HTMLElement) =>
  (s.getAttribute('aria-describedby') ?? '')
    .split(' ')
    .filter(Boolean)
    .map((id) => document.getElementById(id)?.textContent ?? `missing #${id}`);

describe('crafting card affixes', () => {
  it('every roll slider is described by the stat lines it changes, and an affix’s roll shows in those lines', async () => {
    const { container } = render(DisplayItemControls, { info: sample.info, craft: sample.craft, layout: 'stepped', onedit: () => {} });
    const c = within(container);
    const sliders = c.getAllByRole('slider');
    expect(sliders.length).toBe(sample.info.affixes!.controls.filter((x) => x.sliderShown).length + sample.info.mods.filter((m) => m.hasRange).length);
    for (const s of sliders) {
      const name = s.getAttribute('aria-label')!;
      const lines = described(s);
      const affix = /^Roll of affix (\d)$/.exec(name);
      if (affix) {
        const ctl = sample.info.affixes!.controls[Number(affix[1]) - 1];
        expect(lines.map(statTemplate), name).toEqual(ctl.series[ctl.sel].label.split('/').map(statTemplate));
      } else expect([`Roll of ${lines[0]}`], name).toEqual([name]);
    }

    const roll = c.getByRole('slider', { name: 'Roll of affix 2' });
    expect(described(roll)).toEqual(['+98 to Evasion Rating', '+31 to maximum Energy Shield']);
    await fireEvent.input(roll, { target: { value: '0.2' } });
    await waitFor(() => expect(described(roll)).toEqual(['+24 to Evasion Rating', '+24 to maximum Energy Shield']));
  });

  it('a line’s slider sends its range edit, and its line is the one Go’s answer changes', async () => {
    let info: DisplayInfo = sample.info;
    const edits: DisplayEdit[] = [];
    const view = render(DisplayItemControls, { info, craft: sample.craft, layout: 'stepped', onedit: (e: DisplayEdit) => edits.push(e) });
    const s = view.getByRole('slider', { name: 'Roll of +42% to Cold Resistance' });
    expect(described(s)).toEqual(['+42% to Cold Resistance']);
    expect(s.nextElementSibling?.textContent).toBe('86%');
    await fireEvent.input(s, { target: { value: '0.25' } });
    expect(s.nextElementSibling?.textContent).toBe('25%');
    await fireEvent.change(s);
    const index = info.mods.findIndex((m) => m.line === '+42% to Cold Resistance');
    expect(edits).toEqual([{ kind: 'range', index, float: 0.25 }]);
    info = { ...info, mods: info.mods.map((m, k) => (k === index ? { ...m, line: '+33% to Cold Resistance', range: 0.25 } : m)) };
    await view.rerender({ info });
    expect(described(view.getByRole('slider', { name: 'Roll of +33% to Cold Resistance' }))).toEqual(['+33% to Cold Resistance']);
  });

  it('a stat two dropdowns share is one line (Item.lua:2085-2093): held by the first, changed by both', () => {
    const series = (label: string) => ({ label, tiers: 2, haveRange: true, tooltip: [], mid: [] });
    const affixes: AffixControls = {
      sorting: false,
      controls: [
        { type: 'Prefix', series: [series('None'), series('+# to maximum Life')], sel: 1, sliderShown: true, sliderVal: 0.5 },
        { type: 'Prefix', series: [series('None'), series('+# to maximum Energy Shield/+# to maximum Life')], sel: 1, sliderShown: true, sliderVal: 0.5 },
        { type: 'Suffix', series: [series('None'), series('+(10-12)% to Fire Resistance')], sel: 1, sliderShown: false, sliderVal: 0.5 }
      ]
    };
    const lines = ['+20 to maximum Mana', '+31 to maximum Energy Shield', '+130 to maximum Life', '+11% to Fire Resistance', '+15% to Cold Resistance'];
    const got = affixLines(affixes, lines, [1, 2, 3, 4]);
    expect(got.stats).toEqual([[2], [1, 2], [3]]);
    expect(got.slots).toEqual([[2], [1], [3]]);
    expect([...got.shared]).toEqual([2]);
    expect(got.rest).toEqual([4]);
  });
});
