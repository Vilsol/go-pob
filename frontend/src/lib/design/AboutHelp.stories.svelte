<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, fireEvent, userEvent, waitFor, within } from 'storybook/test';
  import AboutDialog from './AboutDialog.svelte';
  import LiveHelp from './stories/LiveHelp.svelte';

  // Main.lua:1359-1447
  const { Story } = defineMeta({ title: 'Pages/Help', component: AboutDialog, parameters: { layout: 'fullscreen' } });

  const vp = (value: 'min' | 'mid' | 'target') => ({ viewport: { value, isRotated: false } });

  const open = async (el: HTMLElement) => {
    const c = within(el);
    const opener = c.getByRole('button', { name: 'Help' });
    await userEvent.click(opener);
    const dlg = c.getByRole('dialog', { name: 'About go-pob' });
    await waitFor(() => expect(dlg).toBeVisible());
    return { c, opener, dlg, d: within(dlg) };
  };

  const headStays = async (dlg: HTMLElement) => {
    const box = dlg.getBoundingClientRect();
    await expect(dlg.scrollTop).toBe(0);
    await expect(dlg.scrollHeight).toBeLessThanOrEqual(dlg.clientHeight + 1);
    const close = within(dlg).getByRole('button', { name: 'Close' }).getBoundingClientRect();
    await expect(close.top).toBeGreaterThanOrEqual(box.top);
    await expect(close.bottom).toBeLessThanOrEqual(Math.min(box.bottom, window.innerHeight));
  };
</script>

<Story
  name="Help at a section"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    let { opener, dlg, d } = await open(canvasElement);
    const nav = d.getByRole('navigation', { name: 'Help sections' });
    await expect(within(nav).getByRole('button', { name: '7. Loadouts' })).toHaveAttribute('aria-current', 'true');
    const text = d.getByRole('region', { name: 'Help text' });
    const heading = d.getByRole('heading', { name: 'Loadouts (7)' });
    await waitFor(() => expect(Math.abs(heading.getBoundingClientRect().top - text.getBoundingClientRect().top)).toBeLessThan(24));
    await headStays(dlg);
    const para = [...text.querySelectorAll('p')].find((p) => (p.textContent ?? '').length > 200)!;
    await expect(para.getBoundingClientRect().width).toBeLessThanOrEqual(760);
    await userEvent.click(within(nav).getByRole('button', { name: '11. Similar Builds Popup' }));
    await headStays(dlg);
    await userEvent.click(d.getByRole('button', { name: 'Close' }));
    await waitFor(() => expect(dlg).not.toBeVisible());
    await expect(opener).toHaveFocus();

    ({ opener, dlg } = await open(canvasElement));
    await userEvent.keyboard('{Escape}');
    await waitFor(() => expect(dlg).not.toBeVisible());
    await expect(opener).toHaveFocus();

    ({ opener, dlg } = await open(canvasElement));
    await fireEvent.click(dlg, { clientX: 4, clientY: 4 });
    await waitFor(() => expect(dlg).not.toBeVisible());
    await expect(opener).toHaveFocus();
  }}>
  <LiveHelp section="Loadouts" />
</Story>

<Story
  name="Help, small window"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { dlg } = await open(canvasElement);
    await headStays(dlg);
  }}>
  <LiveHelp section="Items Tab" />
</Story>
