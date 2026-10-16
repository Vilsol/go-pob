// @vitest-environment jsdom

import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest';
import { cleanup, render, within } from '@testing-library/svelte';

afterEach(() => {
  cleanup();
  vi.unstubAllEnvs();
});

const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
  if (!HTMLDialogElement.prototype.showModal) {
    HTMLDialogElement.prototype.showModal = function () {
      this.open = true;
    };
    HTMLDialogElement.prototype.close = function () {
      this.open = false;
      this.dispatchEvent(new Event('close'));
    };
  }
  vi.stubGlobal('fetch', () => Promise.resolve(new Response('')));
});
afterEach(() => {
  expect(pageErrors.splice(0, pageErrors.length), 'the mounted dialog threw').toEqual([]);
});

const fake = {
  BuildInfo: vi.fn(() => Promise.resolve({ GoVersion: 'go1.27.2', Main: { Version: '(devel)' } })),
  GameDataVersion: vi.fn(() => Promise.resolve('3.29')),
  GetTreeVersion: vi.fn(() => Promise.resolve('3_29'))
};
vi.mock('$lib/go/worker', () => ({ syncWrap: fake, syncWorker: undefined }));

const AppAbout = (await import('./AppAbout.svelte')).default;

const header = async () => {
  const view = render(AppAbout, { open: true });
  const dlg = within(view.getByRole('dialog', { name: 'About go-pob' }));
  await vi.waitFor(() => expect(fake.BuildInfo).toHaveBeenCalled());
  return dlg;
};

describe('AppAbout version', () => {
  it('shows the release version baked in at build time', async () => {
    vi.stubEnv('VITE_VERSION', '0.2.0');
    const dlg = await header();
    expect(await dlg.findAllByText('0.2.0')).toHaveLength(2);
  });

  it('shows the short sha a dev deploy bakes in', async () => {
    vi.stubEnv('VITE_VERSION', '3eb0560');
    const dlg = await header();
    expect(await dlg.findAllByText('3eb0560')).toHaveLength(2);
  });

  it("falls back to Go's module version without one", async () => {
    vi.stubEnv('VITE_VERSION', '');
    const dlg = await header();
    expect(await dlg.findAllByText('(devel)')).toHaveLength(2);
  });
});
