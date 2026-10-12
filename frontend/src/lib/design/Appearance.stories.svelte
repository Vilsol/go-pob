<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, within } from 'storybook/test';
  import CalcsPage from './CalcsPage.svelte';
  import PageFrame from './PageFrame.svelte';
  import LiveTopBar from './stories/LiveTopBar.svelte';
  import LiveSidebar from './stories/LiveSidebar.svelte';
  import LiveCalcs from './stories/LiveCalcs.svelte';
  import LiveItems from './stories/LiveItems.svelte';
  import LiveTree from './stories/LiveTree.svelte';
  import WithAppearance from './stories/WithAppearance.svelte';
  import { fireballBuild, novaBuild } from './fixtures';
  import { expectPageFits, expectSingleLine } from './stories/layout_checks';

  const { Story } = defineMeta({ title: 'Foundations/Appearance', component: CalcsPage, parameters: { layout: 'fullscreen' } });

  const vp = (value: 'min' | 'mid' | 'target') => ({ viewport: { value, isRotated: false } });
  const px = (e: Element) => parseFloat(getComputedStyle(e).fontSize);

  const scaled = async (el: HTMLElement, pt: number, fits: boolean) => {
    const c = within(el);
    const main = c.getByRole('main', { name: 'Calcs' });
    const k = pt / 12;
    await expect(document.documentElement.scrollWidth, 'the document scrolls sideways').toBeLessThanOrEqual(window.innerWidth);
    await expect(main.scrollWidth, 'main scrolls sideways').toBeLessThanOrEqual(main.clientWidth + 1);
    if (fits) await expectPageFits(main);
    await expect(c.getByRole('banner').getBoundingClientRect().height).toBeCloseTo(52 * k, 0);
    const t13 = el.querySelector('.text-13')!;
    await expect(px(t13)).toBeCloseTo(13 * k, 1);
    const content = c.getByRole('region', { name: 'Calculations' });
    await expectSingleLine([...content.querySelectorAll<HTMLElement>('[role="cell"], .font-mono > span')].filter((e) => e.textContent));
  };
  const themed = async (el: HTMLElement, attr: string, ground: string) => {
    await expect(document.documentElement.dataset.theme).toBe(attr);
    await expect(getComputedStyle(document.body).backgroundColor).toBe(ground);
    await expectPageFits(within(el).getByRole('main', { name: 'Calcs' }));
  };
</script>

{#snippet items()}
  <PageFrame label="Items" mainClass="scroll">
    {#snippet topbar()}<LiveTopBar fx={novaBuild} current="#items" />{/snippet}
    {#snippet sidebar()}<LiveSidebar fx={novaBuild} />{/snippet}
    <LiveItems fx={novaBuild} hover="Flask 1" />
  </PageFrame>
{/snippet}

{#snippet tree()}
  <PageFrame label="Tree" mainClass="gap-0 p-0">
    {#snippet topbar()}<LiveTopBar fx={novaBuild} current="#tree" />{/snippet}
    {#snippet sidebar()}<LiveSidebar fx={novaBuild} />{/snippet}
    <LiveTree fx={novaBuild} />
  </PageFrame>
{/snippet}

{#snippet frame()}
  <PageFrame label="Calcs" mainClass="gap-0 px-3 py-0">
    {#snippet topbar()}<LiveTopBar fx={fireballBuild} current="#calcs" />{/snippet}
    {#snippet sidebar()}<LiveSidebar fx={fireballBuild} />{/snippet}
    <LiveCalcs fx={fireballBuild} />
  </PageFrame>
{/snippet}

<Story
  name="Font scaling 6 pt · minimum"
  asChild
  globals={vp('min')}
  parameters={{ a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } } }}
  play={async ({ canvasElement }) => scaled(canvasElement as HTMLElement, 6, false)}>
  <WithAppearance pt={6}>{@render frame()}</WithAppearance>
</Story>

<Story
  name="Font scaling 18 pt · target"
  asChild
  globals={vp('target')}
  parameters={{
    a11y: {
      config: {
        rules: [
          { id: 'color-contrast', selector: '*:not([data-pob-color])' },
          { id: 'scrollable-region-focusable', enabled: false }
        ]
      }
    }
  }}
  play={async ({ canvasElement }) => scaled(canvasElement as HTMLElement, 18, true)}>
  <WithAppearance pt={18}>{@render frame()}</WithAppearance>
</Story>

<Story
  name="Light · minimum"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }) => themed(canvasElement as HTMLElement, 'light', 'rgb(236, 234, 228)')}>
  <WithAppearance theme="light">{@render frame()}</WithAppearance>
</Story>

<Story
  name="High contrast · minimum"
  asChild
  globals={vp('min')}
  parameters={{ a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } } }}
  play={async ({ canvasElement }) => themed(canvasElement as HTMLElement, 'contrast', 'rgb(0, 0, 0)')}>
  <WithAppearance theme="contrast">{@render frame()}</WithAppearance>
</Story>

<Story
  name="Light · Items"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }) => {
    const c = within(canvasElement as HTMLElement);
    await expect(document.documentElement.dataset.theme).toBe('light');
    await expect(c.getByRole('tooltip')).toBeVisible();
    await expectPageFits(c.getByRole('main', { name: 'Items' }));
  }}>
  <WithAppearance theme="light">{@render items()}</WithAppearance>
</Story>

<Story
  name="Light · Tree"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }) => {
    const c = within(canvasElement as HTMLElement);
    await expect(document.documentElement.dataset.theme).toBe('light');
    await expect(document.documentElement.scrollWidth).toBeLessThanOrEqual(window.innerWidth);
    await expect(c.getByRole('main', { name: 'Tree' })).toBeVisible();
  }}>
  <WithAppearance theme="light">{@render tree()}</WithAppearance>
</Story>
