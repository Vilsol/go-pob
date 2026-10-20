// Lists raw form controls in .svelte markup that bypass the design primitives: elements carrying a
// btn class, inputs/textareas carrying the input class, <select> elements and checkbox inputs.
// `node harness/raw_controls.ts` prints file:line per hit; raw_controls.test.ts holds the ceiling.
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, relative, resolve } from 'node:path';
import { parse, type AST } from 'svelte/compiler';

export const KINDS = ['btn', 'input', 'select', 'checkbox'] as const;
export type Kind = (typeof KINDS)[number];
export type Hit = { file: string; line: number; kind: Kind; element: string; classes: string };

// the primitives that render these controls themselves
export const PRIMITIVES = new Set([
  'lib/design/Button.svelte',
  'lib/design/Input.svelte',
  'lib/design/Select.svelte',
  'lib/design/Checkbox.svelte',
  'lib/design/SearchField.svelte',
  'lib/design/NumberInput.svelte',
  'lib/design/Textarea.svelte'
]);

const walk = (dir: string): string[] =>
  readdirSync(dir).flatMap((name) => {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) return walk(p);
    return name.endsWith('.svelte') ? [p] : [];
  });

const STRING = /'([^'\\]*)'|"([^"\\]*)"|`([^`\\$]*)`/g;

const valueParts = (attr: AST.Attribute) => (attr.value === true ? [] : Array.isArray(attr.value) ? attr.value : [attr.value]);

// class names an attribute value can produce: static text, plus string literals inside expressions
const classTokens = (source: string, attr: AST.Attribute) => {
  const out: string[] = [];
  for (const p of valueParts(attr)) {
    if (p.type === 'Text') out.push(...p.data.split(/\s+/));
    else for (const m of source.slice(p.start, p.end).matchAll(STRING)) out.push(...(m[1] ?? m[2] ?? m[3]).split(/\s+/));
  }
  return out.filter(Boolean);
};

const attrText = (source: string, attr: AST.Attribute | undefined) =>
  attr
    ? valueParts(attr)
        .map((p) => (p.type === 'Text' ? p.data : source.slice(p.start, p.end)))
        .join('')
    : '';

const attributes = (el: AST.RegularElement) => el.attributes.filter((a): a is AST.Attribute => a.type === 'Attribute');

const classesOf = (source: string, el: AST.RegularElement) => {
  const cls = attributes(el)
    .filter((a) => a.name === 'class')
    .flatMap((a) => classTokens(source, a));
  for (const d of el.attributes) if (d.type === 'ClassDirective') cls.push(d.name);
  return cls;
};

export function classify(source: string, el: AST.RegularElement): Kind | undefined {
  const cls = classesOf(source, el);
  const type = attrText(
    source,
    attributes(el).find((a) => a.name === 'type')
  );
  if (el.name === 'select') return 'select';
  if (el.name === 'input' && /checkbox/.test(type)) return 'checkbox';
  if ((el.name === 'input' || el.name === 'textarea') && cls.some((t) => t === 'input' || t.startsWith('input-'))) return 'input';
  if (cls.some((t) => t === 'btn' || t.startsWith('btn-'))) return 'btn';
  return undefined;
}

export function scanSource(source: string, file = ''): Hit[] {
  const ast = parse(source, { modern: true, filename: file });
  const hits: Hit[] = [];
  const lineOf = (i: number) => source.slice(0, i).split('\n').length;
  const visit = (node: unknown): void => {
    if (!node || typeof node !== 'object') return;
    if (Array.isArray(node)) return node.forEach(visit);
    if ((node as { type?: string }).type === 'RegularElement') {
      const el = node as AST.RegularElement;
      const kind = classify(source, el);
      if (kind) hits.push({ file, line: lineOf(el.start), kind, element: el.name, classes: classesOf(source, el).join(' ') });
    }
    for (const [k, v] of Object.entries(node)) if (k !== 'metadata' && v && typeof v === 'object') visit(v);
  };
  visit(ast.fragment);
  return hits;
}

export function census(srcDir = resolve(import.meta.dirname, '../src')): Hit[] {
  const hits: Hit[] = [];
  for (const p of walk(srcDir)) {
    const file = relative(srcDir, p);
    if (PRIMITIVES.has(file)) continue;
    hits.push(...scanSource(readFileSync(p, 'utf8'), file));
  }
  return hits.sort((a, b) => a.file.localeCompare(b.file) || a.line - b.line);
}

if (import.meta.url === `file://${process.argv[1]}`) {
  const hits = census();
  for (const h of hits) console.log(`src/${h.file}:${h.line}\t${h.kind}\t<${h.element}>\t${h.classes}`);
  const by = Object.fromEntries(KINDS.map((k) => [k, hits.filter((h) => h.kind === k).length]));
  console.log(`total ${hits.length}`, JSON.stringify(by));
}
