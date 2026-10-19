<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within, waitFor } from 'storybook/test';
  import ItemsPage from '../ItemsPage.svelte';
  import LiveMobile from '../stories/LiveMobile.svelte';
  import LiveItems from '../stories/LiveItems.svelte';
  import { spectreBuild, fireballBuild } from '../fixtures';
  import { phone, expectPhoneFits } from '../stories/mobile_checks';
  import ItemDB from '../ItemDB.svelte';
  import { defaultItemDBFilter, type ItemDBRowPlain } from '../item_db';

  const dbRows: ItemDBRowPlain[] = Array.from({ length: 40 }, (_, i) => ({
    name: `Unique ${i + 1}`,
    title: `Unique ${i + 1}`,
    base: 'Iron Ring',
    rarity: 'UNIQUE',
    type: 'Ring',
    slot: 'Ring',
    level: i + 1,
    league: '',
    source: '',
    power: 0
  }));
  // a tab names its panel and the panel names the selected tab back
  const expectLinked = async (tab: HTMLElement) => {
    const panel = document.getElementById(tab.getAttribute('aria-controls') ?? '');
    await expect(panel).toHaveAttribute('role', 'tabpanel');
    await expect(panel).toHaveAttribute('aria-labelledby', tab.id);
  };
  const dbLists = {
    slots: ['Any slot'],
    types: ['Any type'],
    leagues: ['Any league'],
    requirements: ['Any'],
    obtainable: ['Any'],
    searchModes: ['Anywhere'],
    sortStats: []
  };

  const { Story } = defineMeta({
    title: 'Mobile/Items',
    component: ItemsPage,
    parameters: {
      layout: 'fullscreen',
      a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } },
      docs: {
        description: {
          component:
            'Responsive: on a phone the Board keeps its slots at their own size in four sideways panes (Weapons, Armour, Jewellery and flasks, Jewels) with a section strip above that follows the scroll and jumps on tap; the List is cards, or a sticky-name table (Settings › Wide lists on a phone). Tapping a slot or row opens the item editor, and every dialog but a small confirm fills the screen.'
        }
      }
    },
    globals: phone
  });
</script>

<Story
  name="Phone"
  asChild
  play={async ({ canvasElement }) => {
    await expectPhoneFits(canvasElement, 'Items');
    const t = within(canvasElement).getByRole('table', { name: 'Equipped items' });
    await expect(t).toBeVisible();
    await expect(t.getBoundingClientRect().right).toBeLessThanOrEqual(390);
    await expect(t.scrollWidth).toBeLessThanOrEqual(t.clientWidth);
    await expectLinked(within(canvasElement).getByRole('tab', { name: 'Equipped' }));
  }}>
  <LiveMobile fx={spectreBuild} current="#items" label="Items" mainClass="gap-3 p-3"><LiveItems fx={spectreBuild} layout="list" /></LiveMobile>
</Story>

<Story
  name="List as a table"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await expectPhoneFits(canvasElement, 'Items');
    const t = within(canvasElement).getByRole('table', { name: 'Equipped items' });
    await expect(t.scrollWidth).toBeGreaterThan(t.clientWidth);
    const name = t.querySelector<HTMLElement>('[role=row] [data-col=name]')!;
    t.scrollLeft = 300;
    await waitFor(() => expect(Math.abs(name.getBoundingClientRect().left - t.getBoundingClientRect().left)).toBeLessThanOrEqual(2));
  }}>
  <LiveMobile fx={spectreBuild} current="#items" label="Items" mainClass="gap-3 p-3"
    ><LiveItems fx={spectreBuild} layout="list" phoneLayout="table" /></LiveMobile>
</Story>

<Story
  name="Board"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await expectPhoneFits(canvasElement, 'Items');
    const c = within(canvasElement);
    const panes = c.getByRole('tabpanel', { name: 'Weapons' });
    const inView = (name: string) => {
      const r = within(panes).getByRole('group', { name }).getBoundingClientRect();
      const p = panes.getBoundingClientRect();
      return r.left >= p.left - 1 && r.right <= p.right + 1;
    };
    await expect(inView('Weapon 1 slot')).toBe(true);
    await expect(inView('Helmet slot')).toBe(false);
    await userEvent.click(c.getByRole('tab', { name: 'Armour' }));
    await waitFor(() => expect(inView('Helmet slot')).toBe(true));
    await expect(c.getByRole('tab', { name: 'Armour' })).toHaveAttribute('aria-selected', 'true');
    await userEvent.click(within(panes).getByRole('button', { name: /^Helmet: / }));
    await waitFor(() => expect(within(document.body).getByRole('dialog', { name: /^Edit item / })).toBeVisible());
  }}>
  <LiveMobile fx={fireballBuild} current="#items" label="Items" mainClass="gap-3 p-3"><LiveItems fx={fireballBuild} /></LiveMobile>
</Story>

<Story
  name="Item editor sheet"
  asChild
  play={async () => {
    const d = await waitFor(() => within(document.body).getByRole('dialog', { name: /^Edit item / }));
    const r = d.getBoundingClientRect();
    await expect([Math.round(r.left), Math.round(r.top), Math.round(r.width), Math.round(r.height)]).toEqual([0, 0, 390, 769]);
  }}>
  <LiveMobile fx={spectreBuild} current="#items" label="Items" mainClass="gap-3 p-3"><LiveItems fx={spectreBuild} layout="list" edit="Helmet" /></LiveMobile>
</Story>

<Story
  name="Add modifier"
  asChild
  play={async () => {
    const editor = await waitFor(() => within(document.body).getByRole('dialog', { name: /^Edit item / }));
    await userEvent.click(within(editor).getByRole('button', { name: '+ Add modifier' }));
    const picker = await waitFor(() => within(document.body).getByRole('dialog', { name: 'Add a modifier' }));
    const q = within(picker).getByRole('searchbox', { name: 'Search modifiers' });
    await expect(q.getBoundingClientRect().width).toBeGreaterThanOrEqual(300);
    await expect(picker.scrollWidth).toBeLessThanOrEqual(390);
    const opt = within(picker).getAllByRole('option')[0];
    await expect(opt.getBoundingClientRect().right).toBeLessThanOrEqual(390);
  }}>
  <LiveMobile fx={spectreBuild} current="#items" label="Items" mainClass="gap-3 p-3"><LiveItems fx={spectreBuild} layout="list" edit="Helmet" /></LiveMobile>
</Story>

<Story
  name="Browse items"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await expectPhoneFits(canvasElement, 'Items');
    const c = within(canvasElement);
    const filters = c.getByRole('complementary', { name: 'Filters' }).getBoundingClientRect();
    const list = c.getAllByRole('table').at(-1)!.getBoundingClientRect();
    await expect(list.top).toBeGreaterThanOrEqual(filters.bottom);
    await expect(list.right).toBeLessThanOrEqual(390);
    const main = c.getByRole('main', { name: 'Items' });
    const rows = c.getAllByRole('table').at(-1)!.querySelectorAll<HTMLElement>('[role=row]');
    const last = rows[rows.length - 1];
    main.scrollTop = main.scrollHeight;
    await waitFor(() => expect(last.getBoundingClientRect().bottom).toBeLessThanOrEqual(main.getBoundingClientRect().bottom + 1));
    await expect(last.getBoundingClientRect().bottom).toBeLessThanOrEqual(last.closest('section')!.getBoundingClientRect().bottom + 1);
    await expect(last.getBoundingClientRect().height).toBeGreaterThan(20);
    await expectLinked(c.getByRole('tab', { name: 'Browse items' }));
    await expectLinked(within(c.getByRole('tablist', { name: 'Item type' })).getAllByRole('tab', { selected: true })[0]);
  }}>
  <LiveMobile fx={spectreBuild} current="#items" label="Items" mainClass="gap-3 p-3"><LiveItems fx={spectreBuild} view="browse" /></LiveMobile>
</Story>

<Story
  name="Item database"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await expectPhoneFits(canvasElement, 'Items');
    const c = within(canvasElement);
    const main = c.getByRole('main', { name: 'Items' });
    const rows = c.getByRole('table', { name: 'Unique list' }).querySelectorAll<HTMLElement>('[role=row]');
    const last = rows[rows.length - 1];
    main.scrollTop = main.scrollHeight;
    await waitFor(() => expect(last.getBoundingClientRect().bottom).toBeLessThanOrEqual(main.getBoundingClientRect().bottom + 1));
    await expect(last.getBoundingClientRect().bottom).toBeLessThanOrEqual(last.closest('section')!.getBoundingClientRect().bottom + 1);
    await expect(last.getBoundingClientRect().right).toBeLessThanOrEqual(390);
    await expectLinked(c.getByRole('tab', { name: 'Uniques' }));
  }}>
  <LiveMobile fx={spectreBuild} current="#items" label="Items" mainClass="gap-3 p-3"
    ><ItemDB
      kind="UNIQUE"
      lists={dbLists}
      filter={defaultItemDBFilter()}
      sort=""
      rows={dbRows}
      onadd={() => {}}
      onopen={() => {}}
      oncopy={() => {}} /></LiveMobile>
</Story>
