<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, within } from 'storybook/test';
  import NodeTipSections from './NodeTipSections.svelte';

  const { Story } = defineMeta({
    title: 'Spec/NodeTipSections',
    component: NodeTipSections,
    parameters: { a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } } }
  });
</script>

<Story
  name="All sections"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    await expect(c.getByLabelText('Points and gold')).toHaveTextContent('3 points to node');
    await expect(c.getByLabelText('Reminder text')).toHaveTextContent('Ailments');
    await expect(c.getByLabelText('Compared mastery')).toHaveTextContent('-> 10% increased maximum Life');
  }}>
  <div class="flex w-90 flex-col gap-2 rounded-md border border-line-strong bg-ground px-3 py-2 text-13 text-text">
    <NodeTipSections
      tip={{
        pick: { index: 1, text: '+10% to all Elemental Resistances' },
        comparedMastery: { remove: ['+50 to maximum Life'], add: ['10% increased maximum Life'] },
        affecting: [
          { socket: 1, title: '^xAF6025Thread of Hope', lines: [{ Text: 'Passives in Radius can be Allocated without being connected', Separator: false }] }
        ],
        reminder: ['Ailments are Ignite, Chill, Freeze, Shock, Poison and Bleeding'],
        flavour: ['Life is the measure.'],
        tattooKind: 'tattoo',
        compareOnly: false
      }}
      cost={{ path: 3, leap: false, depends: 0, gold: '1,000', dependsGold: '0', socket: false, allocated: false }} />
  </div>
</Story>
