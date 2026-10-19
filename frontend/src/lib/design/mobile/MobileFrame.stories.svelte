<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within, waitFor } from 'storybook/test';
  import MobileFrame from './MobileFrame.svelte';
  import LiveMobile from '../stories/LiveMobile.svelte';
  import Dialog from '../Dialog.svelte';
  import Button from '../Button.svelte';
  import Toasts from '../Toasts.svelte';
  import { addToast, clearToasts } from '../toast';
  import { spectreBuild, fireballBuild } from '../fixtures';
  import { phone, phoneTall, treeDelta, expectPhoneFits } from '../stories/mobile_checks';

  const { Story } = defineMeta({
    title: 'Mobile/Frame',
    component: MobileFrame,
    parameters: {
      layout: 'fullscreen',
      docs: {
        description: {
          component:
            'Shared phone frame: header (back, undo, redo, save), a stats strip with the last edit’s delta (tap for the full sidebar as a bottom sheet), the page, and a bottom tab bar with a More sheet for the remaining pages and build actions.'
        }
      }
    },
    globals: phone
  });
</script>

{#snippet placeholder()}
  <div class="m-3 flex grow items-center justify-center rounded-md border border-dashed border-line-strong text-text-3">Page content</div>
{/snippet}

<Story
  name="Phone 390×769"
  exportName="Phone"
  asChild
  play={async ({ canvasElement }) => {
    await expectPhoneFits(canvasElement, 'Tree');
    const c = within(canvasElement);
    await expect(c.getByRole('link', { name: 'Tree' })).toHaveAttribute('aria-current', 'page');
    await expect(canvasElement.querySelectorAll('[data-delta]').length).toBeGreaterThan(0);
  }}>
  <LiveMobile fx={spectreBuild} current="#tree" label="Tree" delta={treeDelta(spectreBuild)}>{@render placeholder()}</LiveMobile>
</Story>

<Story
  name="Toolbar hidden 390×834"
  exportName="PhoneTall"
  asChild
  globals={phoneTall}
  play={async ({ canvasElement }) => expectPhoneFits(canvasElement, 'Tree', 834)}>
  <LiveMobile fx={fireballBuild} current="#tree" label="Tree">{@render placeholder()}</LiveMobile>
</Story>

<Story
  name="Stats sheet"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    await userEvent.click(c.getByRole('button', { name: 'Show all stats' }));
    const sheet = await waitFor(() => within(document.body).getByRole('dialog', { name: 'Stats' }));
    await expect(sheet).toBeVisible();
    await expect(Math.round(sheet.getBoundingClientRect().bottom)).toBe(769);
  }}>
  <LiveMobile fx={spectreBuild} current="#tree" label="Tree" delta={treeDelta(spectreBuild)}>{@render placeholder()}</LiveMobile>
</Story>

<Story
  name="More sheet"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    await userEvent.click(c.getByRole('button', { name: 'More' }));
    const sheet = await waitFor(() => within(document.body).getByRole('dialog', { name: 'More' }));
    for (const p of ['Notes', 'Party', 'Compare', 'Import/Export', 'Timeless']) await expect(within(sheet).getByRole('link', { name: p })).toBeVisible();
    await userEvent.click(within(sheet).getByRole('link', { name: 'Notes' }));
    await waitFor(() => expect(c.getByRole('button', { name: 'More' })).toHaveClass('text-accent'));
  }}>
  <LiveMobile fx={spectreBuild} current="#tree" label="Tree">{@render placeholder()}</LiveMobile>
</Story>

<Story
  name="Pinned stats"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const c = within(canvasElement);
    const strip = c.getByRole('button', { name: 'Show all stats' });
    const labels = () => [...strip.querySelectorAll('.truncate.text-text-3')].map((x) => x.textContent?.trim());
    await expect(labels()).toEqual(['Life', 'Energy Shield']);
    await userEvent.click(strip);
    const sheet = await waitFor(() => within(document.body).getByRole('dialog', { name: 'Stats' }));
    await userEvent.click(within(sheet).getAllByRole('button', { name: /^Pin Total Mana/ })[0]);
    await userEvent.click(within(sheet).getByRole('button', { name: 'Close' }));
    await waitFor(() => expect(labels()).toEqual(['Life', 'Energy Shield', 'Mana']));
    const widths = [...strip.querySelectorAll<HTMLElement>('[data-strip-cell]')].map((x) => x.getBoundingClientRect().width);
    for (const w of widths) await expect(Math.abs(w - (strip.clientWidth - 24) / 4)).toBeLessThanOrEqual(1);
  }}>
  <LiveMobile fx={fireballBuild} current="#tree" label="Tree" pinned={['|Total Life|0', '|Energy Shield|0']}>{@render placeholder()}</LiveMobile>
</Story>

<Story
  name="Many pinned stats scroll"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const strip = within(canvasElement).getByRole('button', { name: 'Show all stats' });
    const cells = [...strip.querySelectorAll<HTMLElement>('[data-strip-cell]')];
    await expect(cells).toHaveLength(6);
    await expect(strip.scrollWidth).toBeGreaterThan(strip.clientWidth);
    const box = strip.getBoundingClientRect();
    const inView = cells.filter((x) => x.getBoundingClientRect().right <= box.right);
    await expect(inView).toHaveLength(4);
    await expect(Math.round(box.width)).toBe(390);
  }}>
  <LiveMobile
    fx={fireballBuild}
    current="#tree"
    label="Tree"
    pinned={['|Total Life|0', '|Energy Shield|0', '|Total Mana|0', '|Strength|0', '|Dexterity|0', '|Intelligence|0']}>{@render placeholder()}</LiveMobile>
</Story>

<Story
  name="Toast clears the tab bar"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    clearToasts();
    addToast('Power builder\nCalculating node power: 40%');
    const toast = await waitFor(() => within(document.body).getByRole('status', { name: 'Power builder' }));
    const tabs = within(canvasElement).getByRole('navigation', { name: 'Pages' });
    await expect(toast.getBoundingClientRect().bottom).toBeLessThanOrEqual(tabs.getBoundingClientRect().top);
    clearToasts();
  }}>
  <LiveMobile fx={fireballBuild} current="#tree" label="Tree">{@render placeholder()}</LiveMobile>
  <Toasts />
</Story>

<Story
  name="Small confirm stays a dialog"
  asChild
  play={async () => {
    const small = await waitFor(() => within(document.body).getByRole('dialog', { name: 'Delete Item' }));
    const r = small.getBoundingClientRect();
    await expect(r.width).toBeLessThan(390);
    await expect(r.left).toBeGreaterThan(0);
    await expect(r.height).toBeLessThan(769 / 2);
  }}>
  <LiveMobile fx={spectreBuild} current="#items" label="Items">
    {@render placeholder()}
    <Dialog open title="Delete Item">
      <p class="m-0 text-13">Are you sure you want to remove 'Gloom Barb' from the shared item list?</p>
      {#snippet actions()}<Button size="md" variant="primary">Delete</Button><Button size="md">Cancel</Button>{/snippet}
    </Dialog>
  </LiveMobile>
</Story>
