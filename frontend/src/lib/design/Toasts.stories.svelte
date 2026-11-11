<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import Toasts from './Toasts.svelte';
  import { addToast, clearToasts } from './toast';
  import { showUpdate, updateChanges } from '$lib/pwa';
  import { changesSince, parseChangelog } from '$lib/design/help';
  import changelog from '../../../static/changelog.txt?raw';

  const { Story } = defineMeta({ title: 'Spec/Toasts', component: Toasts });
  const running = parseChangelog(changelog)[2].version;
</script>

<script lang="ts">
  clearToasts();
  addToast('No update available\nYou are running the latest version.');
  addToast('Power builder\nCalculating node power: 40%');
</script>

<Story name="Two toasts" asChild>
  <div class="relative h-60 w-100">
    <Toasts />
  </div>
</Story>

<!-- Main.lua:1300-1334 -->
<Story
  name="Update with changes"
  asChild
  play={async ({ canvasElement }) => {
    clearToasts();
    updateChanges.set(changesSince(changelog, running));
    showUpdate();
    const c = within(canvasElement);
    const prompt = await c.findByRole('alert', { name: 'Update Available' });
    await expect(prompt).toHaveTextContent('A new version of go-pob is available.');
    await expect(c.getAllByRole('alert')).toHaveLength(1);
    await userEvent.click(within(prompt).getByRole('button', { name: 'Changes…' }));
    const first = parseChangelog(changelog)[0];
    await expect(within(prompt).getByText(`Version ${first.version} (${first.date})`)).toBeVisible();
    const r = prompt.getBoundingClientRect();
    await expect(r.top).toBeGreaterThanOrEqual(0);
    await expect(r.right).toBeLessThanOrEqual(window.innerWidth);
  }}>
  <div class="relative h-60 w-100">
    <Toasts />
  </div>
</Story>
