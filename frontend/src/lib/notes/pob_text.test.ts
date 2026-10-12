// @vitest-environment jsdom
import { afterAll, describe, expect, it } from 'vitest';
import fc from 'fast-check';
import { readdirSync, readFileSync, existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { Editor } from '@tiptap/core';

import { COLOR_MARK, docToPlain, plainToDoc, type DocNode } from './pob_text';
import { notesExtensions } from './extensions';

const editor = new Editor({ extensions: notesExtensions() });
afterAll(() => editor.destroy());

const throughEditor = (doc: DocNode): DocNode => {
  editor.commands.setContent(doc, { emitUpdate: false });
  return editor.getJSON() as DocNode;
};

const roundTrip = (plain: string) => docToPlain(plainToDoc(plain));
const roundTripEditor = (plain: string) => docToPlain(throughEditor(plainToDoc(plain)));

const piece = fc.oneof(
  fc.constantFrom('^', '^x', '^7', '^1', '^0', '^9', '^xFF0000', '^xaf6025', '^xAF6025', '^x12345', '^xGGGGGG', '^^1', '\n', '\t', ' ', '  '),
  fc.string({ maxLength: 6 }),
  fc.stringMatching(/^[0-9A-Fa-f]{1,6}$/).map((h) => '^x' + h),
  fc.integer({ min: 0, max: 9 }).map((d) => '^' + d),
  fc.constantFrom('é', '😀', '<b>', '&amp;', 'ünïcode')
);
const caretText = fc.array(piece, { maxLength: 40 }).map((a) => a.join(''));

describe('plain -> rich -> plain is the identity', () => {
  it('for fixed cases PoB builds carry', () => {
    const cases = [
      '',
      'plain',
      '\n',
      'a\n\nb\n',
      '^1red\nstill red',
      '^xD02090^xD02090Upgrades from h^xD02090ere^7^xD02090:^7^7^7',
      '^1a^1b',
      '^7',
      'text^',
      '^x12345 not a code',
      '^xFFFF77\n_____^7^7^7\n^xADAA47\n   ^xAF6025My^7 ^xE05030Youtube^7'
    ];
    for (const c of cases) {
      expect(roundTrip(c)).toBe(c);
      expect(roundTripEditor(c)).toBe(c);
    }
  });

  it('for any caret-coded text', () => {
    fc.assert(
      fc.property(caretText, (s) => roundTrip(s) === s),
      { numRuns: 2000 }
    );
  });

  it('for any caret-coded text held by the editor', () => {
    fc.assert(
      fc.property(caretText, (s) => roundTripEditor(s) === s),
      { numRuns: 500 }
    );
  });

  it('for every corpus build with notes', () => {
    const dir = resolve(import.meta.dirname, '../../../../testdata/corpus-3.29');
    if (!existsSync(dir)) return;
    const decode = (s: string) =>
      s.replace(/&(lt|gt|amp|apos|quot|#x?[0-9A-Fa-f]+);/g, (_, e: string) =>
        e === 'lt'
          ? '<'
          : e === 'gt'
            ? '>'
            : e === 'amp'
              ? '&'
              : e === 'apos'
                ? "'"
                : e === 'quot'
                  ? '"'
                  : String.fromCodePoint(e[1] === 'x' ? parseInt(e.slice(2), 16) : Number(e.slice(1)))
      );
    let n = 0;
    for (const f of readdirSync(dir)) {
      const m = /<Notes>([\s\S]*?)<\/Notes>/.exec(readFileSync(resolve(dir, f), 'utf8'));
      const notes = m ? decode(m[1]).replace(/^[ \t\n\v\f\r]+|[ \t\n\v\f\r]+$/g, '') : '';
      if (!notes) continue;
      n++;
      expect(roundTripEditor(notes), f).toBe(notes);
    }
    expect(n, 'no corpus notes were read').toBeGreaterThan(100);
  });
});

describe('plain -> rich shows PoB colours', () => {
  const colours = (plain: string) =>
    (plainToDoc(plain).content ?? []).map((p) => (p.content ?? []).map((t) => [t.text, t.marks?.find((m) => m.type === COLOR_MARK)?.attrs?.code ?? null]));

  it('carries a colour across line ends until the next code', () => {
    expect(colours('^1red\nstill red^7 plain\nplain')).toEqual([
      [['red', '^1']],
      [
        ['still red', '^1'],
        [' plain', '^7']
      ],
      [['plain', '^7']]
    ]);
  });

  it('reads both code forms and leaves near-misses as text', () => {
    expect(colours('^xAF6025a^3b^x12c')).toEqual([
      [
        ['a', '^xAF6025'],
        ['b^x12c', '^3']
      ]
    ]);
  });
});

const safeText = fc.stringMatching(/^[A-Za-z0-9 .,:;!?'"()\-+%/]{1,12}$/);
const mark = fc.constantFrom<unknown>(
  null,
  { type: 'bold' },
  { type: 'italic' },
  { type: COLOR_MARK, attrs: { code: '^xFF0000', lead: null, run: 0 } },
  { type: COLOR_MARK, attrs: { code: '^3', lead: '^3', run: 0 } },
  { type: COLOR_MARK, attrs: { code: '<script>', lead: '<b>', run: 0 } }
);
const inline = fc.array(
  fc.oneof(
    fc.record({ text: safeText, marks: fc.array(mark, { maxLength: 2 }) }).map(({ text, marks }) => ({ type: 'text', text, marks: marks.filter(Boolean) })),
    fc.constant({ type: 'hardBreak' })
  ),
  { maxLength: 5 }
);
const para = inline.map((content) => ({ type: 'paragraph', content }));
const heading = fc.tuple(fc.constantFrom(1, 2), inline).map(([level, content]) => ({ type: 'heading', attrs: { level }, content }));
const list = fc.letrec<{ list: unknown }>((tie) => ({
  list: fc.record({
    type: fc.constantFrom('bulletList', 'orderedList'),
    content: fc.array(
      fc.array(fc.oneof({ depthSize: 'small' }, para, tie('list')), { minLength: 1, maxLength: 3 }).map((content) => ({ type: 'listItem', content })),
      { minLength: 1, maxLength: 3 }
    )
  })
})).list;
const richDoc = fc.array(fc.oneof(para, heading, list), { maxLength: 6 }).map((content) => ({ type: 'doc', content }) as DocNode);

describe('rich -> plain', () => {
  it('never emits markup, and every caret code it emits is a valid PoB code', () => {
    fc.assert(
      fc.property(richDoc, (doc) => {
        const plain = docToPlain(doc);
        if (/[<>*_#`[\]]/.test(plain)) return false;
        const carets = plain.match(/\^/g)?.length ?? 0;
        const codes = plain.match(/\^x[0-9A-Fa-f]{6}|\^[0-9]/g)?.length ?? 0;
        return carets === codes;
      }),
      { numRuns: 1000 }
    );
  });

  it('flattens structure to lines by the documented rules', () => {
    const doc: DocNode = {
      type: 'doc',
      content: [
        { type: 'heading', attrs: { level: 1 }, content: [{ type: 'text', text: 'Title', marks: [{ type: 'bold' }] }] },
        {
          type: 'bulletList',
          content: [
            { type: 'listItem', content: [{ type: 'paragraph', content: [{ type: 'text', text: 'one' }] }] },
            {
              type: 'listItem',
              content: [
                { type: 'paragraph', content: [{ type: 'text', text: 'two' }] },
                { type: 'orderedList', content: [{ type: 'listItem', content: [{ type: 'paragraph', content: [{ type: 'text', text: 'nested' }] }] }] }
              ]
            }
          ]
        },
        {
          type: 'paragraph',
          content: [
            { type: 'text', text: 'red', marks: [{ type: COLOR_MARK, attrs: { code: '^xFF0000', lead: null, run: 0 } }] },
            { type: 'hardBreak' },
            { type: 'text', text: 'plain' }
          ]
        }
      ]
    };
    expect(docToPlain(doc)).toBe('Title\n• one\n• two\n  1. nested\n^xFF0000red\n^7plain');
  });

  it('flattens the editor-only blocks and marks to text', () => {
    const p = (text: string, marks?: DocNode['marks']): DocNode => ({ type: 'paragraph', content: [{ type: 'text', text, marks }] });
    const task = (checked: boolean, text: string): DocNode => ({ type: 'taskItem', attrs: { checked }, content: [p(text)] });
    const doc = throughEditor({
      type: 'doc',
      content: [
        { type: 'taskList', content: [task(false, 'todo'), task(true, 'done')] },
        { type: 'horizontalRule' },
        { type: 'blockquote', content: [p('quoted', [{ type: 'underline' }, { type: 'strike' }])] },
        { type: 'codeBlock', content: [{ type: 'text', text: 'a\nb' }] },
        p('site', [{ type: 'link', attrs: { href: 'https://pobb.in/x' } }]),
        p('mono', [{ type: 'code' }])
      ]
    });
    expect(docToPlain(doc)).toBe('☐ todo\n☑ done\n----------\nquoted\na\nb\nsite\nmono');
  });
});
