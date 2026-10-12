<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import ComparePage from './ComparePage.svelte';
  import PageFrame from './PageFrame.svelte';
  import LiveTopBar from './stories/LiveTopBar.svelte';
  import LiveSidebar from './stories/LiveSidebar.svelte';
  import LiveCompare from './stories/LiveCompare.svelte';
  import { spectreBuild, fireballBuild, novaBuild, type BuildFixture } from './fixtures';
  import type { CompareView } from './compare';
  import { expectPageFits, expectSingleLine, expectSelectsFloating, expectNotClipped } from './stories/layout_checks';

  const { Story } = defineMeta({
    title: 'Pages/Compare',
    component: ComparePage,
    parameters: {
      layout: 'fullscreen',
      a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } }
    },
    globals: { viewport: { value: 'target', isRotated: false } }
  });

  const vp = (value: 'min' | 'mid' | 'target') => ({ viewport: { value, isRotated: false } });
  const page = (el: HTMLElement) => {
    const c = within(el);
    return { c, main: c.getByRole('main', { name: 'Compare' }) };
  };
  const header = (el: HTMLElement) => within(el).getByRole('region', { name: 'A versus B' });

  const shows = async (el: HTMLElement, fx: BuildFixture) => {
    const { c, main } = page(el);
    await expectPageFits(main);
    await expect(within(header(el)).getByText(fx.compare.a.name)).toBeVisible();
    await expect(within(header(el)).getByText(fx.compare.b.name)).toBeVisible();
    await expect(c.getByRole('tab', { name: 'Summary' })).toHaveAttribute('aria-selected', 'true');
    const stats = c.getByRole('region', { name: 'Stat comparison' });
    const rows = within(stats).getAllByRole('group', { name: /: A .*, B / });
    await expect(rows.length).toBeGreaterThan(5);
    await expect(within(stats).getAllByText(/^A [+−]/).length).toBeGreaterThan(0);
    await expect(within(stats).getAllByText(/^B [+−]/).length).toBeGreaterThan(0);
    await expect(c.getByText(/^A better on \d+ · B better on \d+ · \d+ equal$/)).toBeVisible();
    await expectSingleLine([...stats.querySelectorAll<HTMLElement>('.butterfly-value')].filter((e) => e.textContent));
    const aside = c.getByRole('complementary', { name: 'Biggest differences' });
    await expect(within(aside).getByRole('button', { name: /^Full power report \(\d+ differences\) →$/ })).toBeVisible();
    await expectSelectsFloating(el);
  };
</script>

{#snippet frame(fx: BuildFixture, opts: { view?: CompareView; swapped?: boolean; calcsB?: BuildFixture } = {})}
  <PageFrame label="Compare" mainClass="gap-2.5 px-4.5 py-3">
    {#snippet topbar()}<LiveTopBar {fx} current="#compare" />{/snippet}
    {#snippet sidebar()}<LiveSidebar {fx} />{/snippet}
    <LiveCompare {fx} {...opts} />
  </PageFrame>
{/snippet}

<Story name="Spectre · target" asChild globals={vp('target')} play={async ({ canvasElement }) => shows(canvasElement as HTMLElement, spectreBuild)}>
  {@render frame(spectreBuild)}
</Story>

<Story name="Fireball · minimum" asChild globals={vp('min')} play={async ({ canvasElement }) => shows(canvasElement as HTMLElement, fireballBuild)}>
  {@render frame(fireballBuild)}
</Story>

<Story name="Nova · mid" asChild globals={vp('mid')} play={async ({ canvasElement }) => shows(canvasElement as HTMLElement, novaBuild)}>
  {@render frame(novaBuild)}
</Story>

<Story
  name="Views"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }) => {
    const el = canvasElement as HTMLElement;
    const { c, main } = page(el);
    const fx = spectreBuild.compare;
    await userEvent.click(c.getByRole('button', { name: /^Full power report/ }));
    await expect(c.getByRole('tab', { name: 'Power report' })).toHaveAttribute('aria-selected', 'true');
    const rows = within(c.getByRole('table', { name: 'Power report rows' })).getAllByRole('row');
    await expect(rows.length).toBeGreaterThan(3);
    await expectPageFits(main);
    c.getByRole('tab', { name: 'Power report' }).focus();
    await userEvent.keyboard('{ArrowRight}');
    await expect(c.getByRole('tab', { name: 'Tree' })).toHaveFocus();
    await expect(c.getByRole('region', { name: 'Node differences' })).toBeVisible();
    await expect(c.getByRole('region', { name: 'Tree overlay' }).querySelectorAll('.tree-node').length).toBe(fx.tree.nodes.filter((n) => n.x !== null).length);
    await expectPageFits(main);
    await userEvent.keyboard('{ArrowRight}');
    const items = c.getByRole('table', { name: 'Items by slot' });
    await expect(within(items).getAllByRole('row')).toHaveLength(fx.items.length + 1);
    await expectPageFits(main);
    await userEvent.click(c.getByRole('tab', { name: 'Skills' }));
    await expect(c.getByRole('heading', { name: 'Skill setups: what differs' })).toBeVisible();
    await expect(c.getByText(/^IN BOTH BUILDS · \d+$/)).toBeVisible();
    await expectPageFits(main);
    await userEvent.click(c.getByRole('tab', { name: 'Config' }));
    const cfg = c.getByRole('table', { name: 'Config options' });
    await expect(within(cfg).getAllByRole('row')).toHaveLength(fx.config.length + 1);
    await expectPageFits(main);
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Swap A and B"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }) => {
    const el = canvasElement as HTMLElement;
    const { c } = page(el);
    const fx = fireballBuild.compare;
    const first = () => within(c.getByRole('region', { name: 'Stat comparison' })).getAllByRole('group', { name: /: A .*, B / })[0];
    const before = first().getAttribute('aria-label')!;
    const [, label, va, vb] = /^(.*): A (.*), B (.*?)(, [AB] better|, equal)$/.exec(before)!;
    const tally = c.getByText(/^A better on/).textContent!;
    await userEvent.click(c.getByRole('button', { name: 'Swap A and B' }));
    const heads = within(header(el)).getAllByText(/^(A|B) · /);
    await expect(heads[0]).toHaveTextContent('A · COMPARISON');
    await expect(
      within(header(el))
        .getAllByText(fx.b.name)[0]
        .compareDocumentPosition(within(header(el)).getAllByText(fx.a.name)[0])
    ).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
    await expect(first().getAttribute('aria-label')).toContain(`: A ${vb}, B ${va}`);
    void label;
    const [aw, bw] = /A better on (\d+) · B better on (\d+)/.exec(tally)!.slice(1);
    await expect(c.getByText(/^A better on/)).toHaveTextContent(`A better on ${bw} · B better on ${aw}`);
    await userEvent.click(c.getByRole('tab', { name: 'Items' }));
    const row = within(c.getByRole('table', { name: 'Items by slot' })).getAllByRole('row')[1];
    const [, a, b] = within(row).getAllByRole('cell');
    const orig = fx.items.find((r) => r.slot === within(row).getAllByRole('cell')[0].textContent)!;
    await expect(a).toHaveTextContent(orig.b?.name ?? 'Empty');
    await expect(b).toHaveTextContent(orig.a?.name ?? 'Empty');
    await userEvent.click(c.getByRole('button', { name: 'Swap A and B' }));
    await expect(within(header(el)).getAllByText(/^(A|B) · /)[0]).toHaveTextContent('A · THIS BUILD');
  }}>
  {@render frame(fireballBuild)}
</Story>

<Story
  name="Power report"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }) => {
    const { c, main } = page(canvasElement as HTMLElement);
    await expect(c.getByRole('heading', { name: /^Why is B (better|worse) at Full DPS\?$/ })).toBeVisible();
    const table = () => c.getByRole('table', { name: 'Power report rows' });
    const count = () => within(table()).getAllByRole('row').length;
    const all = count();
    await userEvent.click(c.getByRole('checkbox', { name: /^Config/ }));
    await expect(count()).toBeLessThan(all);
    await userEvent.click(c.getByRole('checkbox', { name: /^Config/ }));
    await expect(count()).toBe(all);
    await userEvent.selectOptions(c.getByRole('combobox', { name: 'Metric' }), 'Effective Hit Pool');
    await expect(c.getByRole('heading', { name: /^Why is B (better|worse) at Effective Hit Pool\?$/ })).toBeVisible();
    await userEvent.click(c.getByRole('button', { name: /^PP/ }));
    await expect(within(table()).getAllByRole('columnheader')[5]).toHaveAttribute('aria-sort', 'descending');
    await expect(within(within(table()).getAllByRole('row')[1]).getAllByRole('cell')[0]).toHaveTextContent('Tree');
    await expectPageFits(main);
    await expectSelectsFloating(canvasElement as HTMLElement);
  }}>
  {@render frame(novaBuild, { view: 'power' })}
</Story>

<!-- ComparePowerReportListControl.lua:124-132 -->
<Story
  name="Power report tooltip"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }) => {
    const { c } = page(canvasElement as HTMLElement);
    const table = c.getByRole('table', { name: 'Power report rows' });
    const tree = within(table)
      .getAllByRole('row')
      .find((r) => r.querySelector('[role=cell]')?.textContent === 'Tree')!;
    const points = tree.querySelectorAll('[role=cell]')[4].textContent?.trim();
    await userEvent.hover(within(tree).getAllByRole('button')[0]);
    const tip = await within(document.body).findByRole('tooltip');
    await expect(tip).toHaveTextContent('Allocating this node and all nodes leading to it will give you:');
    await expect(tip).toHaveTextContent(`${points} points to node`);
  }}>
  {@render frame(novaBuild, { view: 'power' })}
</Story>

<Story
  name="Calcs"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }) => {
    const { c, main } = page(canvasElement as HTMLElement);
    const region = c.getByRole('region', { name: 'Calcs' });
    await expect(within(region).getAllByRole('table').length).toBeGreaterThan(3);
    const cells = within(region).getAllByRole('cell').length;
    await userEvent.click(within(region).getByRole('checkbox', { name: 'Show only differences' }));
    await expect(within(region).getByRole('checkbox', { name: 'Show only differences' })).not.toBeChecked();
    await expect(within(region).getAllByRole('cell').length).toBe(cells);
    await expectPageFits(main);
  }}>
  {@render frame(fireballBuild, { view: 'calcs', calcsB: novaBuild })}
</Story>

<!-- CompareTab.lua:2075-2168 -->
<Story
  name="Config in columns"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, main } = page(canvasElement);
    const cfg = c.getByRole('table', { name: 'Config options' });
    const groups = within(cfg).getAllByRole('rowgroup');
    const lefts = new Set(groups.map((g) => Math.round(g.getBoundingClientRect().left)));
    await expect(lefts.size).toBe(2);
    await expectSingleLine([...cfg.querySelectorAll<HTMLElement>('[role=row][data-diff=true] [role=cell].toned')]);
    await expectNotClipped([...cfg.querySelectorAll<HTMLElement>('[role=row] button')]);
    await expect(
      within(cfg)
        .getAllByRole('row')
        .filter((r) => r.querySelector('[role=columnheader]'))
    ).toHaveLength(1);
    await expect(within(cfg).getAllByRole('row')).toHaveLength(fireballBuild.compare.config.length + 1);
    await expectPageFits(main);
  }}>
  {@render frame(fireballBuild, { view: 'config' })}
</Story>

<Story
  name="Config, small window"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, main } = page(canvasElement);
    const cfg = c.getByRole('table', { name: 'Config options' });
    const lefts = new Set(
      within(cfg)
        .getAllByRole('rowgroup')
        .map((g) => Math.round(g.getBoundingClientRect().left))
    );
    await expect(lefts.size).toBe(1);
    await expectPageFits(main);
  }}>
  {@render frame(fireballBuild, { view: 'config' })}
</Story>
