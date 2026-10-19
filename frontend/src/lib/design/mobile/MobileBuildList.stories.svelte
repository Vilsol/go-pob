<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within, waitFor } from 'storybook/test';
  import BuildListPage from '../BuildListPage.svelte';
  import LiveBuildList from '../stories/LiveBuildList.svelte';
  import { phone } from '../stories/mobile_checks';

  const { Story } = defineMeta({
    title: 'Mobile/Build list',
    component: BuildListPage,
    parameters: {
      layout: 'fullscreen',
      docs: {
        description: {
          component:
            'Responsive: below 640px the folder tree moves into a sheet behind a “Folders” button. Each build is a card (name and actions, then the shown columns), or, with Settings › Wide lists on a phone › Table, a grid that scrolls sideways under a sticky name column. Drag-to-folder has no touch path; ⋯ → Move to… covers it.'
        }
      }
    },
    globals: phone
  });

  let made = 0;
  const fits = async (el: HTMLElement) => {
    await expect(window.innerWidth).toBe(390);
    await expect(document.documentElement.scrollWidth).toBeLessThanOrEqual(390);
    await expect(document.documentElement.scrollHeight).toBeLessThanOrEqual(769);
    await expect(within(el).getByRole('grid', { name: 'Builds' })).toBeVisible();
  };
</script>

<Story
  name="Phone"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await fits(canvasElement);
    const row = canvasElement.querySelector<HTMLElement>('[role=grid] [data-build]')!;
    const name = row.querySelector('[data-col=name]')!.getBoundingClientRect();
    const dps = row.querySelector('[data-col=dps]')!.getBoundingClientRect();
    await expect(dps.top).toBeGreaterThanOrEqual(name.bottom - 1);
    await expect(row.getBoundingClientRect().right).toBeLessThanOrEqual(390);
    const folder = canvasElement.querySelector<HTMLElement>('[role=grid] [data-folder]')!;
    const fname = folder.querySelector('button')!.getBoundingClientRect();
    await expect(Math.round(fname.left - folder.getBoundingClientRect().left)).toBe(Math.round(name.left - row.getBoundingClientRect().left));
    const add = within(canvasElement).getByRole('button', { name: 'New folder' });
    await expect(add).toBeVisible();
    await userEvent.click(add);
    await expect(made).toBe(1);
  }}>
  <LiveBuildList onnewfolder={() => made++} />
</Story>

<Story
  name="Table"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await fits(canvasElement);
    const grid = within(canvasElement).getByRole('grid', { name: 'Builds' });
    const scroller = grid.parentElement!;
    await expect(scroller.scrollWidth).toBeGreaterThan(scroller.clientWidth);
    const name = grid.querySelector<HTMLElement>('[data-build] [data-col=name]')!;
    scroller.scrollLeft = 200;
    await waitFor(() => expect(Math.abs(name.getBoundingClientRect().left - scroller.getBoundingClientRect().left)).toBeLessThanOrEqual(2));
    const box = scroller.getBoundingClientRect();
    const folders = [...grid.querySelectorAll<HTMLElement>('[data-folder]')];
    await expect(folders.length).toBeGreaterThan(0);
    for (const f of folders) {
      const r = f.getBoundingClientRect();
      await expect(Math.abs(r.left - box.left)).toBeLessThanOrEqual(2);
      await expect(r.right).toBeLessThanOrEqual(box.right + 1);
      for (const cell of f.querySelectorAll<HTMLElement>('[role=gridcell]'))
        await expect(cell.getBoundingClientRect().right).toBeLessThanOrEqual(box.right + 1);
    }
  }}>
  <LiveBuildList phoneLayout="table" />
</Story>

<Story
  name="Folders sheet"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await fits(canvasElement);
    const c = within(canvasElement);
    await userEvent.click(c.getByRole('button', { name: /^Folders · / }));
    const sheet = await waitFor(() => within(document.body).getByRole('dialog', { name: 'Folders' }));
    await userEvent.click(
      within(sheet)
        .getAllByRole('button')
        .find((x) => !x.hasAttribute('aria-label') && x.textContent?.includes('League starters'))!
    );
    await waitFor(() => expect(sheet).not.toBeVisible());
    await expect(c.getByRole('button', { name: /^Folders · League starters/ })).toBeVisible();
  }}>
  <LiveBuildList />
</Story>
