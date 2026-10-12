import { getSchema } from '@tiptap/core';

import { notesExtensions } from './extensions';
import { plainToDoc, type DocNode } from './pob_text';

let schema: ReturnType<typeof getSchema> | undefined;

export const parseRichDoc = (raw: string): DocNode | undefined => {
  schema ??= getSchema(notesExtensions());
  try {
    const json: unknown = JSON.parse(raw);
    schema.nodeFromJSON(json).check();
    return json as DocNode;
  } catch {
    return undefined;
  }
};

export type LoadedNotes = {
  doc: DocNode;
  banner: 'mismatch' | 'unreadable' | null;
  rich?: DocNode;
};

export const loadNotes = (status: string, plain: string, rawDoc: string): LoadedNotes => {
  if (status === 'none') return { doc: plainToDoc(plain), banner: null };
  const rich = parseRichDoc(rawDoc);
  if (!rich) return { doc: plainToDoc(plain), banner: 'unreadable' };
  if (status === 'match') return { doc: rich, banner: null };
  return { doc: plainToDoc(plain), banner: 'mismatch', rich };
};
