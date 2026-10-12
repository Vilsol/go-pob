<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import GemRow from './GemRow.svelte';

  const { Story } = defineMeta({
    title: 'Spec/GemRow',
    component: GemRow,
    args: { n: 1, name: 'Blade Vortex', level: 20, quality: 20, tone: 'dex', socket: 'G' }
  });
</script>

<script lang="ts">
  let links = $state([true, true, false, false]);
</script>

<Story name="Active gem" asChild>
  <div class="panel relative flex w-110 flex-col px-1 text-13"><GemRow n={1} name="Blade Vortex" level={20} quality={20} tone="dex" socket="G" /></div>
</Story>

<Story
  name="Linked group"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const link = c.getByRole('button', { name: 'Sockets 3–4: not linked' });
    await expect(link).toHaveAttribute('aria-pressed', 'false');
    await userEvent.click(link);
    const now = c.getByRole('button', { name: 'Sockets 3–4: linked' });
    await expect(now).toHaveAttribute('aria-pressed', 'true');
    await userEvent.click(now);
    await expect(c.getByRole('button', { name: 'Sockets 3–4: not linked' })).toHaveAttribute('aria-pressed', 'false');
  }}>
  <div class="panel relative flex w-110 flex-col px-1 text-13">
    <GemRow n={1} name="Blade Vortex" level={20} quality={20} tone="dex" socket="G" />
    <GemRow n={2} name="Arcane Surge Support" level={1} quality={0} tone="int" socket="B" support bind:linked={links[0]} />
    <GemRow n={3} name="Culling Strike Support" level={20} quality={0} tone="dex" socket="G" support bind:linked={links[1]} />
    <GemRow n={4} name="Faster Casting Support" level={20} quality={0} tone="int" socket="B" support bind:linked={links[2]} />
    <GemRow n={5} name="Unleash Support" level={18} quality={0} tone="int" socket="B" support from="Storm Grip" bind:linked={links[3]} />
  </div>
</Story>

<Story name="Gem error" asChild parameters={{ a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } } }}>
  <div class="panel relative flex w-110 flex-col px-1 text-13">
    <GemRow
      n={1}
      name="Fire"
      level={20}
      quality={0}
      tone="white"
      socket="R"
      enabled
      onlevel={() => {}}
      onquality={() => {}}
      error="Ambiguous gem name 'Fire': matches 'Fireball', 'Firestorm'" />
  </div>
</Story>
