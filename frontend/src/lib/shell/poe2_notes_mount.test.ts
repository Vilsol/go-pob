// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import type { Editor } from '@tiptap/core';

import { docToPlain, type DocNode } from '$lib/notes/pob_text';

afterEach(cleanup);

const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
});
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted page threw').toEqual([]);
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

// the PoE1 worker answers nothing: no PoE1 build is open
const poe1 = vi.hoisted(() => ({}) as Record<string, ReturnType<typeof vi.fn>>);
const syncWrap = new Proxy(poe1, { get: (t, k: string) => t[k] ?? (() => Promise.resolve(undefined)) });
vi.mock('$lib/go/worker', () => ({ syncWrap, syncWorker: undefined }));
vi.mock('$app/navigation', () => ({ goto: vi.fn(() => Promise.resolve()), pushState: vi.fn() }));
vi.mock('$app/state', () => ({ page: { state: {}, url: new URL('http://localhost/notes') } }));

// the PoE2 worker's notes as wasm2/exposition keeps them: the Notes text and the rich document beside it
const held = vi.hoisted(() => ({ status: 'none', plain: '', doc: '' }));
const w2 = vi.hoisted(() => ({
  Notes: vi.fn(() => Promise.resolve({ ...held })),
  SetNotes: vi.fn((plain: string, doc: string) => {
    Object.assign(held, { status: 'match', plain, doc });
    return Promise.resolve();
  })
}));
vi.mock('$lib/go/worker2', () => ({ poe2Worker: () => Promise.resolve(w2) }));

const { poe2Build } = await import('$lib/global');
const { opening } = await import('./build_name.svelte');
const calc = () => ({ className: 'Sorceress', ascendClassName: 'None', level: 40, output: { Life: 683 }, skillFlags: {}, socketGroups: [] });

afterEach(() => {
  poe2Build.set(undefined);
  opening.modFlag = false;
  w2.SetNotes.mockClear();
});

const mount = async () => {
  const Page = (await import('../../routes/notes/+page.svelte')).default;
  const view = render(Page);
  const box = await view.findByRole('textbox', { name: 'Notes' });
  const editor = (box as unknown as { editor: Editor }).editor;
  expect(editor, 'Tiptap did not attach to the editor element').toBeDefined();
  return { ...view, box, editor };
};

const RICH: DocNode = {
  type: 'doc',
  content: [
    { type: 'heading', attrs: { level: 1 }, content: [{ type: 'text', text: 'Plan' }] },
    { type: 'paragraph', content: [{ type: 'text', text: 'spark' }] }
  ]
};

describe('the Notes page for a PoE2 build', () => {
  // NotesTab.lua:75-82 (PoB-PoE2)
  it("shows the build's <Notes> text with its colour codes converted, and mounting writes nothing", async () => {
    Object.assign(held, { status: 'none', plain: '^xE05030fire^7 and ^4blue', doc: '' });
    poe2Build.set(calc());
    const { box, findByRole } = await mount();
    await vi.waitFor(() => expect(box.textContent).toBe('fire and blue'));
    expect(box.querySelector('span[data-pob-color="^xE05030"]')?.textContent).toBe('fire');
    expect((await findByRole('status')).textContent).toContain('Colour codes converted: 3 ');
    await new Promise((r) => setTimeout(r, 600));
    expect(w2.SetNotes).not.toHaveBeenCalled();
  });

  // NotesTab.lua:84-88, 111 (PoB-PoE2): the edit is what Save writes, and leaves the build unsaved
  it('an edit writes the PoB text and the document together, and leaves the build unsaved', async () => {
    Object.assign(held, { status: 'none', plain: 'plain words', doc: '' });
    poe2Build.set(calc());
    const bar = render((await import('./AppTopBar.svelte')).default);
    await bar.findByRole('button', { name: 'Save (no unsaved changes)' });
    const { box, editor } = await mount();
    await vi.waitFor(() => expect(box.textContent).toBe('plain words'));

    editor.chain().focus('end').insertContent(' added').run();
    await fireEvent.blur(box);

    await vi.waitFor(() => expect(w2.SetNotes).toHaveBeenCalledTimes(1));
    const [plain, doc] = w2.SetNotes.mock.calls[0];
    expect(plain).toBe('plain words added');
    expect(docToPlain(JSON.parse(doc) as DocNode)).toBe(plain);
    await bar.findByRole('button', { name: 'Save (unsaved changes)' });
  });

  it('a stored rich document that matches <Notes> is what opens', async () => {
    Object.assign(held, { status: 'match', plain: docToPlain(RICH), doc: JSON.stringify(RICH) });
    poe2Build.set(calc());
    const { box, queryByRole } = await mount();
    await vi.waitFor(() => expect(box.querySelector('h1')?.textContent).toBe('Plan'));
    expect(queryByRole('status')).toBeNull();
  });

  it('an edit still pending when another PoE2 build opens is not written into that build', async () => {
    Object.assign(held, { status: 'none', plain: 'first build', doc: '' });
    poe2Build.set(calc());
    const { box, editor } = await mount();
    await vi.waitFor(() => expect(box.textContent).toBe('first build'));
    editor.chain().focus('end').insertContent(' edited').run();

    Object.assign(held, { status: 'none', plain: 'second build', doc: '' });
    poe2Build.set(calc());
    await vi.waitFor(() => expect(box.ownerDocument.querySelector('[aria-label="Notes"]')?.textContent).toBe('second build'));
    await new Promise((r) => setTimeout(r, 600));
    expect(w2.SetNotes).not.toHaveBeenCalled();
  });
});
