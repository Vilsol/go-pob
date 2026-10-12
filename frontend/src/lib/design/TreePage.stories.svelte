<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, waitFor, within } from 'storybook/test';
  import TreePage from './TreePage.svelte';
  import PageFrame from './PageFrame.svelte';
  import LiveTopBar from './stories/LiveTopBar.svelte';
  import LiveSidebar from './stories/LiveSidebar.svelte';
  import LiveTree from './stories/LiveTree.svelte';
  import { spectreBuild, fireballBuild, novaBuild, type BuildFixture } from './fixtures';
  import { expectFloating, expectSingleLine, expectSelectsFloating } from './stories/layout_checks';

  const { Story } = defineMeta({ title: 'Pages/Tree', component: TreePage, parameters: { layout: 'fullscreen' } });

  const vp = (value: 'min' | 'mid' | 'target') => ({ viewport: { value, isRotated: false } });
  const page = (el: HTMLElement) => {
    const c = within(el);
    const main = c.getByRole('main', { name: 'Tree' });
    return { c, main, area: within(main).getByRole('region', { name: 'Passive tree' }) };
  };
  const node = (el: HTMLElement, id: number | string) => el.querySelector<HTMLElement>(`button[data-node="${id}"]`)!;
  const inside = (r: DOMRect, b: DOMRect, pad = 0) => r.left >= b.left + pad && r.right <= b.right - pad && r.top >= b.top + pad && r.bottom <= b.bottom - pad;
  const visible = async (el: HTMLElement, fx: BuildFixture, want: (id: number) => boolean) => {
    let id: number | undefined;
    await waitFor(() => {
      const box = page(el).area.getBoundingClientRect();
      id = fx.tree.nodes.map((n) => n.id).find((i) => want(i) && node(el, i) && inside(node(el, i).getBoundingClientRect(), box, 80));
      if (id === undefined) throw new Error('no such node in view');
    });
    return node(el, id!);
  };
  const nodeTip = (el: HTMLElement) => el.querySelector<HTMLElement>('[role="tooltip"][id$="-tip"]');
  const tipped = (fx: BuildFixture, alloc: boolean) => (id: number) => !!fx.tree.tips[id]?.length && !!fx.tree.nodes.find((n) => n.id === id)?.a === alloc;

  const shows = async (el: HTMLElement, fx: BuildFixture) => {
    const { c, main, area } = page(el);
    await expect(document.documentElement.scrollWidth).toBeLessThanOrEqual(window.innerWidth);
    await expect(main.scrollWidth).toBeLessThanOrEqual(main.clientWidth + 1);
    const m = main.getBoundingClientRect();
    const a = area.getBoundingClientRect();
    await expect(c.queryByRole('complementary', { name: 'Tree details' })).toBeNull();
    await expect(Math.round(a.right)).toBe(Math.round(m.right));
    for (const bar of [a.top - m.top, m.bottom - a.bottom]) await expect(Math.round(bar)).toBe(52);
    await expect(within(main).getByText(/^Canvas placeholder/)).toBeVisible();
    await waitFor(() => expect(area.querySelectorAll('.tree-node').length).toBe(fx.tree.nodes.length));
    await expect(within(main).getByText(`${fx.tree.nodes.filter((n) => n.a).length} allocated in view`, { exact: false })).toBeVisible();
    await expect(main.querySelector('[data-points]')).toHaveTextContent(`Points ${fx.progress.used} / ${fx.progress.usedMax}`);
    for (const f of main.querySelectorAll<HTMLElement>('.field-inline')) await expect(f.getBoundingClientRect().height).toBeLessThanOrEqual(36);
    const bar = main.querySelector<HTMLElement>('[data-toolbar]')!;
    const ver = within(bar).getByRole('combobox', { name: 'Version' });
    const br = bar.getBoundingClientRect();
    const vr = ver.getBoundingClientRect();
    await expect(Math.abs(br.right - parseFloat(getComputedStyle(bar).paddingRight) - vr.right)).toBeLessThanOrEqual(1);
    for (const k of bar.querySelectorAll<HTMLElement>('select, [data-points]'))
      if (k !== ver) await expect(k.getBoundingClientRect().right, k.outerHTML.slice(0, 60)).toBeLessThanOrEqual(vr.left);
    await expectSingleLine([...main.querySelectorAll<HTMLElement>('.map-note')]);
    await expectSelectsFloating(el);
  };
</script>

{#snippet frame(fx: BuildFixture, q = '')}
  <PageFrame label="Tree" mainClass="gap-0 p-0">
    {#snippet topbar()}<LiveTopBar {fx} current="#tree" />{/snippet}
    {#snippet sidebar()}<LiveSidebar {fx} />{/snippet}
    <LiveTree {fx} {q} />
  </PageFrame>
{/snippet}

<Story
  name="Spectre · target"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => shows(canvasElement, spectreBuild)}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Fireball · minimum"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => shows(canvasElement, fireballBuild)}>
  {@render frame(fireballBuild)}
</Story>

<Story name="Nova · mid" asChild globals={vp('mid')} play={async ({ canvasElement }: { canvasElement: HTMLElement }) => shows(canvasElement, novaBuild)}>
  {@render frame(novaBuild)}
</Story>

<Story
  name="Search and help"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, main, area } = page(canvasElement);
    const field = c.getByRole('searchbox', { name: 'Search the tree' });
    const help = c.getByRole('tooltip', { hidden: true });
    await expect(help).not.toBeVisible();
    await userEvent.hover(c.getByRole('button', { name: 'Search syntax' }));
    await waitFor(() => expect(help).toBeVisible());
    await expectFloating(c.getByRole('button', { name: 'Search syntax' }), help);
    await expectFloating(field, help);
    await userEvent.unhover(c.getByRole('button', { name: 'Search syntax' }));
    await userEvent.type(field, 'life');
    await expect(help).toBeVisible();
    const lit = area.querySelectorAll('.tree-node[data-match="true"]');
    await expect(lit.length).toBeGreaterThan(0);
    const results = within(c.getByRole('region', { name: 'Search results' }));
    const aside = c.getByRole('complementary', { name: 'Tree details' }).getBoundingClientRect();
    await expect(Math.round(aside.width)).toBe(360);
    await expect(Math.round(aside.right)).toBe(Math.round(main.getBoundingClientRect().right));
    await expect(area.getBoundingClientRect().right).toBeLessThanOrEqual(aside.left + 1);
    await expect(results.getByText(/^\d+ matches · \d+ allocated$/)).toBeVisible();
    const n = Number(/^(\d+)/.exec(results.getByText(/^\d+ matches/).textContent!)![1]);
    await expect(n).toBe(lit.length);
    const box = area.getBoundingClientRect();
    const centreOut = (e: Element) => {
      const r = e.getBoundingClientRect();
      const x = r.left + r.width / 2;
      const y = r.top + r.height / 2;
      return x < box.left || x > box.right || y < box.top || y > box.bottom;
    };
    const out = [...lit].filter(centreOut).length;
    const marks = within(area).queryAllByRole('note');
    await expect(out).toBeGreaterThan(0);
    await expect(marks.reduce((s, m) => s + Number(/^(\d+)/.exec(m.getAttribute('aria-label')!)![1]), 0)).toBe(out);
    for (const m of marks) await expect(inside(m.getBoundingClientRect(), box)).toBe(true);
    await userEvent.clear(field);
    await userEvent.type(field, 'notable life');
    for (const r of main.querySelectorAll<HTMLElement>('[data-result]')) await expect(node(canvasElement, r.dataset.result!)).toHaveAttribute('data-kind', 'n');
    await userEvent.click(field);
    await userEvent.clear(field);
    await userEvent.keyboard('{Escape}');
    await expect(within(area).queryAllByRole('note')).toHaveLength(0);
    await userEvent.type(field, 'keystone');
    const first = main.querySelector<HTMLElement>('[data-result]')!;
    await expect(first).not.toBeNull();
    await userEvent.hover(first);
    await expect(node(canvasElement, first.dataset.result!)).toHaveAttribute('data-hover', 'true');
  }}>
  {@render frame(fireballBuild)}
</Story>

<Story
  name="Hover costs"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const fx = spectreBuild;
    const byId = new Map(fx.tree.nodes.map((x) => [x.id, x]));
    const adjacent = (id: number) => {
      const x = byId.get(id)!;
      return !x.a && x.k !== 'm' && x.k !== 'c' && x.next !== undefined && !!byId.get(x.next)?.a;
    };
    const n = await visible(canvasElement, fx, adjacent);
    await userEvent.hover(n);
    await waitFor(() => expect(nodeTip(canvasElement)).not.toBeNull());
    await expect(nodeTip(canvasElement)).toHaveTextContent('Allocating the path: 1 point (blue)');
    await userEvent.unhover(n);

    const leaf = (id: number) => {
      const x = byId.get(id)!;
      return !!x.a && x.k !== 'm' && x.k !== 'c' && x.k !== 'j' && (x.loseIds ?? []).length === 0;
    };
    const l = await visible(canvasElement, fx, leaf);
    await userEvent.hover(l);
    await waitFor(() => expect(nodeTip(canvasElement)).toHaveTextContent('Deallocating'));
    await expect(nodeTip(canvasElement)).not.toHaveTextContent(/removes \d+ nodes/);
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Node tooltip"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const n = await visible(canvasElement, spectreBuild, tipped(spectreBuild, false));
    const id = Number(n.dataset.node);
    const fx = spectreBuild.tree.nodes.find((x) => x.id === id)!;
    await userEvent.hover(n);
    await waitFor(() => expect(nodeTip(canvasElement)).not.toBeNull());
    const tip = nodeTip(canvasElement)!;
    await expect(tip).toHaveTextContent(fx.name);
    await expect(tip).toHaveTextContent('Allocating this node will give you:');
    await expect(tip).toHaveTextContent(`Allocating the path: ${fx.pts} point`);
    await expect(n).toHaveAttribute('aria-describedby', tip.id);
    await expectFloating(n, tip);
    await expect(canvasElement.querySelectorAll('.tree-node[data-state="path"]').length).toBeGreaterThan(0);
    await userEvent.click(n);
    await waitFor(() => expect(tip).toHaveTextContent(/Notable · allocated|Keystone · allocated/));
    await expect(tip).toHaveTextContent('Deallocating');
    await expect(n).toHaveAttribute('data-alloc', 'true');
    await userEvent.click(n);
    await waitFor(() => expect(tip).toHaveTextContent('Allocating the path'));
    await userEvent.unhover(n);
    await waitFor(() => expect(nodeTip(canvasElement)).toBeNull());
    const a = await visible(canvasElement, spectreBuild, tipped(spectreBuild, true));
    a.focus();
    await waitFor(() => expect(nodeTip(canvasElement)).toHaveTextContent('Unallocating this node will give you:'));
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Context menu"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, area } = page(canvasElement);
    const n = await visible(canvasElement, novaBuild, (id) => novaBuild.tree.nodes.find((x) => x.id === id)?.k === 'n');
    const name = novaBuild.tree.nodes.find((x) => x.id === Number(n.dataset.node))!.name;
    await userEvent.pointer({ keys: '[MouseRight]', target: n });
    const menu = await c.findByRole('menu', { name: `Actions for ${name}` });
    await expectFloating(n, menu);
    await expect(inside(menu.getBoundingClientRect(), area.getBoundingClientRect())).toBe(true);
    const m = within(menu);
    await expect(m.getByRole('menuitem', { name: 'Tattoo / runegraft…' })).toBeEnabled();
    await expect(m.getByRole('menuitem', { name: 'Socket a jewel…' })).toBeDisabled();
    await expect(m.getByRole('menuitem', { name: 'Tattoo / runegraft…' })).toHaveFocus();
    await userEvent.keyboard('{ArrowDown}');
    await expect(m.getByRole('menuitem', { name: 'Copy node text' })).toHaveFocus();
    await userEvent.keyboard('{Escape}');
    await expect(c.queryByRole('menu')).toBeNull();
    await expect(n).toHaveFocus();
    await userEvent.keyboard('{Shift>}{F10}{/Shift}');
    await expect(await c.findByRole('menu', { name: `Actions for ${name}` })).toBeVisible();
    await userEvent.keyboard('{ArrowDown}{Enter}');
    await expect(canvasElement.querySelector('[data-log]')).toHaveTextContent(`copy ${name}`);
    await expect(c.queryByRole('menu')).toBeNull();
    const j = await visible(canvasElement, novaBuild, (id) => novaBuild.tree.nodes.find((x) => x.id === id)?.k === 'j');
    await userEvent.pointer({ keys: '[MouseRight]', target: j });
    await expect(within(await c.findByRole('menu')).getByRole('menuitem', { name: 'Socket a jewel…' })).toBeEnabled();
    await userEvent.click(c.getByRole('banner'));
    await expect(c.queryByRole('menu')).toBeNull();
  }}>
  {@render frame(novaBuild)}
</Story>

<Story
  name="Build controls and results"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, main } = page(canvasElement);
    const log = () => canvasElement.querySelector('[data-log]');
    await userEvent.selectOptions(c.getByRole('combobox', { name: 'Class' }), 'Witch');
    await expect(log()).toHaveTextContent('class Witch');
    await userEvent.selectOptions(c.getByRole('combobox', { name: 'Ascendancy' }), novaBuild.tree.ascendancies[1]);
    await expect(log()).toHaveTextContent(`ascendancy ${novaBuild.tree.ascendancies[1]}`);
    const level = c.getByRole('spinbutton', { name: 'Character level' });
    await expect(level).toHaveValue(String(novaBuild.level));
    await userEvent.clear(level);
    await userEvent.type(level, '42');
    await expect(log()).toHaveTextContent('level 42');
    await expect(c.getByRole('status', { name: 'Points used' })).toHaveTextContent(`Points ${novaBuild.progress.used} / ${novaBuild.progress.usedMax}`);
    const n = novaBuild.tree.nodes.find((x) => x.k === 's' && !x.a && x.next !== undefined && novaBuild.tree.nodes.find((y) => y.id === x.next)?.a)!;
    await userEvent.type(c.getByRole('searchbox', { name: 'Search the tree' }), `#${n.id}`);
    const result = within(c.getByRole('region', { name: 'Search results' })).getByRole('button', { name: `${n.name} #${n.id}` });
    await expect(result).toHaveAttribute('aria-pressed', 'false');
    await userEvent.click(result);
    await expect(result).toHaveAttribute('aria-pressed', 'true');
    await expect(node(canvasElement, n.id)).toHaveAttribute('data-alloc', 'true');
    await expect(main.querySelectorAll('[data-result]')).toHaveLength(1);
  }}>
  {@render frame(novaBuild)}
</Story>

<Story
  name="Mastery"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c } = page(canvasElement);
    const fx = fireballBuild.tree;
    const n = await visible(canvasElement, fireballBuild, (id) => {
      const x = fx.nodes.find((y) => y.id === id);
      return x?.k === 'm' && !!x.a && !!x.effects;
    });
    const mastery = fx.nodes.find((x) => x.id === Number(n.dataset.node))!;
    await userEvent.click(n);
    await expect(n).toHaveAttribute('data-alloc', 'true');
    // TreeTab.lua:1011-1036
    const dialog = () => within(c.getByRole('dialog', { name: mastery.name }));
    const effects = () =>
      dialog()
        .getAllByRole('button')
        .filter((b) => b.hasAttribute('aria-pressed'));
    await expect(effects()).toHaveLength(mastery.effects!.length);
    const other = (mastery.effect! + 1) % mastery.effects!.length;
    await userEvent.click(effects()[other]);
    await waitFor(() => expect(c.queryByRole('dialog', { name: mastery.name })).toBeNull());
    await userEvent.click(n);
    await expect(effects()[other]).toHaveAttribute('aria-pressed', 'true');
    await userEvent.keyboard('{Escape}');
    await waitFor(() => expect(c.queryByRole('dialog', { name: mastery.name })).toBeNull());
  }}>
  {@render frame(fireballBuild)}
</Story>
