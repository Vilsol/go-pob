<script lang="ts">
  import { pressedOutside } from './floating';
  import { onDestroy, onMount, tick, untrack } from 'svelte';
  import { Editor } from '@tiptap/core';
  import { colorCodes } from '$lib/display/colors';
  import { notesExtensions } from '$lib/notes/extensions';
  import { COLOR_MARK, DEFAULT_CODE, codeToCss, docToPlain, type DocNode } from '$lib/notes/pob_text';
  import Checkbox from './Checkbox.svelte';

  type Banner = 'mismatch' | 'unreadable' | null;
  type Props = {
    doc: DocNode;
    banner?: Banner;
    converted?: number;
    onchange?: (doc: DocNode) => void;
    onblur?: () => void;
    onrestore?: () => void;
    ondismiss?: () => void;
    ondismissimport?: () => void;
    oncopy?: (text: string) => void;
  };
  let { doc, banner = null, converted = 0, onchange, onblur, onrestore, ondismiss, ondismissimport, oncopy }: Props = $props();

  // NotesTab.lua:21-32
  const SWATCHES = (['NORMAL', 'MAGIC', 'RARE', 'UNIQUE', 'FIRE', 'COLD', 'LIGHTNING', 'CHAOS', 'STRENGTH', 'DEXTERITY', 'INTELLIGENCE'] as const).map(
    (name) => ({ name, code: '^x' + colorCodes[name].slice(1).toUpperCase() })
  );

  let element = $state<HTMLDivElement>();
  let editor = $state<Editor>();
  let version = $state(0);
  let paletteOpen = $state(false);
  let infoShown = $state(false);
  let pobView = $state(false);
  let importDismissed = $state(false);
  let paletteRoot = $state<HTMLElement>();
  let linkOpen = $state(false);
  let linkHref = $state('');
  let linkInput = $state<HTMLInputElement>();

  let shownDoc: DocNode | undefined;
  $effect(() => {
    const d = doc;
    const ed = editor;
    if (!ed || d === shownDoc) return;
    untrack(() => {
      const first = shownDoc === undefined;
      shownDoc = d;
      if (!first && JSON.stringify(ed.getJSON()) !== JSON.stringify(d)) ed.commands.setContent(d, { emitUpdate: false });
    });
  });

  const active = (name: string, attrs?: Record<string, unknown>) => {
    void version;
    return editor?.isActive(name, attrs) ?? false;
  };
  const currentCode = $derived.by(() => {
    void version;
    return (editor?.getAttributes(COLOR_MARK).code as string | undefined) ?? DEFAULT_CODE;
  });
  const pobText = $derived.by(() => {
    void version;
    return editor ? docToPlain(editor.getJSON() as DocNode) : '';
  });
  const history = $derived.by(() => {
    void version;
    return { undo: editor?.can().undo() ?? false, redo: editor?.can().redo() ?? false };
  });
  const countText = $derived.by(() => {
    void version;
    if (!editor) return '';
    const words = editor.getText({ blockSeparator: '\n' }).split(/\s+/).filter(Boolean).length;
    const chars = editor.state.doc.textContent.length;
    return `${words} word${words === 1 ? '' : 's'} · ${chars} character${chars === 1 ? '' : 's'}`;
  });

  const setColor = (code: string) => {
    paletteOpen = false;
    const chain = editor?.chain().focus();
    if (!chain) return;
    if (code === DEFAULT_CODE) chain.unsetMark(COLOR_MARK).run();
    else chain.setMark(COLOR_MARK, { code, lead: null, run: 0 }).run();
  };

  const customHex = $derived(currentCode.startsWith('^x') ? '#' + currentCode.slice(2).toLowerCase() : '#ffffff');

  type Tool = { name: string; label: string; keys?: string; is?: () => boolean; run: () => void; cls?: string };
  const cmd = () => editor!.chain().focus();
  const HISTORY: (Tool & { can: () => boolean })[] = [
    { name: 'Undo', label: '↶', keys: 'Ctrl+Z', can: () => history.undo, run: () => cmd().undo().run() },
    { name: 'Redo', label: '↷', keys: 'Ctrl+Shift+Z', can: () => history.redo, run: () => cmd().redo().run() }
  ];
  const HEADINGS: Tool[] = ([1, 2, 3] as const).map((level) => ({
    name: `Heading ${level}`,
    label: `H${level}`,
    keys: `Ctrl+Alt+${level}`,
    is: () => active('heading', { level }),
    run: () => cmd().toggleHeading({ level }).run()
  }));
  const INLINE: Tool[] = [
    { name: 'Bold', label: 'B', keys: 'Ctrl+B', is: () => active('bold'), run: () => cmd().toggleBold().run(), cls: 'font-bold' },
    { name: 'Italic', label: 'I', keys: 'Ctrl+I', is: () => active('italic'), run: () => cmd().toggleItalic().run(), cls: 'italic' },
    { name: 'Underline', label: 'U', keys: 'Ctrl+U', is: () => active('underline'), run: () => cmd().toggleUnderline().run(), cls: 'underline' },
    { name: 'Strikethrough', label: 'S', keys: 'Ctrl+Shift+S', is: () => active('strike'), run: () => cmd().toggleStrike().run(), cls: 'line-through' },
    { name: 'Inline code', label: '</>', keys: 'Ctrl+E', is: () => active('code'), run: () => cmd().toggleCode().run(), cls: 'font-mono text-12' }
  ];
  const BLOCKS: Tool[] = [
    { name: 'Bulleted list', label: '•', keys: 'Ctrl+Shift+8', is: () => active('bulletList'), run: () => cmd().toggleBulletList().run(), cls: 'text-16' },
    { name: 'Numbered list', label: '1.', keys: 'Ctrl+Shift+7', is: () => active('orderedList'), run: () => cmd().toggleOrderedList().run() },
    { name: 'Task list', label: '☑', keys: 'Ctrl+Shift+9', is: () => active('taskList'), run: () => cmd().toggleTaskList().run() },
    { name: 'Quote', label: '❝', keys: 'Ctrl+Shift+B', is: () => active('blockquote'), run: () => cmd().toggleBlockquote().run(), cls: 'text-15' },
    { name: 'Code block', label: '{ }', keys: 'Ctrl+Alt+C', is: () => active('codeBlock'), run: () => cmd().toggleCodeBlock().run(), cls: 'font-mono text-12' },
    { name: 'Horizontal rule', label: '―', run: () => cmd().setHorizontalRule().run() }
  ];
  const tip = (t: Tool) => (t.keys ? `${t.name} (${t.keys})` : t.name);

  const clearFormatting = () => {
    if (!editor) return;
    const sel = editor.state.selection;
    const chain = cmd();
    if (sel.empty) chain.setTextSelection({ from: sel.$from.start(), to: sel.$from.end() });
    chain.unsetAllMarks().clearNodes().run();
  };

  async function openLink() {
    if (!editor || pobView) return;
    paletteOpen = false;
    linkHref = (editor.getAttributes('link').href as string | undefined) ?? '';
    linkOpen = true;
    await tick();
    linkInput?.focus();
    linkInput?.select();
  }
  const closeLink = () => {
    linkOpen = false;
    editor?.commands.focus();
  };
  const applyLink = () => {
    if (!editor) return;
    const href = linkHref.trim();
    const bare = editor.state.selection.empty && !editor.isActive('link');
    linkOpen = false;
    if (!href) cmd().extendMarkRange('link').unsetLink().run();
    else if (bare)
      cmd()
        .insertContent({ type: 'text', text: href, marks: [{ type: 'link', attrs: { href } }] })
        .run();
    else cmd().extendMarkRange('link').setLink({ href }).run();
  };
  const removeLink = () => {
    linkOpen = false;
    cmd().extendMarkRange('link').unsetLink().run();
  };
  let linkRoot = $state<HTMLElement>();

  onMount(() => {
    editor = new Editor({
      element,
      extensions: notesExtensions(),
      content: doc,
      editorProps: {
        attributes: { role: 'textbox', 'aria-multiline': 'true', 'aria-label': 'Notes', class: 'notes-doc' },
        handleKeyDown: (_view, e) => {
          if (!(e.ctrlKey || e.metaKey) || e.altKey || e.shiftKey || e.key.toLowerCase() !== 'k') return false;
          void openLink();
          return true;
        },
        handleDOMEvents: {
          click: (_view, e) => {
            if ((e.target as HTMLElement | null)?.closest('a')) e.preventDefault();
            return false;
          }
        }
      },
      onTransaction: () => untrack(() => version++),
      onUpdate: () => onchange?.(editor!.getJSON() as DocNode),
      onBlur: () => onblur?.()
    });
  });
  onDestroy(() => editor?.destroy());

  const outside = (e: PointerEvent) => {
    if (paletteOpen && paletteRoot && pressedOutside(e, paletteRoot)) paletteOpen = false;
    if (linkOpen && linkRoot && pressedOutside(e, linkRoot)) linkOpen = false;
  };
</script>

<svelte:window onpointerdown={outside} />

{#if banner === 'mismatch'}
  <div role="status" class="flex items-center gap-2.5 rounded border border-warn-line bg-warn-bg-2 px-3 py-2 text-13">
    <span class="text-11 font-semibold tracking-caps whitespace-nowrap text-worse">CHANGED OUTSIDE GO-POB</span>
    <span>These notes were edited as PoB text, so that text is shown. The earlier rich version (formatting and colours) is still stored.</span>
    <div class="grow"></div>
    <button class="btn btn-sm btn-primary whitespace-nowrap" onclick={onrestore}>Restore rich version</button>
    <button class="btn btn-sm whitespace-nowrap" onclick={ondismiss}>Keep PoB text</button>
  </div>
{:else if banner === 'unreadable'}
  <div role="status" class="rounded border border-warn-line bg-warn-bg-2 px-3 py-2 text-13">
    The stored rich notes could not be read; the PoB text is shown.
  </div>
{:else if converted && !importDismissed}
  <div
    role="status"
    class="flex items-center gap-2.5 rounded border border-info-line bg-info-bg px-3 py-2 text-13 max-phone:grid max-phone:grid-cols-[minmax(0,1fr)_auto] max-phone:gap-y-1">
    <span class="text-11 font-semibold tracking-caps whitespace-nowrap text-better">IMPORTED FROM POB</span>
    <span class="max-phone:col-span-2 max-phone:row-start-2"
      >Colour codes converted: {converted} ^xRRGGBB / ^0–^9 codes became text colours. Export writes them back as caret codes.</span>
    <div class="grow max-phone:hidden"></div>
    <button
      class="btn btn-sm btn-ghost max-phone:col-start-2 max-phone:row-start-1"
      onclick={() => {
        importDismissed = true;
        ondismissimport?.();
      }}>Dismiss</button>
  </div>
{/if}

{#snippet tool(t: Tool)}
  <button
    class={[
      'h-7.5 min-w-7.5 rounded border border-transparent bg-transparent px-1.5 font-sans text-13 text-text hover:bg-hover disabled:opacity-45 disabled:hover:bg-transparent aria-pressed:border-accent aria-pressed:bg-surface-3',
      t.cls
    ]}
    aria-label={t.name}
    aria-pressed={t.is?.()}
    title={tip(t)}
    disabled={pobView}
    onclick={t.run}>{t.label}</button>
{/snippet}
{#snippet sep()}<span class="mx-0.5 h-5.5 w-px shrink-0 bg-line" aria-hidden="true"></span>{/snippet}

<div role="toolbar" aria-label="Formatting" class="panel relative flex flex-wrap items-center gap-0.5 p-1.5 max-phone:pr-8">
  {#each HISTORY as t (t.name)}
    <button
      class="h-7.5 min-w-7.5 rounded border border-transparent bg-transparent px-1.5 font-sans text-15 text-text hover:bg-hover disabled:opacity-45 disabled:hover:bg-transparent"
      aria-label={t.name}
      title={tip(t)}
      disabled={pobView || !t.can()}
      onclick={t.run}>{t.label}</button>
  {/each}
  {@render sep()}
  {#each HEADINGS as t (t.name)}{@render tool(t)}{/each}
  {@render sep()}
  {#each INLINE as t (t.name)}{@render tool(t)}{/each}
  <div class="relative" bind:this={paletteRoot}>
    <button
      class="btn btn-sm gap-1.5"
      aria-label="Text colour"
      aria-expanded={paletteOpen}
      title="Text colour: PoB's colours or a custom one"
      disabled={pobView}
      onclick={() => (paletteOpen = !paletteOpen)}
      onkeydown={(e) => {
        if (e.key === 'Escape') paletteOpen = false;
      }}>A<span class="toned h-1 w-3.5 rounded-hair bg-(--tone,var(--color-text))" style:--tone={codeToCss(currentCode)} data-pob-color></span>▾</button>
    {#if paletteOpen}
      <div
        class="absolute top-9 left-0 z-overlay flex w-82.5 flex-col gap-1.5 rounded-md border border-line-strong bg-surface-2 p-2 shadow-pop"
        role="presentation"
        onkeydown={(e) => {
          if (e.key === 'Escape') {
            paletteOpen = false;
            paletteRoot?.querySelector('button')?.focus();
          }
        }}>
        <div role="listbox" aria-label="PoB colours" tabindex="-1" class="grid grid-cols-3 gap-1">
          {#each SWATCHES as s (s.name)}
            <button
              role="option"
              aria-selected={currentCode === s.code}
              data-pob-color
              class="toned flex h-6.5 items-center gap-1.5 rounded border border-line bg-surface-1 px-1.5 font-sans text-11 font-semibold aria-selected:border-accent"
              style:--tone={codeToCss(s.code)}
              onclick={() => setColor(s.code)}><span class="size-2.5 rounded-xs bg-(--tone)"></span>{s.name}</button>
          {/each}
          <button
            role="option"
            aria-selected={currentCode === DEFAULT_CODE}
            class="flex h-6.5 items-center gap-1.5 rounded border border-line bg-surface-1 px-1.5 font-sans text-11 font-semibold text-text aria-selected:border-accent"
            onclick={() => setColor(DEFAULT_CODE)}><span class="size-2.5 rounded-xs bg-text"></span>DEFAULT</button>
        </div>
        <label class="flex items-center gap-2 text-12 text-text-2"
          >Custom<input
            type="color"
            aria-label="Custom colour"
            class="h-6.5 w-9 cursor-pointer rounded border border-line-strong bg-ground p-0"
            value={customHex}
            onchange={(e) => setColor('^x' + e.currentTarget.value.slice(1).toUpperCase())} /></label>
        <span class="note-sm">Colours are PoB’s colour codes, so export maps each back to its ^x code.</span>
      </div>
    {/if}
  </div>
  {@render sep()}
  {#each BLOCKS as t (t.name)}{@render tool(t)}{/each}
  {@render sep()}
  <div class="relative" bind:this={linkRoot}>
    {@render tool({ name: 'Link', label: 'Link', keys: 'Ctrl+K', is: () => active('link'), run: () => void openLink(), cls: 'text-12 underline' })}
    {#if linkOpen}
      <div
        role="dialog"
        aria-label="Link"
        tabindex="-1"
        class="absolute top-9 left-0 z-overlay w-80 rounded-md border border-line-strong bg-surface-2 p-2 shadow-pop"
        onkeydown={(e) => {
          if (e.key !== 'Escape') return;
          e.preventDefault();
          closeLink();
        }}>
        <form
          class="flex items-center gap-1.5"
          onsubmit={(e) => {
            e.preventDefault();
            applyLink();
          }}>
          <input bind:this={linkInput} type="url" class="input input-xs min-w-0 grow" aria-label="Address" placeholder="https://…" bind:value={linkHref} />
          <button class="btn btn-sm btn-primary" type="submit">Apply</button>
          {#if active('link')}<button class="btn btn-sm" type="button" onclick={removeLink}>Remove link</button>{/if}
        </form>
      </div>
    {/if}
  </div>
  <button
    class="h-7.5 min-w-7.5 rounded border border-transparent bg-transparent px-1.5 font-sans text-12 text-text hover:bg-hover disabled:opacity-45"
    aria-label="Clear formatting"
    title="Clear formatting (the selection, or the whole block)"
    disabled={pobView}
    onclick={clearFormatting}>T<sub>×</sub></button>
  <span class="relative ml-1 inline-flex max-phone:absolute max-phone:top-2.5 max-phone:right-1.5">
    <button
      class="info-btn size-5.5 text-12"
      aria-label="How notes export to PoB"
      aria-describedby="notes-info"
      onmouseenter={() => (infoShown = true)}
      onmouseleave={() => (infoShown = false)}
      onfocus={() => (infoShown = true)}
      onclick={() => (infoShown = true)}
      onblur={() => (infoShown = false)}>i</button>
    <span
      role="tooltip"
      id="notes-info"
      class={[
        'tooltip tooltip-info -top-1.5 right-7 left-auto w-85.5 max-phone:fixed max-phone:inset-x-3 max-phone:top-auto max-phone:bottom-18 max-phone:z-tooltip max-phone:w-auto',
        !infoShown && 'hidden'
      ]}
      >Rich text (formatting, colours, lists, links) is stored in the build. PoB export flattens it to plain text with ^xRRGGBB colour codes: formatting and
      links export as their text, list items as “• ” / “1. ” / “☐ ” lines, a rule as a line of dashes.</span>
  </span>
</div>

<section aria-label="Notes editor" class={['panel scroll min-h-0 grow', pobView && 'hidden']}>
  <div bind:this={element} class="h-full"></div>
</section>
{#if pobView}
  <section aria-label="PoB text" class="scroll min-h-0 grow rounded-md border border-line bg-ground px-5 py-4">
    <pre class="m-0 font-mono text-13 leading-[1.7] whitespace-pre-wrap text-text-2" data-pob-text>{pobText}</pre>
  </section>
{/if}
<div data-notes-footer class="flex shrink-0 items-center gap-3 px-1 text-12 text-text-2">
  <span data-word-count aria-live="polite">{countText}</span>
  <div class="grow"></div>
  <Checkbox compact class="gap-2 text-13" bind:checked={pobView}>View as PoB text</Checkbox>
  <button class="btn btn-sm" onclick={() => oncopy?.(pobText)}>Copy as PoB text</button>
</div>
