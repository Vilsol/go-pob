import { readFileSync, readdirSync, statSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { compile } from 'tailwindcss';
import { describe, expect, it } from 'vitest';

const here = dirname(fileURLToPath(import.meta.url));
const src = resolve(here, '../..');
const tailwindDir = resolve(here, '../../../node_modules/tailwindcss');

const loadStylesheet = (id: string, base: string) => {
  const path = id.startsWith('tailwindcss/') ? join(tailwindDir, id.slice('tailwindcss/'.length)) : resolve(base, id);
  return Promise.resolve({ content: readFileSync(path, 'utf8'), base: dirname(path) });
};

const walk = (dir: string): string[] =>
  readdirSync(dir).flatMap((name) => {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) return walk(p);
    return /\.(svelte|ts|css)$/.test(name) && !/\.test\.ts$/.test(name) ? [p] : [];
  });

// colour-, tone- and shadow-taking utilities, with any variant prefix and opacity modifier
const UTILITY =
  /^(?:[\w-]+:)*!?(?:bg|text|border(?:-[trblxyse])?|outline|ring|ring-offset|fill|stroke|from|via|to|divide|placeholder|caret|accent|decoration|shadow|inset-shadow|drop-shadow|tone)-[a-z][\w-]*(?:\/\d+)?!?$/;

const TOKEN = /[^\s"'`{}(),;<>=]+/g;

const candidates = () => {
  const found = new Map<string, string[]>();
  for (const file of walk(src)) {
    const text = readFileSync(file, 'utf8');
    for (const m of text.matchAll(TOKEN)) {
      // an attribute name (stroke-width=, style:border-color=) is not a class
      if (text[m.index + m[0].length] === '=' && !m[0].startsWith('class:')) continue;
      const token = m[0].replace(/^class:/, '');
      if (!UTILITY.test(token)) continue;
      const files = found.get(token) ?? [];
      found.set(token, files);
      files.push(file.slice(src.length + 1));
    }
  }
  return found;
};

describe('colour, tone and shadow utilities resolve against tokens.css', () => {
  it('every one generates CSS', async () => {
    const css = readFileSync(join(here, 'styles/spec.css'), 'utf8');
    const compiler = await compile(css, { base: join(here, 'styles'), loadStylesheet });
    let size = compiler.build([]).length;
    const dead: string[] = [];
    const found = candidates();
    expect(found.size).toBeGreaterThan(200);
    expect(found.has('text-accent') && found.has('shadow-pop') && found.has('tone-worse')).toBe(true);
    for (const [token, files] of found) {
      const next = compiler.build([token]).length;
      if (next === size) dead.push(`${token} in ${[...new Set(files)].join(', ')}`);
      size = next;
    }
    expect(dead).toEqual([]);
  });

  it('no colour utility hard-codes a hex or rgb() colour, so every theme recolours it', () => {
    const raw = walk(src).flatMap((file) =>
      [
        ...readFileSync(file, 'utf8').matchAll(
          /\b(?:bg|text|border|outline|ring|fill|stroke|shadow|from|via|to|decoration|divide|caret|accent)(?:-[trblxy])?-\[(?:#|rgba?\()[^\]\s]*\]/g
        )
      ].map((m) => `${file.slice(src.length + 1)}: ${m[0]}`)
    );
    expect(raw).toEqual([]);
  });
});
