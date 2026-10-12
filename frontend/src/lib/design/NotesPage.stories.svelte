<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, waitFor, within } from 'storybook/test';
  import NotesPage from './NotesPage.svelte';
  import PageFrame from './PageFrame.svelte';
  import LiveTopBar from './stories/LiveTopBar.svelte';
  import LiveSidebar from './stories/LiveSidebar.svelte';
  import LiveNotes from './stories/LiveNotes.svelte';
  import { spectreBuild, fireballBuild, novaBuild, notesSamples, type BuildFixture } from './fixtures';
  import { expectPageFits, expectFloating, expectDarkCheckboxes, expectSelectsFloating } from './stories/layout_checks';

  const { Story } = defineMeta({
    title: 'Pages/Notes',
    component: NotesPage,
    parameters: {
      layout: 'fullscreen',
      a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } }
    }
  });

  const vp = (value: 'min' | 'mid' | 'target') => ({ viewport: { value, isRotated: false } });
  const page = (el: HTMLElement) => {
    const c = within(el);
    return { c, main: c.getByRole('main', { name: 'Notes' }), editor: c.getByRole('textbox', { name: 'Notes' }) };
  };
  const [plain, lists, headings] = notesSamples;

  const ready = async (el: HTMLElement) => {
    const { c, editor } = page(el);
    await waitFor(() => expect(editor.querySelector('p')).not.toBeNull());
    return { c, editor, bar: within(c.getByRole('toolbar', { name: 'Formatting' })) };
  };
  const last = (el: HTMLElement, sel: string) => [...el.querySelectorAll<HTMLElement>(sel)].at(-1)!;
  const newLine = async (editor: HTMLElement) => {
    await userEvent.click(last(editor, ':scope > p'));
    await waitFor(() => expect(editor).toHaveFocus());
    await userEvent.keyboard('{Enter}');
  };
  const ctrl = (k: string, mod = '') => (mod ? `{Control>}{${mod}>}${k}{/${mod}}{/Control}` : `{Control>}${k}{/Control}`);
  const INLINE = [
    ['Bold', ctrl('b'), 'Ctrl+B', 'strong'],
    ['Italic', ctrl('i'), 'Ctrl+I', 'em'],
    ['Underline', ctrl('u'), 'Ctrl+U', 'u'],
    ['Strikethrough', ctrl('s', 'Shift'), 'Ctrl+Shift+S', 's'],
    ['Inline code', ctrl('e'), 'Ctrl+E', 'code']
  ] as const;
  const BLOCKS = [
    ['Heading 1', ctrl('1', 'Alt'), 'Ctrl+Alt+1', 'h1'],
    ['Heading 2', ctrl('2', 'Alt'), 'Ctrl+Alt+2', 'h2'],
    ['Heading 3', ctrl('3', 'Alt'), 'Ctrl+Alt+3', 'h3'],
    ['Bulleted list', ctrl('8', 'Shift'), 'Ctrl+Shift+8', 'ul:not([data-type]) li'],
    ['Numbered list', ctrl('7', 'Shift'), 'Ctrl+Shift+7', 'ol li'],
    ['Task list', ctrl('9', 'Shift'), 'Ctrl+Shift+9', 'ul[data-type="taskList"] li'],
    ['Quote', ctrl('b', 'Shift'), 'Ctrl+Shift+B', 'blockquote'],
    ['Code block', ctrl('c', 'Alt'), 'Ctrl+Alt+C', 'pre']
  ] as const;

  const shows = async (el: HTMLElement, coloured: boolean) => {
    const { c, main, editor } = page(el);
    await expectPageFits(main);
    await waitFor(() => expect(editor.textContent?.length).toBeGreaterThan(20));
    const bar = c.getByRole('toolbar', { name: 'Formatting' }).getBoundingClientRect();
    const box = c.getByRole('region', { name: 'Notes editor' }).getBoundingClientRect();
    const foot = main.querySelector<HTMLElement>('[data-notes-footer]')!.getBoundingClientRect();
    await expect(box.top).toBeGreaterThanOrEqual(bar.bottom);
    await expect(foot.top).toBeGreaterThanOrEqual(box.bottom);
    await expect(foot.top - box.bottom).toBeLessThanOrEqual(12);
    await expect(Math.round(main.getBoundingClientRect().bottom - foot.bottom)).toBeLessThanOrEqual(20);
    await expect(Math.round(bar.height)).toBeLessThanOrEqual(46);
    await expect(editor.querySelectorAll('[data-pob-color]').length > 0).toBe(coloured);
    await expect(editor.textContent).not.toMatch(/\^x[0-9A-F]{6}|\^\d/);
    await expectDarkCheckboxes(main);
    await expectSelectsFloating(el);
  };
</script>

{#snippet frame(fx: BuildFixture, text: string, mode: 'imported' | 'mismatch' | 'match' = 'imported')}
  <PageFrame label="Notes" mainClass="gap-2.5 py-4">
    {#snippet topbar()}<LiveTopBar {fx} current="#notes" />{/snippet}
    {#snippet sidebar()}<LiveSidebar {fx} />{/snippet}
    <LiveNotes {text} {mode} />
  </PageFrame>
{/snippet}

<Story
  name="Imported · target"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await shows(canvasElement, true);
    const { main } = page(canvasElement);
    const banner = within(main).getByRole('status');
    await expect(banner).toHaveTextContent(/^IMPORTED FROM POB/);
    await expect(banner).toHaveTextContent(/Colour codes converted: \d+ /);
    await userEvent.click(within(banner).getByRole('button', { name: 'Dismiss' }));
    await expect(within(main).queryByRole('status')).toBeNull();
  }}>
  {@render frame(spectreBuild, headings.text)}
</Story>

<Story name="Lists · minimum" asChild globals={vp('min')} play={async ({ canvasElement }: { canvasElement: HTMLElement }) => shows(canvasElement, true)}>
  {@render frame(fireballBuild, lists.text)}
</Story>

<Story name="Plain · mid" asChild globals={vp('mid')} play={async ({ canvasElement }: { canvasElement: HTMLElement }) => shows(canvasElement, false)}>
  {@render frame(novaBuild, plain.text, 'match')}
</Story>

<Story
  name="Formatting"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, editor } = page(canvasElement);
    const bar = within(c.getByRole('toolbar', { name: 'Formatting' }));
    await waitFor(() => expect(editor.querySelector('p')).not.toBeNull());
    await userEvent.click(editor.querySelector('p')!);
    await userEvent.click(bar.getByRole('button', { name: 'Heading 1' }));
    await waitFor(() => expect(editor.querySelector('h1')).not.toBeNull());
    await expect(bar.getByRole('button', { name: 'Heading 1' })).toHaveAttribute('aria-pressed', 'true');
    await userEvent.click(editor.querySelectorAll('p')[1]);
    await userEvent.click(bar.getByRole('button', { name: 'Bold' }));
    await expect(bar.getByRole('button', { name: 'Bold' })).toHaveAttribute('aria-pressed', 'true');
    await waitFor(() => expect(editor).toHaveFocus());
    await userEvent.keyboard('Bold words');
    await expect(editor.querySelector('strong')).toHaveTextContent('Bold words');
    const colour = bar.getByRole('button', { name: 'Text colour' });
    await userEvent.click(colour);
    const palette = c.getByRole('listbox', { name: 'PoB colours' });
    await expectFloating(colour, palette);
    await userEvent.click(within(palette).getByRole('option', { name: 'FIRE' }));
    await expect(c.queryByRole('listbox', { name: 'PoB colours' })).toBeNull();
    await waitFor(() => expect(editor).toHaveFocus());
    await userEvent.keyboard(' on fire');
    await waitFor(() => expect(editor.querySelector('[data-pob-color="^xB97123"]')).toHaveTextContent('on fire'));
    await userEvent.click(colour);
    await expect(within(c.getByRole('listbox', { name: 'PoB colours' })).getByRole('option', { name: 'FIRE' })).toHaveAttribute('aria-selected', 'true');
    await userEvent.keyboard('{Escape}');
    await expect(c.queryByRole('listbox', { name: 'PoB colours' })).toBeNull();
    await userEvent.click(c.getByRole('checkbox', { name: 'View as PoB text' }));
    const pob = c.getByRole('region', { name: 'PoB text' });
    await expect(pob).toHaveTextContent(/Bold words\^xB97123 ?on fire/);
    await expect(pob.textContent).not.toMatch(/<|\*\*|#/);
    await expect(canvasElement.querySelector('section[aria-label="Notes editor"]')).not.toBeVisible();
    await userEvent.click(c.getByRole('button', { name: 'Copy as PoB text' }));
    await expect(canvasElement.querySelector('[data-copied]')!.textContent).toMatch(/\^xB97123 ?on fire/);
    await userEvent.click(c.getByRole('checkbox', { name: 'View as PoB text' }));
    await expect(editor).toBeVisible();
  }}>
  {@render frame(novaBuild, plain.text, 'match')}
</Story>

<Story
  name="Export info"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c } = page(canvasElement);
    const info = c.getByRole('button', { name: 'How notes export to PoB' });
    const tip = c.getByRole('tooltip', { hidden: true });
    await userEvent.hover(info);
    await waitFor(() => expect(tip).toBeVisible());
    await expectFloating(info, tip);
    await userEvent.unhover(info);
    await waitFor(() => expect(tip).not.toBeVisible());
    info.focus();
    await waitFor(() => expect(tip).toBeVisible());
    info.blur();
    await waitFor(() => expect(tip).not.toBeVisible());
  }}>
  {@render frame(fireballBuild, lists.text)}
</Story>

<Story
  name="Restore rich version"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { editor, main } = page(canvasElement);
    const banner = within(main).getByRole('status');
    await expect(banner).toHaveTextContent(/^CHANGED OUTSIDE GO-POB/);
    await waitFor(() => expect(editor.querySelector('p')).not.toBeNull());
    await expect(editor.querySelector('h1')).toBeNull();
    await expect(editor.querySelector('ul')).toBeNull();
    await expectPageFits(main);
    await userEvent.click(within(banner).getByRole('button', { name: 'Restore rich version' }));
    await expect(within(main).queryByRole('status')).toBeNull();
    await waitFor(() => expect(editor.querySelector('ul')).not.toBeNull());
    await expect(editor.querySelectorAll('ul > li').length).toBeGreaterThan(3);
    await expect(editor.querySelector('li [data-pob-color]')).not.toBeNull();
  }}>
  {@render frame(fireballBuild, lists.text, 'mismatch')}
</Story>

<Story
  name="Keep PoB text"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { editor, main } = page(canvasElement);
    await userEvent.click(within(within(main).getByRole('status')).getByRole('button', { name: 'Keep PoB text' }));
    await expect(within(main).queryByRole('status')).toBeNull();
    await expect(editor.querySelector('h1')).toBeNull();
  }}>
  {@render frame(spectreBuild, headings.text, 'mismatch')}
</Story>

<Story
  name="Inline formats"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { editor, bar } = await ready(canvasElement);
    await newLine(editor);
    for (const [name, keys, shown, tag] of INLINE) {
      const btn = bar.getByRole('button', { name });
      await expect(btn.title, `${name}: shortcut in its tooltip`).toContain(shown);
      await userEvent.click(btn);
      await expect(btn).toHaveAttribute('aria-pressed', 'true');
      await waitFor(() => expect(editor).toHaveFocus());
      await userEvent.keyboard(`${name} tool`);
      await expect(last(editor, tag)).toHaveTextContent(`${name} tool`);
      await userEvent.click(btn);
      await expect(btn).toHaveAttribute('aria-pressed', 'false');
      await waitFor(() => expect(editor).toHaveFocus());
      await userEvent.keyboard('{Enter}');
      await userEvent.keyboard(keys);
      await expect(btn).toHaveAttribute('aria-pressed', 'true');
      await userEvent.keyboard(`${name} key`);
      await expect(last(editor, tag)).toHaveTextContent(`${name} key`);
      await userEvent.keyboard(keys);
      await expect(btn).toHaveAttribute('aria-pressed', 'false');
      await userEvent.keyboard('{Enter}');
    }
    await userEvent.click(bar.getByRole('button', { name: 'Text colour' }));
    const custom = within(canvasElement).getByLabelText<HTMLInputElement>('Custom colour');
    custom.value = '#12ab34';
    custom.dispatchEvent(new Event('change', { bubbles: true }));
    await waitFor(() => expect(editor).toHaveFocus());
    await userEvent.keyboard('custom hue');
    await expect(editor.querySelector('[data-pob-color="^x12AB34"]')).toHaveTextContent('custom hue');
  }}>
  {@render frame(novaBuild, plain.text, 'match')}
</Story>

<Story
  name="Blocks and history"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { editor, bar } = await ready(canvasElement);
    await expect(bar.getByRole('button', { name: 'Undo' })).toBeDisabled();
    await expect(bar.getByRole('button', { name: 'Redo' })).toBeDisabled();
    await newLine(editor);
    for (const [name, keys, shown, sel] of BLOCKS) {
      const btn = bar.getByRole('button', { name });
      await expect(btn.title, `${name}: shortcut in its tooltip`).toContain(shown);
      await userEvent.click(btn);
      await expect(btn).toHaveAttribute('aria-pressed', 'true');
      await waitFor(() => expect(editor).toHaveFocus());
      await userEvent.keyboard(`${name} tool`);
      await expect(last(editor, sel)).toHaveTextContent(`${name} tool`);
      await userEvent.click(btn);
      await expect(btn).toHaveAttribute('aria-pressed', 'false');
      await waitFor(() => expect(editor).toHaveFocus());
      await userEvent.keyboard('{Enter}');
      await userEvent.keyboard(keys);
      await expect(btn).toHaveAttribute('aria-pressed', 'true');
      await userEvent.keyboard(`${name} key`);
      await expect(last(editor, sel)).toHaveTextContent(`${name} key`);
      await userEvent.keyboard(keys);
      await expect(btn).toHaveAttribute('aria-pressed', 'false');
      await userEvent.keyboard('{Enter}');
    }
    const rules = () => editor.querySelectorAll('hr').length;
    await userEvent.click(bar.getByRole('button', { name: 'Horizontal rule' }));
    await expect(rules()).toBe(1);
    await expect(bar.getByRole('button', { name: 'Undo' })).toBeEnabled();
    await expect(bar.getByRole('button', { name: 'Undo' }).title).toContain('Ctrl+Z');
    await expect(bar.getByRole('button', { name: 'Redo' }).title).toContain('Ctrl+Shift+Z');
    await userEvent.click(bar.getByRole('button', { name: 'Undo' }));
    await expect(rules()).toBe(0);
    await userEvent.click(bar.getByRole('button', { name: 'Redo' }));
    await expect(rules()).toBe(1);
    await waitFor(() => expect(editor).toHaveFocus());
    await userEvent.keyboard('{Control>}z{/Control}');
    await expect(rules()).toBe(0);
    await userEvent.keyboard('{Control>}{Shift>}z{/Shift}{/Control}');
    await expect(rules()).toBe(1);
    await userEvent.keyboard('{Control>}{Alt>}2{/Alt}{/Control}');
    await userEvent.keyboard('{Control>}b{/Control}loud{Control>}b{/Control} words');
    await expect(last(editor, 'h2 strong')).toHaveTextContent('loud');
    const h2s = editor.querySelectorAll('h2').length;
    await userEvent.click(bar.getByRole('button', { name: 'Clear formatting' }));
    await expect(editor.querySelectorAll('h2').length).toBe(h2s - 1);
    await expect(last(editor, 'p')).toHaveTextContent('loud words');
    await expect(last(editor, 'p').querySelector('strong')).toBeNull();
    await userEvent.keyboard(`{Enter}${ctrl('9', 'Shift')}tick me`);
    const task = last(editor, 'ul[data-type="taskList"] li');
    await expect(task).toHaveTextContent('tick me');
    await userEvent.click(task.querySelector('input[type="checkbox"]')!);
    await expect(task.querySelector('input[type="checkbox"]')).toBeChecked();
    await expect(task).toHaveAttribute('data-checked', 'true');
  }}>
  {@render frame(fireballBuild, plain.text, 'match')}
</Story>

<Story
  name="Links"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, editor, bar } = await ready(canvasElement);
    await newLine(editor);
    const link = bar.getByRole('button', { name: 'Link' });
    await expect(link.title).toContain('Ctrl+K');
    await userEvent.keyboard('{Control>}k{/Control}');
    const form = () => c.getByRole('dialog', { name: 'Link' });
    await expectFloating(link, form());
    const url = () => within(form()).getByRole('textbox', { name: 'Address' });
    await waitFor(() => expect(url()).toHaveFocus());
    await userEvent.keyboard('https://pobb.in/abc{Enter}');
    await expect(c.queryByRole('dialog', { name: 'Link' })).toBeNull();
    await waitFor(() => expect(editor.querySelectorAll('a')).toHaveLength(1));
    const a = () => editor.querySelector('a')!;
    await expect(a()).toHaveAttribute('href', 'https://pobb.in/abc');
    await expect(a()).toHaveTextContent('https://pobb.in/abc');
    await userEvent.click(a());
    await expect(link).toHaveAttribute('aria-pressed', 'true');
    await userEvent.click(link);
    await expect(url()).toHaveValue('https://pobb.in/abc');
    await userEvent.clear(url());
    await userEvent.type(url(), 'https://pobb.in/xyz{Enter}');
    await waitFor(() => expect(a()).toHaveAttribute('href', 'https://pobb.in/xyz'));
    await expect(editor.querySelectorAll('a')).toHaveLength(1);
    await userEvent.click(a());
    await userEvent.keyboard('{Control>}k{/Control}');
    await userEvent.click(within(form()).getByRole('button', { name: 'Remove link' }));
    await expect(editor.querySelector('a')).toBeNull();
    await expect(editor).toHaveTextContent('https://pobb.in/abc');
    await userEvent.click(link);
    await userEvent.keyboard('{Escape}');
    await expect(c.queryByRole('dialog', { name: 'Link' })).toBeNull();
    await expect(editor.querySelector('a')).toBeNull();
  }}>
  {@render frame(novaBuild, plain.text, 'match')}
</Story>

<Story
  name="Word count"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { editor } = await ready(canvasElement);
    const count = canvasElement.querySelector<HTMLElement>('[data-word-count]')!;
    await expect(count, 'the word count footer').not.toBeNull();
    const read = () =>
      count
        .textContent!.match(/^(\d+) words? · (\d+) characters?$/)!
        .slice(1)
        .map(Number);
    const text = editor.textContent!;
    await expect(read()).toEqual([editor.innerText.split(/\s+/).filter(Boolean).length, text.length]);
    const [w, ch] = read();
    await newLine(editor);
    await userEvent.keyboard('two more');
    await waitFor(() => expect(read()).toEqual([w + 2, ch + 8]));
  }}>
  {@render frame(novaBuild, plain.text, 'match')}
</Story>
