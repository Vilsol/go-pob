// @vitest-environment jsdom
// ItemsTab.lua:696-715

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import DisplayItemControls from './DisplayItemControls.svelte';
import type { DisplayEdit, DisplayInfo } from './items';

afterEach(cleanup);

const info = (skill: string, nodeCount: number): DisplayInfo => ({
  quality: 0,
  hasQuality: false,
  variants: [],
  variant: 0,
  altVariants: [],
  sockets: '',
  socketLimit: 0,
  defaultColor: 'W',
  canInfluence: false,
  influence: [],
  showsCatalyst: false,
  catalyst: 0,
  catalystQuality: 0,
  mods: [],
  cluster: {
    skills: [
      { id: 'a', label: 'Skill A' },
      { id: 'b', label: 'Skill B' }
    ],
    skill,
    nodeCount,
    min: 8,
    max: 12
  }
});

describe('DisplayItemControls cluster jewel', () => {
  it('builds the slider edit on a skill edit Go has not answered yet, and on Go once it has', async () => {
    const edits: DisplayEdit[] = [];
    const { getByRole, rerender } = render(DisplayItemControls, { info: info('a', 8), onedit: (e: DisplayEdit) => edits.push(e) });
    const skill = getByRole('combobox', { name: 'Cluster skill' }) as HTMLSelectElement;
    const slider = getByRole('slider', { name: 'Added Passives' }) as HTMLInputElement;

    skill.value = 'b';
    await fireEvent.change(skill);
    slider.value = '11';
    await fireEvent.change(slider);
    expect(edits).toEqual([
      { kind: 'cluster', text: 'b', value: 8 },
      { kind: 'cluster', text: 'b', value: 11 }
    ]);

    await rerender({ info: info('b', 11) });
    skill.value = 'a';
    await fireEvent.change(skill);
    expect(edits.at(-1)).toEqual({ kind: 'cluster', text: 'a', value: 11 });
  });
});
