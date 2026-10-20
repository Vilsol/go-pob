<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import Disclosure from './Disclosure.svelte';

  const { Story } = defineMeta({ title: 'Spec/Disclosure', component: Disclosure });
</script>

<script lang="ts">
  let open = $state(false);
</script>

<Story
  name="Indicators"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const b = c.getByRole('button', { name: 'Offence' });
    await expect(b).toHaveAttribute('aria-expanded', 'false');
    await userEvent.click(b);
    await expect(b).toHaveAttribute('aria-expanded', 'true');
    await expect(c.getByText('Total DPS')).toBeVisible();
  }}>
  <div class="flex w-80 flex-col gap-3">
    <div class="flex flex-col">
      <Disclosure class="stat-group-head" bind:open controls="dis-offence">Offence</Disclosure>
      <div id="dis-offence" class="pl-4 text-12" hidden={!open}>Total DPS</div>
    </div>
    <Disclosure class="btn btn-2xs self-start" indicator="caret">Comparison sets</Disclosure>
    <Disclosure class="btn-bare self-start px-1 text-13 text-text-2" indicator="plusminus" open aria-label="Collapse General" />
  </div>
</Story>
