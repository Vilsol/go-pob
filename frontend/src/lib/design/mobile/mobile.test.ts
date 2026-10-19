import { describe, expect, it } from 'vitest';
import type { Sidebar } from '$lib/display/sidebar';
import { headlines } from './mobile';

const line = (label: string, value = '1') => ({ label, value, labelColor: '', valueColor: '' });
const sb: Sidebar = {
  warnings: [],
  sections: [
    {
      groups: [
        [line('Hit DPS', '120'), line('Combined DPS', '700'), line('Full DPS', '900')],
        [line('Total Life', '4000'), line('Total Mana', '800'), line('Energy Shield', '50')]
      ]
    },
    { header: 'Minion', groups: [[line('Hit DPS', '5000'), line('Total Life', '2000')]] }
  ]
};
const shown = (pinned?: string[]) => headlines(sb, pinned).map((h) => `${h.section}:${h.line.label}`);

describe('the phone stats strip', () => {
  it('with nothing pinned shows Life, Mana, ES and Full DPS', () => {
    expect(shown()).toEqual(['Player:Total Life', 'Player:Total Mana', 'Player:Energy Shield', 'Player:Full DPS']);
  });

  it('shows only the pinned stats, in pin order, matching the player section by its empty header', () => {
    expect(shown(['|Hit DPS|0', 'Minion|Total Life|1'])).toEqual(['Player:Hit DPS', 'Minion:Total Life']);
  });

  it('shows every pin, past four, so the strip can scroll', () => {
    const pins = ['|Hit DPS|0', '|Combined DPS|1', '|Full DPS|2', '|Total Life|0', '|Total Mana|1', '|Energy Shield|2'];
    expect(shown(pins)).toEqual(['Player:Hit DPS', 'Player:Combined DPS', 'Player:Full DPS', 'Player:Total Life', 'Player:Total Mana', 'Player:Energy Shield']);
  });

  it('ignores pins that name no line, and falls back to the defaults when none match', () => {
    expect(shown(['|Nothing|0', 'Minion|Hit DPS|0'])).toEqual(['Minion:Hit DPS']);
    expect(shown(['|Nothing|0'])).toEqual(['Player:Total Life', 'Player:Total Mana', 'Player:Energy Shield', 'Player:Full DPS']);
  });
});
