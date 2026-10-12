<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, within } from 'storybook/test';
  import PageFrame from './PageFrame.svelte';
  import LiveTopBar from './stories/LiveTopBar.svelte';
  import LiveSidebar from './stories/LiveSidebar.svelte';
  import { spectreBuild, fireballBuild } from './fixtures';
  import { expectThemedScrollbars } from './stories/layout_checks';

  const { Story } = defineMeta({ title: 'Shell/PageFrame', component: PageFrame, parameters: { layout: 'fullscreen' } });

  const fills = async (canvasElement: HTMLElement, w: number, h: number) => {
    const c = within(canvasElement);
    await expect(window.innerWidth).toBe(w);
    const main = c.getByRole('main', { name: 'Tree' });
    const box = main.getBoundingClientRect();
    await expect(box.left).toBe(320);
    await expect(box.top).toBe(52);
    await expect(Math.round(box.bottom)).toBe(h);
    await expect(document.documentElement.scrollHeight).toBeLessThanOrEqual(h);
  };
</script>

{#snippet placeholder()}
  <div class="flex grow items-center justify-center rounded-md border border-dashed border-line-strong text-text-3">Page content</div>
{/snippet}

<Story
  name="Target 1920×1080"
  exportName="Target"
  asChild
  globals={{ viewport: { value: 'target', isRotated: false } }}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await fills(canvasElement, 1920, 1080);
    await expectThemedScrollbars(canvasElement);
  }}>
  <PageFrame label="Tree">
    {#snippet topbar()}<LiveTopBar fx={spectreBuild} />{/snippet}
    {#snippet sidebar()}<LiveSidebar fx={spectreBuild} />{/snippet}
    {@render placeholder()}
  </PageFrame>
</Story>

<Story
  name="Minimum 1280×720"
  exportName="Minimum"
  asChild
  globals={{ viewport: { value: 'min', isRotated: false } }}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await fills(canvasElement, 1280, 720);
    await expectThemedScrollbars(canvasElement);
  }}>
  <PageFrame label="Tree">
    {#snippet topbar()}<LiveTopBar fx={fireballBuild} />{/snippet}
    {#snippet sidebar()}<LiveSidebar fx={fireballBuild} />{/snippet}
    {@render placeholder()}
  </PageFrame>
</Story>

<Story name="Rail collapsed" asChild globals={{ viewport: { value: 'min', isRotated: false } }}>
  <PageFrame label="Tree">
    {#snippet topbar()}<LiveTopBar fx={spectreBuild} dirty={false} />{/snippet}
    {#snippet sidebar()}<LiveSidebar fx={spectreBuild} collapsed pinned={['Player|Total Life|1', 'Minion|Hit DPS|1']} />{/snippet}
    {@render placeholder()}
  </PageFrame>
</Story>
