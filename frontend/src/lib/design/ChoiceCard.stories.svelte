<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import ChoiceCard from './ChoiceCard.svelte';

  const { Story } = defineMeta({ title: 'Spec/ChoiceCard', component: ChoiceCard });

  const chars = [
    { name: 'Spectrewitch', line: 'Necromancer · level 94', tone: 'tone-cls-witch' },
    { name: 'MineShadow', line: 'Saboteur · level 90', tone: 'tone-cls-shadow' }
  ];
</script>

<script lang="ts">
  let pick = $state(0);
</script>

<Story
  name="Characters"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const witch = c.getByRole('button', { name: /Spectrewitch/ });
    const shadow = c.getByRole('button', { name: /MineShadow/ });
    await expect(witch).toHaveAttribute('aria-pressed', 'true');
    await userEvent.click(shadow);
    await expect(shadow).toHaveAttribute('aria-pressed', 'true');
    await expect(witch).toHaveAttribute('aria-pressed', 'false');
  }}>
  <div class="flex gap-2">
    {#each chars as ch, i (ch.name)}
      <ChoiceCard pressed={pick === i} onclick={() => (pick = i)}>
        <span class="font-semibold">{ch.name}</span><span class={['toned text-12', ch.tone]}>{ch.line}</span>
      </ChoiceCard>
    {/each}
  </div>
</Story>
