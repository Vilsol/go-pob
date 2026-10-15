// @vitest-environment jsdom

import { afterEach, describe, expect, it, vi } from 'vitest';

const worker = vi.hoisted(() => ({ ImportCode: vi.fn((code: string) => Promise.resolve(void code)) }));
const goto = vi.hoisted(() => vi.fn(() => Promise.resolve()));
vi.mock('$lib/go/worker', () => ({ syncWrap: worker, syncWorker: undefined }));
vi.mock('$app/navigation', () => ({ goto }));

vi.stubEnv('VITE_DESKTOP', '1');
(window as Window & { pobDesktopPending?: string[] }).pobDesktopPending = ['gopob-file:EARLY'];
const { openImportLink } = await import('./link_import');
const { markBackendAsLoaded } = await import('$lib/global.js');
const { toasts, clearToasts } = await import('$lib/design/toast');
const { get } = await import('svelte/store');

afterEach(() => {
  vi.unstubAllGlobals();
  worker.ImportCode.mockClear();
  goto.mockClear();
  clearToasts();
});

const textResponse = (body: string, status = 200) => new Response(body, { status, headers: { 'Content-Type': 'text/plain; charset=utf-8' } });

describe('desktop build files', () => {
  it('holds a handed-over build until the game data is in, then opens it', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn((url: string) => Promise.resolve(textResponse(`code-of-${url}`)))
    );
    window.dispatchEvent(new CustomEvent('pob:desktop-import', { detail: 'gopob-file:COLD' }));
    await new Promise((r) => setTimeout(r, 20));
    expect(worker.ImportCode).not.toHaveBeenCalled();
    get(markBackendAsLoaded)();
    await vi.waitFor(() => expect(goto).toHaveBeenCalledTimes(2));
    expect(worker.ImportCode.mock.calls.map(([code]) => code)).toEqual(['code-of-/desktop/opened/EARLY', 'code-of-/desktop/opened/COLD']);
    expect((window as Window & { pobDesktopPending?: string[] }).pobDesktopPending).toEqual([]);
  });

  it('reads an opened file back as its code', async () => {
    const fetchMock = vi.fn((url: string) => Promise.resolve(textResponse(` code-of-${url} \n`)));
    vi.stubGlobal('fetch', fetchMock);
    await openImportLink('gopob-file:TOKEN1');
    expect(fetchMock).toHaveBeenCalledWith('/desktop/opened/TOKEN1');
    expect(worker.ImportCode).toHaveBeenCalledWith('code-of-/desktop/opened/TOKEN1');
  });

  it("opens the host's event and shows the tree, or says why not", async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(textResponse('eNoLcvVz8QQABIYBug==')))
    );
    window.dispatchEvent(new CustomEvent('pob:desktop-import', { detail: 'gopob-file:T2' }));
    await vi.waitFor(() => expect(goto).toHaveBeenCalledWith('/tree'));
    expect(worker.ImportCode).toHaveBeenCalledWith('eNoLcvVz8QQABIYBug==');

    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(textResponse('this build file is no longer open; open it again', 404)))
    );
    window.dispatchEvent(new CustomEvent('pob:desktop-import', { detail: 'gopob-file:gone' }));
    await vi.waitFor(() => expect(get(toasts).length).toBe(1));
    expect(get(toasts)[0].message).toContain('could not read the build from the build file');
  });
});
