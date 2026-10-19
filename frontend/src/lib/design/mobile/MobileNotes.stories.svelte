<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within, waitFor } from 'storybook/test';
  import NotesPage from '../NotesPage.svelte';
  import LiveMobile from '../stories/LiveMobile.svelte';
  import LiveNotes from '../stories/LiveNotes.svelte';
  import { spectreBuild, notesSamples } from '../fixtures';
  import { phone, expectPhoneFits } from '../stories/mobile_checks';

  const { Story } = defineMeta({
    title: 'Mobile/Notes',
    component: NotesPage,
    parameters: {
      layout: 'fullscreen',
      docs: {
        description: {
          component:
            'Responsive: the toolbar wraps and the editor fills the width; the “Imported from PoB” note puts its title and Dismiss on one row above the text, and the “i” tip docks above the tab bar. Notes is reached through the More sheet.'
        }
      }
    },
    globals: phone
  });
</script>

<Story
  name="Phone"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await expectPhoneFits(canvasElement, 'Notes');
    const c = within(canvasElement);
    await expect(c.getByRole('toolbar', { name: 'Formatting' })).toBeVisible();
    await expect(c.getByRole('textbox', { name: 'Notes' })).toBeVisible();
    const banner = c.getByText('IMPORTED FROM POB').parentElement!;
    const title = c.getByText('IMPORTED FROM POB').getBoundingClientRect();
    const dismiss = within(banner).getByRole('button', { name: 'Dismiss' }).getBoundingClientRect();
    const text = within(banner)
      .getByText(/^Colour codes converted/)
      .getBoundingClientRect();
    await expect(Math.abs(title.top + title.height / 2 - (dismiss.top + dismiss.height / 2))).toBeLessThanOrEqual(4);
    await expect(text.top).toBeGreaterThanOrEqual(Math.max(title.bottom, dismiss.bottom) - 1);
    const bar = c.getByRole('toolbar', { name: 'Formatting' });
    const info = c.getByRole('button', { name: 'How notes export to PoB' }).getBoundingClientRect();
    const others = [...bar.querySelectorAll('button')]
      .filter((b) => b.getAttribute('aria-label') !== 'How notes export to PoB')
      .map((b) => b.getBoundingClientRect());
    const sameRow = others.filter((o) => info.top < o.bottom && info.bottom > o.top);
    await expect(sameRow.length, 'the info button shares a row with other toolbar buttons').toBeGreaterThan(0);
    await expect(info.right).toBeLessThanOrEqual(bar.getBoundingClientRect().right);
    await userEvent.click(c.getByRole('button', { name: 'How notes export to PoB' }));
    const tip = await waitFor(() => c.getByRole('tooltip'));
    const r = tip.getBoundingClientRect();
    await expect(r.left).toBeGreaterThanOrEqual(0);
    await expect(r.right).toBeLessThanOrEqual(390);
  }}>
  <LiveMobile fx={spectreBuild} current="#notes" label="Notes" mainClass="gap-2.5 p-3"><LiveNotes text={notesSamples[1].text} /></LiveMobile>
</Story>
