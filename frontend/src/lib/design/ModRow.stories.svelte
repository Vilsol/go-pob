<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import ModRow from './ModRow.svelte';

  const { Story } = defineMeta({ title: 'Spec/ModRow', component: ModRow });

  const mods = [
    { text: '+94 to maximum Life', meta: 'PREFIX · T3', tone: 'magic' as const },
    { text: '+1 to Level of Socketed Minion Gems', meta: 'CRAFTED', tone: 'crafted' as const },
    { text: '+42% to Cold Resistance', meta: 'SUFFIX · FRACTURED', tone: 'fractured' as const }
  ];
</script>

<script lang="ts">
  let cur = $state(0);
</script>

<Story
  name="Item mods"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const life = c.getByRole('button', { name: /maximum Life/ });
    const cold = c.getByRole('button', { name: /Cold Resistance/ });
    await expect(life).toHaveAttribute('aria-current', 'true');
    await userEvent.click(cold);
    await expect(cold).toHaveAttribute('aria-current', 'true');
    await expect(life).not.toHaveAttribute('aria-current');
  }}>
  <div class="panel flex w-110 flex-col px-2.5 py-1.5">
    {#each mods as m, i (m.text)}<ModRow {...m} current={cur === i} onclick={() => (cur = i)} />{/each}
  </div>
</Story>
