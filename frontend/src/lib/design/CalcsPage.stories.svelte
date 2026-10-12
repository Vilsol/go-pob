<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within, waitFor } from 'storybook/test';
  import CalcsPage from './CalcsPage.svelte';
  import PageFrame from './PageFrame.svelte';
  import LiveTopBar from './stories/LiveTopBar.svelte';
  import LiveSidebar from './stories/LiveSidebar.svelte';
  import LiveCalcs from './stories/LiveCalcs.svelte';
  import modDebugSample from './stories/mod_debug_sample.txt?raw';
  import { spectreBuild, fireballBuild, novaBuild, type BuildFixture } from './fixtures';
  import { expectPageFits, expectDocked, expectSingleLine, expectNotClipped, expectCapped, expectSelectsFloating } from './stories/layout_checks';

  const { Story } = defineMeta({
    title: 'Pages/Calcs',
    component: CalcsPage,
    parameters: {
      layout: 'fullscreen',
      a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } }
    },
    globals: { viewport: { value: 'target', isRotated: false } }
  });

  const vp = (value: 'min' | 'mid' | 'target') => ({ viewport: { value, isRotated: false } });
  const page = (el: HTMLElement) => {
    const c = within(el);
    const main = c.getByRole('main', { name: 'Calcs' });
    return { c, main, content: c.getByRole('region', { name: 'Calculations' }), panel: c.getByRole('complementary', { name: 'Breakdown' }) };
  };
  const rowBtn = (el: HTMLElement, section: string, label: string) =>
    within(within(el).getByRole('region', { name: section })).getAllByRole('button', { name: label })[0];
  const heading = (panel: HTMLElement) => within(panel).getByRole('heading', { level: 2 });

  const CAPPED = [
    /^(Fire|Cold|Lightning|Chaos) Resist$/,
    /^(Spell )?Block Chance$/,
    /Evade Chance$/,
    /^(Dodge|Spell Ddg\.) Chance$/,
    /^Suppression Ch\.$/,
    /Dmg\. Reduct$/,
    /^Avoid .+ Ch\.$/
  ];
  const capsShown = async (content: HTMLElement, fx: BuildFixture) => {
    let n = 0;
    for (const s of fx.calcs.sections.filter((x) => x.applicable && !x.wide))
      for (const [j, b] of (s.blocks ?? []).entries())
        for (const [k, r] of b.rows.entries()) {
          if (!CAPPED.some((re) => re.test(r.label))) continue;
          await expect(typeof r.gauge?.cap, `${s.id} · ${r.label}: its cap`).toBe('number');
          const row = content.querySelector<HTMLElement>(`[data-key="${s.id}:${j}:${k}"]`)!;
          await expect(row, `${s.id} · ${r.label}: row`).not.toBeNull();
          if (r.gauge!.cap! < 100) await expectCapped(row, r.gauge!.cap!);
          n++;
        }
    await expect(n, 'capped rows').toBeGreaterThanOrEqual(6);
  };

  const shows = async (el: HTMLElement, fx: BuildFixture) => {
    const { c, main, content, panel } = page(el);
    await expectPageFits(main);
    await expectDocked(main, panel, content);
    for (const s of fx.calcs.sections.filter((x) => x.applicable)) await expect(c.getByRole('region', { name: s.title })).toBeVisible();
    await expect(c.queryByRole('region', { name: fx.calcs.sections.find((x) => !x.applicable)!.title })).toBeNull();
    await expectSingleLine([...content.querySelectorAll<HTMLElement>('[role="cell"], .font-mono > span')].filter((e) => e.textContent));
    const wideCells = [...content.querySelectorAll<HTMLElement>('[role="cell"]')].filter((e) => e.textContent);
    if (fx.calcs.sections.some((x) => x.wide && x.applicable)) await expectNotClipped(wideCells);
    await expect(heading(panel)).toHaveTextContent('Breakdown');
    await capsShown(content, fx);
    await expectSelectsFloating(el);
  };
</script>

{#snippet frame(fx: BuildFixture, opts: { showAll?: boolean; pin?: string; moddebug?: () => Promise<string> } = {})}
  <PageFrame label="Calcs" mainClass="gap-0 px-3 py-0">
    {#snippet topbar()}<LiveTopBar {fx} current="#calcs" />{/snippet}
    {#snippet sidebar()}<LiveSidebar {fx} />{/snippet}
    <LiveCalcs {fx} {...opts} />
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
  name="Hover preview"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }) => {
    const { main, content, panel } = page(canvasElement as HTMLElement);
    const btn = rowBtn(content, 'Resists', 'Fire Resist');
    const row = btn.closest<HTMLElement>('[data-key]')!;
    await userEvent.hover(row.lastElementChild as HTMLElement);
    await expect(heading(panel)).toHaveTextContent('Resists · Fire Resist');
    await expect(within(panel).getByText('PREVIEW')).toBeVisible();
    await expect(btn).toHaveAttribute('aria-describedby', 'calcs-breakdown');
    await expect(within(panel).getAllByText(/Fire/).length).toBeGreaterThan(0);
    await expectDocked(main, panel, content);
    await userEvent.unhover(row);
    await expect(heading(panel)).toHaveTextContent('Breakdown');
  }}>
  {@render frame(fireballBuild)}
</Story>

<Story
  name="Pinned panel"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }) => {
    const { main, content, panel } = page(canvasElement as HTMLElement);
    const life = rowBtn(content, 'Life', 'Total');
    await userEvent.click(life);
    await expect(life).toHaveAttribute('aria-pressed', 'true');
    await expect(heading(panel)).toHaveTextContent('Life · Total');
    await expect(within(panel).getByText('PINNED')).toBeVisible();
    const other = rowBtn(content, 'Resists', 'Cold Resist');
    await userEvent.hover(other);
    await expect(heading(panel)).toHaveTextContent('Resists · Cold Resist');
    await expect(within(panel).getByText(/^Pinned: Total/)).toBeVisible();
    await expectDocked(main, panel, content);
    await userEvent.unhover(other);
    await expect(heading(panel)).toHaveTextContent('Life · Total');
    await userEvent.click(within(panel).getByRole('button', { name: 'Unpin' }));
    await expect(heading(panel)).toHaveTextContent('Breakdown');
    await expect(life).toHaveAttribute('aria-pressed', 'false');
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Row click pins"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }) => {
    const { content, panel } = page(canvasElement as HTMLElement);
    for (const [section, label] of [
      ['Life', 'Total'],
      ['Resists', 'Cold Resist']
    ]) {
      const btn = rowBtn(content, section, label);
      const row = btn.closest<HTMLElement>('[data-key]')!;
      const value = row.lastElementChild as HTMLElement;
      await userEvent.click(value);
      await expect(btn).toHaveAttribute('aria-pressed', 'true');
      await expect(heading(panel)).toHaveTextContent(`${section} · ${label}`);
      await userEvent.click(value);
      await expect(btn).toHaveAttribute('aria-pressed', 'false');
      for (let i = 1; i <= 5; i++) {
        await userEvent.click(i % 2 ? btn : value);
        await expect(btn).toHaveAttribute('aria-pressed', String(i % 2 === 1));
      }
      await userEvent.click(btn);
    }
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Keyboard focus"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }) => {
    const { content, panel } = page(canvasElement as HTMLElement);
    const btn = rowBtn(content, 'Resists', 'Fire Resist');
    btn.focus();
    await waitFor(() => expect(heading(panel)).toHaveTextContent('Resists · Fire Resist'));
    await userEvent.keyboard('{Enter}');
    await expect(btn).toHaveAttribute('aria-pressed', 'true');
    await userEvent.tab();
    await expect(heading(panel)).toHaveTextContent('Resists · Cold Resist');
    await expect(within(panel).getByText(/^Pinned: Fire Resist/)).toBeVisible();
    await userEvent.tab({ shift: true });
    await expect(within(panel).getByText('PINNED')).toBeVisible();
  }}>
  {@render frame(novaBuild)}
</Story>

<Story
  name="Show all, search, collapse"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }) => {
    const { c, main } = page(canvasElement as HTMLElement);
    const na = spectreBuild.calcs.sections.find((x) => !x.applicable && !x.wide)!;
    await userEvent.click(c.getByRole('checkbox', { name: 'Show all panels' }));
    const sec = c.getByRole('region', { name: na.title });
    await expect(within(sec).getByText('not applicable')).toBeVisible();
    await expectPageFits(main);
    await userEvent.type(c.getByRole('searchbox', { name: 'Search stats' }), 'Resist');
    await waitFor(() => expect(c.queryByRole('region', { name: 'Life' })).toBeNull());
    await expect(within(c.getByRole('region', { name: 'Resists' })).getAllByRole('button', { name: /Resist$/ }).length).toBe(4);
    await userEvent.clear(c.getByRole('searchbox', { name: 'Search stats' }));
    const head = within(c.getByRole('region', { name: 'Life' })).getByRole('button', { name: /Life/, expanded: true });
    await userEvent.click(head);
    await expect(head).toHaveAttribute('aria-expanded', 'false');
    await expect(within(c.getByRole('region', { name: 'Life' })).queryByRole('button', { name: 'Unreserved' })).toBeNull();
  }}>
  {@render frame(spectreBuild)}
</Story>

<!-- Calcs.lua:37-70 -->
<Story
  name="Mod debug (dev mode)"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }) => {
    const { c, main, content } = page(canvasElement as HTMLElement);
    const btn = c.getByRole('button', { name: 'Mod Debug' });
    await userEvent.click(btn);
    const pre = await within(content).findByLabelText('Mod debug');
    await expect(pre).toHaveTextContent('=== Modifiers ===');
    await expect(pre).toHaveTextContent('=== Main Skill Flags ===');
    await expect(btn).toHaveAttribute('aria-expanded', 'true');
    await expectPageFits(main);
  }}>
  {@render frame(fireballBuild, { moddebug: () => Promise.resolve(modDebugSample) })}
</Story>
