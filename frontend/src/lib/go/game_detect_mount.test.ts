// @vitest-environment jsdom

import { describe, expect, it, beforeAll, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

vi.mock('$lib/go/worker', async () => {
  const { remoteWorker } = await import('../../../harness/component/go_runtime');
  return { syncWrap: remoteWorker(), syncWorker: undefined };
});

const { bootWorker } = await import('../../../harness/component/go_runtime');
const { currentBuild } = await import('$lib/global');
const types = await import('$lib/types');
const { decodeCode, poe2Xml, rootElement } = await import('./game_detect');

const ROOT = resolve(process.cwd(), '..');
const FIXTURES = [
  'wasm2/exposition/testdata/sorceress_spark.xml',
  'wasm2/exposition/testdata/sorceress_items.xml',
  'testdata/builds/Fireball.xml',
  'testdata/builds/Fireball-full.xml'
].map((f) => readFileSync(resolve(ROOT, f), 'utf8'));

const b64 = (xml: string) => deflateSync(xml).toString('base64');
const b64url = (xml: string) => b64(xml).replaceAll('+', '-').replaceAll('/', '_');

const goDecode = (code: string) => {
  const r = types.pob.DecodeDecompress(code);
  return r.ok ? r.value : undefined;
};

beforeAll(async () => {
  await bootWorker(currentBuild);
}, 120_000);

describe('decodeCode', () => {
  const codesOf = () =>
    FIXTURES.flatMap((xml) => {
      const url = b64url(xml);
      return [
        url,
        b64(xml),
        types.pob.CompressEncode(xml).unwrap(),
        url.replace(/(.{76})/g, '$1\r\n'),
        url.slice(0, url.length >> 1),
        url.slice(0, -4),
        url + '!',
        ` ${url}`,
        url.replace(/=+$/, '')
      ];
    });
  const bad = ['', 'not a build code', 'AAAA', Buffer.from('<PathOfBuilding2/>').toString('base64'), 'eJ@@'];

  it('decodes a code exactly as pob.DecodeDecompress does, and never one it refuses', async () => {
    const codes = codesOf();
    const goOk = codes.filter((c) => goDecode(c) !== undefined);
    expect(goOk.length, 'the fixtures gave Go nothing to decode, so agreement proves nothing').toBeGreaterThanOrEqual(FIXTURES.length * 4);
    expect(codes.length - goOk.length, 'no code Go refuses, so refusing proves nothing').toBeGreaterThan(FIXTURES.length);
    for (const code of codes.filter((_, i) => i % 9 < 4)) expect(await decodeCode(code), code.slice(-12)).toBe(goDecode(code));
    // the rest may be refused here and left to Go, as long as nothing Go refuses decodes
    for (const code of [...codes, ...bad]) {
      const ours = await decodeCode(code);
      if (ours !== undefined) expect(ours, code.slice(-12)).toBe(goDecode(code));
      else if (goDecode(code) !== undefined) expect(code.endsWith('!'), code.slice(-12)).toBe(true);
    }
  });
});

describe('rootElement', () => {
  const docs = [
    ...FIXTURES,
    '<PathOfBuilding2/>',
    '<?xml version="1.0"?>\n<!-- saved -->\n<PathOfBuilding2>\n</PathOfBuilding2>',
    '﻿<PathOfBuilding><Build targetVersion="3_0"/></PathOfBuilding>',
    '<!DOCTYPE x>\n<PathOfBuilding2 a="1"></PathOfBuilding2>',
    '<PathOfBuilding2Extra/>',
    '<Other><PathOfBuilding2/></Other>',
    'PathOfBuilding2',
    ''
  ];

  it('is PathOfBuilding2 exactly where builds.DetectGame says PoE2', () => {
    const poe2 = docs.filter((d) => types.builds.DetectGame(new Uint8Array(new TextEncoder().encode(d))) === types.builds.Game.GamePoE2);
    expect(poe2.length).toBeGreaterThanOrEqual(4);
    for (const d of docs) expect(rootElement(d) === 'PathOfBuilding2', d.slice(0, 60)).toBe(poe2.includes(d));
  });

  it('hands back a PoE2 code its XML, and nothing for any other code', async () => {
    for (const xml of FIXTURES) expect(await poe2Xml(b64url(xml))).toBe(xml.includes('<PathOfBuilding2>') ? xml : undefined);
    expect(await poe2Xml('not a build code')).toBeUndefined();
  });
});
