<script lang="ts">
  import NotesPage from '../NotesPage.svelte';
  import { loadNotes } from '$lib/notes/load';
  import { countCodes, plainToDoc, type DocNode } from '$lib/notes/pob_text';

  type Props = { text: string; mode?: 'imported' | 'mismatch' | 'match' };
  let { text, mode = 'imported' }: Props = $props();

  const richVersion = (d: DocNode): DocNode => {
    const out: DocNode[] = [];
    for (const [i, b] of (d.content ?? []).entries()) {
      const first = b.content?.[0];
      if (i === 0 && b.content?.length) out.push({ ...b, type: 'heading', attrs: { level: 1 } });
      else if (first?.text?.startsWith('- ')) {
        const rest = [{ ...first, text: first.text.slice(2) }, ...(b.content ?? []).slice(1)].filter((n) => n.text);
        const item = { type: 'listItem', content: [{ ...b, content: rest }] };
        const prev = out.at(-1);
        if (prev?.type === 'bulletList') prev.content!.push(item);
        else out.push({ type: 'bulletList', content: [item] });
      } else out.push(b);
    }
    return { ...d, content: out };
  };

  const stored = JSON.stringify(richVersion(plainToDoc(text)));
  const loaded = loadNotes(mode === 'imported' ? 'none' : mode, text, mode === 'imported' ? '' : stored);

  let s = $state({ doc: loaded.doc, banner: loaded.banner, saved: '', copied: '' });
</script>

<NotesPage
  doc={s.doc}
  banner={s.banner}
  converted={mode === 'imported' ? countCodes(text) : 0}
  onchange={(d: DocNode) => (s.saved = JSON.stringify(d).slice(0, 80))}
  onrestore={() => {
    if (loaded.rich) s.doc = loaded.rich;
    s.banner = null;
  }}
  ondismiss={() => (s.banner = null)}
  oncopy={(t: string) => (s.copied = t)} />
<span class="sr-only" data-copied>{s.copied}</span>
