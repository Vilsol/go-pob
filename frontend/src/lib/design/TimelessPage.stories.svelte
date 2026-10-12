<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, waitFor, within } from 'storybook/test';
  import TimelessPage from './TimelessPage.svelte';
  import PageFrame from './PageFrame.svelte';
  import LiveTopBar from './stories/LiveTopBar.svelte';
  import LiveSidebar from './stories/LiveSidebar.svelte';
  import LiveTimeless from './stories/LiveTimeless.svelte';
  import { spectreBuild, fireballBuild, novaBuild, type BuildFixture } from './fixtures';
  import { socketLabel } from './timeless';
  import { expectPageFits, expectFloating, expectValuesVisible, expectSingleLine, expectSelectsFloating } from './stories/layout_checks';

  const { Story } = defineMeta({
    title: 'Pages/Timeless search',
    component: TimelessPage,
    parameters: {
      layout: 'fullscreen',
      a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } }
    }
  });

  const vp = (value: 'min' | 'mid' | 'target') => ({ viewport: { value, isRotated: false } });
  const page = (el: HTMLElement) => {
    const c = within(el);
    return { c, main: c.getByRole('main', { name: 'Timeless search' }) };
  };
  const view = (el: HTMLElement) => within(el).getByRole('region', { name: 'Passive tree around the build' });
  const socket = (el: HTMLElement, id: number) => view(el).querySelector<HTMLElement>(`button[data-node="${id}"]`)!;
  const current = (el: HTMLElement) => within(el).getByRole('list', { name: 'Steps' }).querySelector('[aria-current="step"]')!.textContent;

  const treeShows = async (el: HTMLElement, fx: BuildFixture) => {
    const { main } = page(el);
    await expectPageFits(main);
    const v = view(el);
    await waitFor(() => expect(v.querySelectorAll('.tree-node').length).toBe(fx.tree.nodes.length));
    await expect(v.querySelectorAll('.tree-edge').length).toBe(fx.tree.edges.length);
    await expect(v.querySelectorAll('.tree-node[data-alloc="true"]').length).toBe(fx.tree.nodes.filter((n) => n.a).length);
    const box = v.getBoundingClientRect();
    for (const n of v.querySelectorAll<HTMLElement>('.tree-node')) {
      const r = n.getBoundingClientRect();
      await expect(r.left >= box.left - 14 && r.right <= box.right + 14).toBe(true);
    }
    await expect(v.querySelectorAll('button.tree-node')).toHaveLength(fx.tree.sockets.length);
  };
</script>

{#snippet frame(fx: BuildFixture, step = 1)}
  <PageFrame label="Timeless search" mainClass="scroll gap-2.5 bg-tree-ground px-4 py-3">
    {#snippet topbar()}<LiveTopBar {fx} current="#tree" />{/snippet}
    {#snippet sidebar()}<LiveSidebar {fx} />{/snippet}
    <LiveTimeless {fx} {step} />
  </PageFrame>
{/snippet}

<Story
  name="Jewel · target"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, main } = page(canvasElement);
    await expectPageFits(main);
    const jewels = within(c.getByRole('group', { name: 'Jewels' }));
    await expect(jewels.getAllByRole('button')).toHaveLength(6);
    await expect(c.queryByRole('combobox', { name: 'Devotion modifier 1' })).toBeNull();
    await userEvent.click(jewels.getByRole('button', { name: /^Militant Faith/ }));
    await expect(c.getByRole('combobox', { name: 'Devotion modifier 1' })).toBeVisible();
    await expect(c.getByRole('combobox', { name: 'Devotion modifier 2' })).toBeVisible();
    await expectSelectsFloating(canvasElement);
    const conq = within(c.getByRole('region', { name: 'Conqueror' }));
    await expect(conq.getAllByRole('button')).toHaveLength(4);
    await userEvent.click(conq.getByRole('button', { name: /^Dominus/ }));
    await expect(c.getByText(/· Dominus \(Inner Conviction\)$/)).toBeVisible();
    await expectSingleLine([...c.getByRole('list', { name: 'Steps' }).querySelectorAll<HTMLElement>('button')]);
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Sockets · mid"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await treeShows(canvasElement, spectreBuild);
    const { c } = page(canvasElement);
    const [s0, s1] = spectreBuild.tree.sockets;
    const node = socket(canvasElement, s1.id);
    await userEvent.hover(node);
    const tip = await c.findByRole('tooltip');
    await expect(tip).toHaveTextContent(socketLabel(s1));
    await expect(tip).toHaveTextContent(`${s1.notables.length} notables`);
    await expectFloating(node, tip);
    await expect(node).toHaveAttribute('aria-describedby', tip.id);
    await expect(view(canvasElement).querySelector('.radius-ring[data-hot]')).not.toBeNull();
    await expect(view(canvasElement).querySelectorAll('.tree-node[data-conq="true"]').length).toBeGreaterThan(0);
    await userEvent.unhover(node);
    await waitFor(() => expect(c.queryByRole('tooltip')).toBeNull());
    await expect(socket(canvasElement, s0.id)).toHaveAttribute('data-picked', 'true');
    await userEvent.click(node);
    await expect(node).toHaveAttribute('data-picked', 'true');
    await expect(socket(canvasElement, s0.id)).not.toHaveAttribute('data-picked');
    const list = within(c.getByRole('region', { name: 'Sockets' }));
    await expect(list.getByRole('button', { name: new RegExp(`^${socketLabel(s1).replace(/[#.]/g, '\\$&')}`) })).toHaveAttribute('aria-pressed', 'true');
    await expect(view(canvasElement).querySelectorAll('.radius-ring')).toHaveLength(1);
    await userEvent.click(list.getByRole('button', { name: 'All sockets' }));
    await expect(socket(canvasElement, s0.id)).toHaveAttribute('data-picked', 'true');
    await expect(node).toHaveAttribute('data-picked', 'true');
    await userEvent.click(node);
    socket(canvasElement, s0.id).focus();
    await expect(await c.findByRole('tooltip')).toHaveTextContent(socketLabel(s0));
  }}>
  {@render frame(spectreBuild, 2)}
</Story>

<Story
  name="Sockets · minimum"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => treeShows(canvasElement, fireballBuild)}>
  {@render frame(fireballBuild, 2)}
</Story>

<Story
  name="Requirements"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, main } = page(canvasElement);
    const req = c.getByRole('region', { name: 'Requirements' });
    await expectValuesVisible(req);
    await expectPageFits(main);
    await expectSelectsFloating(canvasElement);
    const w = within(req).getByRole('spinbutton', { name: '#% increased maximum Life weight' });
    await userEvent.tripleClick(w);
    await userEvent.keyboard('5{Enter}');
    await expect(w).toHaveValue('5');
    const add = within(req).getByRole('combobox', { name: 'Add a required stat' });
    await userEvent.click(add);
    const options = () =>
      within(c.getByRole('listbox', { name: 'Add a required stat options' }))
        .getAllByRole('option')
        .map((o) => o.textContent?.trim());
    await expect(options()).not.toContain('#% increased maximum Life');
    await userEvent.keyboard('armour');
    await expect(options()).toEqual(['#% increased Armour']);
    await userEvent.keyboard('{Enter}');
    await expect(within(req).getByText('#% increased Armour')).toBeVisible();
    await expect(within(req).getByRole('spinbutton', { name: '#% increased Armour weight' })).toHaveValue('1');
    await userEvent.click(within(req).getByRole('checkbox', { name: '#% increased Armour required' }));
    await expect(within(req).getByRole('spinbutton', { name: '#% increased Armour minimum' })).toBeDisabled();
    await userEvent.click(within(req).getByRole('button', { name: 'Remove #% increased Armour' }));
    await expect(within(req).queryByText('#% increased Armour')).toBeNull();
    await expectValuesVisible(req);
  }}>
  {@render frame(novaBuild, 3)}
</Story>

<Story
  name="Ranking"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c } = page(canvasElement);
    const rank = within(c.getByRole('region', { name: 'Ranking' }));
    await userEvent.selectOptions(rank.getByRole('combobox', { name: 'Fallback Weight Mode' }), 'Sort by Life');
    await userEvent.click(rank.getByRole('button', { name: 'Generate' }));
    await expect(canvasElement.querySelector('[data-log]')).toHaveTextContent('generated');
    await userEvent.click(rank.getByRole('button', { name: 'Fallback Nodes' }));
    await expect(rank.getByRole('textbox', { name: 'Fallback nodes' })).toBeVisible();
    await expect(c.getByRole('button', { name: 'Search' })).toBeVisible();
    await expectSelectsFloating(canvasElement);
  }}>
  {@render frame(fireballBuild, 4)}
</Story>

<Story
  name="Results"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, main } = page(canvasElement);
    await expectPageFits(main);
    const rows = within(c.getByRole('listbox', { name: 'Search results' })).getAllByRole('option');
    await expect(rows[0]).toHaveAttribute('data-on', 'true');
    const s2 = novaBuild.tree.sockets[1];
    await userEvent.click(rows[2]);
    await expect(rows[2]).toHaveAttribute('data-on', 'true');
    await expect(
      within(c.getByRole('group', { name: 'Result preview' })).getByText(new RegExp(`in ${socketLabel(s2).replace(/[#.]/g, '\\$&')}$`))
    ).toBeVisible();
    const ring = view(canvasElement).querySelector<HTMLElement>('.radius-ring[data-hot]')!;
    const node = socket(canvasElement, s2.id).getBoundingClientRect();
    const r = ring.getBoundingClientRect();
    await expect(Math.abs(r.left + r.width / 2 - (node.left + node.width / 2))).toBeLessThanOrEqual(1.5);
    await expect(view(canvasElement).querySelectorAll('.tree-node[data-conq="true"]').length).toBeGreaterThan(0);
    await userEvent.keyboard('{ArrowDown}');
    await waitFor(() => expect(rows[3]).toHaveAttribute('data-on', 'true'));
    await userEvent.click(c.getByRole('button', { name: 'Add to the build' }));
    await expect(canvasElement.querySelector('[data-log]')).toHaveTextContent(/^added \d+$/);
    await expect(c.getByRole('button', { name: 'Added to the build' })).toBeDisabled();
    for (const o of rows.slice(0, 6)) {
      const cells = ['seed', 'socket', 'weight', 'nodes'].map((k) => o.querySelector<HTMLElement>(`[data-cell="${k}"]`)!);
      await expectSingleLine([...cells.slice(0, 3), ...cells[3].querySelectorAll<HTMLElement>(':scope > span')]);
      await expect(new Set([...cells[3].children].map((e) => Math.round(e.getBoundingClientRect().top))).size).toBeLessThanOrEqual(1);
      for (const cell of cells) await expect(cell.getBoundingClientRect().height).toBeLessThanOrEqual(o.getBoundingClientRect().height);
    }
    await expect(rows[1].querySelector('[data-cell="seed"]')).toHaveTextContent(/^\d+$/);
    await expect(rows[1].querySelectorAll('[data-cell="nodes"] > span').length).toBeGreaterThan(1);
    const ringOn = (id: number) => {
      const ringBox = view(canvasElement).querySelector<HTMLElement>('.radius-ring[data-hot]')!.getBoundingClientRect();
      const n = socket(canvasElement, id).getBoundingClientRect();
      return Math.abs(ringBox.left + ringBox.width / 2 - (n.left + n.width / 2));
    };
    const s3 = novaBuild.tree.sockets[2 % novaBuild.tree.sockets.length];
    await expect(s3.id).not.toBe(s2.id);
    await userEvent.hover(rows[4]);
    await waitFor(() => expect(ringOn(s2.id)).toBeLessThanOrEqual(1.5));
    await userEvent.unhover(c.getByRole('listbox', { name: 'Search results' }));
    await waitFor(() => expect(ringOn(s3.id)).toBeLessThanOrEqual(1.5));
  }}>
  {@render frame(novaBuild, 5)}
</Story>

<Story
  name="Step navigation"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c } = page(canvasElement);
    await expect(current(canvasElement)).toBe('1 Jewel');
    await expect(c.getByRole('button', { name: 'Back' })).toBeDisabled();
    for (const next of ['2 Sockets', '3 Requirements', '4 Ranking']) {
      await userEvent.click(c.getByRole('button', { name: 'Next' }));
      await expect(current(canvasElement)).toBe(next);
    }
    await userEvent.click(c.getByRole('button', { name: 'Search' }));
    await expect(current(canvasElement)).toBe('5 Results');
    await userEvent.click(c.getByRole('button', { name: 'Back' }));
    await expect(current(canvasElement)).toBe('4 Ranking');
    await userEvent.click(within(c.getByRole('list', { name: 'Steps' })).getByRole('button', { name: '2 Sockets' }));
    await expect(current(canvasElement)).toBe('2 Sockets');
    await userEvent.click(c.getByRole('button', { name: 'Close timeless search' }));
    await expect(canvasElement.querySelector('[data-log]')).toHaveTextContent('closed');
  }}>
  {@render frame(spectreBuild)}
</Story>
