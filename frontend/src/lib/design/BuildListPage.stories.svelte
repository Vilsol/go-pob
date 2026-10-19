<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, fireEvent, userEvent, waitFor, within } from 'storybook/test';
  import BuildListPage from './BuildListPage.svelte';
  import LiveBuildList from './stories/LiveBuildList.svelte';
  import {
    expectPageFits,
    expectFloating,
    expectModalTrap,
    expectSingleLine,
    expectNotClipped,
    expectValuesVisible,
    expectDarkCheckboxes
  } from './stories/layout_checks';

  const { Story } = defineMeta({ title: 'Pages/Build list', component: BuildListPage, parameters: { layout: 'fullscreen' } });

  const vp = (value: 'min' | 'mid' | 'target') => ({ viewport: { value, isRotated: false } });
  const tab = (shift: boolean) => userEvent.tab({ shift });
  const page = (el: HTMLElement) => {
    const c = within(el);
    const main = c.getByRole('main');
    return { c, main, grid: c.getByRole('grid', { name: 'Builds' }), tree: c.getByRole('tree', { name: 'Folder tree' }) };
  };
  const names = (grid: HTMLElement) => [...grid.querySelectorAll<HTMLElement>('[data-build]')].map((r) => r.dataset.build);
  const treeRow = (tree: HTMLElement, path: string) => tree.querySelector<HTMLElement>(`[data-folder="${path}"]`)!;
  const folderButton = (tree: HTMLElement, name: string) => within(tree).getByRole('button', { name: new RegExp(`^${name}\\s*\\d+$`) });
  const hashFolder = () => decodeURIComponent(location.hash.replace(/^#folder=/, ''));

  const shows = async (el: HTMLElement) => {
    const { c, main, grid, tree } = page(el);
    await expectPageFits(main);
    await expect(document.documentElement.scrollWidth).toBeLessThanOrEqual(window.innerWidth);
    await expect(c.getByRole('banner').getBoundingClientRect().height).toBe(56);
    await expect(tree.getBoundingClientRect().right).toBeLessThanOrEqual(main.getBoundingClientRect().left + 1);
    await expect(names(grid).length).toBeGreaterThan(0);
    await expectNotClipped([...grid.querySelectorAll<HTMLElement>('[role="columnheader"] button')]);
    await expectSingleLine([...tree.querySelectorAll<HTMLElement>('[role="treeitem"] .truncate')]);
    await expectSingleLine([...el.querySelectorAll<HTMLElement>('header .btn')]);
  };
</script>

<Story name="All builds · target" asChild globals={vp('target')} play={async ({ canvasElement }: { canvasElement: HTMLElement }) => shows(canvasElement)}>
  <LiveBuildList />
</Story>

<Story
  name="In a folder · mid"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await shows(canvasElement);
    const { grid } = page(canvasElement);
    await expect(within(grid).getByRole('button', { name: 'Necromancer' })).toBeVisible();
    await expect(names(grid)).toEqual(['Winter Orb Elementalist']);
  }}>
  <LiveBuildList folder="League starters" />
</Story>

<Story name="Minimum" asChild globals={vp('min')} play={async ({ canvasElement }: { canvasElement: HTMLElement }) => shows(canvasElement)}>
  <LiveBuildList folder="Bossers" />
</Story>

<Story
  name="Folder navigation and Back"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, tree } = page(canvasElement);
    const entries = history.length;
    await userEvent.click(folderButton(tree, 'Bossers'));
    await waitFor(() => expect(hashFolder()).toBe('Bossers'));
    const crumbs = c.getByRole('navigation', { name: 'Breadcrumb' });
    await expect(within(crumbs).getByRole('button', { name: 'Bossers' })).toHaveAttribute('aria-current', 'page');
    await userEvent.click(within(page(canvasElement).grid).getByRole('button', { name: 'Uber' }));
    await waitFor(() => expect(hashFolder()).toBe('Bossers/Uber'));
    await expect(names(page(canvasElement).grid)).toEqual(['Blade Vortex Occultist']);
    await expect(folderButton(tree, 'Uber')).toHaveAttribute('aria-current', 'page');
    await expect(history.length, 'two folder picks pushed two entries').toBe(entries + 2);
    history.back();
    await waitFor(() => expect(within(crumbs).getByRole('button', { name: 'Bossers' })).toHaveAttribute('aria-current', 'page'));
    history.back();
    await waitFor(() => expect(within(crumbs).getByRole('button', { name: 'All builds' })).toHaveAttribute('aria-current', 'page'));
    await userEvent.click(folderButton(tree, 'Imported'));
    await expect(treeRow(tree, 'Imported')).toHaveAttribute('aria-expanded', 'true');
    await expect(treeRow(tree, 'Imported/pobb.in')).not.toBeNull();
    await userEvent.click(within(tree).getByRole('button', { name: 'Collapse Imported' }));
    await expect(treeRow(tree, 'Imported')).toHaveAttribute('aria-expanded', 'false');
    await expect(treeRow(tree, 'Imported/pobb.in')).toBeNull();
    await userEvent.click(within(crumbs).getByRole('button', { name: 'All builds' }));
    await waitFor(() => expect(hashFolder()).toBe(''));
  }}>
  <LiveBuildList />
</Story>

<Story
  name="Search results"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, grid } = page(canvasElement);
    const q = c.getByRole('searchbox', { name: 'Search builds' });
    await expect(q).toHaveAttribute('placeholder', 'Search in League starters and its subfolders');
    await userEvent.type(q, 'spectre');
    await waitFor(() => expect(names(grid)).toEqual(['Spectre Necro – mapping', 'Spectre Necro – budget']));
    const subs = [...grid.querySelectorAll<HTMLElement>('[data-subpath]')].map((s) => s.textContent);
    await expect(subs).toEqual(['Necromancer/ ', 'Necromancer/Budget/ ']);
    await expect(getComputedStyle(grid.querySelector('[data-subpath]')!).fontWeight).toBe('400');
    await expectSingleLine([...grid.querySelectorAll<HTMLElement>('[data-col="name"]')]);
    await userEvent.clear(q);
    await userEvent.type(q, 'class:necro budget');
    await waitFor(() => expect(names(grid)).toEqual(['Spectre Necro – budget']));
    await userEvent.clear(q);
    await userEvent.type(q, 'nothing like this');
    await expect(c.getByText('No builds match. Clear the search or pick another class.')).toBeVisible();
    await userEvent.clear(q);
    await userEvent.type(q, 'e');
    await waitFor(() => expect(names(grid)).toHaveLength(3));
    await userEvent.click(c.getByRole('button', { name: 'Shadow' }));
    await expect(c.getByRole('button', { name: 'Shadow' })).toHaveAttribute('aria-pressed', 'true');
    await expect(names(grid)).toEqual([]);
    await userEvent.click(c.getByRole('button', { name: 'Witch' }));
    await expect(names(grid)).toHaveLength(3);
  }}>
  <LiveBuildList folder="League starters" />
</Story>

<Story
  name="Sort and columns"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, main, grid } = page(canvasElement);
    const level = () => within(grid).getByRole('columnheader', { name: /^LEVEL/ });
    const levels = () => [...grid.querySelectorAll<HTMLElement>('[data-build] [data-col="level"]')].map((e) => Number(e.textContent));
    await userEvent.click(within(level()).getByRole('button'));
    await expect(level()).toHaveAttribute('aria-sort', 'descending');
    await expect(levels()).toEqual([...levels()].sort((a, b) => b - a));
    await userEvent.click(within(level()).getByRole('button'));
    await expect(level()).toHaveAttribute('aria-sort', 'ascending');
    await expect(levels()).toEqual([...levels()].sort((a, b) => a - b));
    const open = c.getByRole('button', { name: /^Columns · 9 \/ 11/ });
    await userEvent.click(open);
    const menu = c.getByRole('group', { name: 'Visible columns' });
    await expectFloating(open, menu);
    await expectDarkCheckboxes(menu);
    await expect(within(menu).getByRole('checkbox', { name: 'Name' })).toBeDisabled();
    await userEvent.click(within(menu).getByRole('checkbox', { name: 'Class' }));
    await expect(within(grid).getByRole('columnheader', { name: /^CLASS/ })).toBeVisible();
    await userEvent.click(within(menu).getByRole('checkbox', { name: 'Life' }));
    await expect(within(grid).queryByRole('columnheader', { name: /^LIFE/ })).toBeNull();
    await expect(open).toHaveTextContent('Columns · 9 / 11');
    await expectPageFits(main);
    await userEvent.click(within(menu).getByRole('button', { name: 'Reset to default' }));
    await expect(within(grid).queryByRole('columnheader', { name: /^CLASS/ })).toBeNull();
    await userEvent.keyboard('{Escape}');
    await expect(c.queryByRole('group', { name: 'Visible columns' })).toBeNull();
    await userEvent.click(open);
    await userEvent.click(c.getByRole('banner'));
    await expect(c.queryByRole('group', { name: 'Visible columns' })).toBeNull();
  }}>
  <LiveBuildList />
</Story>

<Story
  name="Drag into a folder"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, grid, tree } = page(canvasElement);
    const row = grid.querySelector<HTMLElement>('[data-build="Spectre Necro – mapping"]')!;
    const at = (el: HTMLElement) => {
      const r = el.getBoundingClientRect();
      return { clientX: r.left + r.width / 2, clientY: r.top + r.height / 2, pointerId: 1, isPrimary: true, button: 0, pointerType: 'mouse' };
    };
    await fireEvent.pointerDown(row, at(row));
    const target = treeRow(tree, 'Imported');
    await fireEvent.pointerMove(target, at(target));
    await waitFor(() => expect(target.className).toContain('drop-target'));
    await expect(within(target).getByText('drop here')).toBeVisible();
    await fireEvent.pointerUp(target, at(target));
    await waitFor(() => expect(names(grid)).not.toContain('Spectre Necro – mapping'));
    await expect(c.getByText('Spectre Necro – mapping moved to Imported')).toBeVisible();
    await expect(names(grid)).toEqual([]);
    await expect(target.className).not.toContain('drop-target');
    const next = grid.querySelector<HTMLElement>('[data-build="Spectre Necro – budget"]');
    await expect(next).toBeNull();
    await userEvent.click(folderButton(tree, 'League starters'));
    const winter = page(canvasElement).grid.querySelector<HTMLElement>('[data-build="Winter Orb Elementalist"]')!;
    await fireEvent.pointerDown(winter, at(winter));
    const sub = page(canvasElement).grid.querySelector<HTMLElement>('[data-folder="League starters/Necromancer"]')!;
    await fireEvent.pointerMove(sub, at(sub));
    await waitFor(() => expect(within(sub).getByText('Drop to move into Necromancer')).toBeVisible());
    await fireEvent.pointerUp(sub, at(sub));
    await waitFor(() => expect(names(page(canvasElement).grid)).toEqual([]));
    await userEvent.click(folderButton(tree, 'Necromancer'));
    await waitFor(() => expect(names(page(canvasElement).grid)).toEqual(['Winter Orb Elementalist']));
    const actions = c.getByRole('button', { name: 'Actions for Winter Orb Elementalist' });
    actions.focus();
    await userEvent.keyboard('{ArrowDown}');
    const menu = c.getByRole('menu', { name: 'Actions for Winter Orb Elementalist' });
    await expectFloating(actions, menu);
    await expect(within(menu).getByRole('menuitem', { name: 'Necromancer (current)' })).toBeDisabled();
    await userEvent.click(within(menu).getByRole('menuitem', { name: 'Bossers' }));
    await waitFor(() => expect(names(page(canvasElement).grid)).not.toContain('Winter Orb Elementalist'));
  }}>
  <LiveBuildList folder="League starters/Necromancer" />
</Story>

<Story
  name="Recalculate all"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, grid } = page(canvasElement);
    const stale = () => grid.querySelectorAll<HTMLElement>('[data-stale]');
    await expect(stale().length).toBeGreaterThan(0);
    const cell = stale()[0];
    await expect(cell.textContent).toMatch(/^↻ /);
    await expect(cell.getAttribute('title')).toMatch(/^Saved with 3\.2[78] data/);
    await userEvent.click(c.getByRole('button', { name: /^↻ Recalculate all · 3 stale$/ }));
    await expect(c.getByRole('status')).toHaveTextContent(/Recalculating \d \/ 3 in the background/);
    await waitFor(() => expect(stale()).toHaveLength(0), { timeout: 3000 });
    await expect(c.queryByRole('button', { name: /Recalculate all/ })).toBeNull();
  }}>
  <LiveBuildList />
</Story>

<Story
  name="Settings open"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c } = page(canvasElement);
    const opener = c.getByRole('button', { name: 'Settings' });
    await userEvent.click(opener);
    const dlg = c.getByRole('dialog', { name: 'Settings' });
    await waitFor(() => expect(dlg).toBeVisible());
    await expectModalTrap(dlg, tab);
    const d = within(dlg);
    const general = d.getByRole('tab', { name: /^General/ });
    general.focus();
    await userEvent.keyboard('{ArrowDown}');
    await expect(d.getByRole('tab', { name: /^Numbers/ })).toHaveAttribute('aria-selected', 'true');
    await expect(d.getByText('Preview: Full DPS 2,060,418')).toBeVisible();
    await userEvent.selectOptions(d.getByRole('combobox', { name: 'Thousands separator' }), 'space');
    await expect(d.getByText('Preview: Full DPS 2 060 418')).toBeVisible();
    await userEvent.click(d.getByRole('checkbox', { name: 'Show thousands separators' }));
    await expect(d.getByText('Preview: Full DPS 2060418')).toBeVisible();
    await expectDarkCheckboxes(dlg);
    await userEvent.click(d.getByRole('button', { name: 'Reset this section' }));
    await expect(d.getByText('Preview: Full DPS 2,060,418')).toBeVisible();
    await userEvent.click(d.getByRole('tab', { name: /^Items and new builds/ }));
    await expectValuesVisible(dlg);
    await expectNotClipped([...dlg.querySelectorAll<HTMLElement>('[role="tab"]')]);
    await userEvent.keyboard('{Escape}');
    await waitFor(() => expect(dlg).not.toBeVisible());
    await expect(opener).toHaveFocus();
  }}>
  <LiveBuildList />
</Story>

<Story
  name="Settings: data and storage"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c } = page(canvasElement);
    await userEvent.click(c.getByRole('button', { name: 'Settings' }));
    const dlg = c.getByRole('dialog', { name: 'Settings' });
    await waitFor(() => expect(dlg).toBeVisible());
    const d = within(dlg);
    await userEvent.click(d.getByRole('tab', { name: /^Data and storage/ }));
    await expectNotClipped([...d.getByRole('tabpanel').querySelectorAll<HTMLElement>('button, [id$="-dataVersion"], [id$="-cache"]')]);
    const taken = c.getByRole('status', { name: 'Actions taken' });
    await userEvent.click(d.getByRole('button', { name: 'Export all builds' }));
    await expect(taken).toHaveTextContent('setting exportAll');
    await userEvent.click(d.getByRole('button', { name: 'Clear cached data' }));
    await expect(d.getByRole('alert')).toHaveTextContent('Clear the cached game data? Builds are kept.');
    await expect(taken).not.toHaveTextContent('clearCache');
    await expectNotClipped([d.getByRole('button', { name: 'Keep' }), d.getByRole('button', { name: 'Clear cached data' })]);
    await userEvent.click(d.getByRole('button', { name: 'Keep' }));
    await expect(d.queryByRole('alert')).toBeNull();
    await userEvent.click(d.getByRole('button', { name: 'Clear cached data' }));
    await userEvent.click(d.getByRole('button', { name: 'Clear cached data' }));
    await expect(taken).toHaveTextContent('setting clearCache');
  }}>
  <LiveBuildList app />
</Story>

<Story
  name="About open"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c } = page(canvasElement);
    const opener = c.getByRole('button', { name: 'About go-pob' });
    await userEvent.click(opener);
    const dlg = c.getByRole('dialog', { name: 'About go-pob' });
    await waitFor(() => expect(dlg).toBeVisible());
    await expectModalTrap(dlg, tab);
    const d = within(dlg);
    await expect(d.getByText('GAME DATA')).toBeVisible();
    await userEvent.click(d.getByRole('tab', { name: 'Dependencies' }));
    const deps = d.getByRole('table', { name: 'Dependencies' });
    await expect(within(deps).getByText('github.com/Vilsol/go-pob-data')).toBeVisible();
    await userEvent.click(d.getByRole('button', { name: 'Copy diagnostics' }));
    await expect(d.getByRole('button', { name: 'Copied ✓' })).toBeVisible();
    await userEvent.keyboard('{Escape}');
    await waitFor(() => expect(dlg).not.toBeVisible());
    await expect(opener).toHaveFocus();
  }}>
  <LiveBuildList />
</Story>

<Story
  name="In the app"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await shows(canvasElement);
    const { c, grid, tree } = page(canvasElement);
    const log = () => c.getByRole('status', { name: 'Actions taken' }).textContent;
    await expect([...grid.querySelectorAll<HTMLElement>('[role="columnheader"] button')].map((b) => b.title)).toEqual([
      'Sort by Name',
      'Sort by Class',
      'Sort by Level',
      'Sort by Modified'
    ]);
    await userEvent.click(c.getByRole('button', { name: /^Columns · 4 \/ 4/ }));
    await expect(within(c.getByRole('group', { name: 'Visible columns' })).getAllByRole('checkbox')).toHaveLength(4);
    await userEvent.keyboard('{Escape}');
    const nf = within(tree.parentElement!).getByRole('button', { name: 'New folder' });
    await expectSingleLine([nf]);
    await userEvent.click(nf);
    await userEvent.click(c.getByRole('button', { name: 'New build' }));
    await userEvent.click(c.getByRole('button', { name: 'Import build' }));
    await expect(log()).toBe('new folder; new; import');
    await userEvent.click(c.getByRole('button', { name: 'Actions for Spectre Necro – budget' }));
    const menu = c.getByRole('menu', { name: 'Actions for Spectre Necro – budget' });
    await userEvent.click(within(menu).getByRole('menuitem', { name: 'Rename…' }));
    await userEvent.click(c.getByRole('button', { name: 'Actions for Spectre Necro – budget' }));
    await userEvent.click(within(c.getByRole('menu', { name: 'Actions for Spectre Necro – budget' })).getByRole('menuitem', { name: 'Delete…' }));
    await userEvent.click(c.getByRole('button', { name: 'Copy Spectre Necro – budget' }));
    await expect(log()).toBe(
      'new folder; new; import; rename League starters/Necromancer/Budget/Spectre Necro – budget; delete League starters/Necromancer/Budget/Spectre Necro – budget; copy League starters/Necromancer/Budget/Spectre Necro – budget'
    );
  }}>
  <LiveBuildList app />
</Story>
