<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, within } from 'storybook/test';
  import PartyPage from '../PartyPage.svelte';
  import LiveMobile from '../stories/LiveMobile.svelte';
  import LiveParty from '../stories/LiveParty.svelte';
  import { spectreBuild } from '../fixtures';
  import { phone, expectPhoneFits } from '../stories/mobile_checks';

  const { Story } = defineMeta({
    title: 'Mobile/Party',
    component: PartyPage,
    parameters: {
      layout: 'fullscreen',
      docs: {
        description: {
          component:
            'Responsive: the import column and the members column stack (the frame passes a column instead of a row), and each member card already fits 390px.'
        }
      }
    },
    globals: phone
  });
</script>

<Story
  name="Phone"
  asChild
  play={async ({ canvasElement }) => {
    await expectPhoneFits(canvasElement, 'Party');
    await expect(within(canvasElement).getByRole('region', { name: 'Party Member Stats' })).toBeInTheDocument();
  }}>
  <LiveMobile fx={spectreBuild} current="#party" label="Party" mainClass="gap-4 p-3"><LiveParty fx={spectreBuild} /></LiveMobile>
</Story>

<Story name="Advanced" asChild play={async ({ canvasElement }) => expectPhoneFits(canvasElement, 'Party')}>
  <LiveMobile fx={spectreBuild} current="#party" label="Party" mainClass="gap-4 p-3"><LiveParty fx={spectreBuild} advanced /></LiveMobile>
</Story>
