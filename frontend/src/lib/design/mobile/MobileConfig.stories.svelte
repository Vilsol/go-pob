<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, waitFor, within } from 'storybook/test';
  import ConfigPage from '../ConfigPage.svelte';
  import LiveMobile from '../stories/LiveMobile.svelte';
  import LiveConfig from '../stories/LiveConfig.svelte';
  import { spectreBuild, fireballBuild } from '../fixtures';
  import { phone, phoneTall, expectPhoneFits } from '../stories/mobile_checks';

  const { Story } = defineMeta({
    title: 'Mobile/Config',
    component: ConfigPage,
    parameters: {
      layout: 'fullscreen',
      a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } },
      docs: {
        description: {
          component:
            'Responsive: the page is a list of labelled controls that only needs one column and a full-width filter, so it reflows in place below 640px. Tap a control to see its tooltip (focus shows it).'
        }
      }
    },
    globals: phone
  });

  const shows = async (el: HTMLElement, h?: number) => {
    await expectPhoneFits(el, 'Config', h);
    const c = within(el);
    for (const r of ['Most used', 'Enemy Stats', 'Custom modifiers']) await expect(c.getByRole('region', { name: r })).toBeInTheDocument();
    await expect(c.getByRole('searchbox', { name: 'Filter options' }).getBoundingClientRect().width).toBeGreaterThan(300);
    const custom = c.getByRole('region', { name: 'Custom modifiers' });
    custom.scrollIntoView({ block: 'end' });
    for (const x of custom.querySelectorAll('*')) await expect(x.getBoundingClientRect().right).toBeLessThanOrEqual(390);
    const groups = within(custom).getByRole('list', { name: 'Mod groups' }).getBoundingClientRect();
    await expect(within(custom).getByRole('list', { name: 'Modifier lines' }).parentElement!.getBoundingClientRect().top).toBeGreaterThanOrEqual(groups.bottom);
    const browser = within(custom).getByRole('button', { name: 'Mod browser' });
    browser.scrollIntoView({ block: 'nearest' });
    const b = browser.getBoundingClientRect();
    await expect(b.left).toBeGreaterThanOrEqual(0);
    await expect(b.right).toBeLessThanOrEqual(390);
    await expect(b.width).toBeGreaterThan(0);
    await userEvent.click(browser);
    const d = await waitFor(() => within(document.body).getByRole('dialog', { name: 'Mod browser' }));
    await waitFor(() => expect(within(d).getAllByRole('option').length).toBeGreaterThan(0));
    await expect(Math.round(d.getBoundingClientRect().width)).toBeLessThanOrEqual(390);
    await userEvent.click(within(d).getByRole('button', { name: 'Close' }));
  };
</script>

<Story name="Phone" asChild play={async ({ canvasElement }) => shows(canvasElement)}>
  <LiveMobile fx={spectreBuild} current="#config" label="Config" mainClass="gap-2.5 overflow-hidden p-3"><LiveConfig fx={spectreBuild} /></LiveMobile>
</Story>

<Story name="Toolbar hidden" asChild globals={phoneTall} play={async ({ canvasElement }) => shows(canvasElement, 834)}>
  <LiveMobile fx={fireballBuild} current="#config" label="Config" mainClass="gap-2.5 overflow-hidden p-3"><LiveConfig fx={fireballBuild} /></LiveMobile>
</Story>
