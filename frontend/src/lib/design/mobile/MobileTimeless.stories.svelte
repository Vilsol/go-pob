<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import TimelessPage from '../TimelessPage.svelte';
  import LiveMobile from '../stories/LiveMobile.svelte';
  import LiveTimeless from '../stories/LiveTimeless.svelte';
  import { spectreBuild } from '../fixtures';
  import { phone, expectPhoneFits } from '../stories/mobile_checks';

  const { Story } = defineMeta({
    title: 'Mobile/Timeless',
    component: TimelessPage,
    parameters: {
      layout: 'fullscreen',
      a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } },
      docs: {
        description: {
          component:
            'Responsive: the page is already a five-step wizard, so on a phone each step stacks (jewels two to a row, the conqueror list and the socket map full width) and the step chips scroll sideways.'
        }
      }
    },
    globals: phone
  });
</script>

<Story name="Jewel" asChild play={async ({ canvasElement }) => expectPhoneFits(canvasElement, 'Timeless search')}>
  <LiveMobile fx={spectreBuild} current="#timeless" label="Timeless search" mainClass="gap-2.5 bg-tree-ground p-3"
    ><LiveTimeless fx={spectreBuild} step={1} /></LiveMobile>
</Story>

<Story name="Sockets" asChild play={async ({ canvasElement }) => expectPhoneFits(canvasElement, 'Timeless search')}>
  <LiveMobile fx={spectreBuild} current="#timeless" label="Timeless search" mainClass="gap-2.5 bg-tree-ground p-3"
    ><LiveTimeless fx={spectreBuild} step={2} /></LiveMobile>
</Story>

<Story name="Requirements" asChild play={async ({ canvasElement }) => expectPhoneFits(canvasElement, 'Timeless search')}>
  <LiveMobile fx={spectreBuild} current="#timeless" label="Timeless search" mainClass="gap-2.5 bg-tree-ground p-3"
    ><LiveTimeless fx={spectreBuild} step={3} /></LiveMobile>
</Story>

<Story name="Ranking" asChild play={async ({ canvasElement }) => expectPhoneFits(canvasElement, 'Timeless search')}>
  <LiveMobile fx={spectreBuild} current="#timeless" label="Timeless search" mainClass="gap-2.5 bg-tree-ground p-3"
    ><LiveTimeless fx={spectreBuild} step={4} /></LiveMobile>
</Story>

<Story name="Results" asChild play={async ({ canvasElement }) => expectPhoneFits(canvasElement, 'Timeless search')}>
  <LiveMobile fx={spectreBuild} current="#timeless" label="Timeless search" mainClass="gap-2.5 bg-tree-ground p-3"
    ><LiveTimeless fx={spectreBuild} step={5} /></LiveMobile>
</Story>
