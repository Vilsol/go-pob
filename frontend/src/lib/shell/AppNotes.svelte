<script module lang="ts">
  let importDismissedFor: number | undefined;
</script>

<script lang="ts">
  import { onDestroy } from 'svelte';
  import { resource } from 'runed';
  import { buildRevision, currentBuild } from '$lib/global.js';
  import { logError } from '$lib/utils';
  import { loadNotes } from '$lib/notes/load';
  import { countCodes, docToPlain, type DocNode } from '$lib/notes/pob_text';
  import NotesPage from '$lib/design/NotesPage.svelte';

  type Build = NonNullable<typeof $currentBuild>;
  const write = (build: Build, doc: DocNode) => {
    build.SetRichNotes(docToPlain(doc), JSON.stringify(doc)).catch(logError);
  };

  let pending: { build: Build; doc: DocNode } | undefined;
  let timer: ReturnType<typeof setTimeout> | undefined;
  const flush = () => {
    clearTimeout(timer);
    timer = undefined;
    const p = pending;
    pending = undefined;
    if (p) write(p.build, p.doc);
  };

  const notes = resource(
    () => [$buildRevision, $currentBuild === undefined] as const,
    async () => {
      flush();
      const build = $currentBuild;
      if (!build) return undefined;
      const [status, plain, doc] = await Promise.all([build.RichNotesStatus(), build.Notes, build.RichNotesDoc()]);
      const converted = status === 'none' ? countCodes(plain ?? '') : 0;
      return { build, key: $buildRevision, converted, ...loadNotes(status, plain ?? '', doc) };
    }
  );

  let restored = $state<{ key: number; doc: DocNode } | undefined>();
  let dismissed = $state<number | undefined>();
  let importDismissed = $state(importDismissedFor);

  const loaded = $derived(notes.current);
  const shown = $derived(loaded && restored?.key === loaded.key ? restored.doc : loaded?.doc);
  const banner = $derived(loaded && dismissed !== loaded.key && restored?.key !== loaded.key ? loaded.banner : null);
  const converted = $derived(loaded && importDismissed !== loaded.key ? loaded.converted : 0);

  const onchange = (doc: DocNode) => {
    if (!loaded) return;
    pending = { build: loaded.build, doc };
    clearTimeout(timer);
    timer = setTimeout(flush, 400);
  };
  onDestroy(flush);

  const onrestore = () => {
    if (!loaded?.rich) return;
    restored = { key: loaded.key, doc: loaded.rich };
    pending = undefined;
    write(loaded.build, loaded.rich);
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
