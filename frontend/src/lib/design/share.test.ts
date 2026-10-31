// BuildSiteTools.lua:100
import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { downloadUrl, DROPPED_SITES, importLink, jsonVerdict, SITES, urlVerdict } from './share';

const pinRows = readFileSync(resolve(__dirname, 'testdata/build_sites.tsv'), 'utf8')
  .trim()
  .split('\n')
  .map((l) => l.split('\t'));
const dropped = new Set(DROPPED_SITES.map((s) => s.label));
const rows = pinRows.filter((r) => !dropped.has(r[0]));
// PoB-PoE2's rows, by the label go-pob gives a row that differs from PoE1's
const POE2_LABELS: Record<string, string> = { Maxroll: 'Maxroll (PoE2)', 'poe.ninja': 'poe.ninja (PoE2)' };
const PoE1_SHARED = new Set(['pobb.in', 'Pastebin.com', 'PastebinP.com']);
const poe2Rows = readFileSync(resolve(__dirname, 'testdata/build_sites_poe2.tsv'), 'utf8')
  .trim()
  .split('\n')
  .map((l) => l.split('\t'))
  .filter((r) => !dropped.has(r[0]))
  .map(([label, link, raw]) => [POE2_LABELS[label] ?? label, link, raw]);

describe('downloadUrl', () => {
  it('covers every site PoB lists', () => {
    expect(new Set(rows.map((r) => r[0]))).toEqual(new Set(SITES.filter((s) => !s.game).map((s) => s.label)));
  });

  it.each(rows)('%s: %s → %s', (label, link, raw) => {
    expect(downloadUrl(link)).toEqual({ site: label, url: raw });
  });

  it('leaves out only the sites go-pob drops, and refuses their links with a reason', () => {
    expect(pinRows.filter((r) => dropped.has(r[0])).map((r) => r[0])).toEqual(['Rentry.co']);
    const link = pinRows.find((r) => r[0] === 'Rentry.co')![1];
    expect(downloadUrl(link)).toBeUndefined();
    expect(urlVerdict(link)).toEqual({ valid: false, text: DROPPED_SITES[0].text });
  });

  it('covers every site PoB-PoE2 lists, its own rows marked as PoE2', () => {
    const labels = new Set(poe2Rows.map((r) => r[0]));
    expect(labels).toEqual(new Set(SITES.filter((s) => s.game === 'poe2' || PoE1_SHARED.has(s.label)).map((s) => s.label)));
  });

  it.each(poe2Rows)('PoB-PoE2 %s: %s → %s', (label, link, raw) => {
    expect(downloadUrl(link)).toEqual({ site: label, url: raw });
  });

  it('is undefined for text that is no site link', () => {
    expect(downloadUrl('eNrtXVtz2zgSfp')).toBeUndefined();
    expect(downloadUrl('https://example.com/pob/abc')).toBeUndefined();
  });
});

// ImportTab.lua:590-594
describe('importLink', () => {
  const cases = JSON.parse(readFileSync(resolve(__dirname, 'testdata/import_redirects.json'), 'utf8')) as { input: string; url: string }[];
  it.each(cases)('$input → $url', ({ input, url }) => {
    expect(importLink(input)).toBe(url);
  });

  it('a wrapped site link is recognised and downloaded as that site', () => {
    expect(urlVerdict(cases[0].input)?.site).toBe('pobb.in');
    expect(downloadUrl(cases[1].input)?.site).toBe('Pastebin.com');
  });
});

// ImportTab.lua:609-628
describe('jsonVerdict', () => {
  it('names each refusal as PoB does', () => {
    expect(jsonVerdict('{')).toBeUndefined();
    expect(jsonVerdict('{ nope }')).toEqual({ valid: false, text: 'Invalid JSON format (decode error)' });
    expect(jsonVerdict('{"x":1}')).toEqual({ valid: false, text: 'Invalid JSON format (character missing)' });
    expect(jsonVerdict('{"character":{"equipment":[]}}')).toEqual({ valid: false, text: 'Invalid JSON format (equipment or passives missing)' });
  });

  it("takes PoB's sample character, and is undefined for a build code", () => {
    const sample = readFileSync(resolve(__dirname, '../../../../testdata/import/sample_character.json'), 'utf8');
    expect(jsonVerdict(`  ${sample}\n`)).toEqual({ valid: true, text: 'JSON is valid', json: true });
    expect(jsonVerdict('eNrtXVtz2zgSfp')).toBeUndefined();
  });
});
