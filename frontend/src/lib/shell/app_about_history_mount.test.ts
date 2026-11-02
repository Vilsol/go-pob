// @vitest-environment jsdom
// Main.lua:1338-1358, 1434-1437

import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest';
import { cleanup, render, within } from '@testing-library/svelte';
import { parseChangelog } from '$lib/design/help';

afterEach(cleanup);

const own = readFileSync(resolve(__dirname, '../../../static/changelog.txt'), 'utf8');
const pob = readFileSync(resolve(__dirname, '../../../static/pob-changelog.txt'), 'utf8');

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
  vi.stubGlobal('fetch', (url: string) => {
    const file = String(url).split('/').at(-1);
    return Promise.resolve(new Response(file === 'changelog.txt' ? own : file === 'pob-changelog.txt' ? pob : ''));
  });
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

describe('AppAbout version history', () => {
  it("lists go-pob's versions first, then Path of Building's under its own sub-title at the bottom", async () => {
    const view = render(AppAbout, { open: true, tab: 'history' });
    const dlg = within(view.getByRole('dialog', { name: 'About go-pob' }));
    const heading = (v: { version: string; date: string }) => `Version ${v.version} (${v.date})`;
    const [mine, theirs] = [parseChangelog(own), parseChangelog(pob)];
    // a role query that throws before the history renders makes that render several times slower in jsdom
    const versions = () => dlg.queryAllByRole('heading', { name: /^Version / }).map((h) => h.textContent);
    await vi.waitFor(() => expect(versions().length).toBe(mine.length + theirs.length));
    const headings = versions();
    expect(headings[0]).toBe(heading(mine[0]));
    expect(headings[mine.length - 1]).toBe(heading({ ...mine.at(-1)!, version: '0.1.0' }));
    expect(headings[mine.length]).toBe(heading(theirs[0]));
    expect(headings.at(-1)).toBe(heading(theirs.at(-1)!));
    const part = dlg.getByText('--- Path of Building ---');
    const first = dlg.getByRole('heading', { name: heading(theirs[0]) });
    expect(part.compareDocumentPosition(first) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(dlg.getByRole('heading', { name: heading(mine.at(-1)!) }).compareDocumentPosition(part) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });
});
