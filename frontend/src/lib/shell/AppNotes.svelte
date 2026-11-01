<script module lang="ts">
  let importDismissedFor: string | undefined;
</script>

<script lang="ts">
  import { onDestroy } from 'svelte';
  import { resource } from 'runed';
  import { currentEngine } from '$lib/go/engine';
  import { logError } from '$lib/utils';
  import { loadNotes } from '$lib/notes/load';
  import { countCodes, docToPlain, type DocNode } from '$lib/notes/pob_text';
  import NotesPage from '$lib/design/NotesPage.svelte';

  // NotesTab.lua: the open build's notes, whichever game it is
  const engine = currentEngine().notes;
  const revision = engine.revision;

  type Write = (plain: string, doc: string) => unknown;
  const write = (to: Write, doc: DocNode) => {
    void Promise.resolve(to(docToPlain(doc), JSON.stringify(doc))).catch(logError);
  };

  let pending: { to: Write; doc: DocNode } | undefined;
  let timer: ReturnType<typeof setTimeout> | undefined;
  const flush = () => {
    clearTimeout(timer);
    timer = undefined;
    const p = pending;
    pending = undefined;
    if (p) write(p.to, p.doc);
  };

  const notes = resource(
    () => $revision,
    async (key) => {
      flush();
      const read = await engine.read();
      if (!read) return undefined;
      const converted = read.status === 'none' ? countCodes(read.plain) : 0;
      return { to: read.write, key, converted, ...loadNotes(read.status, read.plain, read.doc) };
    }
  );

  let restored = $state<{ key: string; doc: DocNode } | undefined>();
  let dismissed = $state<string | undefined>();
  let importDismissed = $state(importDismissedFor);

  const loaded = $derived(notes.current);
  const shown = $derived(loaded && restored?.key === loaded.key ? restored.doc : loaded?.doc);
  const banner = $derived(loaded && dismissed !== loaded.key && restored?.key !== loaded.key ? loaded.banner : null);
  const converted = $derived(loaded && importDismissed !== loaded.key ? loaded.converted : 0);

  const onchange = (doc: DocNode) => {
    if (!loaded) return;
    pending = { to: loaded.to, doc };
    clearTimeout(timer);
    timer = setTimeout(flush, 400);
  };
  onDestroy(flush);

  const onrestore = () => {
    if (!loaded?.rich) return;
    restored = { key: loaded.key, doc: loaded.rich };
    pending = undefined;
    write(loaded.to, loaded.rich);
  };

  const copy = (text: string) => {
    void navigator.clipboard?.writeText(text).catch(logError);
  };
</script>

{#if loaded && shown}
  {#key loaded.key}
    <NotesPage
      doc={shown}
      {banner}
      {converted}
      {onchange}
      onblur={flush}
      {onrestore}
      ondismiss={() => (dismissed = loaded?.key)}
      ondismissimport={() => (importDismissed = importDismissedFor = loaded?.key)}
      oncopy={copy} />
  {/key}
{/if}
