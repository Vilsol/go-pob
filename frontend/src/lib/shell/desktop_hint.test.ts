// @vitest-environment jsdom

import { expect, it, vi } from 'vitest';

vi.mock('$lib/go/worker', () => ({ syncWrap: undefined, syncWorker: undefined }));
vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.stubEnv('VITE_DESKTOP', '1');

it('shows the hint the host answers with, once', async () => {
  const fetchMock = vi.fn(() => Promise.resolve(new Response('Builds folder\nchoose it', { status: 200 })));
  vi.stubGlobal('fetch', fetchMock);
  await import('./link_import');
  const { toasts } = await import('$lib/design/toast');
  const { get } = await import('svelte/store');
  await vi.waitFor(() => expect(get(toasts).map((t) => t.message)).toEqual(['Builds folder\nchoose it']));
  expect(fetchMock).toHaveBeenCalledWith('/desktop/hint', { method: 'POST' });
  expect(fetchMock).toHaveBeenCalledTimes(1);
});

it('shows nothing when the host has no hint', async () => {
  vi.resetModules();
  vi.stubGlobal(
    'fetch',
    vi.fn(() => Promise.resolve(new Response(null, { status: 204 })))
  );
  await import('./link_import');
  const { toasts } = await import('$lib/design/toast');
  const { get } = await import('svelte/store');
  await new Promise((r) => setTimeout(r, 20));
  expect(get(toasts)).toEqual([]);
});
