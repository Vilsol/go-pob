<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import PwaStatus from './PwaStatus.svelte';
  import { updateAvailable, updateChanges } from './index';
  import { changesSince, parseChangelog } from '$lib/design/help';
  import changelog from '../../../static/changelog.txt?raw';

  // Main.lua:1300-1334
  const { Story } = defineMeta({ title: 'App/Update prompt', component: PwaStatus, parameters: { layout: 'fullscreen' } });
  const running = parseChangelog(changelog)[2].version;
</script>

<Story
  name="Update with changes"
  play={async ({ canvasElement }) => {
    updateChanges.set(changesSince(changelog, running));
    updateAvailable.set(true);
    const c = within(canvasElement);
    const prompt = await c.findByRole('alert');
    await expect(prompt).toHaveTextContent('A new version of go-pob is available.');
    await userEvent.click(within(prompt).getByRole('button', { name: 'Changes…' }));
    const first = parseChangelog(changelog)[0];
    await expect(within(prompt).getByText(`Version ${first.version} (${first.date})`)).toBeVisible();
    const r = prompt.getBoundingClientRect();
    await expect(r.top).toBeGreaterThanOrEqual(0);
    await expect(r.right).toBeLessThanOrEqual(window.innerWidth);
  }} />
