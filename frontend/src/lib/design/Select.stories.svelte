<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect } from 'storybook/test';
  import Select from './Select.svelte';
  import { expectSelectArrowRoom } from './stories/layout_checks';

  const { Story } = defineMeta({
    title: 'Spec/Select',
    component: Select,
    argTypes: { size: { control: 'inline-radio', options: ['xs', 'default', 'lg'] } },
    args: { size: 'lg', options: ['Raise Spectre', 'Blade Vortex', 'Stormblast Mine'], 'aria-label': 'Main skill' }
  });

  const arrowRoom = async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    for (const s of canvasElement.querySelectorAll('select')) await expectSelectArrowRoom(s);
  };
</script>

<Story name="Playground" play={arrowRoom} />

<Story
  name="Compact option"
  play={arrowRoom}
  args={{ size: 'xs', class: 'max-w-37.5', options: ['Uber Pinnacle Boss', 'Pinnacle Boss', 'Boss', 'None'], 'aria-label': 'Is the enemy a Boss?' }} />

<Story name="Sizes" asChild play={arrowRoom}>
  <div class="flex items-center gap-2">
    <Select size="xs" class="w-24" options={['Uber Pinnacle Boss']} aria-label="xs" />
    <Select class="w-28" options={['Stormblast Mine of the long name']} aria-label="default" />
    <Select size="lg" class="w-32" options={['Raise Spectre with a long name']} aria-label="lg" />
  </div>
</Story>

<Story
  name="Cancelled change"
  asChild
  play={async ({ canvasElement }) => {
    const { userEvent } = await import('storybook/test');
    const select = (canvasElement as HTMLElement).querySelector('select')!;
    await userEvent.selectOptions(select, 'b');
    await new Promise((r) => setTimeout(r, 50));
    await expect(select.value).toBe('a');
  }}>
  <Select
    aria-label="Cancelled"
    options={['a', 'b']}
    value="a"
    onchange={(e: Event & { currentTarget: HTMLSelectElement }) => {
      e.currentTarget.value = 'a';
    }} />
</Story>
