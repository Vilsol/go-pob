<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import StatPicker from './StatPicker.svelte';

  const { Story } = defineMeta({
    title: 'Spec/StatPicker',
    component: StatPicker,
    args: {
      label: 'Desired stat',
      class: 'w-80',
      onpick: () => {},
      stats: [
        { id: 'total', label: 'Total Mods', descriptions: [], total: true },
        { id: '1', label: 'Life', descriptions: ['+1% to maximum Life'] },
        { id: '2', label: 'Elemental Resistances', descriptions: ['+2% to all Elemental Resistances'] },
        {
          id: '3',
          label: 'Divine Flesh',
          descriptions: ['All Damage taken bypasses Energy Shield', '50% of Elemental Damage taken as Chaos Damage', '+5% to maximum Chaos Resistance']
        },
        {
          id: '4',
          label: 'Minion Damage while you have at least one nearby Ally',
          descriptions: ['Minions deal 2% increased Damage while you have at least one nearby Ally with a long line that wraps']
        }
      ]
    }
  });
</script>

<Story
  name="Open"
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const c = within(canvasElement);
    await userEvent.click(c.getByRole('combobox', { name: 'Desired stat' }));
    const options = c.getAllByRole('option');
    await expect(options.length).toBe(5);
    for (const o of options) {
      const box = o.getBoundingClientRect();
      await expect(o.scrollHeight, o.textContent ?? '').toBeLessThanOrEqual(o.clientHeight);
      for (const t of o.children) {
        const r = t.getBoundingClientRect();
        await expect(r.top, t.textContent ?? '').toBeGreaterThanOrEqual(box.top);
        await expect(r.bottom, t.textContent ?? '').toBeLessThanOrEqual(box.bottom);
      }
    }
  }} />
