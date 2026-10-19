<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within, waitFor } from 'storybook/test';
  import CalcsPage from '../CalcsPage.svelte';
  import LiveMobile from '../stories/LiveMobile.svelte';
  import LiveCalcs from '../stories/LiveCalcs.svelte';
  import { spectreBuild, fireballBuild } from '../fixtures';
  import { phone, expectPhoneFits } from '../stories/mobile_checks';

  const { Story } = defineMeta({
    title: 'Mobile/Calcs',
    component: CalcsPage,
    parameters: {
      layout: 'fullscreen',
      a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } },
      docs: {
        description: {
          component:
            'Responsive, with one touch change: the sections already fall to one 17rem column, but the side breakdown pane has no room, so on a phone it is hidden until a row is tapped (pinned) and then docks as a bottom sheet; tap the row again or Unpin to close it.'
        }
      }
    },
    globals: phone
  });

  const dock = (el: HTMLElement) => el.querySelector<HTMLElement>('[data-breakdown-dock]')!;
</script>

<Story
  name="Phone"
  asChild
  play={async ({ canvasElement }) => {
    await expectPhoneFits(canvasElement, 'Calcs');
    const c = within(canvasElement);
    await expect(c.queryByRole('complementary', { name: 'Breakdown' })).toBeNull();
    const row = within(c.getByRole('region', { name: 'Calculations' })).getAllByRole('button', { pressed: false })[0];
    await userEvent.click(row);
    const panel = await waitFor(() => within(dock(canvasElement)).getByRole('complementary', { name: 'Breakdown' }));
    const r = panel.getBoundingClientRect();
    await expect(Math.round(r.width)).toBe(390);
    await expect(Math.round(r.bottom)).toBe(769);
  }}>
  <LiveMobile fx={spectreBuild} current="#calcs" label="Calcs" mainClass="overflow-hidden"><LiveCalcs fx={spectreBuild} /></LiveMobile>
</Story>

<Story
  name="Folded breakdown"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const c = within(canvasElement);
    await userEvent.click(within(c.getByRole('region', { name: 'Calculations' })).getAllByRole('button', { pressed: false })[0]);
    const d = dock(canvasElement);
    await userEvent.click(await waitFor(() => within(d).getByRole('button', { name: /Fold the breakdown down/ })));
    const bar = c.getByRole('navigation', { name: 'Pages' }).getBoundingClientRect();
    await waitFor(() => expect(Math.abs(d.getBoundingClientRect().bottom - bar.top)).toBeLessThanOrEqual(1.5));
    await expect(within(d).queryByRole('complementary', { name: 'Breakdown' })).toBeNull();
    await userEvent.click(within(d).getByRole('button', { name: /Show the breakdown/ }));
    await waitFor(() => expect(within(d).getByRole('complementary', { name: 'Breakdown' })).toBeVisible());
  }}>
  <LiveMobile fx={spectreBuild} current="#calcs" label="Calcs" mainClass="overflow-hidden"><LiveCalcs fx={spectreBuild} /></LiveMobile>
</Story>

<Story
  name="Show all panels"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await expectPhoneFits(canvasElement, 'Calcs');
    const wide = [...canvasElement.querySelectorAll<HTMLElement>('[data-calcs-wide]')].find((w) => w.scrollWidth > w.clientWidth)!;
    const label = wide.querySelector<HTMLElement>('[role=rowheader]')!;
    wide.scrollLeft = 150;
    await waitFor(() => expect(Math.abs(label.getBoundingClientRect().left - wide.getBoundingClientRect().left)).toBeLessThanOrEqual(2));
  }}>
  <LiveMobile fx={fireballBuild} current="#calcs" label="Calcs" mainClass="overflow-hidden"><LiveCalcs fx={fireballBuild} showAll /></LiveMobile>
</Story>
