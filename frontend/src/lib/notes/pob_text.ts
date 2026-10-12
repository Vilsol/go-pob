export type PobColorAttrs = { code: string; lead: string | null; run: number };
export type Mark = { type: string; attrs?: Record<string, unknown> };
export type DocNode = {
  type: string;
  attrs?: Record<string, unknown>;
  content?: DocNode[];
  text?: string;
  marks?: Mark[];
};

export const COLOR_MARK = 'pobColor';
export const DEFAULT_CODE = '^7';

const TOKEN = /\^x[0-9A-Fa-f]{6}|\^[0-9]/g;
const ONE_CODE = /^(?:\^x[0-9A-Fa-f]{6}|\^[0-9])$/;
const CODE_RUN = /^(?:\^x[0-9A-Fa-f]{6}|\^[0-9])+$/;

export const isCode = (s: unknown): s is string => typeof s === 'string' && ONE_CODE.test(s);
const isRun = (s: unknown): s is string => typeof s === 'string' && CODE_RUN.test(s);
const lastCode = (run: string) => run.match(TOKEN)!.at(-1)!;

const DIGITS = ['#000000', '#FF0000', '#00FF00', '#0000FF', '#FFFF00', '#FF00FF', '#00FFFF', '#FFFFFF', '#B2B2B2', '#666666'];

export const codeToCss = (code: string): string | null => {
  if (!isCode(code)) return null;
  if (code === DEFAULT_CODE) return null;
  return code[1] === 'x' ? `#${code.slice(2).toUpperCase()}` : DIGITS[Number(code[1])];
};

const colorMark = (code: string, lead: string | null, run: number): Mark => ({ type: COLOR_MARK, attrs: { code, lead, run } });

export const plainToDoc = (plain: string): DocNode => {
  let state: string | null = null;
  const content: DocNode[] = plain.split('\n').map((line) => {
    const inline: DocNode[] = [];
    const push = (text: string, mark: Mark | null) => {
      if (text) inline.push(mark ? { type: 'text', text, marks: [mark] } : { type: 'text', text });
    };

    let runStart = -1;
    let runEnd = -1;
    let trail: string | null = null;
    let prev: PobColorAttrs | null = null;

    const flushRun = (textEnd: number) => {
      const run = line.slice(runStart, runEnd);
      const code = lastCode(run);
      const text = line.slice(runEnd, textEnd);
      if (text) {
        const n = prev && prev.code === code && prev.lead === run ? prev.run + 1 : 0;
        push(text, colorMark(code, run, n));
        prev = { code, lead: run, run: n };
      } else {
        trail = run;
      }
      state = code;
    };

    for (const m of line.matchAll(TOKEN)) {
      const at = m.index;
      if (runStart >= 0 && at === runEnd) {
        runEnd = at + m[0].length;
        continue;
      }
      if (runStart >= 0) {
        flushRun(at);
      } else {
        push(line.slice(0, at), state ? colorMark(state, null, 0) : null);
      }
      runStart = at;
      runEnd = at + m[0].length;
    }
    if (runStart >= 0) {
      flushRun(line.length);
    } else {
      push(line, state ? colorMark(state, null, 0) : null);
    }

    return { type: 'paragraph', attrs: { trail }, content: inline.length ? inline : undefined };
  });
  return { type: 'doc', content };
};

type Line = { prefix: string; inline: DocNode[]; trail: string | null };

const colorOf = (n: DocNode): PobColorAttrs | null => {
  const m = n.marks?.find((k) => k.type === COLOR_MARK);
  if (!m || !isCode(m.attrs?.code)) return null;
  const lead = isRun(m.attrs?.lead) && lastCode(m.attrs.lead) === m.attrs.code ? m.attrs.lead : null;
  const run = typeof m.attrs?.run === 'number' ? m.attrs.run : 0;
  return { code: m.attrs.code, lead, run };
};

const sameColor = (a: PobColorAttrs | null, b: PobColorAttrs | null) => !!a && !!b && a.code === b.code && a.lead === b.lead && a.run === b.run;

const inlineLines = (block: DocNode, prefix: string, indent: string): Line[] => {
  const lines: Line[] = [{ prefix, inline: [], trail: null }];
  for (const n of block.content ?? []) {
    if (n.type === 'hardBreak') lines.push({ prefix: indent, inline: [], trail: null });
    else if (typeof n.text === 'string') lines.at(-1)!.inline.push(n);
    else lines.at(-1)!.inline.push(...inlineLines(n, '', '').flatMap((l) => l.inline));
  }
  const trail = block.attrs?.trail;
  if (isRun(trail)) lines.at(-1)!.trail = trail;
  return lines;
};

const LISTS = new Set(['bulletList', 'orderedList', 'taskList']);

const blockLines = (block: DocNode, prefix: string, indent: string, depth: number): Line[] => {
  switch (block.type) {
    case 'bulletList':
    case 'orderedList':
    case 'taskList': {
      const start = typeof block.attrs?.start === 'number' ? block.attrs.start : 1;
      const pad = '  '.repeat(depth);
      return (block.content ?? []).flatMap((item, i) => {
        const marker = block.type === 'bulletList' ? '• ' : block.type === 'taskList' ? (item.attrs?.checked ? '☑ ' : '☐ ') : `${start + i}. `;
        const children = item.content ?? [];
        if (children.length === 0) return [{ prefix: pad + marker, inline: [], trail: null }];
        return children.flatMap((child, j) =>
          LISTS.has(child.type)
            ? blockLines(child, '', '', depth + 1)
            : blockLines(child, j === 0 ? pad + marker : pad + ' '.repeat(marker.length), pad + ' '.repeat(marker.length), depth + 1)
        );
      });
    }
    case 'horizontalRule':
      return [{ prefix: prefix + '----------', inline: [], trail: null }];
    case 'paragraph':
    case 'heading':
      return inlineLines(block, prefix, indent);
    default:
      if (block.content?.some((c) => typeof c.text === 'string' || c.type === 'hardBreak')) return inlineLines(block, prefix, indent);
      return (block.content ?? []).flatMap((c) => blockLines(c, prefix, indent, depth));
  }
};

export const docToPlain = (doc: DocNode): string => {
  const lines = (doc.content ?? []).flatMap((b) => blockLines(b, '', '', 0));
  let state: string | null = null;
  const out = lines.map((line) => {
    let s = line.prefix;
    let prev: PobColorAttrs | null = null;
    for (const n of line.inline) {
      const c = colorOf(n);
      if (c && c.lead !== null && !sameColor(prev, c)) {
        s += c.lead;
        state = c.code;
      } else if (c && state !== c.code) {
        s += c.code;
        state = c.code;
      } else if (!c && state !== null && state !== DEFAULT_CODE) {
        s += DEFAULT_CODE;
        state = DEFAULT_CODE;
      }
      s += n.text ?? '';
      prev = c;
    }
    if (line.trail) {
      s += line.trail;
      state = lastCode(line.trail);
    }
    return s;
  });
  return out.join('\n');
};

export const countCodes = (plain: string) => (plain.match(TOKEN) ?? []).length;
