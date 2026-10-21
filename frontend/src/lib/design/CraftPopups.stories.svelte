<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, waitFor, within } from 'storybook/test';
  import EnchantDialog from './EnchantDialog.svelte';
  import AnointDialog from './AnointDialog.svelte';
  import CorruptDialog from './CorruptDialog.svelte';
  import AddImplicitDialog from './AddImplicitDialog.svelte';
  import AddModifierDialog from './AddModifierDialog.svelte';
  import AffixControls from './AffixControls.svelte';
  import DisplayItemControls from './DisplayItemControls.svelte';
  import CrucibleDialog from './CrucibleDialog.svelte';
  import { allAffixSliders, type DisplayInfo } from './items';
  import type { AffixControls as AffixData, AnointNotable, CorruptMod, CraftApi } from './crafting';

  // ItemsTab.lua:2661-3015
  const { Story } = defineMeta({
    title: 'Items/CraftPopups',
    parameters: {
      layout: 'fullscreen',
      a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } }
    }
  });

  const listOf = (box: HTMLElement) => box.ownerDocument.getElementById(box.getAttribute('aria-controls') ?? '') as HTMLElement;
  const openList = async (box: HTMLElement) => {
    if (box.getAttribute('aria-expanded') !== 'true') await userEvent.click(box);
    await waitFor(() => expect(listOf(box)).toBeTruthy());
    return listOf(box);
  };
  const entries = async (box: HTMLElement) => {
    const out = within(await openList(box))
      .getAllByRole('option')
      .map((o) => o.textContent?.trim());
    await userEvent.click(box);
    return out;
  };
  const chooseAt = async (box: HTMLElement, i: number) => {
    await userEvent.click(within(await openList(box)).getAllByRole('option')[i]);
    await waitFor(() => expect(box).toHaveAttribute('aria-expanded', 'false'));
  };

  const notable = (id: number, name: string, stat: string): AnointNotable => ({ id, name, stats: [stat], reminder: [], recipe: ['Clear Oil'] });
  const notables = [
    notable(1, 'Constitution', '+20 to maximum Life'),
    notable(2, 'Acuity', '20% increased Accuracy Rating'),
    notable(3, 'Heart of Oak', '+30 to maximum Life')
  ];
  const lines = ['10% increased Fireball Damage', '40% increased Fireball Damage', '20% increased Fireball Damage'];
  const api: CraftApi = {
    enchantChoices: () =>
      Promise.resolve({
        haveSkills: true,
        skills: [
          { skill: 'Fireball', sources: [{ name: 'MERCILESS', label: 'Merciless Labyrinth', lines }] },
          { skill: 'Ice Nova', sources: [{ name: 'MERCILESS', label: 'Merciless Labyrinth', lines: ['10% increased Ice Nova Damage'] }] }
        ],
        used: ['Fireball']
      }),
    notables: () => Promise.resolve(notables),
    sortStats: () =>
      Promise.resolve([
        { stat: 'FullDPS', label: 'Full DPS' },
        { stat: 'Life', label: 'Life' }
      ]),
    preview: (e) => Promise.resolve({ item: 'Doom Visor', base: 'Hubris Circlet', rarity: 'rare', mods: [e.text], impactTitle: '', impact: [] }),
    enchantSort: (_slot, l) => Promise.resolve(l.map((x) => Number(/\d+/.exec(x)?.[0] ?? 0))),
    anointPowers: (_slot, ids) => Promise.resolve(ids.map((id) => ({ 1: 20, 2: 0, 3: 30 })[id] ?? 0)),
    anointCompare: (_slot, id) =>
      Promise.resolve(
        id === 2
          ? { note: '^7Anointing Acuity changes nothing because this node is already allocated on the tree.', blocks: [] }
          : { blocks: [{ header: 'Anointing Constitution will give you: ', sections: [{ lines: [{ label: 'Life', value: '+20', better: true }] }] }] }
      )
  };
  const cmod = (id: string, line: string, group: string): CorruptMod => ({ id, label: line, lines: [line], group, modTags: [] });
  const corruptApi: CraftApi = {
    ...api,
    corruptChoices: () =>
      Promise.resolve({
        unique: true,
        implicitsCannotBeChanged: false,
        maxImplicits: 2,
        glimpse: [],
        corrupted: [
          cmod('A', '(4-6)% increased maximum Life', 'Life'),
          cmod('B', '(5-6)% increased maximum Life', 'Life'),
          cmod('C', '+1 to Maximum Power Charges', 'Power')
        ],
        scourgeUpside: [],
        scourgeDownside: [],
        rolls: [{ index: 0, line: '+50 to maximum Life', value: 1 }]
      }),
    rangeLine: (_i, v) => Promise.resolve(`+${Math.round(50 * v)} to maximum Life`),
    corruptText: () => Promise.resolve('Rarity: UNIQUE'),
    corruptSort: (_s, _t, ids) => Promise.resolve(ids.map((id) => ({ A: 1, B: 2, C: 0 })[id] ?? 0)),
    modCompare: () => Promise.resolve([{ header: 'Adding this mod will give: ', sections: [{ lines: [{ label: 'Life', value: '+40', better: true }] }] }]),
    tipOf: () => Promise.resolve({ item: 'Test Crown', base: 'Hubris Circlet', rarity: 'unique', mods: ['Corrupted'], impactTitle: '', impact: [] })
  };
  const imod = (id: string, line: string) => ({ id, label: line, lines: [line], modTags: [] });
  const implicitApi: CraftApi = {
    ...corruptApi,
    implicitChoices: () =>
      Promise.resolve({
        sources: [
          { id: 'EXARCH', label: 'Searing Exarch' },
          { id: 'CUSTOM', label: 'Custom' }
        ],
        groups: {
          EXARCH: [
            { label: '#% increased Spell Damage', mods: [imod('S1', '10% increased Spell Damage'), imod('S2', '12% increased Spell Damage')] },
            { label: '+# to maximum Life', mods: [imod('L1', '+20 to maximum Life'), imod('L2', '+30 to maximum Life')] }
          ]
        }
      }),
    implicitText: () => Promise.resolve('Rarity: RARE'),
    implicitSort: (_s, ids) => Promise.resolve(ids.map((id) => ({ S1: 1, S2: 2, L1: 3, L2: 4 })[id] ?? 0))
  };
  const customApi: CraftApi = {
    ...implicitApi,
    customModChoices: () =>
      Promise.resolve({
        sources: [
          { id: 'MASTER', label: 'Crafting Bench' },
          { id: 'CUSTOM', label: 'Custom' }
        ],
        mods: {
          MASTER: [
            { id: 'm1', label: '+(25-34) to maximum Mana ^8(Prefix)', lines: ['+(25-34) to maximum Mana'], modTags: [] },
            { id: 'm2', label: '+(25-34) to maximum Life ^8(Prefix)', lines: ['+(25-34) to maximum Life'], modTags: [] }
          ]
        }
      }),
    customModText: () => Promise.resolve('Rarity: RARE'),
    customModSort: (_s, ids) => Promise.resolve(ids.map((id) => ({ m1: 1, m2: 2 })[id] ?? 0))
  };
  // the preview arrives only when the play function releases it
  let releasePreview = () => {};
  const latePreviewApi: CraftApi = {
    ...customApi,
    tipOf: () =>
      new Promise((done) => {
        releasePreview = () =>
          done({
            item: 'Test Crown',
            base: 'Hubris Circlet',
            rarity: 'rare',
            mods: Array.from({ length: 12 }, (_, i) => `+${i} to maximum Life`),
            impactTitle: '',
            impact: []
          });
      })
  };
  const series = (label: string, tiers: number) => ({ label, tiers, haveRange: tiers > 0, tooltip: [`^7${tiers} Tiers`, label], mid: [label] });
  const affixData: AffixData = {
    sorting: true,
    controls: [
      {
        type: 'Prefix',
        series: [series('None', 0), series('+# to maximum Mana', 12), series('+# to maximum Life', 10)],
        sel: 2,
        sliderShown: true,
        sliderVal: 0.55
      },
      { type: 'Suffix', series: [series('None', 0), series('+# to Intelligence', 10)], sel: 0, sliderShown: false, sliderVal: 0.5 }
    ]
  };
  const affixApi: CraftApi = {
    ...customApi,
    affixEdit: () => Promise.resolve(undefined),
    affixRollTooltip: (_slot, v) => Promise.resolve([`+${Math.round(3 + 141 * v)} to maximum Life`, '', '^7Affix: Tier 1 (Prime)']),
    affixPowers: (slot) => Promise.resolve(slot === 0 ? [0, 1, 5] : [0, 2])
  };
  const clusterInfo: DisplayInfo = {
    quality: 0,
    hasQuality: false,
    variants: [],
    variant: 0,
    altVariants: [],
    sockets: '',
    socketLimit: 0,
    defaultColor: '',
    canInfluence: false,
    influence: [],
    showsCatalyst: false,
    catalyst: 0,
    catalystQuality: 0,
    mods: [],
    cluster: {
      skills: [
        { id: 'a', label: 'Attack Damage' },
        { id: 'b', label: 'Fire Damage' }
      ],
      skill: 'b',
      nodeCount: 9,
      min: 8,
      max: 12
    }
  };
  const rangedInfo: DisplayInfo = {
    ...clusterInfo,
    cluster: undefined,
    mods: [
      { line: '+(80-100) to maximum Life', kind: 'explicit', hasRange: true, range: 0.25, disabled: false, removable: false, removal: '' },
      { line: 'Cannot be Frozen', kind: 'explicit', hasRange: false, range: 0, disabled: false, removable: false, removal: '' },
      { line: '(20-30)% increased Attack Speed', kind: 'explicit', hasRange: true, range: 0.75, disabled: false, removable: false, removal: '' }
    ]
  };
  const crucibleApi: CraftApi = {
    ...affixApi,
    crucibleChoices: () =>
      Promise.resolve({
        nodes: [
          [
            { id: 'a', label: '10% increased Attack Speed - Tier: 3', lines: ['10% increased Attack Speed'] },
            { id: 'b', label: '20% increased Physical Damage - Tier: 2', lines: ['20% increased Physical Damage'] }
          ],
          [],
          [],
          [],
          []
        ],
        selected: ['b', '', '', '', '']
      }),
    crucibleText: () => Promise.resolve('Rarity: RARE')
  };
</script>

<Story
  name="Enchant: sort and pick"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement.ownerDocument.body);
    const dialog = await c.findByRole('dialog', { name: 'Enchant Item' });
    const d = within(dialog);
    await waitFor(() => expect(d.getByRole('combobox', { name: 'Skill' })).toHaveDisplayValue('Fireball'));
    await userEvent.selectOptions(d.getByRole('combobox', { name: 'Sort by' }), 'Full DPS');
    await waitFor(async () =>
      expect(await entries(d.getByRole('combobox', { name: 'Enchantment' }))).toEqual([
        '40% increased Fireball Damage',
        '20% increased Fireball Damage',
        '10% increased Fireball Damage'
      ])
    );
    await waitFor(() => expect(d.getByRole('group', { name: 'Enchanted item' })).toHaveTextContent('40% increased Fireball Damage'));
  }}>
  <div class="h-150"><EnchantDialog open slot={1} craft={api} onedit={() => {}} /></div>
</Story>

<Story
  name="Anoint: search, sort and compare"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement.ownerDocument.body);
    const dialog = await c.findByRole('dialog', { name: 'Anoint Item' });
    const d = within(dialog);
    const list = d.getByRole('listbox', { name: 'Notables' });
    await waitFor(() => expect(within(list).getAllByRole('option')).toHaveLength(3));
    await userEvent.selectOptions(d.getByRole('combobox', { name: 'Search in' }), 'Modifiers');
    await userEvent.type(d.getByRole('searchbox', { name: 'Search notables' }), 'life');
    await userEvent.selectOptions(d.getByRole('combobox', { name: 'Sort notables' }), 'Sort by Life');
    await waitFor(() =>
      expect(
        within(list)
          .getAllByRole('option')
          .map((o) => o.textContent)
      ).toEqual(['Heart of Oak', 'Constitution'])
    );
    await userEvent.click(within(list).getByRole('option', { name: 'Constitution' }));
    await expect(d.getByRole('button', { name: 'Anoint Constitution' })).toBeVisible();
    await waitFor(() => expect(d.getByRole('region', { name: 'Anoint comparison' })).toHaveTextContent('Anointing Constitution will give you:'));
  }}>
  <div class="h-150"><AnointDialog open slot={1} anointed={['Acuity']} craft={api} onedit={() => {}} /></div>
</Story>

<Story
  name="Corrupt: implicits and roll ranges"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement.ownerDocument.body);
    const dialog = await c.findByRole('dialog', { name: 'Corrupted Item' });
    const d = within(dialog);
    const first = await d.findByRole('combobox', { name: 'Implicit #1' });
    await waitFor(async () => expect(await entries(first)).toHaveLength(4));
    await chooseAt(first, 1);
    await expect(await entries(d.getByRole('combobox', { name: 'Implicit #2' }))).toEqual(['None', '+1 to Maximum Power Charges']);
    await waitFor(() => expect(d.getByRole('group', { name: 'Implicit comparison' })).toHaveTextContent('Adding this mod will give:'));
    await userEvent.click(d.getByRole('tab', { name: 'Roll Ranges' }));
    await expect(d.getByRole('group', { name: 'Roll Ranges' })).toHaveTextContent('+50 to maximum Life');
    const tab = d.getByRole('tab', { name: 'Roll Ranges' });
    await expect(d.getByRole('tabpanel', { name: 'Roll Ranges' })).toHaveAttribute('id', tab.getAttribute('aria-controls'));
  }}>
  <div class="h-150"><CorruptDialog open craft={corruptApi} /></div>
</Story>

<Story
  name="Add implicit: types, sort and custom"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement.ownerDocument.body);
    const dialog = await c.findByRole('dialog', { name: 'Add Implicit to Item' });
    const d = within(dialog);
    const type = await d.findByRole('combobox', { name: 'Type' });
    await waitFor(() => expect(within(type).getAllByRole('option')).toHaveLength(2));
    await userEvent.selectOptions(d.getByRole('combobox', { name: 'Sort by' }), 'Life');
    await waitFor(() =>
      expect(
        within(d.getByRole('combobox', { name: 'Type' }))
          .getAllByRole('option')
          .map((o) => o.textContent)
      ).toEqual(['+# to maximum Life', '#% increased Spell Damage'])
    );
    await expect(d.getByRole('combobox', { name: 'Modifier' })).toHaveTextContent('+30 to maximum Life');
    await userEvent.selectOptions(d.getByRole('combobox', { name: 'Source' }), 'Custom');
    await expect(d.getByRole('textbox', { name: 'Custom implicit' })).toBeVisible();
  }}>
  <div class="h-150"><AddImplicitDialog open craft={implicitApi} /></div>
</Story>

<Story
  name="Add modifier: sort and custom"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement.ownerDocument.body);
    const dialog = await c.findByRole('dialog', { name: 'Add Modifier to Item' });
    const d = within(dialog);
    const mod = await d.findByRole('combobox', { name: 'Modifier' });
    await waitFor(async () => expect(await entries(mod)).toHaveLength(2));
    await expect((await entries(mod))[0]).toBe('+(25-34) to maximum Mana (Prefix)');
    await userEvent.selectOptions(d.getByRole('combobox', { name: 'Sort by' }), 'Life');
    await waitFor(() => expect(d.getByRole('combobox', { name: 'Modifier' })).toHaveTextContent('+(25-34) to maximum Life (Prefix)'));
    await userEvent.selectOptions(d.getByRole('combobox', { name: 'Source' }), 'Custom');
    await expect(d.getByRole('textbox', { name: 'Custom modifier' })).toBeVisible();
  }}>
  <div class="h-150"><AddModifierDialog open craft={customApi} /></div>
</Story>

<Story
  name="Add modifier: a late preview leaves the open list in place"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement.ownerDocument.body);
    const dialog = await c.findByRole('dialog', { name: 'Add Modifier to Item' });
    const mod = await within(dialog).findByRole('combobox', { name: 'Modifier' });
    const list = await openList(mod);
    await waitFor(() => expect(within(list).getAllByRole('option')).toHaveLength(2));
    const option = within(list).getAllByRole('option')[1];
    await userEvent.hover(option);
    const before = [mod.getBoundingClientRect().top, option.getBoundingClientRect().top];
    const height = dialog.getBoundingClientRect().height;
    releasePreview();
    await within(dialog).findByRole('group', { name: 'New item' });
    await waitFor(() => expect(dialog.getBoundingClientRect().height).toBeGreaterThan(height + 100));
    await new Promise((r) => setTimeout(r, 300));
    await expect([mod.getBoundingClientRect().top, option.getBoundingClientRect().top], 'the hovered option moved out from under the pointer').toEqual(before);
    await expect(c.getByRole('tooltip', { name: 'Modifier tooltip' })).toBeVisible();
  }}>
  <div class="h-150"><AddModifierDialog open craft={latePreviewApi} /></div>
</Story>

<Story
  name="Affixes: sorting, roll and comparison"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const first = c.getByRole('combobox', { name: 'Affix 1' });
    await expect(first).toHaveTextContent('+# to maximum Life');
    await userEvent.selectOptions(c.getByRole('combobox', { name: 'Modifier sorting' }), 'Life');
    await waitFor(async () => expect(await entries(first)).toEqual(['None', '+# to maximum Life', '+# to maximum Mana']));
    await userEvent.click(first);
    await waitFor(() => expect(c.getByRole('group', { name: 'Affix comparison' })).toHaveTextContent('Adding this mod will give:'));
  }}>
  <div class="w-110 p-4"><AffixControls affixes={affixData} craft={affixApi} /></div>
</Story>

<Story
  name="Cluster jewel: skill and added passives"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    await expect(c.getByRole('combobox', { name: 'Cluster skill' })).toHaveDisplayValue('Fire Damage');
    await expect(c.getByRole('slider', { name: 'Added Passives' })).toHaveValue('9');
  }}>
  <div class="w-110 p-4"><DisplayItemControls info={clusterInfo} onedit={() => {}} /></div>
</Story>

<!-- Main.lua:1032-1035, ItemsTab.lua:1025-1089 -->
<Story
  name="Affix sliders: all, then one"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    await expect(c.getAllByRole('slider')).toHaveLength(2);
    allAffixSliders.set(false);
    try {
      await waitFor(() => expect(c.getAllByRole('slider')).toHaveLength(1));
      await userEvent.selectOptions(c.getByRole('combobox', { name: 'Modifier range' }), '(20-30)% increased Attack Speed');
      await expect(c.getByRole('slider', { name: 'Roll of (20-30)% increased Attack Speed' })).toHaveValue('0.75');
    } finally {
      allAffixSliders.set(true);
    }
  }}>
  <div class="w-110 p-4"><DisplayItemControls info={rangedInfo} onedit={() => {}} /></div>
</Story>

<Story
  name="Crucible: nodes"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement.ownerDocument.body);
    const dialog = await c.findByRole('dialog', { name: 'Add Crucible Modifier to Item' });
    const d = within(dialog);
    await waitFor(() => expect(d.getByRole('combobox', { name: 'Node 1' })).toHaveDisplayValue('20% increased Physical Damage - Tier: 2'));
    await userEvent.selectOptions(d.getByRole('combobox', { name: 'Node 1' }), 'a');
    await waitFor(() => expect(d.getByRole('group', { name: 'Crucible comparison' })).toHaveTextContent('Adding this mod will give:'));
  }}>
  <div class="h-150"><CrucibleDialog open craft={crucibleApi} /></div>
</Story>
