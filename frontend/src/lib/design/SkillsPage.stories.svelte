<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within, waitFor } from 'storybook/test';
  import SkillsPage from './SkillsPage.svelte';
  import PageFrame from './PageFrame.svelte';
  import LiveTopBar from './stories/LiveTopBar.svelte';
  import LiveSidebar from './stories/LiveSidebar.svelte';
  import LiveSkills from './stories/LiveSkills.svelte';
  import { spectreBuild, fireballBuild, novaBuild, type BuildFixture } from './fixtures';
  import { expectPageFits, expectFloating, expectValuesVisible, expectDarkCheckboxes, expectSelectsFloating } from './stories/layout_checks';

  const { Story } = defineMeta({
    title: 'Pages/Skills',
    component: SkillsPage,
    parameters: { layout: 'fullscreen' },
    globals: { viewport: { value: 'target', isRotated: false } }
  });

  const vp = (value: 'min' | 'mid' | 'target') => ({ viewport: { value, isRotated: false } });
  const page = (el: HTMLElement) => {
    const c = within(el);
    return { c, main: c.getByRole('main', { name: 'Skills' }) };
  };
  const groupsOf = (fx: BuildFixture) => fx.skills.sets[fx.skills.activeSet].groups;
  const firstWithGems = (fx: BuildFixture, min: number) => groupsOf(fx).findIndex((g) => g.gems.length >= min);

  const live = (el: HTMLElement) => el.querySelector<HTMLElement>('[data-reorder-live]')!;
  const gemOrder = (ed: HTMLElement) =>
    within(ed)
      .getAllByRole('checkbox', { name: / enabled$/ })
      .map((b) => b.getAttribute('aria-label')!.replace(/ enabled$/, ''));
  const ptr = (el: Element, type: string, x: number, y: number) =>
    el.dispatchEvent(
      new PointerEvent(type, {
        bubbles: true,
        cancelable: true,
        pointerId: 7,
        isPrimary: true,
        pointerType: 'mouse',
        button: 0,
        buttons: type === 'pointerup' ? 0 : 1,
        clientX: x,
        clientY: y
      })
    );
  const centre = (el: Element) => {
    const r = el.getBoundingClientRect();
    return { x: r.left + r.width / 2, y: r.top + r.height / 2 };
  };
  const ghostOf = (box: HTMLElement) => box.querySelector<HTMLElement>('[data-reorder-ghost]');
  const tops = (els: HTMLElement[]) => els.map((e) => Math.round(e.getBoundingClientRect().top));
  const ghostAt = (box: HTMLElement, x: number, y: number) =>
    waitFor(async () => {
      const g = ghostOf(box)?.getBoundingClientRect();
      await expect(g, 'a ghost follows the pointer').toBeDefined();
      await expect(x >= g!.left && x <= g!.right && y >= g!.top && y <= g!.bottom, `ghost ${JSON.stringify(g)} holds (${x}, ${y})`).toBe(true);
    });
  const dragRow = async (box: HTMLElement, rows: () => HTMLElement[], press: HTMLElement, from: number, to: number, cancel = false) => {
    const before = tops(rows());
    const stride = before[1] - before[0];
    const p = centre(press);
    const t = rows()[to].getBoundingClientRect();
    const y = to > from ? t.top + t.height * 0.75 : t.top + t.height * 0.25;
    ptr(press, 'pointerdown', p.x, p.y);
    ptr(press, 'pointermove', p.x + 1, p.y + 2);
    await expect(ghostOf(box), 'no drag before 4px of movement').toBeNull();
    ptr(press, 'pointermove', p.x + 3, (p.y + y) / 2);
    await ghostAt(box, p.x + 3, (p.y + y) / 2);
    ptr(press, 'pointermove', p.x, y);
    await ghostAt(box, p.x, y);
    await expect(getComputedStyle(ghostOf(box)!).cursor).toBe('grabbing');
    const lo = Math.min(from, to);
    const hi = Math.max(from, to);
    await waitFor(async () => {
      const now = tops(rows());
      for (const [k, top] of now.entries()) {
        if (k === from) continue;
        const want = k < lo || k > hi ? 0 : to > from ? -stride : stride;
        await expect(Math.abs(top - before[k] - want), `row ${k} moved ${top - before[k]}, want ${want}`).toBeLessThanOrEqual(3);
      }
      const slot = box.querySelector<HTMLElement>('[data-reorder-source]');
      await expect(slot, 'the source row marks the slot').not.toBeNull();
      await expect(Math.abs(slot!.getBoundingClientRect().top - before[to]), 'the slot sits where the row lands').toBeLessThanOrEqual(3);
    });
    if (cancel) await userEvent.keyboard('{Escape}');
    else {
      ptr(press, 'pointerup', p.x, y);
      press.click();
    }
    await waitFor(() => expect(ghostOf(box)).toBeNull());
    await waitFor(() => expect(tops(rows())).toEqual(before));
    await expect(box.querySelector('[data-reorder-source]')).toBeNull();
  };

  const shows = async (el: HTMLElement, fx: BuildFixture) => {
    const { c, main } = page(el);
    await expectPageFits(main);
    const list = c.getByRole('region', { name: 'Socket groups' });
    const groups = groupsOf(fx);
    await expect(within(list).getAllByRole('button').length).toBeGreaterThanOrEqual(groups.length);
    const sel = within(list).getByRole('button', { current: true });
    await expect(sel).toHaveTextContent(groups[fx.mainSocketGroup].label);
    const ed = c.getByRole('region', { name: 'Selected group' });
    await expect(within(ed).getByRole('textbox', { name: 'Label' })).toHaveValue(groups[fx.mainSocketGroup].label);
    await expect(within(ed).getAllByRole('checkbox', { name: / enabled$/ })).toHaveLength(groups[fx.mainSocketGroup].gems.length);
    await expectValuesVisible(ed);
    await expectDarkCheckboxes(main);
    await expectSelectsFloating(el, ['Sort by']);
  };
</script>

{#snippet frame(fx: BuildFixture, selected?: number)}
  <PageFrame label="Skills" mainClass="flex-row gap-3.5 px-4">
    {#snippet topbar()}<LiveTopBar {fx} current="#skills" />{/snippet}
    {#snippet sidebar()}<LiveSidebar {fx} />{/snippet}
    <LiveSkills {fx} {selected} />
  </PageFrame>
{/snippet}

<Story name="Spectre · target" asChild globals={vp('target')} play={async ({ canvasElement }) => shows(canvasElement, spectreBuild)}>
  {@render frame(spectreBuild)}
</Story>

<Story name="Fireball · minimum" asChild globals={vp('min')} play={async ({ canvasElement }) => shows(canvasElement, fireballBuild)}>
  {@render frame(fireballBuild)}
</Story>

<Story name="Nova · mid" asChild globals={vp('mid')} play={async ({ canvasElement }) => shows(canvasElement, novaBuild)}>
  {@render frame(novaBuild)}
</Story>

<Story
  name="Select a group"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }) => {
    const { c } = page(canvasElement as HTMLElement);
    const groups = groupsOf(spectreBuild);
    const to = groups.findIndex((g, i) => i !== spectreBuild.mainSocketGroup && g.gems.length > 1);
    const list = c.getByRole('region', { name: 'Socket groups' });
    const btn = within(list)
      .getAllByRole('button')
      .find((b) => b.textContent?.includes(groups[to].label))!;
    await userEvent.click(btn);
    await expect(btn).toHaveAttribute('aria-current', 'true');
    const ed = c.getByRole('region', { name: 'Selected group' });
    await expect(within(ed).getByRole('textbox', { name: 'Label' })).toHaveValue(groups[to].label);
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Edit gems"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }) => {
    const { c } = page(canvasElement as HTMLElement);
    const g = groupsOf(spectreBuild)[firstWithGems(spectreBuild, 2)];
    const ed = c.getByRole('region', { name: 'Selected group' });
    const gem = g.gems[0].name;
    const level = within(ed).getByRole('spinbutton', { name: `${gem} level` });
    await userEvent.tripleClick(level);
    await userEvent.keyboard('12{Enter}');
    await expect(level).toHaveValue('12');
    const quality = within(ed).getByRole('spinbutton', { name: `${gem} quality` });
    const q = Number((quality as HTMLInputElement).value);
    await userEvent.click(quality);
    await userEvent.keyboard('{ArrowUp}');
    await expect(quality).toHaveValue(String(q + 1));
    const on = within(ed).getByRole('checkbox', { name: `${gem} enabled` });
    await userEvent.click(on);
    await expect(on).not.toBeChecked();
    const link = within(ed).getAllByRole('button', { name: /^Sockets 1–2/ })[0];
    const was = link.getAttribute('aria-pressed');
    const summary = within(ed).getByText(/^Links: /);
    const before = summary.textContent;
    await userEvent.click(link);
    await expect(link.getAttribute('aria-pressed')).not.toBe(was);
    await expect(summary.textContent).not.toBe(before);
    const last = g.gems[g.gems.length - 1].name;
    await userEvent.click(within(ed).getByRole('button', { name: `Remove ${last}` }));
    await expect(within(ed).queryByRole('checkbox', { name: `${last} enabled` })).toBeNull();
  }}>
  {@render frame(spectreBuild, firstWithGems(spectreBuild, 2))}
</Story>

<Story
  name="Gem search"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }) => {
    const { c } = page(canvasElement as HTMLElement);
    const ed = c.getByRole('region', { name: 'Selected group' });
    const box = within(ed).getByRole('combobox', { name: 'Add a gem' });
    await userEvent.click(box);
    await userEvent.keyboard('Elemental Focus');
    const list = within(ed).getByRole('listbox', { name: 'Gem results' });
    await expect(box).toHaveAttribute('aria-expanded', 'true');
    await expectFloating(box, list);
    await userEvent.keyboard('{ArrowDown}');
    const active = box.getAttribute('aria-activedescendant')!;
    await expect(document.getElementById(active)).toHaveTextContent('Elemental Focus Support');
    await userEvent.keyboard('{Enter}');
    await expect(within(ed).getByRole('checkbox', { name: 'Elemental Focus Support enabled' })).toBeChecked();
    await waitFor(() => expect(within(ed).queryByRole('listbox')).toBeNull());
    await userEvent.keyboard('Faster');
    await expect(within(ed).getByRole('listbox', { name: 'Gem results' })).toBeVisible();
    await userEvent.keyboard('{Escape}');
    await expect(within(ed).queryByRole('listbox')).toBeNull();
    await expect(box).toHaveAttribute('aria-expanded', 'false');
  }}>
  {@render frame(fireballBuild)}
</Story>

<Story
  name="Gem search scrolls"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }) => {
    const { c } = page(canvasElement as HTMLElement);
    const ed = c.getByRole('region', { name: 'Selected group' });
    const box = within(ed).getByRole('combobox', { name: 'Add a gem' });
    await userEvent.click(box);
    const list = within(ed).getByRole('listbox', { name: 'Gem results' });
    await expect(list.scrollHeight).toBeGreaterThan(list.clientHeight);
    await expect(list.getBoundingClientRect().top).toBeGreaterThanOrEqual(0);
    // GemSelectControl.lua:536-541
    await expect(within(list).getAllByRole('option').length).toBeLessThan(60);
    await userEvent.keyboard('{ArrowUp}');
    const last = document.getElementById(box.getAttribute('aria-activedescendant')!)!;
    await waitFor(() => expect(last.getBoundingClientRect().bottom).toBeLessThanOrEqual(list.getBoundingClientRect().bottom + 1));
    await expect(list.scrollTop).toBeGreaterThan(0);
  }}>
  {@render frame(fireballBuild)}
</Story>

<Story
  name="Manage skill sets"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }) => {
    const { c } = page(canvasElement as HTMLElement);
    const select = within(c.getByRole('region', { name: 'Socket groups' })).getByRole('combobox', { name: 'Skill set' });
    await userEvent.selectOptions(select, 'Manage skill sets…');
    const dialog = await within(document.body).findByRole('dialog', { name: 'Manage skill sets' });
    const d = within(dialog);
    const before = d.getAllByRole('option').length;
    await userEvent.click(d.getByRole('button', { name: 'New' }));
    await expect(d.getByRole('button', { name: 'Save' })).toBeDisabled();
    await userEvent.type(d.getByRole('textbox', { name: 'Name for this skill set' }), 'Bossing');
    await userEvent.click(d.getByRole('button', { name: 'Save' }));
    await waitFor(() => expect(d.getAllByRole('option')).toHaveLength(before + 1));
    await expect(d.getAllByRole('option').at(-1)).toHaveTextContent('Bossing');
    await userEvent.click(d.getByRole('button', { name: 'Done' }));
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Gem search by tag"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }) => {
    const { c } = page(canvasElement as HTMLElement);
    const ed = c.getByRole('region', { name: 'Selected group' });
    const box = within(ed).getByRole('combobox', { name: 'Add a gem' });
    await userEvent.click(box);
    await userEvent.keyboard(':aura:cold:-vaal:-mine');
    const list = within(ed).getByRole('listbox', { name: 'Gem results' });
    const names = () =>
      within(list)
        .getAllByRole('option')
        .map((o) => o.textContent ?? '');
    await waitFor(() => expect(names().length).toBe(2));
    for (const [i, name] of ['Hatred', 'Purity of Ice'].entries()) await expect(names()[i]).toContain(name);
    await userEvent.keyboard('{Escape}');
  }}>
  {@render frame(fireballBuild)}
</Story>

<Story
  name="Drag to reorder"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }) => {
    const { c } = page(canvasElement as HTMLElement);
    const groups = groupsOf(spectreBuild);
    const list = c.getByRole('region', { name: 'Socket groups' });
    const cards = () => [...list.querySelectorAll<HTMLElement>('[data-reorder-item]')];
    await expect(cards(), 'one reorderable card per group').toHaveLength(groups.length);
    await expect(getComputedStyle(cards()[0]).cursor).toBe('grab');
    await dragRow(list, cards, within(cards()[0]).getByRole('button'), 0, 2);
    const want = [groups[1], groups[2], groups[0], ...groups.slice(3)].map((g) => g.label || '<No active skills>');
    await expect(cards()).toHaveLength(groups.length);
    for (const [k, e] of cards().entries()) await expect(e.textContent!.trim().startsWith(want[k]), `${k}: ${e.textContent}`).toBe(true);
    await expect(within(cards()[2]).getByRole('button', { current: true })).toBeInTheDocument();
    await expect(live(list)).toHaveTextContent(`Moved ${want[2]} to position 3 of ${groups.length}`);

    const ed = c.getByRole('region', { name: 'Selected group' });
    const gems = groups[0].gems.map((g) => g.name);
    const rows = () => [...ed.querySelectorAll<HTMLElement>('[data-reorder-item]')];
    await dragRow(ed, rows, within(rows()[0]).getByText(gems[0]), 0, 1);
    await expect(gemOrder(ed)).toEqual([gems[1], gems[0], ...gems.slice(2)]);
    const last = rows()[gems.length - 1];
    await dragRow(ed, rows, last.querySelector<HTMLElement>('.socket')!, gems.length - 1, 0);
    await expect(gemOrder(ed)).toEqual([gems[gems.length - 1], gems[1], gems[0], ...gems.slice(2, -1)]);
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Drag: cancel and controls"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }) => {
    const { c } = page(canvasElement as HTMLElement);
    const groups = groupsOf(spectreBuild);
    const ed = c.getByRole('region', { name: 'Selected group' });
    const gems = groups[spectreBuild.mainSocketGroup].gems.map((g) => g.name);
    const rows = () => [...ed.querySelectorAll<HTMLElement>('[data-reorder-item]')];
    await dragRow(ed, rows, within(rows()[0]).getByText(gems[0]), 0, 2, true);
    await expect(gemOrder(ed)).toEqual(gems);
    const list = c.getByRole('region', { name: 'Socket groups' });
    const cards = () => [...list.querySelectorAll<HTMLElement>('[data-reorder-item]')];
    await dragRow(list, cards, within(cards()[1]).getByRole('button'), 1, 0, true);
    for (const [k, e] of cards().entries()) await expect(e.textContent!.trim().startsWith(groups[k].label || '<No active skills>')).toBe(true);

    const level = within(ed).getByRole('spinbutton', { name: `${gems[1]} level` });
    const on = within(ed).getByRole('checkbox', { name: `${gems[1]} enabled` });
    for (const el of [level, on]) {
      const p = centre(el);
      ptr(el, 'pointerdown', p.x, p.y);
      ptr(el, 'pointermove', p.x, p.y + 20);
      ptr(el, 'pointermove', p.x, p.y + 60);
      await expect(ghostOf(ed), `pressing ${el.getAttribute('aria-label')} starts no drag`).toBeNull();
      await expect(ed.querySelector('[data-reorder-source]')).toBeNull();
      ptr(el, 'pointerup', p.x, p.y + 60);
    }
    await userEvent.tripleClick(level);
    await userEvent.keyboard('7{Enter}');
    await expect(level).toHaveValue('7');
    await userEvent.click(on);
    await expect(on).not.toBeChecked();
    await expect(gemOrder(ed)).toEqual(gems);
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Main group, delete, count, options"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }) => {
    const { c } = page(canvasElement as HTMLElement);
    const groups = groupsOf(spectreBuild);
    const list = c.getByRole('region', { name: 'Socket groups' });
    const card = (label: string) =>
      within(list)
        .getAllByRole('button')
        .find((b) => b.hasAttribute('aria-current') && b.textContent?.includes(label))!;
    const other = groups.findIndex((g, i) => i !== spectreBuild.mainSocketGroup && g.label && groups.filter((o) => o.label.includes(g.label)).length === 1);
    await expect(card(groups[spectreBuild.mainSocketGroup].label)).toHaveTextContent('MAIN');
    await userEvent.pointer({ keys: '[MouseRight]', target: card(groups[other].label) });
    await expect(card(groups[other].label)).toHaveTextContent('MAIN');
    await expect(card(groups[spectreBuild.mainSocketGroup].label)).not.toHaveTextContent('MAIN');
    await userEvent.click(card(groups[spectreBuild.mainSocketGroup].label));
    await userEvent.click(within(list).getByRole('button', { name: 'Set as main' }));
    await expect(card(groups[spectreBuild.mainSocketGroup].label)).toHaveTextContent('MAIN');
    await expect(within(list).getByRole('button', { name: 'Set as main' })).toBeDisabled();

    const ed = c.getByRole('region', { name: 'Selected group' });
    const gem = groups[spectreBuild.mainSocketGroup].gems[0].name;
    const count = within(ed).getByRole('spinbutton', { name: `${gem} count` });
    await userEvent.tripleClick(count);
    await userEvent.keyboard('3{Enter}');
    await expect(count).toHaveValue('3');
    await expectValuesVisible(ed);

    const options = c.getByRole('region', { name: 'Gem options' });
    const level = within(options).getByRole('combobox', { name: 'Default gem level' });
    await expect(level).toHaveDisplayValue('Normal Maximum');
    await userEvent.selectOptions(level, 'characterLevel');
    await expect(level).toHaveDisplayValue('Match Character Level');
    await expect(level).toHaveAttribute('title', expect.stringMatching(/^All gems default to their highest valid non-corrupted gem level that your character/));
    const quality = within(options).getByRole('spinbutton', { name: 'Default gem quality' });
    await userEvent.tripleClick(quality);
    await userEvent.keyboard('20{Enter}');
    await expect(quality).toHaveValue('20');
    await userEvent.click(within(options).getByRole('checkbox', { name: 'Show legacy gems' }));
    await expect(within(options).getByRole('checkbox', { name: 'Show legacy gems' })).toBeChecked();

    const n = within(list)
      .getAllByRole('button')
      .filter((b) => b.hasAttribute('aria-current')).length;
    await userEvent.click(within(list).getByRole('button', { name: 'Delete group' }));
    await expect(
      within(list)
        .getAllByRole('button')
        .filter((b) => b.hasAttribute('aria-current'))
    ).toHaveLength(n - 1);
    await expectPageFits(c.getByRole('main', { name: 'Skills' }));
    await expectDarkCheckboxes(c.getByRole('main', { name: 'Skills' }));
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Reorder by keyboard"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }) => {
    const { c } = page(canvasElement as HTMLElement);
    const groups = groupsOf(spectreBuild);
    const list = c.getByRole('region', { name: 'Socket groups' });
    const cards = () => [...list.querySelectorAll<HTMLElement>('[data-reorder-item]')];
    await expect(cards(), 'one reorderable card per group').toHaveLength(groups.length);
    cards()[1].focus();
    await userEvent.keyboard('{Alt>}{ArrowUp}{/Alt}');
    await expect(cards()[0].textContent!.trim().startsWith(groups[1].label)).toBe(true);
    await expect(cards()[1].textContent!.trim().startsWith(groups[0].label)).toBe(true);
    await expect(cards()[0]).toHaveFocus();
    await expect(live(list)).toHaveTextContent(`Moved ${groups[1].label} to position 1 of ${groups.length}`);
    await userEvent.keyboard('{Enter}');
    await expect(within(cards()[0]).getByRole('button')).toHaveAttribute('aria-current', 'true');
    await userEvent.tab();
    await expect(cards()[1]).toHaveFocus();
    await userEvent.keyboard('{Enter}');
    await expect(within(cards()[1]).getByRole('button')).toHaveAttribute('aria-current', 'true');

    const ed = c.getByRole('region', { name: 'Selected group' });
    const sel = groups[0].gems.map((g) => g.name);
    const rows = () => [...ed.querySelectorAll<HTMLElement>('[data-reorder-item]')];
    rows()[0].focus();
    await userEvent.keyboard(' ');
    await expect(rows()[0]).toHaveAttribute('data-grabbed');
    await expect(live(ed)).toHaveTextContent(`Grabbed ${sel[0]}, position 1 of ${sel.length}`);
    await userEvent.keyboard('{ArrowDown}{ArrowDown}');
    await expect(rows()[2]).toHaveFocus();
    await userEvent.keyboard(' ');
    await expect(rows()[2]).not.toHaveAttribute('data-grabbed');
    await expect(live(ed)).toHaveTextContent(`Dropped ${sel[0]} at position 3 of ${sel.length}`);
    const moved = [sel[1], sel[2], sel[0], ...sel.slice(3)];
    await expect(gemOrder(ed)).toEqual(moved);
    await userEvent.keyboard(' {ArrowUp}');
    await expect(gemOrder(ed)[1]).toBe(sel[0]);
    await userEvent.keyboard('{Escape}');
    await expect(gemOrder(ed)).toEqual(moved);
    await expect(rows()[2]).toHaveFocus();
    await expect(live(ed)).toHaveTextContent(`${sel[0]} back at position 3`);
    const level = within(rows()[0]).getByRole('spinbutton', { name: / level$/ });
    const v = Number((level as HTMLInputElement).value);
    await userEvent.click(level);
    await userEvent.keyboard('{ArrowUp}');
    await expect(level).toHaveValue(String(v + 1));
    await expect(gemOrder(ed)).toEqual(moved);
  }}>
  {@render frame(spectreBuild)}
</Story>
