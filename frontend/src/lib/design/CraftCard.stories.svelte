<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, waitFor, within } from 'storybook/test';
  import DisplayItemControls from './DisplayItemControls.svelte';
  import ImpactBlocks from './ImpactBlocks.svelte';
  import Button from './Button.svelte';
  import { tone as toneClass } from './tone';
  import { craftCardSample as sample } from './craft-card/fixture';
  import type { ImpactBlock } from './items';

  const { Story } = defineMeta({
    title: 'Items/Crafting card',
    parameters: {
      layout: 'padded',
      a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } }
    }
  });

  const top = (el: Element) => el.getBoundingClientRect().top;
  const headings = (root: HTMLElement) =>
    within(root)
      .getAllByRole('heading', { level: 3 })
      .map((h) => h.textContent?.trim());
</script>

{#snippet head()}
  <div class={['flex flex-col', toneClass(sample.item.rarity)]}>
    <span data-pob-color class="toned text-15 font-semibold">{sample.item.name}</span>
    <span class="text-12 text-text-2">Rare {sample.item.base} → fits {sample.item.fits.join(', ')}</span>
  </div>
{/snippet}
{#snippet result()}
  <section aria-label="Equip impact" class="flex flex-col gap-2.5 font-mono text-13">
    <ImpactBlocks blocks={sample.blocks}>
      {#snippet action(b: ImpactBlock)}
        {#if b.slot}<Button size="sm" variant="primary" class="shrink-0 font-sans">Equip in {b.slot}</Button>{/if}
      {/snippet}
    </ImpactBlocks>
  </section>
  <div class="flex flex-wrap gap-2"><Button size="md">Save to build</Button></div>
{/snippet}

<Story
  name="A · Stepped sections"
  asChild
  play={async ({ canvasElement }) => {
    const card = within(canvasElement).getByRole('group', { name: 'Crafting card' });
    await expect(headings(card)).toEqual(['Base', 'Implicits', 'Affixes', 'Result']);
    const sections = ['Base', 'Implicits', 'Affixes', 'Result'].map((n) => within(card).getByRole('region', { name: n }));
    for (let i = 1; i < sections.length; i++) await expect(top(sections[i])).toBeGreaterThan(sections[i - 1].getBoundingClientRect().bottom);
    await expect(within(sections[1]).getByRole('list', { name: 'Implicit modifiers' })).toHaveTextContent('10% increased maximum Life');
    await expect(within(sections[2]).getByRole('combobox', { name: 'Affix 1' })).toBeVisible();
  }}>
  <div class="w-115"><DisplayItemControls info={sample.info} craft={sample.craft} onedit={() => {}} layout="stepped" {head} {result} /></div>
</Story>

<Story
  name="C · Tabs with result footer"
  asChild
  play={async ({ canvasElement }) => {
    const card = within(canvasElement).getByRole('group', { name: 'Crafting card' });
    const c = within(card);
    await expect(c.getByRole('combobox', { name: 'Affix 1' })).toBeVisible();
    await userEvent.click(c.getByRole('tab', { name: /Base/ }));
    await waitFor(() => expect(c.getByRole('spinbutton', { name: 'Quality' })).toBeVisible());
    await expect(c.queryByRole('combobox', { name: 'Affix 1' })).toBeNull();
    await expect(c.getByRole('region', { name: 'Result' })).toBeVisible();
  }}>
  <div class="w-115"><DisplayItemControls info={sample.info} craft={sample.craft} onedit={() => {}} layout="tabs" {head} {result} /></div>
</Story>

<Story
  name="A · Stepped, wide (side by side)"
  asChild
  play={async ({ canvasElement }) => {
    const card = within(canvasElement).getByRole('group', { name: 'Crafting card' });
    const [base, implicits, affixes, outcome] = ['Base', 'Implicits', 'Affixes', 'Result'].map((n) =>
      within(card).getByRole('region', { name: n }).getBoundingClientRect()
    );
    await expect(implicits.left).toBeGreaterThan(base.right);
    await expect(implicits.top).toBe(base.top);
    await expect(outcome.left).toBeGreaterThan(affixes.right);
    await expect(outcome.top).toBe(affixes.top);
    await expect(affixes.top).toBeGreaterThan(Math.max(base.bottom, implicits.bottom));
  }}>
  <div class="w-240"><DisplayItemControls info={sample.info} craft={sample.craft} onedit={() => {}} layout="stepped" {head} {result} /></div>
</Story>

<Story
  name="Affixes: hybrid entries, a long line, six slots"
  asChild
  play={async ({ canvasElement }) => {
    const prefixes = within(canvasElement).getByRole('group', { name: 'Prefixes' });
    const hybrid = within(prefixes).getByRole('group', { name: 'Prefix 2' });
    await expect(within(hybrid).getByRole('list', { name: 'Lines of affix 2' }).children).toHaveLength(2);
    for (const s of within(hybrid).getAllByRole('slider')) await expect(s.getAttribute('aria-describedby')).toBeTruthy();
  }}>
  <div class="w-115"><DisplayItemControls info={sample.info} craft={sample.craft} onedit={() => {}} layout="stepped" /></div>
</Story>
