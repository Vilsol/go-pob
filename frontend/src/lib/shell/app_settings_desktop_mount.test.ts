// @vitest-environment jsdom

import { expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, fireEvent, cleanup, within, waitFor } from '@testing-library/svelte';
import { get } from 'svelte/store';

afterEach(cleanup);

const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
});
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted dialog threw').toEqual([]);
});

if (!HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = function () {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function () {
    this.open = false;
    this.dispatchEvent(new Event('close'));
  };
}
globalThis.matchMedia = ((q: string) => ({
  matches: false,
  media: q,
  addEventListener: () => {},
  removeEventListener: () => {}
})) as unknown as typeof matchMedia;

const fake = {
  SetShowFlavourText: vi.fn(() => Promise.resolve()),
  SetItemModDebug: vi.fn(() => Promise.resolve()),
  SetDefaultItemAffixQuality: vi.fn(() => Promise.resolve()),
  GameDataVersion: vi.fn(() => Promise.resolve('3.29')),
  BuildInfo: vi.fn(() => Promise.resolve({ Deps: {} })),
  ExportBuilds: vi.fn(() => Promise.resolve([{ path: 'Top build.xml', data: new TextEncoder().encode('<PathOfBuilding/>') }]))
};
vi.mock('$lib/go/worker', () => ({ syncWrap: fake, syncWorker: undefined }));
vi.mock('$lib/pwa', () => ({ checkForUpdate: vi.fn(() => Promise.resolve()) }));

vi.stubEnv('VITE_DESKTOP', '1');
const AppSettings = (await import('./AppSettings.svelte')).default;
const { toasts, clearToasts } = await import('$lib/design/toast');

it('Export all builds goes through the save dialog: the saved path, or nothing when cancelled', async () => {
  const click = vi.spyOn(HTMLAnchorElement.prototype, 'click');
  const answers = [new Response('/home/u/go-pob-builds.zip', { status: 200 }), new Response(null, { status: 204 })];
  const fetchMock = vi.fn<(url: string, init?: RequestInit) => Promise<Response>>((url) =>
    Promise.resolve(url.endsWith('?size') ? Response.json({ bytes: 0 }) : (answers.shift() ?? new Response('', { status: 500 })))
  );
  const saves = () => fetchMock.mock.calls.filter(([url]) => url.startsWith('/desktop/save'));
  vi.stubGlobal('fetch', fetchMock);
  clearToasts();

  const view = render(AppSettings, { open: true });
  const s = within(view.getByRole('dialog', { name: 'Settings' }));
  await fireEvent.click(s.getByRole('tab', { name: /^Data and storage/ }));
  expect(s.getByText('Every build of the builds folder in one .zip')).toBeTruthy();

  await fireEvent.click(s.getByRole('button', { name: 'Export all builds' }));
  await waitFor(() => expect(get(toasts).map((t) => t.message)).toEqual(['Exported 1 build\n/home/u/go-pob-builds.zip']));
  const [url, init] = saves()[0];
  expect(url).toMatch(/^\/desktop\/save\?name=go-pob-builds-\d{4}-\d{2}-\d{2}\.zip$/);
  expect(init?.method).toBe('POST');
  expect([...(init?.body as Uint8Array).slice(0, 4)]).toEqual([0x50, 0x4b, 0x03, 0x04]);

  await fireEvent.click(s.getByRole('button', { name: 'Export all builds' }));
  await waitFor(() => expect(saves()).toHaveLength(2));
  expect(get(toasts)).toHaveLength(1);
  expect(click).not.toHaveBeenCalled();
  click.mockRestore();
  vi.unstubAllGlobals();
});

it("Cached data is the size of the host's cache folder, not the webview's storage estimate", async () => {
  Object.defineProperty(navigator, 'storage', { configurable: true, value: { estimate: () => Promise.resolve({ usage: 999 * 1048576 }) } });
  const fetchMock = vi.fn((url: string) =>
    Promise.resolve(url === '/desktop/fs/cache/?size' ? Response.json({ bytes: 2.4 * 1048576 }) : new Response('', { status: 404 }))
  );
  vi.stubGlobal('fetch', fetchMock);

  const view = render(AppSettings, { open: true });
  const s = within(view.getByRole('dialog', { name: 'Settings' }));
  await fireEvent.click(s.getByRole('tab', { name: /^Data and storage/ }));
  await waitFor(() => expect(s.getByText('2.4 MB')).toBeTruthy());
  expect(s.queryByText('999.0 MB')).toBeNull();
  vi.unstubAllGlobals();
});
