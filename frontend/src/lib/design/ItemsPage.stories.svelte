<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, fn, userEvent, within, waitFor, screen } from 'storybook/test';
  import ItemsPage from './ItemsPage.svelte';
  import PageFrame from './PageFrame.svelte';
  import LiveTopBar from './stories/LiveTopBar.svelte';
  import LiveSidebar from './stories/LiveSidebar.svelte';
  import LiveItems from './stories/LiveItems.svelte';
  import { spectreBuild, fireballBuild, novaBuild, type BuildFixture } from './fixtures';
  import { BOARD_SLOTS } from './items';
  import { expectPageFits, expectFloating, expectModalTrap, expectSingleLine, expectDarkCheckboxes, expectSelectsFloating } from './stories/layout_checks';

  const { Story } = defineMeta({
    title: 'Pages/Items',
    component: ItemsPage,
    parameters: {
      layout: 'fullscreen',
      a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } }
    },
    globals: { viewport: { value: 'target', isRotated: false } }
  });

  const vp = (value: 'min' | 'mid' | 'target') => ({ viewport: { value, isRotated: false } });
  const tab = (shift: boolean) => userEvent.tab({ shift });
  const page = (el: HTMLElement) => {
    const c = within(el);
    return { c, main: c.getByRole('main', { name: 'Items' }) };
  };
  const equippedName = (fx: BuildFixture, slot: string) => {
    const set = fx.items.sets.find((s) => s.id === fx.items.activeSet)!;
    const it = set.slots.find((x) => x.slot === slot);
    return it ? fx.items.items[it.item].name : 'Empty';
  };
  const tile = (c: { getByRole: typeof screen.getByRole }, slot: string) => c.getByRole('button', { name: new RegExp(`^${slot}: `) });
  const editText = fn();
  const lineCount = (fx: BuildFixture, slot: string) => fx.items.impact[slot].sections.reduce((n, s) => n + s.lines.length, 0);
</script>

{#snippet frame(fx: BuildFixture, opts: Record<string, unknown> = {})}
  <PageFrame label="Items" mainClass="scroll">
    {#snippet topbar()}<LiveTopBar {fx} current="#items" />{/snippet}
    {#snippet sidebar()}<LiveSidebar {fx} />{/snippet}
    <LiveItems {fx} {...opts} />
  </PageFrame>
{/snippet}

<Story
  name="Spectre · board"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }) => {
    const { c, main } = page(canvasElement);
    await expectPageFits(main);
    await expectDarkCheckboxes(main);
    await expectSelectsFloating(canvasElement);
    await expect(c.getByRole('tab', { name: 'Equipped' })).toHaveAttribute('aria-selected', 'true');
    const board = c.getByRole('region', { name: 'Equipped items' });
    for (const b of BOARD_SLOTS) await expect(within(board).getByRole('button', { name: new RegExp(`^${b.slot}: `) })).toBeVisible();
    const helmet = tile(c, 'Helmet');
    await expect(helmet).toHaveAccessibleName(`Helmet: ${equippedName(spectreBuild, 'Helmet')}`);
    await userEvent.hover(helmet);
    const tip = await c.findByRole('tooltip');
    await expect(tip).toHaveTextContent('UNEQUIP IMPACT');
    await expect(helmet).toHaveAttribute('aria-describedby', tip.id);
    await expectFloating(helmet, tip);
    await userEvent.unhover(helmet);
    await waitFor(() => expect(c.queryByRole('tooltip')).toBeNull());
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Fireball · minimum"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }) => {
    const { c, main } = page(canvasElement);
    await expectPageFits(main);
    await expect(c.getByRole('region', { name: 'Tree jewels' })).toBeInTheDocument();
    await expect(c.getByRole('region', { name: 'Paste or craft an item' })).toBeVisible();
    const w2 = tile(c, 'Weapon 2');
    await userEvent.hover(w2);
    await expectFloating(w2, await c.findByRole('tooltip'));
  }}>
  {@render frame(fireballBuild)}
</Story>

<!-- ItemsTab.lua:193-206 -->
<Story
  name="Passive tree · Manage"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }) => {
    const { c, main } = page(canvasElement as HTMLElement);
    await expectPageFits(main);
    const select = c.getByRole('combobox', { name: 'Passive tree' });
    const manage = c.getByRole('button', { name: 'Manage passive trees' });
    await expect(manage).toHaveTextContent('Manage…');
    await expect(Math.abs(manage.getBoundingClientRect().top - select.getBoundingClientRect().top)).toBeLessThan(8);
    await expect(manage.getBoundingClientRect().left).toBeGreaterThan(select.getBoundingClientRect().right);
    await userEvent.click(manage);
    await expect(c.getByRole('status', { name: 'Tree manager asked' })).toHaveTextContent('1');
  }}>
  {@render frame(novaBuild, { trees: true })}
</Story>

<Story
  name="Nova · set switching"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }) => {
    const { c, main } = page(canvasElement as HTMLElement);
    await expectPageFits(main);
    const other = novaBuild.items.sets[1];
    const empty = BOARD_SLOTS.find((b) => !other.slots.some((x) => x.slot === b.slot) && equippedName(novaBuild, b.slot) !== 'Empty')!.slot;
    await expect(tile(c, empty)).not.toHaveAccessibleName(`${empty}: Empty`);
    await userEvent.selectOptions(c.getByRole('combobox', { name: 'Item set' }), other.title);
    await expect(tile(c, empty)).toHaveAccessibleName(`${empty}: Empty`);
    await expectPageFits(main);
  }}>
  {@render frame(novaBuild)}
</Story>

<Story
  name="Hovered flask"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }) => {
    const { c } = page(canvasElement);
    const tip = c.getByRole('tooltip');
    await expect(tip).toHaveTextContent(novaBuild.items.impact['Flask 1'].header.replace(/:$/, '').toUpperCase());
    await expectFloating(tile(c, 'Flask 1'), tip);
  }}>
  {@render frame(novaBuild, { hover: 'Flask 1' })}
</Story>

<Story
  name="Flask toggle"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }) => {
    const { c } = page(canvasElement);
    const box = c.getByRole('checkbox', { name: 'Flask 2 active' });
    await expect(box).toBeChecked();
    await userEvent.click(box);
    await expect(box).not.toBeChecked();
    await expect(tile(c, 'Flask 2')).toHaveAttribute('data-off', 'true');
    await userEvent.hover(tile(c, 'Flask 2'));
    await expect(await c.findByRole('tooltip')).toHaveTextContent('Inactive (not counted)');
    await userEvent.click(box);
    await expect(tile(c, 'Flask 2')).not.toHaveAttribute('data-off');
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Equip through the slot"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }) => {
    const { c } = page(canvasElement);
    const trigger = c.getByRole('button', { name: 'Change Ring 1' });
    await userEvent.click(trigger);
    const menu = c.getByRole('menu', { name: 'Ring 1' });
    await expectFloating(trigger, menu);
    await userEvent.click(within(menu).getByRole('menuitem', { name: 'Unequip' }));
    await expect(tile(c, 'Ring 1')).toHaveAccessibleName('Ring 1: Empty');
    await userEvent.click(trigger);
    const ring2 = equippedName(spectreBuild, 'Ring 2');
    await userEvent.click(within(c.getByRole('menu', { name: 'Ring 1' })).getByRole('menuitem', { name: new RegExp(`^Equip ${ring2}`) }));
    await expect(tile(c, 'Ring 1')).toHaveAccessibleName(`Ring 1: ${ring2}`);
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="List view"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }) => {
    const { c, main } = page(canvasElement as HTMLElement);
    await userEvent.click(c.getByRole('button', { name: 'List' }));
    await expect(c.getByRole('button', { name: 'List' })).toHaveAttribute('aria-pressed', 'true');
    const list = c.getByRole('table', { name: 'Equipped items' });
    await expectPageFits(main);
    const sockets = [...list.querySelectorAll<HTMLElement>('[role="row"]')]
      .map((r) => r.querySelectorAll<HTMLElement>('[role="cell"]')[4])
      .filter((cell) => cell?.textContent?.trim());
    await expectSingleLine(sockets);
    const box = within(list).getByRole('checkbox', { name: 'Flask 1 active' });
    const was = (box as HTMLInputElement).checked;
    await userEvent.click(box);
    await expect((box as HTMLInputElement).checked).toBe(!was);
    const link = within(list).getByRole('button', { name: equippedName(spectreBuild, 'Helmet') });
    await userEvent.hover(link);
    await expectFloating(link, await c.findByRole('tooltip'));
    await userEvent.unhover(link);
    await waitFor(() => expect(c.queryByRole('tooltip')).toBeNull());
    await userEvent.hover(within(link.closest('[role="row"]') as HTMLElement).getAllByRole('cell')[1]);
    await expect(await c.findByRole('tooltip')).toBeVisible();
    await userEvent.click(c.getByRole('button', { name: 'Board' }));
    await expect(c.getByRole('region', { name: 'Equipped items' })).toBeVisible();
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Stash"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }) => {
    const { c, main } = page(canvasElement as HTMLElement);
    await userEvent.click(c.getByRole('tab', { name: /^Stash/ }));
    const table = c.getByRole('table', { name: 'Stash' });
    await expectPageFits(main);
    await expectSelectsFloating(canvasElement as HTMLElement, ['Sort']);
    const first = novaBuild.items.stash.find((s) => s.impact)!;
    const name = novaBuild.items.items[first.item].name;
    const row = within(table).getByRole('button', { name });
    await userEvent.hover(row);
    const tip = await c.findByRole('tooltip');
    await expect(tip).toHaveTextContent('EQUIP IMPACT');
    await expectFloating(row, tip);
    await userEvent.unhover(row);
    await waitFor(() => expect(c.queryByRole('tooltip')).toBeNull());
    await userEvent.hover(within(row.closest('[role="row"]') as HTMLElement).getAllByRole('cell')[1]);
    await expect(await c.findByRole('tooltip')).toHaveTextContent('EQUIP IMPACT');
  }}>
  {@render frame(novaBuild)}
</Story>

<Story
  name="Shared"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }) => {
    const { c, main } = page(canvasElement);
    await userEvent.click(c.getByRole('tab', { name: 'Shared' }));
    await expectPageFits(main);
    const info = c.getByRole('button', { name: 'About shared items' });
    await userEvent.hover(info);
    await expectFloating(info, await c.findByRole('tooltip'));
  }}>
  {@render frame(spectreBuild, { view: 'shared' })}
</Story>

<Story
  name="Browse items"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }) => {
    const { c, main } = page(canvasElement);
    await userEvent.click(c.getByRole('tab', { name: 'Browse items' }));
    await expectPageFits(main);
    await expectSelectsFloating(canvasElement);
    await userEvent.click(c.getByRole('tab', { name: 'Flasks' }));
    await expect(c.getByRole('tab', { name: 'Flasks' })).toHaveAttribute('aria-selected', 'true');
    const stashTab = c.getByRole('tab', { name: /^Stash/ });
    const before = stashTab.textContent;
    const add = c.getAllByRole('button', { name: 'Add' })[0];
    await userEvent.click(add);
    await expect(add).toHaveTextContent('Added ✓');
    await expect(add).toBeDisabled();
    await expect(within(main).getByRole('status')).toHaveTextContent(/^Added /);
    await expect(stashTab.textContent).not.toBe(before);
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Manage sets"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }) => {
    const { c } = page(canvasElement);
    const opener = c.getByRole('button', { name: 'Manage…' });
    await userEvent.click(opener);
    const dlg = c.getByRole('dialog', { name: 'Manage item sets' });
    await waitFor(() => expect(dlg).toBeVisible());
    await expectModalTrap(dlg, tab);
    const rows = () => within(within(dlg).getByRole('listbox', { name: 'Item sets' })).getAllByRole('option');
    const n = rows().length;
    await userEvent.click(within(dlg).getByRole('button', { name: 'New' }));
    const save = within(dlg).getByRole('button', { name: 'Save' });
    await expect(save).toBeDisabled();
    await userEvent.type(within(dlg).getByRole('textbox', { name: 'Name for this item set' }), 'Bossing');
    await userEvent.click(save);
    await expect(rows()).toHaveLength(n + 1);
    const ag = within(dlg).getByRole('option', { name: new RegExp(`^${novaBuild.items.sets[1].title}`) });
    await userEvent.dblClick(ag);
    await waitFor(() => expect(ag).toHaveTextContent('current'));
    await userEvent.click(within(dlg).getByRole('button', { name: 'Done' }));
    await waitFor(() => expect(dlg).not.toBeVisible());
    await expect(opener).toHaveFocus();
    await expect(c.getByRole('combobox', { name: 'Item set' })).toHaveDisplayValue(novaBuild.items.sets[1].title);
  }}>
  {@render frame(novaBuild)}
</Story>

<Story
  name="Item editor"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }) => {
    const { c } = page(canvasElement);
    const body = tile(c, 'Body Armour');
    await userEvent.click(body);
    const ed = c.getByRole('dialog', { name: /^Edit item / });
    await waitFor(() => expect(ed).toBeVisible());
    await expectModalTrap(ed, tab);
    const history = within(ed).getByRole('group', { name: 'Item editor history' });
    await expect(history).toHaveTextContent('0 / 0');
    const add = within(ed).getByRole('button', { name: '+ Add modifier' });
    await userEvent.click(add);
    const picker = c.getByRole('dialog', { name: 'Add a modifier' });
    await waitFor(() => expect(picker).toBeVisible());
    await expectModalTrap(picker, tab);
    await expectSelectsFloating(picker);
    await userEvent.type(within(picker).getByRole('searchbox', { name: 'Search modifiers' }), 'Cold');
    const opts = within(within(picker).getByRole('listbox', { name: 'Modifiers' })).getAllByRole('option');
    await expect(opts).toHaveLength(1);
    await userEvent.click(opts[0]);
    await expect(opts[0]).toHaveAttribute('aria-selected', 'true');
    await userEvent.click(within(picker).getByRole('button', { name: 'Add modifier' }));
    await waitFor(() => expect(picker).not.toBeVisible());
    const line = within(ed).getByRole('button', { name: '+(46–48)% to Cold Resistance' });
    await expect(history).toHaveTextContent('1 / 1');
    await userEvent.keyboard('{Control>}z{/Control}');
    await expect(line).not.toBeInTheDocument();
    await expect(history).toHaveTextContent('0 / 1');
    await userEvent.keyboard('{Control>}{Shift>}z{/Shift}{/Control}');
    await expect(within(ed).getByRole('button', { name: '+(46–48)% to Cold Resistance' })).toBeVisible();
    await userEvent.click(add);
    await waitFor(() => expect(picker).toBeVisible());
    await userEvent.keyboard('{Escape}');
    await waitFor(() => expect(picker).not.toBeVisible());
    await expect(ed).toBeVisible();
    await userEvent.keyboard('{Escape}');
    await waitFor(() => expect(ed).not.toBeVisible());
    await expect(body).toHaveFocus();
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Editor open"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }) => {
    const { c } = page(canvasElement);
    const ed = c.getByRole('dialog', { name: /^Edit item / });
    await waitFor(() => expect(ed).toBeVisible());
    await expect(within(ed).getByRole('region', { name: 'Impact' })).toHaveTextContent('Removing this item from Body Armour');
    await expectSelectsFloating(ed);
    await expectModalTrap(ed, tab);
  }}>
  {@render frame(fireballBuild, { edit: 'Body Armour' })}
</Story>

<Story
  name="App: Go's what-ifs, read-only controls"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }) => {
    const { c, main } = page(canvasElement as HTMLElement);
    await expectPageFits(main);
    await expect(c.getByRole('combobox', { name: 'Item set' })).toBeDisabled();
    for (const name of ['Manage…', 'New item', 'Craft item…']) await expect(c.queryByRole('button', { name })).toBeNull();
    for (const name of ['Shared', 'Browse items']) await expect(c.queryByRole('tab', { name })).toBeNull();
    await expect(c.queryByRole('group', { name: 'Weapon set' })).toBeNull();
    await expect(c.getByText('Weapon set I')).toBeVisible();
    for (const box of within(c.getByRole('region', { name: 'Tree jewels' })).queryAllByRole('combobox')) await expect(box).toBeDisabled();

    const helmet = tile(c, 'Helmet');
    await userEvent.hover(helmet);
    const tip = await c.findByRole('tooltip');
    const diff = within(tip).getByRole('group', { name: 'Stat differences' });
    await expect(diff).toHaveAttribute('data-slot', 'Helmet');
    await expect(diff).toHaveAttribute('aria-busy', 'false');
    await expect(diff.querySelectorAll('[data-line]')).toHaveLength(lineCount(spectreBuild, 'Helmet'));
    await expect(diff.querySelector('[data-header]')).toHaveAttribute('data-header', spectreBuild.items.impact.Helmet.header);
    await expectFloating(helmet, tip);
    await userEvent.unhover(helmet);

    await userEvent.click(c.getByRole('button', { name: 'Change Ring 1' }));
    await expect(
      within(c.getByRole('menu', { name: 'Ring 1' }))
        .getAllByRole('menuitem')
        .map((m) => m.textContent?.trim())
    ).toEqual(['Unequip']);
    await userEvent.keyboard('{Escape}');

    const paste = c.getByRole('region', { name: 'Paste an item' });
    await expect(within(paste).getByRole('article', { name: `Item: ${spectreBuild.items.paste.item.name}` })).toBeVisible();
    const impact = within(paste).getByRole('region', { name: 'Equip impact' });
    await expect(within(impact).getByRole('button', { name: `Equip in ${spectreBuild.items.paste.impact!.slot}` })).toBeVisible();
    await expect(within(paste).getByRole('button', { name: 'Save to build' })).toBeVisible();

    await userEvent.click(tile(c, 'Body Armour'));
    const ed = c.getByRole('dialog', { name: /^Item / });
    await waitFor(() => expect(ed).toBeVisible());
    await expectModalTrap(ed, tab);
    await expect(within(ed).getByRole('button', { name: 'Mod table' })).toBeDisabled();
    await expect(within(ed).queryByRole('button', { name: '+ Add modifier' })).toBeNull();
    await expect(within(ed).queryByRole('group', { name: 'Item editor history' })).toBeNull();
    await expect(within(ed).queryByRole('button', { name: /^Save/ })).toBeNull();
    const region = within(ed).getByRole('region', { name: 'Impact' });
    await expect(region).toHaveAttribute('data-slot', 'Body Armour');
    await expect(region.querySelectorAll('[data-line]')).toHaveLength(lineCount(spectreBuild, 'Body Armour'));
    editText.mockClear();
    await userEvent.click(within(ed).getByRole('button', { name: 'Edit as text' }));
    await expect(editText).toHaveBeenCalledTimes(1);
    await waitFor(() => expect(ed).not.toBeVisible());
  }}>
  {@render frame(spectreBuild, { app: true, onedittext: editText })}
</Story>
