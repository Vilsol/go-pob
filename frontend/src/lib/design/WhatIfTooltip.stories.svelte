<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import WhatIfTooltip from './WhatIfTooltip.svelte';
  import { expect } from 'storybook/test';

  const { Story } = defineMeta({
    title: 'Spec/WhatIfTooltip',
    component: WhatIfTooltip,
    args: {
      title: 'Death Attunement',
      kind: 'Notable',
      mods: ['Minions deal 20% increased Damage', 'Minions have 10% increased Attack and Cast Speed'],
      action: 'Allocating (3 points)',
      sections: [
        {
          title: 'Minion',
          lines: [
            { label: 'Total DPS', text: '▲ +38,912 (9.4%)', dir: 'up', perPoint: '+12,971 pp' },
            { label: 'Life', text: '▲ +1,204', dir: 'up', perPoint: '+401 pp' }
          ]
        },
        { title: 'Player', lines: [{ label: 'Mana Cost', text: '▼ +4', dir: 'down' }] }
      ]
    }
  });
</script>

<Story
  name="Allocating a path"
  args={{
    mods: ['Minions deal 20% increased Damage', 'Minions have 10% increased Attack and Cast Speed while you have Arcane Surge']
  }}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const tip = canvasElement.querySelector<HTMLElement>('[role="tooltip"]')!;
    const lineHeight = (el: Element) => (1.9 / 1.6) * parseFloat(getComputedStyle(el).fontSize);
    for (const el of tip.querySelectorAll('.text-magic > span, [data-line]')) {
      await expect(el.getBoundingClientRect().height, el.textContent ?? '').toBeLessThan(1.6 * lineHeight(el));
    }
  }} />

<Story
  name="Single node"
  args={{
    title: 'Heart and Soul',
    mods: ['8% increased maximum Life'],
    action: 'Allocating (1 point)',
    sections: [{ lines: [{ label: 'Life', text: '▲ +221', dir: 'up' }] }]
  }} />

<!-- PassiveTreeView.lua:1578-1599 -->
<Story
  name="Unallocated mastery"
  args={{
    title: 'Life Mastery',
    kind: 'Mastery',
    modsTitle: 'Available Mastery node options are:',
    mods: ['+50 to maximum Life', '20% of Life Recovery from Flasks is applied to nearby Allies'],
    action: 'Allocating the path: 3 points (blue)',
    sections: [],
    empty: 'No changes from allocating this node or the nodes leading to it.'
  }} />

<!-- TreeTab.lua:51-104 -->
<Story
  name="Tree set"
  args={{
    title: 'Until Lab',
    kind: undefined,
    mods: [],
    info: ['Class: Witch', 'Ascendancy: Elementalist', 'Points used: 36', 'Jewel sockets: 1'],
    footer: 'Game Version: 3.29',
    action: '',
    sections: [
      {
        title: 'Switching to this tree will give you:',
        lines: [
          { label: 'Life', text: '▼ -812 (-19.1%)', dir: 'down' },
          { label: 'Total DPS', text: '▼ -41,203 (-37.5%)', dir: 'down' }
        ]
      }
    ]
  }} />

<Story
  name="Over the tree"
  args={{ translucent: true }}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const bg = getComputedStyle(canvasElement.querySelector('[role="tooltip"]')!).backgroundColor;
    const alpha = Number(/[\d.]+(?=\)$)/.exec(bg.replace(/\s/g, ''))?.[0] ?? 1);
    await expect(alpha, bg).toBeLessThan(1);
    await expect(alpha, bg).toBeGreaterThan(0.8);
  }} />
