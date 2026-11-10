<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, waitFor, within } from 'storybook/test';
  import ActivityIndicator from './ActivityIndicator.svelte';

  const { Story } = defineMeta({ title: 'Spec/Activity', component: ActivityIndicator });
</script>

<script lang="ts">
  import { onMount } from 'svelte';
  import ActivityBar from './ActivityBar.svelte';
  import PageLoading from './PageLoading.svelte';
  import Toasts from './Toasts.svelte';
  import { clearToasts } from './toast';
  import { resetJobs, startJob } from '$lib/activity/jobs';
  import { followLongJobs } from '$lib/activity/job_toasts';

  // jobs started in the past, so they are drawn at once
  const ago = (s: number) => ({ startedAt: Date.now() - s * 1000 });
  onMount(() => {
    resetJobs();
    clearToasts();
    const stop = followLongJobs();
    const saved = startJob('save', 'Saving Necromancer', ago(3));
    saved.phase('Writing');
    // a job that ends before it is drawn is never shown; end this one once it is
    const ended = setTimeout(() => saved.end(), 50);
    startJob('open', 'Opening Old build', ago(2)).end('error', new Error('The build file is not valid XML.'));
    startJob('import', 'Importing build', { ...ago(4), long: true })
      .phase('Decoding')
      .phase('Calculating in the worker');
    startJob('data', 'Loading game data', { ...ago(6), long: true })
      .phase('Downloading')
      .progress(14_200_000, 38_000_000, 'bytes');
    return () => {
      clearTimeout(ended);
      stop();
      resetJobs();
      clearToasts();
    };
  });
</script>

<Story
  name="Running and finished"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement as HTMLElement);
    const btn = await c.findByRole('button', { name: /^Activity: Loading game data · Downloading/ });
    await waitFor(() => expect(c.getByRole('progressbar', { name: 'Activity' }).getAttribute('aria-valuenow')).toBe('37'));
    await userEvent.click(btn);
    const panel = await c.findByRole('dialog', { name: 'Activity' });
    await expect(within(panel).getByText('The build file is not valid XML.')).toBeTruthy();
    await expect(await within(panel).findByRole('list', { name: 'Phases of Saving Necromancer' })).toBeTruthy();
    await userEvent.keyboard('{Escape}');
    await waitFor(() => expect(c.queryByRole('dialog', { name: 'Activity' })).toBeNull());
    await expect(document.activeElement).toBe(btn);
  }}>
  <div class="relative h-130 w-180 bg-ground p-4">
    <ActivityBar />
    <div class="flex justify-end"><ActivityIndicator /></div>
    <Toasts />
  </div>
</Story>

<Story name="Page loading" asChild>
  <div class="flex h-40 w-120 bg-ground"><PageLoading label="Loading the tree…" /></div>
</Story>
