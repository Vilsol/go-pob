<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, within } from 'storybook/test';
  import SectionHeading from './SectionHeading.svelte';

  const { Story } = defineMeta({ title: 'Spec/SectionHeading', component: SectionHeading });
</script>

<Story
  name="Sizes"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    await expect(c.getByRole('heading', { level: 3, name: 'GEM OPTIONS' })).toBeTruthy();
    await expect(c.getByRole('heading', { name: 'Custom Modifiers · 3 groups' })).toBeTruthy();
  }}>
  <div class="flex w-90 flex-col gap-3">
    <SectionHeading level={3}>GEM OPTIONS</SectionHeading>
    <SectionHeading size="md">REQUIRED NODES</SectionHeading>
    <SectionHeading size="lg">BUILD SHARING</SectionHeading>
    <SectionHeading size="md" class="uppercase" count="· 3 groups">Custom Modifiers</SectionHeading>
    <SectionHeading size="md">SEARCH{#snippet actions()}<span class="note">12 matches · 4 allocated</span>{/snippet}</SectionHeading>
  </div>
</Story>

<Story name="Eyebrow" asChild>
  <div class="flex flex-col gap-2">
    <span class="eyebrow text-text-2">Offence</span>
    <span class="eyebrow eyebrow-xs text-text-3">Skill type</span>
  </div>
</Story>
