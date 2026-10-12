// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, fireEvent, cleanup } from '@testing-library/svelte';
import { get } from 'svelte/store';
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

vi.mock('$lib/go/worker', async () => {
  const { remoteWorker } = await import('../../../harness/component/go_runtime');
  return { syncWrap: remoteWorker(), syncWorker: undefined };
});

const { bootWorker, loadCorpusBuild } = await import('../../../harness/component/go_runtime');
const { currentBuild } = await import('$lib/global');
const { browser } = await import('$app/environment');
const { workerInstance } = await import('$lib/go/sync_worker');

const BUILD = '0001-necromancer-unknown.xml';

beforeAll(async () => {
  await bootWorker(currentBuild);
  await loadCorpusBuild(BUILD);
}, 120_000);

const go = () => workerInstance.currentBuild!;

const mount = async () => {
  const Page = (await import('./+page.svelte')).default;
  const view = render(Page);
  const box = await view.findByRole('textbox', { name: 'Notes' });
  const editor = (box as unknown as { editor: Editor }).editor;
  expect(editor, 'Tiptap did not attach to the editor element').toBeDefined();
  return { ...view, box, editor };
};

const atEnd = (editor: Editor, text: string) =>
  editor
    .chain()
    .setTextSelection(editor.state.doc.content.size - 1)
    .insertContent(text)
    .run();

const RICH: DocNode = {
  type: 'doc',
  content: [
    { type: 'heading', attrs: { level: 1 }, content: [{ type: 'text', text: 'Plan' }] },
    {
      type: 'paragraph',
      content: [{ type: 'text', text: 'fire', marks: [{ type: 'pobColor', attrs: { code: '^xB97123', lead: null, run: 0 } }, { type: 'bold' }] }]
    }
  ]
};

describe('the harness is not vacuous', () => {
  it('resolves the BROWSER build of $app/environment, so pages take their real path', () => {
    expect(browser).toBe(true);
  });

  it('booted the real Go module and holds a real build with the rich-notes surface', () => {
    expect(workerInstance.booted).toBe(true);
    expect(typeof go().SetRichNotes).toBe('function');
    expect(get(currentBuild)).toBeDefined();
  });
});

describe('notes/+page.svelte', () => {
  it('shows <Notes> converted: colour codes become coloured text', async () => {
    go().SetNotes('^xFF0000red\nstill red^7 plain');
    const { box } = await mount();
    await vi.waitFor(() => expect(box.textContent).toBe('redstill red plain'));
    const red = box.querySelectorAll('span[data-pob-color="^xFF0000"]');
    expect([...red].map((s) => s.textContent)).toEqual(['red', 'still red']);
    expect((red[0] as HTMLElement).style.color).toBe('rgb(255, 0, 0)');
  });

  it('mounting alone writes nothing', async () => {
    go().SetNotes('^1untouched');
    const { box } = await mount();
    await vi.waitFor(() => expect(box.textContent).toBe('untouched'));
    await new Promise((r) => setTimeout(r, 600));
    expect(go().Notes).toBe('^1untouched');
    expect(go().RichNotesStatus()).toBe('none');
  });

  it('an edit writes the document and its PoB text together', async () => {
    go().SetNotes('^xAF6025unique^7 text');
    const { box, editor } = await mount();
    await vi.waitFor(() => expect(box.textContent).toBe('unique text'));

    editor.chain().focus('end').insertContent(' added').run();
    await fireEvent.blur(box);

    await vi.waitFor(() => expect(go().Notes).toBe('^xAF6025unique^7 text added'));
    expect(go().RichNotesStatus()).toBe('match');
    expect(docToPlain(JSON.parse(go().RichNotesDoc()) as DocNode)).toBe(go().Notes);
  });

  it('the colour palette recolours the selection with a PoB colour code', async () => {
    go().SetNotes('plain words');
    const { box, editor, getByRole } = await mount();
    await vi.waitFor(() => expect(box.textContent).toBe('plain words'));

    editor.commands.selectAll();
    await fireEvent.click(getByRole('button', { name: 'Text colour' }));
    await fireEvent.click(getByRole('option', { name: 'FIRE' }));

    await vi.waitFor(() => expect(go().Notes).toBe('^xB97123plain words'));
  });

  it('a stored rich document that matches <Notes> is what opens', async () => {
    go().SetRichNotes(docToPlain(RICH), JSON.stringify(RICH));
    const { box, queryByRole } = await mount();
    await vi.waitFor(() => expect(box.querySelector('h1')?.textContent).toBe('Plan'));
    expect(box.querySelector('strong span[data-pob-color="^xB97123"], span[data-pob-color="^xB97123"] strong')?.textContent).toBe('fire');
    expect(queryByRole('status')).toBeNull();
  });

  it('<Notes> edited outside go-pob: the PoB text opens, and the banner restores the rich version', async () => {
    go().SetRichNotes(docToPlain(RICH), JSON.stringify(RICH));
    go().SetNotes('edited in PoB');
    const { findByRole, getByRole } = await mount();

    const banner = await findByRole('status');
    expect(banner.textContent).toContain('CHANGED OUTSIDE GO-POB');
    await vi.waitFor(() => expect(getByRole('textbox', { name: 'Notes' }).textContent).toBe('edited in PoB'));
    expect(go().Notes, 'showing the banner must not write').toBe('edited in PoB');

    await fireEvent.click(getByRole('button', { name: 'Restore rich version' }));
    await vi.waitFor(() => expect(go().Notes).toBe(docToPlain(RICH)));
    expect(go().RichNotesStatus()).toBe('match');
    expect(go().RichNotesDoc(), 'the restore writes the stored document as it was').toBe(JSON.stringify(RICH));
    await vi.waitFor(() => expect(getByRole('textbox', { name: 'Notes' }).querySelector('h1')?.textContent).toBe('Plan'));
  });

  it('View as PoB text shows exactly the text <Notes> would hold', async () => {
    go().SetRichNotes(docToPlain(RICH), JSON.stringify(RICH));
    const { getByRole, findByRole } = await mount();
    await fireEvent.click(getByRole('checkbox', { name: 'View as PoB text' }));
    const pob = await findByRole('region', { name: 'PoB text' });
    expect(pob.textContent).toBe(go().Notes);
    expect(pob.textContent).toBe('Plan\n^xB97123fire');
  });

  it('Copy as PoB text puts the <Notes> text on the clipboard', async () => {
    go().SetRichNotes(docToPlain(RICH), JSON.stringify(RICH));
    const writeText = vi.fn(() => Promise.resolve());
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true });
    const { getByRole, box } = await mount();
    await vi.waitFor(() => expect(box.querySelector('h1')?.textContent).toBe('Plan'));
    await fireEvent.click(getByRole('button', { name: 'Copy as PoB text' }));
    expect(writeText).toHaveBeenCalledWith('Plan\n^xB97123fire');
  });

  it('an edit waits for the debounce, and a blur writes it at once', async () => {
    go().SetNotes('first');
    const { box, editor } = await mount();
    await vi.waitFor(() => expect(box.textContent).toBe('first'));

    atEnd(editor, ' second');
    await new Promise((r) => setTimeout(r, 100));
    expect(go().Notes, 'written before the debounce').toBe('first');
    await fireEvent.blur(box);
    await vi.waitFor(() => expect(go().Notes).toBe('first second'), { timeout: 150 });
  });

  it('an edit still pending when the page is left is written', async () => {
    go().SetNotes('leaving');
    const { box, editor, unmount } = await mount();
    await vi.waitFor(() => expect(box.textContent).toBe('leaving'));

    atEnd(editor, ' now');
    unmount();
    await vi.waitFor(() => expect(go().Notes).toBe('leaving now'), { timeout: 150 });
  });

  it('<Notes> with colour codes and no rich version: the import banner counts them, and a dismissal holds for the build', async () => {
    await loadCorpusBuild(BUILD);
    expect(go().RichNotesStatus()).toBe('none');
    go().SetNotes('^xFF0000red^7 and ^4blue');
    const first = await mount();
    const banner = await first.findByRole('status');
    expect(banner.textContent).toContain('IMPORTED FROM POB');
    expect(banner.textContent).toContain('Colour codes converted: 3 ');
    await fireEvent.click(first.getByRole('button', { name: 'Dismiss' }));
    expect(first.queryByRole('status')).toBeNull();
    first.unmount();

    const again = await mount();
    await vi.waitFor(() => expect(again.box.textContent).toBe('red and blue'));
    expect(again.queryByRole('status'), 'the dismissed banner came back on the same build').toBeNull();
    expect(go().RichNotesStatus(), 'dismissing writes nothing').toBe('none');
  });
});
