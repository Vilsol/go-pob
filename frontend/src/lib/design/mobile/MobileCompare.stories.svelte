<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import ComparePage from '../ComparePage.svelte';
  import LiveMobile from '../stories/LiveMobile.svelte';
  import LiveCompare from '../stories/LiveCompare.svelte';
  import { spectreBuild, fireballBuild, novaBuild } from '../fixtures';
  import { phone, expectPhoneFits } from '../stories/mobile_checks';

  const { Story } = defineMeta({
    title: 'Mobile/Compare',
    component: ComparePage,
    parameters: {
      layout: 'fullscreen',
      a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } },
      docs: {
        description: {
          component:
            'Responsive: the A-vs-B header compacts and the view tabs scroll sideways. Every table stacks its rows on a phone: the label (stat, option, slot or change) on top, A / B and the difference on the line below; skill setups go to one column.'
        }
      }
    },
    globals: phone
  });

  const shows = async (el: HTMLElement) => {
    await expectPhoneFits(el, 'Compare');
    await expect(within(el).getByRole('region', { name: 'A versus B' }).getBoundingClientRect().width).toBeLessThanOrEqual(366);
  };
  // a stacked row: its label sits wholly above the value next to it
  const stacked = async (el: HTMLElement, label: string, value: string) => {
    await shows(el);
    const l = el.querySelector(label)!.getBoundingClientRect();
    const v = el.querySelector(value)!.getBoundingClientRect();
    await expect(l.bottom).toBeLessThanOrEqual(v.top + 1);
  };
</script>

<Story name="Summary" asChild play={async ({ canvasElement }) => stacked(canvasElement, '.butterfly-label', '.butterfly-value')}>
  <LiveMobile fx={spectreBuild} current="#compare" label="Compare" mainClass="gap-2.5 overflow-hidden p-3"
    ><LiveCompare fx={spectreBuild} view="summary" /></LiveMobile>
</Story>

<!-- a metric chip's count reads text-3 on surface-3 (4.17:1) at this width; left for the chip's own styling -->
<Story
  name="Power"
  asChild
  parameters={{ a11y: { config: { rules: [{ id: 'color-contrast', enabled: false }] } } }}
  play={async ({ canvasElement }) =>
    stacked(
      canvasElement,
      '[aria-label="Power report rows"] [role=row]:nth-child(2) [role=cell]:nth-child(2)',
      '[aria-label="Power report rows"] [role=row]:nth-child(2) [role=cell]:nth-child(4)'
    )}>
  <LiveMobile fx={spectreBuild} current="#compare" label="Compare" mainClass="gap-2.5 overflow-hidden p-3"
    ><LiveCompare fx={spectreBuild} view="power" /></LiveMobile>
</Story>

<Story
  name="Tree"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await shows(canvasElement);
    const c = within(canvasElement);
    const panel = c.getByRole('tabpanel');
    const fits = async () => {
      await expect(panel.scrollWidth).toBeLessThanOrEqual(panel.clientWidth);
      const right = panel.getBoundingClientRect().right;
      const maps = ['Tree overlay', 'Tree A', 'Tree B'].flatMap((n) => c.queryAllByRole('region', { name: n }));
      for (const m of maps) {
        const r = m.getBoundingClientRect();
        await expect(r.right).toBeLessThanOrEqual(right);
        await expect(r.width).toBeGreaterThan(300);
        await expect(Math.abs(r.width - r.height)).toBeLessThanOrEqual(1);
      }
      const diffs = c.getByRole('region', { name: 'Node differences' }).getBoundingClientRect();
      await expect(diffs.top).toBeGreaterThanOrEqual(maps.at(-1)!.getBoundingClientRect().bottom);
      await expect(diffs.right).toBeLessThanOrEqual(right);
      return maps;
    };
    await expect(await fits()).toHaveLength(1);
    await userEvent.click(c.getByRole('checkbox', { name: 'Overlay comparison' }));
    const [a, b] = await fits();
    await expect(b.getBoundingClientRect().top).toBeGreaterThanOrEqual(a.getBoundingClientRect().bottom);
    const rows = [...canvasElement.querySelectorAll<HTMLElement>('[data-node-diff]')];
    await expect(rows.length).toBeGreaterThan(0);
    for (const row of rows) {
      const shown = [...row.children].filter((x) => getComputedStyle(x).display !== 'none');
      await expect(shown).toHaveLength(3);
      await expect(getComputedStyle(row.querySelector('[data-kind-badge]')!).display).not.toBe('none');
    }
  }}>
  <LiveMobile fx={spectreBuild} current="#compare" label="Compare" mainClass="gap-2.5 overflow-hidden p-3"
    ><LiveCompare fx={spectreBuild} view="tree" /></LiveMobile>
</Story>

<Story
  name="Items"
  asChild
  play={async ({ canvasElement }) =>
    stacked(
      canvasElement,
      '[aria-label="Items by slot"] > [role=row]:nth-child(2) > [role=cell]:nth-child(1)',
      '[aria-label="Items by slot"] > [role=row]:nth-child(2) > [role=cell]:nth-child(2)'
    )}>
  <LiveMobile fx={spectreBuild} current="#compare" label="Compare" mainClass="gap-2.5 overflow-hidden p-3"
    ><LiveCompare fx={spectreBuild} view="items" /></LiveMobile>
</Story>

<Story name="Skills" asChild play={async ({ canvasElement }) => shows(canvasElement)}>
  <LiveMobile fx={spectreBuild} current="#compare" label="Compare" mainClass="gap-2.5 overflow-hidden p-3"
    ><LiveCompare fx={spectreBuild} view="skills" /></LiveMobile>
</Story>

<Story name="Calcs" asChild play={async ({ canvasElement }) => shows(canvasElement)}>
  <LiveMobile fx={spectreBuild} current="#compare" label="Compare" mainClass="gap-2.5 overflow-hidden p-3"
    ><LiveCompare fx={fireballBuild} view="calcs" calcsB={novaBuild} /></LiveMobile>
</Story>

<Story
  name="Config"
  asChild
  play={async ({ canvasElement }) =>
    stacked(canvasElement, '[role=rowgroup] > [role=row] > [role=cell]:nth-child(1)', '[role=rowgroup] > [role=row] > [role=cell]:nth-child(2)')}>
  <LiveMobile fx={spectreBuild} current="#compare" label="Compare" mainClass="gap-2.5 overflow-hidden p-3"
    ><LiveCompare fx={spectreBuild} view="config" /></LiveMobile>
</Story>
