// @vitest-environment jsdom

import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';
import ActivityIndicator from './ActivityIndicator.svelte';
import ActivityBar from './ActivityBar.svelte';
import Toasts from './Toasts.svelte';
import PageLoading from './PageLoading.svelte';
import { addToast, clearToasts, toasts } from './toast';
import { SHOW_AFTER_MS, activityPanelOpen, resetJobs, startJob } from '$lib/activity/jobs';
import { followLongJobs } from '$lib/activity/job_toasts';
import { forgetDurations } from '$lib/activity/durations';

const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
});
beforeEach(() => {
  vi.useFakeTimers();
  resetJobs();
  forgetDurations();
});
afterEach(() => {
  cleanup();
  clearToasts();
  resetJobs();
  vi.useRealTimers();
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted component threw').toEqual([]);
});

const advance = async (ms: number) => {
  vi.advanceTimersByTime(ms);
  await tick();
};
const indicator = () => screen.queryByRole('button', { name: /^Activity/ });

describe('the activity indicator', () => {
  it('does not appear for a job that ends inside the reveal delay', async () => {
    render(ActivityIndicator);
    render(ActivityBar);
    const j = startJob('save', 'Saving');
    await advance(SHOW_AFTER_MS - 50);
    expect(indicator()).toBeNull();
    j.end();
    await advance(SHOW_AFTER_MS);
    expect(indicator()).toBeNull();
    expect(screen.queryByRole('progressbar')).toBeNull();
  });

  it('appears for a slow job with its label and phase, and the bar sweeps', async () => {
    render(ActivityIndicator);
    render(ActivityBar);
    startJob('import', 'Importing build').phase('Decoding');
    await advance(SHOW_AFTER_MS);
    expect(indicator()?.getAttribute('aria-label')).toBe('Activity: Importing build · Decoding');
    expect(screen.getByRole('status').textContent).toBe('Importing build · Decoding');
    const bar = screen.getByRole('progressbar', { name: 'Activity' });
    expect(bar.getAttribute('aria-valuenow')).toBeNull();
  });

  it('turns the bar determinate once a running job knows its total', async () => {
    render(ActivityBar);
    startJob('data', 'Loading game data').progress(1, 4, 'bytes');
    await advance(SHOW_AFTER_MS);
    expect(screen.getByRole('progressbar', { name: 'Activity' }).getAttribute('aria-valuenow')).toBe('25');
  });

  it('opens the panel from the keyboard-reachable button and closes it with Escape, back on the button', async () => {
    render(ActivityIndicator);
    const j = startJob('open', 'Opening Witch');
    j.phase('Reading and calculating');
    await advance(SHOW_AFTER_MS + 1200);
    j.end('error', new Error('not a build'));
    await tick();
    const btn = indicator()!;
    expect(btn.getAttribute('aria-label')).toBe('Activity: 1 failed');
    await fireEvent.click(btn);
    const panel = screen.getByRole('dialog', { name: 'Activity' });
    expect(document.activeElement).toBe(panel);
    expect(panel.textContent).toContain('Opening Witch');
    expect(screen.getByRole('alert').textContent).toBe('not a build');
    expect(screen.getByRole('list', { name: 'Phases of Opening Witch' }).textContent).toContain('Reading and calculating');
    await fireEvent.keyDown(panel, { key: 'Escape' });
    expect(screen.queryByRole('dialog')).toBeNull();
    expect(document.activeElement).toBe(btn);
  });
});

describe('progress toasts', () => {
  it('follow a long job and go the moment it finishes', async () => {
    render(Toasts);
    const stop = followLongJobs();
    const j = startJob('boot', 'Starting the engine', { long: true });
    await advance(SHOW_AFTER_MS);
    j.phase('Downloading').progress(50, 200, 'bytes');
    await tick();
    const toast = screen.getByRole('status', { name: 'Starting the engine' });
    expect(toast.textContent).toContain('Downloading · 50 B / 200 B');
    expect(screen.getByRole('progressbar', { name: 'Starting the engine' }).getAttribute('aria-valuenow')).toBe('25');
    await advance(1700);
    j.end();
    await tick();
    expect(get(toasts), 'a finished job leaves nothing over the page').toEqual([]);
    stop();
  });

  it('collapse into the panel with Details and are not shown again', async () => {
    render(Toasts);
    const stop = followLongJobs();
    const j = startJob('import', 'Importing build', { long: true });
    await advance(SHOW_AFTER_MS);
    await fireEvent.click(screen.getByRole('button', { name: 'Details' }));
    expect(get(activityPanelOpen)).toBe(true);
    j.phase('Calculating');
    await tick();
    expect(get(toasts)).toEqual([]);
    j.end('error', 'refused');
    await tick();
    expect(get(toasts)).toEqual([]);
    stop();
  });

  it('stay up when a drawn long job fails', async () => {
    render(Toasts);
    const stop = followLongJobs();
    startJob('boot', 'Starting the engine', { long: true }).end('error', 'not a wasm module');
    await tick();
    expect(get(toasts), 'a failure before the job is drawn is shown where it was started').toEqual([]);
    const j = startJob('data', 'Loading game data', { long: true });
    await advance(SHOW_AFTER_MS);
    j.end('error', 'offline');
    await advance(60_000);
    expect(screen.getByRole('status', { name: 'Loading game data failed' }).textContent).toContain('offline');
    stop();
  });

  it('share the corner with the floating indicator, which shows only what no toast already says', async () => {
    render(Toasts, { activity: true });
    const stop = followLongJobs();
    const boot = startJob('boot', 'Starting the engine', { long: true });
    await advance(SHOW_AFTER_MS);
    expect(screen.getByRole('status', { name: 'Starting the engine' })).toBeTruthy();
    expect(indicator(), 'the job already has its toast').toBeNull();
    const save = startJob('save', 'Saving');
    await advance(SHOW_AFTER_MS);
    expect(indicator()?.getAttribute('aria-label')).toBe('Activity: Saving');
    save.end();
    boot.end('error', 'offline');
    await tick();
    expect(indicator(), 'the failure is on its toast').toBeNull();
    await fireEvent.click(screen.getByRole('button', { name: 'Dismiss' }));
    expect(indicator()?.getAttribute('aria-label')).toBe('Activity: 1 failed');
    stop();
  });

  it('leave a plain toast as it was: no timeout, no spinner, no progress', async () => {
    render(Toasts);
    addToast('Link copied\nhttps://example');
    await advance(60_000);
    expect(screen.getByRole('status', { name: 'Link copied' })).toBeTruthy();
    expect(screen.queryByRole('progressbar')).toBeNull();
  });
});

describe('a loading page', () => {
  it('says nothing before the reveal delay, then shows its label', async () => {
    render(PageLoading, { label: 'Loading the tree…' });
    expect(screen.getByRole('status').textContent?.trim()).toBe('');
    await advance(SHOW_AFTER_MS);
    expect(screen.getByRole('status').textContent?.trim()).toBe('Loading the tree…');
  });
});
