<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within, waitFor } from 'storybook/test';
  import ConfigPage from './ConfigPage.svelte';
  import PageFrame from './PageFrame.svelte';
  import LiveTopBar from './stories/LiveTopBar.svelte';
  import LiveSidebar from './stories/LiveSidebar.svelte';
  import LiveConfig from './stories/LiveConfig.svelte';
  import { spectreBuild, fireballBuild, novaBuild, type BuildFixture } from './fixtures';
  import { expectPageFits, expectFloating, expectValuesVisible, expectSingleLine, expectDarkCheckboxes } from './stories/layout_checks';

  const { Story } = defineMeta({
    title: 'Pages/Config',
    component: ConfigPage,
    parameters: {
      layout: 'fullscreen',
      a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } }
    },
    globals: { viewport: { value: 'target', isRotated: false } }
  });

  const vp = (value: 'min' | 'mid' | 'target') => ({ viewport: { value, isRotated: false } });
  const page = (el: HTMLElement) => {
    const c = within(el);
    return { c, main: c.getByRole('main', { name: 'Config' }) };
  };
  const summary = (el: HTMLElement) => within(el).getByText(/ shown · \d+ modified · \d+ in total$/);
  const count = (el: HTMLElement, what: 'shown' | 'modified') => Number(new RegExp(`(\\d+) ${what}`).exec(summary(el).textContent ?? '')![1]);
  const row = (main: HTMLElement, v: string, where: 'most' | 'sec' = 'sec') => {
    const most = within(main).getByRole('region', { name: 'Most used' });
    const rows = [...main.querySelectorAll<HTMLElement>(`[data-var="${v}"]`)];
    const r = rows.find((x) => most.contains(x) === (where === 'most'));
    if (!r) throw new Error(`no ${where} row for ${v}`);
    return r;
  };
  const choose = async (r: HTMLElement, label: string) => {
    await userEvent.click(within(r).getByRole('combobox'));
    await userEvent.click(within(r).getByRole('option', { name: label }));
  };

  const shows = async (el: HTMLElement, fx: BuildFixture) => {
    const { c, main } = page(el);
    await expectPageFits(main);
    await expect(c.getByRole('region', { name: 'Most used' })).toBeVisible();
    await expect(c.getByRole('region', { name: 'Enemy Stats' })).toBeVisible();
    await expect(c.getByRole('region', { name: 'Custom modifiers' })).toBeVisible();
    await expect(count(main, 'shown')).toBeGreaterThan(0);
    await expect(summary(main).textContent).toMatch(/in total$/);
    await expect(c.getAllByText(/^\+ \d+ more \(hidden: only shown when the build uses them\)$/).length).toBeGreaterThan(0);
    await expectValuesVisible(main);
    await expectSingleLine([summary(main)]);
    await expectDarkCheckboxes(main);
    void fx;
  };
</script>

{#snippet frame(fx: BuildFixture, opts: { showAll?: boolean; modOnly?: boolean } = {})}
  <PageFrame label="Config" mainClass="gap-2.5 overflow-hidden px-4 py-3">
    {#snippet topbar()}<LiveTopBar {fx} current="#config" />{/snippet}
    {#snippet sidebar()}<LiveSidebar {fx} />{/snippet}
    <LiveConfig {fx} {...opts} />
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
  name="Show all"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }) => {
    const { c, main } = page(canvasElement as HTMLElement);
    const gated = count(main, 'shown');
    const toggle = c.getByRole('checkbox', { name: 'Show options that don’t affect this build' });
    await userEvent.click(toggle);
    await expect(count(main, 'shown')).toBeGreaterThan(gated);
    await expect(c.queryAllByText(/^\+ \d+ more/)).toHaveLength(0);
    await expect(main.querySelectorAll('.option-row.text-text-3').length).toBeGreaterThan(0);
    await expectPageFits(main);
    await userEvent.click(toggle);
    await expect(count(main, 'shown')).toBe(gated);
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Edit and reset"
  asChild
  globals={vp('min')}
  parameters={{ chromatic: { disableSnapshot: true } }}
  play={async ({ canvasElement }) => {
    const { c, main } = page(canvasElement as HTMLElement);
    const before = count(main, 'modified');
    const moving = row(main, 'conditionMoving');
    const box = within(moving).getByRole('checkbox');
    await userEvent.click(box);
    await expect(box).toBeChecked();
    await expect(within(moving).getByText('(changed)')).toBeInTheDocument();
    await expect(within(row(main, 'conditionMoving', 'most')).getByRole('checkbox')).toBeChecked();
    await choose(row(main, 'enemyIsBoss'), 'Uber Pinnacle Boss');
    await expect(within(row(main, 'enemyIsBoss', 'most')).getByRole('combobox')).toHaveTextContent('Uber Pinnacle Boss');
    const level = within(row(main, 'enemyLevel')).getByRole('spinbutton');
    await userEvent.tripleClick(level);
    await userEvent.keyboard('80{Enter}');
    await expect(level).toHaveValue('80');
    await expect(within(row(main, 'enemyLevel', 'most')).getByRole('spinbutton')).toHaveValue('80');
    await expect(count(main, 'modified')).toBeGreaterThanOrEqual(before + 3);
    await expectValuesVisible(main);
    await userEvent.click(c.getByRole('checkbox', { name: 'Modified only' }));
    await expect(count(main, 'shown')).toBe(count(main, 'modified'));
    await userEvent.click(c.getByRole('checkbox', { name: 'Modified only' }));
    await userEvent.click(c.getByRole('button', { name: 'Reset to defaults' }));
    await expect(count(main, 'modified')).toBe(0);
    await expect(within(row(main, 'conditionMoving')).getByRole('checkbox')).not.toBeChecked();
  }}>
  {@render frame(fireballBuild, { showAll: true })}
</Story>

<Story
  name="Option tooltip"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }) => {
    const { c, main } = page(canvasElement as HTMLElement);
    const r = row(main, 'enemyIsBoss', 'most');
    await userEvent.hover(r.querySelector('.dot')!);
    const tip = await c.findByRole('tooltip');
    await expectFloating(r, tip);
    await expect(within(r).getByRole('combobox')).toHaveAttribute('aria-describedby', tip.id);
    await userEvent.unhover(r);
    await waitFor(() => expect(c.queryByRole('tooltip')).toBeNull());
    within(row(main, 'conditionMoving', 'most'))
      .getByRole('checkbox')
      .focus();
    await expect(await c.findByRole('tooltip')).toBeVisible();
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Bandit benefit tooltip"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }) => {
    const { c, main } = page(canvasElement as HTMLElement);
    const r = row(main, 'bandit');
    await choose(r, 'Help Oak');
    await userEvent.unhover(r);
    await userEvent.hover(r);
    await waitFor(() => expect(c.getByRole('tooltip')).toHaveTextContent('+40 to Maximum Life'));
    await choose(r, 'Help Alira');
    await userEvent.unhover(r);
    await userEvent.hover(r);
    await waitFor(() => expect(c.getByRole('tooltip')).toHaveTextContent('+15% to all Elemental Resistances'));
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Toggle while hovered"
  asChild
  globals={vp('target')}
  parameters={{ chromatic: { disableSnapshot: true } }}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, main } = page(canvasElement);
    const secs = [...main.querySelectorAll<HTMLElement>('[data-var]')].filter((r) => !within(main).getByRole('region', { name: 'Most used' }).contains(r));
    const boxed = secs.filter((r) => r.querySelector('input[type="checkbox"]'));
    const rows = [...new Set([...boxed.slice(0, 6), ...boxed.filter((r) => r.querySelector<HTMLInputElement>('input[type="checkbox"]')!.checked)])];
    await expect(rows.some((r) => r.dataset.var === 'raiseSpectreEnableBuffs')).toBe(true);
    await expect(rows.length).toBeGreaterThan(0);
    for (const r of rows) {
      const box = r.querySelector<HTMLInputElement>('input[type="checkbox"]')!;
      const start = box.checked;
      await userEvent.hover(r);
      const tip = () => c.queryByRole('tooltip')?.textContent ?? '';
      const modifiedNow = () => tip().includes('modified from default');
      await expect(modifiedNow(), `${r.dataset.var} at load`).toBe(false);
      for (const expected of [!start, start]) {
        await userEvent.click(box);
        await expect(box.checked, `${r.dataset.var} box`).toBe(expected);
        await waitFor(() => expect(modifiedNow(), `${r.dataset.var} tooltip after → ${expected}`).toBe(expected !== start));
      }
      await userEvent.unhover(r);
    }
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Custom modifiers"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }) => {
    const { c } = page(canvasElement as HTMLElement);
    const panel = c.getByRole('region', { name: 'Custom modifiers' });
    const p = within(panel);
    const n = p.queryAllByRole('textbox', { name: /^Modifier line/ }).length;
    await userEvent.click(p.getByRole('button', { name: '+ Add modifier' }));
    const first = p.getByRole('textbox', { name: `Modifier line ${n + 1}` });
    await expect(first).toHaveFocus();
    await userEvent.keyboard('10% increased Damage');
    const cellOf = (i: HTMLElement) => i.closest<HTMLElement>('.mod-cell')!;
    await expect(cellOf(first)).toHaveAttribute('data-k', 'ok');
    await expect(within(cellOf(first)).getByText('✓')).toBeVisible();
    await userEvent.click(p.getByRole('button', { name: '+ Add modifier' }));
    await userEvent.keyboard('Crit Immune{Enter}');
    const second = p.getByRole('textbox', { name: `Modifier line ${n + 2}` });
    await expect(cellOf(second)).toHaveAttribute('data-k', 'bad');
    await expect(within(cellOf(second)).getByText('✗ no match')).toBeVisible();
    await expect(p.getByText(/1 applied · 1 not recognised/)).toBeVisible();
    await userEvent.hover(cellOf(first));
    const tip = await c.findByRole('tooltip');
    await expect(tip).toHaveTextContent('Damage INC 10');
    await expectFloating(cellOf(first), tip);
    await userEvent.unhover(cellOf(first));
    await userEvent.click(p.getByRole('button', { name: `Move line ${n + 2} earlier` }));
    await expect(p.getByRole('textbox', { name: `Modifier line ${n + 1}` })).toHaveValue('Crit Immune');
    await userEvent.click(p.getByRole('checkbox', { name: /^Enable / }));
    await expect(p.getByText('Group disabled — its lines are not applied')).toBeVisible();
    await expect(cellOf(first)).toHaveAttribute('data-off', 'true');
    await expectSingleLine(Array.from(panel.querySelectorAll<HTMLElement>('.mod-cell-badge')));
  }}>
  {@render frame(spectreBuild)}
</Story>
