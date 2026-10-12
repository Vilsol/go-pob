<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import Dropdown from './Dropdown.svelte';
  import { expectSelectArrowRoom } from './stories/layout_checks';

  const { Story } = defineMeta({
    title: 'Spec/Dropdown',
    component: Dropdown,
    argTypes: { size: { control: 'inline-radio', options: ['xs', 'default', 'lg'] } },
    args: {
      size: 'xs',
      class: 'w-37.5',
      'aria-label': 'Is the enemy a Boss?',
      value: 'None',
      options: ['None', 'Boss', 'Pinnacle Boss', 'Uber Pinnacle Boss'].map((v) => ({ value: v, label: v }))
    }
  });
</script>

<Story
  name="Playground"
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const c = within(canvasElement);
    await expectSelectArrowRoom(c.getByRole('combobox', { name: 'Is the enemy a Boss?' }));
    await userEvent.click(c.getByRole('combobox', { name: 'Is the enemy a Boss?' }));
    await expect(c.getByRole('option', { selected: true })).toHaveTextContent('None');
    await userEvent.hover(c.getByRole('option', { name: 'Boss' }));
  }} />

<Story name="Closed" args={{ size: 'default', class: 'w-60', value: 'Pinnacle Boss' }} />
