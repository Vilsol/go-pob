// Lists template whitespace that survives at the start or end of an element's text. Svelte trims
// whitespace at the edge of an element, block branch, snippet or component children (after dropping
// comments and {@const}), but not beside a sibling that can render nothing:
// `<p>{#if a}<i />{/if}\n  Name</p>` renders "<p> Name</p>" when `a` is false. Only whitespace that
// then touches text ({expression}, {@html}, children) is reported; {expression} counts as non-empty.
// `node harness/edge_whitespace.ts` prints file:line per hit; edge_whitespace.test.ts holds the ceiling.
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, relative, resolve } from 'node:path';
import { parse, type AST } from 'svelte/compiler';

export type Side = 'start' | 'end';
export type Hit = { file: string; line: number; side: Side };

type Node = AST.Fragment['nodes'][number];
type Frag = AST.Fragment | null | undefined;

const walk = (dir: string): string[] =>
  readdirSync(dir).flatMap((name) => {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) return walk(p);
    return name.endsWith('.svelte') ? [p] : [];
  });

const blank = (n: Node) => n.type === 'Text' && n.data.trim() === '';

const SPECIAL = new Set(['SvelteHead', 'SvelteWindow', 'SvelteDocument', 'SvelteBody', 'SvelteOptions', 'TitleElement']);
const transparent = (n: Node) => ['Comment', 'ConstTag', 'DebugTag', 'SnippetBlock'].includes(n.type) || SPECIAL.has(n.type);

const branches = (n: Node): Frag[] => {
  switch (n.type) {
    case 'IfBlock':
      return [n.consequent, n.alternate];
    case 'EachBlock':
      return [n.body, n.fallback];
    case 'KeyBlock':
      return [n.fragment];
    case 'AwaitBlock':
      return [n.pending, n.then, n.catch];
    default:
      return [];
  }
};

const PRESERVE = new Set(['pre', 'textarea', 'script', 'style']);

export function scanSource(source: string, file = ''): Hit[] {
  const ast = parse(source, { modern: true, filename: file });
  const snippets = new Map<string, AST.Fragment>();
  const found = new Map<number, Hit>();
  const lineOf = (i: number) => source.slice(0, i).split('\n').length;

  // the local snippet a {@render x()} or {@render x?.()} calls, if any
  const callee = (n: AST.RenderTag) => {
    let e = n.expression as { type: string; expression?: unknown; callee?: { type: string; name?: string } };
    if (e.type === 'ChainExpression') e = e.expression as typeof e;
    return e.callee?.type === 'Identifier' ? e.callee.name : undefined;
  };
  const target = (n: AST.RenderTag) => snippets.get(callee(n) ?? '');

  const seen = new Set<AST.Fragment>();
  function mayBeEmpty(n: Node): boolean {
    const fragEmpty = (f: Frag): boolean => {
      if (!f) return true;
      if (seen.has(f)) return false;
      seen.add(f);
      const r = f.nodes.every((c) => blank(c) || transparent(c) || mayBeEmpty(c));
      seen.delete(f);
      return r;
    };
    switch (n.type) {
      case 'Text':
        return n.data.trim() === '';
      case 'HtmlTag':
      case 'AwaitBlock':
        return true;
      case 'RenderTag': {
        const s = target(n);
        return s ? fragEmpty(s) : true;
      }
      case 'IfBlock':
        return fragEmpty(n.consequent) || fragEmpty(n.alternate);
      case 'EachBlock':
        return !n.fallback || fragEmpty(n.body) || fragEmpty(n.fallback);
      case 'KeyBlock':
        return fragEmpty(n.fragment);
      default:
        return transparent(n);
    }
  }

  // what the first rendered thing in `nodes` is, read from `side`
  const kindSeen = new Set<AST.Fragment>();
  function kind(nodes: Node[], side: Side): 'text' | 'element' | 'none' {
    const kept = nodes.filter((n) => !transparent(n));
    for (const n of side === 'start' ? kept : kept.reverse()) {
      if (blank(n)) continue;
      if (n.type === 'Text' || n.type === 'ExpressionTag' || n.type === 'HtmlTag') return 'text';
      let inner: Frag[] = branches(n);
      if (n.type === 'RenderTag') {
        const s = target(n);
        if (!s) return callee(n) === 'children' ? 'text' : 'element';
        inner = [s];
      } else if (!inner.length) return 'element';
      let k: 'element' | 'none' = 'none';
      for (const f of inner) {
        if (!f || kindSeen.has(f)) continue;
        kindSeen.add(f);
        const r = kind(f.nodes, side);
        kindSeen.delete(f);
        if (r === 'text') return 'text';
        if (r === 'element') k = 'element';
      }
      if (k === 'element' && !mayBeEmpty(n)) return 'element';
    }
    return 'none';
  }

  // Walks one edge of a trimmed fragment: once a sibling that can render nothing has been passed,
  // whitespace before the first node that always renders something survives.
  function edge(nodes: Node[], side: Side) {
    const kept = nodes.filter((n) => !transparent(n));
    const trimmed = [kept[0], kept.at(-1)].filter((n) => n && blank(n));
    const order = side === 'start' ? kept : [...kept].reverse();
    let passed = false;
    for (const n of order) {
      if (trimmed.includes(n)) continue;
      if (n.type === 'Text') {
        const ws = side === 'start' ? /^\s/.test(n.data) : /\s$/.test(n.data);
        const at = kept.indexOf(n);
        const touches = !blank(n) || kind(side === 'start' ? kept.slice(at + 1) : kept.slice(0, at), side) === 'text';
        if (passed && ws && touches) found.set(n.start, { file, line: lineOf(side === 'start' ? n.start : n.start + n.data.trimEnd().length), side });
        if (!blank(n)) return;
        continue;
      }
      for (const b of branches(n)) if (b) edge(b.nodes, side);
      if (!mayBeEmpty(n)) return;
      passed = true;
    }
  }
  const contentEdges = (f: AST.Fragment) => {
    edge(f.nodes, 'start');
    edge(f.nodes, 'end');
  };

  type Visited = { type?: string; name?: string; fragment?: AST.Fragment; body?: AST.Fragment; expression?: { name: string } };
  const each = (node: unknown, fn: (n: Visited) => boolean | void): void => {
    if (!node || typeof node !== 'object') return;
    if (Array.isArray(node)) return node.forEach((c) => each(c, fn));
    if (fn(node as Visited) === false) return;
    for (const [k, v] of Object.entries(node)) if (k !== 'metadata' && v && typeof v === 'object') each(v, fn);
  };
  each(ast.fragment, (n) => {
    if (n.type === 'SnippetBlock') snippets.set(n.expression!.name, n.body!);
  });
  contentEdges(ast.fragment);
  each(ast.fragment, (n) => {
    if (n.type === 'RegularElement' && PRESERVE.has(n.name!)) return false;
    if (n.type === 'SnippetBlock') contentEdges(n.body!);
    else if (n.fragment && ['RegularElement', 'SvelteElement', 'Component', 'SvelteComponent'].includes(n.type!)) contentEdges(n.fragment);
  });
  return [...found.values()].sort((a, b) => a.line - b.line || a.side.localeCompare(b.side));
}

export function census(srcDir = resolve(import.meta.dirname, '../src')): Hit[] {
  const hits: Hit[] = [];
  for (const p of walk(srcDir)) hits.push(...scanSource(readFileSync(p, 'utf8'), relative(srcDir, p)));
  return hits.sort((a, b) => a.file.localeCompare(b.file) || a.line - b.line);
}

if (import.meta.url === `file://${process.argv[1]}`) {
  const hits = census();
  for (const h of hits) console.log(`src/${h.file}:${h.line}\t${h.side}`);
  console.log(`total ${hits.length}`);
}
