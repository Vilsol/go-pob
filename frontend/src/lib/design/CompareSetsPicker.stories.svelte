<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, fn, userEvent, within } from 'storybook/test';
  import CompareSetsPicker from './CompareSetsPicker.svelte';
  import type { CompareSets } from './compare';

  const { Story } = defineMeta({ title: 'Spec/CompareSetsPicker', component: CompareSetsPicker });
  const list = (labels: string[], active = 0) => ({ labels, active });
  const SETS: CompareSets = {
    tree: list(['Leveling', 'Endgame'], 1),
    skill: list(['Default']),
    item: list(['Default', 'Bossing']),
    config: list(['Default']),
    group: list(['Arc', 'Herald of Thunder']),
    skills: { ...list(['Arc']), enabled: true },
    stage: '3'
  };
  const onset = fn();
  const onconvert = fn();
</script>

<Story
  name="Sets"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    await userEvent.click(c.getByRole('button', { name: /^Comparison sets/ }));
    await userEvent.selectOptions(c.getByRole('combobox', { name: 'Comparison item set' }), 'Bossing');
    await expect(onset).toHaveBeenCalledWith('item', 1);
    await userEvent.click(c.getByRole('button', { name: /^This build's sets/ }));
    await userEvent.selectOptions(c.getByRole('combobox', { name: 'Primary tree version' }), '3_27');
    const dialog = within(document.body).getByRole('dialog', { name: /^Convert to Version/ });
    await userEvent.click(within(dialog).getByRole('button', { name: 'Copy + Convert' }));
    await expect(onconvert).toHaveBeenCalledWith('a', '3_27', false);
  }}>
  <div class="w-240">
    <CompareSetsPicker
      sets={SETS}
      primarySets={{ tree: list(['Main']), item: { ...list(['Default']), ids: [1] }, versions: { a: '3_28', b: '3_28' } }}
      aConfigSets={{ sets: [{ id: 1 }], active: 0 }}
      aLoadouts={{ loadouts: [], selected: -1 }}
      bLoadouts={{ loadouts: ['Mapping', 'Bossing'], selected: 0 }}
      versionList={[
        { value: '3_27', label: '3.27' },
        { value: '3_28', label: '3.28' }
      ]}
      onloadout={() => {}}
      onprimary={() => {}}
      {onset}
      oncount={() => {}}
      {onconvert} />
  </div>
</Story>
