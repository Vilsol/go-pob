<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, within } from 'storybook/test';
  import ImportExportPage from '../ImportExportPage.svelte';
  import LiveMobile from '../stories/LiveMobile.svelte';
  import LiveImportExport from '../stories/LiveImportExport.svelte';
  import { fireballBuild } from '../fixtures';
  import { phone, expectPhoneFits } from '../stories/mobile_checks';

  const { Story } = defineMeta({
    title: 'Mobile/Import Export',
    component: ImportExportPage,
    parameters: {
      layout: 'fullscreen',
      docs: {
        description: {
          component:
            'Responsive: the three panels stack in one scrolling column; inside Character import the account row wraps, and the character list sits above the two import panels, which stack.'
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
    await expectPhoneFits(canvasElement, 'Import / Export');
    const c = within(canvasElement);
    for (const name of ['Build sharing', 'Build import', 'Character import']) {
      const r = c.getByRole('region', { name }).getBoundingClientRect();
      await expect(Math.round(r.right)).toBeLessThanOrEqual(390);
    }
  }}>
  <LiveMobile fx={fireballBuild} current="#import" label="Import / Export" mainClass="gap-4 p-3"><LiveImportExport fx={fireballBuild} /></LiveMobile>
</Story>

<Story
  name="Characters"
  asChild
  parameters={{ a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } } }}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await expectPhoneFits(canvasElement, 'Import / Export');
    const r = within(canvasElement).getByRole('region', { name: 'Character import' });
    r.scrollIntoView({ block: 'start' });
    for (const x of r.querySelectorAll('*')) await expect(Math.round(x.getBoundingClientRect().right)).toBeLessThanOrEqual(390);
  }}>
  <LiveMobile fx={fireballBuild} current="#import" label="Import / Export" mainClass="gap-4 p-3"><LiveImportExport fx={fireballBuild} fetched /></LiveMobile>
</Story>
