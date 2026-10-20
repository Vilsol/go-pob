<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import SpectreLibrary from './SpectreLibrary.svelte';
  import type { SpectreRow } from './spectres';

  // Build.lua:1453-1482
  const { Story } = defineMeta({
    title: 'Shell/SpectreLibrary',
    component: SpectreLibrary,
    parameters: {
      layout: 'fullscreen',
      a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } }
    }
  });

  const row = (id: string, name: string, skills: string[], life: string): SpectreRow => ({
    id,
    name,
    skills,
    tooltip: [
      `^7${name}`,
      `^7Life Multiplier: x${life}`,
      '^7Resistances: ^xB97123-25^7/^x3F6DB340^7/^xADAA470^7/^xD020900',
      ...skills.map((s) => `^7Skill: ${s}`)
    ]
  });
  const library = [
    row('A', 'Arena Master', ['Cyclone'], '1.50'),
    row('B', 'Bone Husk', ['Fireball', 'Ice Nova'], '0.90'),
    row('C', 'Cannibal Fire-eater', ['Flame Dash', 'Fire Storm'], '1.20'),
    row('D', 'Forged Frostbearer', ['Cold Snap'], '2.00')
  ];
</script>

<Story
  name="Add, search, save"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement.ownerDocument.body);
    const src = c.getByRole('listbox', { name: 'Available Spectres' });
    await userEvent.type(c.getByRole('searchbox', { name: 'Search spectres' }), 'fire');
    await expect(
      within(src)
        .getAllByRole('option')
        .map((o) => o.textContent)
    ).toEqual(['Cannibal Fire-eater']);
    await userEvent.dblClick(within(src).getByRole('option', { name: 'Cannibal Fire-eater' }));
    const dest = c.getByRole('listbox', { name: 'Spectres in Build' });
    await expect(within(dest).getAllByRole('option')).toHaveLength(2);
    await userEvent.hover(within(dest).getByRole('option', { name: 'Cannibal Fire-eater' }));
    await expect(c.getByRole('tooltip')).toHaveTextContent('Skill: Fire Storm');
  }}>
  <div class="h-150"><SpectreLibrary open {library} spectres={[library[3]]} onsave={() => {}} /></div>
</Story>
