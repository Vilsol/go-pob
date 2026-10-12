<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { userEvent, within } from 'storybook/test';
  import ConfigCustom from './ConfigCustom.svelte';
  import type { CustomOps } from './config';
  import type { CustomGroup } from './fixtures';
  import { expectFloating } from './stories/layout_checks';

  const { Story } = defineMeta({ title: 'Spec/ConfigCustom', component: ConfigCustom, parameters: { layout: 'fullscreen' } });

  const groups: CustomGroup[] = ['Boss setup', 'Flasks', 'Auras'].map((title, i) => ({
    title,
    enabled: i !== 1,
    cells: [{ text: '10% increased Damage', k: 'ok', parsed: 'Damage INC 10' }]
  }));
  const lines = Array.from({ length: 14 }, (_, i) => ({ label: `Stat ${i + 1}`, value: `+${i + 1}`, better: true }));
  const noop = () => {};
  const ops: CustomOps = {
    onselect: noop,
    ontoggle: noop,
    onrename: noop,
    onaddgroup: noop,
    ondeletegroup: noop,
    oncell: noop,
    oncommit: noop,
    onmove: noop,
    onremove: noop,
    onaddcell: noop,
    togglewhatif: () => Promise.resolve([{ lines }])
  };
</script>

<Story
  name="Group what-if on screen"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const boxes = within(c.getByRole('list', { name: 'Mod groups' })).getAllByRole('checkbox');
    const last = boxes[boxes.length - 1];
    await userEvent.hover(last);
    const tip = await c.findByRole('tooltip', { name: 'Group toggle what-if' });
    await within(tip).findByText('Stat 14');
    await expectFloating(last, tip);
  }}>
  <div class="flex h-screen flex-col justify-end">
    <ConfigCustom {groups} selected={0} {ops} />
  </div>
</Story>
