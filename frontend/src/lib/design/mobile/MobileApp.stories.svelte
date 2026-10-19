<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within, waitFor } from 'storybook/test';
  import MobileFrame from './MobileFrame.svelte';
  import LiveMobileApp from '../stories/LiveMobileApp.svelte';
  import { spectreBuild, fireballBuild } from '../fixtures';
  import { phone, phoneTall, expectPhoneFits } from '../stories/mobile_checks';

  const { Story } = defineMeta({
    title: 'Mobile/App',
    component: MobileFrame,
    parameters: {
      layout: 'fullscreen',
      a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color], .map-note, .map-note *)' }] } },
      docs: { description: { component: 'Every phone draft behind the real tab bar and More sheet, for clicking through the whole app at phone size.' } }
    },
    globals: phone
  });
</script>

<Story
  name="Phone"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    await expectPhoneFits(canvasElement, 'Tree');
    await userEvent.click(c.getByRole('link', { name: 'Config' }));
    await waitFor(() => expect(c.getByRole('main', { name: 'Config' })).toBeVisible());
    await userEvent.click(c.getByRole('button', { name: 'More' }));
    await userEvent.click(within(await waitFor(() => within(document.body).getByRole('dialog', { name: 'More' }))).getByRole('link', { name: 'Party' }));
    await waitFor(() => expect(c.getByRole('main', { name: 'Party' })).toBeVisible());
    await expectPhoneFits(canvasElement, 'Party');
  }}>
  <LiveMobileApp fx={spectreBuild} />
</Story>

<Story name="Toolbar hidden" asChild globals={phoneTall} play={async ({ canvasElement }) => expectPhoneFits(canvasElement, 'Tree', 834)}>
  <LiveMobileApp fx={fireballBuild} />
</Story>
