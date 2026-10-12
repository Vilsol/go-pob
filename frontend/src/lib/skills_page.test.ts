import { expect, describe, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const raw = readFileSync(fileURLToPath(new URL('./shell/AppSkills.svelte', import.meta.url)), 'utf8');

const code = raw
  .replace(/<!--[\s\S]*?-->/g, '')
  .split('\n')
  .map((l) => l.replace(/(^|\s)\/\/.*$/, '$1'))
  .join('\n');

const body = (name: string) => {
  const start = code.search(new RegExp(`const ${name} = `));
  expect(start, `${name} not found`).toBeGreaterThanOrEqual(0);
  const end = code.indexOf('\n  };', start);
  expect(end, `${name} has no end`).toBeGreaterThan(start);
  return code.slice(start, end);
};

describe('the comment stripper actually strips', () => {
  it('removes prose but keeps code', () => {
    expect(raw).toContain('THE LOAD PATH NEVER WRITES');
    expect(code).not.toContain('THE LOAD PATH NEVER WRITES');
    expect(code).toContain('SetSocketGroupGems');
  });
});

describe('the load path never writes', () => {
  it('has no two-way store at all', () => {
    expect(code).not.toMatch(/\bwritable\(/);
    expect(code).not.toMatch(/\.subscribe\(/);
  });

  it('reads Go in load() without calling a setter or assigning a Go field', () => {
    const load = body('load');
    expect(load).toContain('GetSocketGroupSummaries');
    expect(load).not.toMatch(/\.Set[A-Z]\w*\(|AddNewSocketGroup|DeleteSocketGroup|DeleteAllSocketGroups|UITick|= send\(/);
  });

  it('never awaits the live Skills wrapper itself, only its scalars and the plain gem slice', () => {
    expect(code).not.toMatch(/await\s+(\$currentBuild|build)\??\.Skills\s*[;)]/);
    expect(code).not.toMatch(/await\s+skills\s*[;)]/);
    expect(code).not.toMatch(/\$currentBuild\??\.Skills\??\.then/);
  });
});

describe('the main socket group is written only from a user action', () => {
  it('calls SetMainSocketGroup from exactly one place, the commit helper', () => {
    expect(code.match(/SetMainSocketGroup/g) ?? []).toHaveLength(1);
    expect(body('commitMainSocketGroup')).toContain('SetMainSocketGroup');
  });
});

describe('an edit outlives a read that started before it', () => {
  it('load() drops its result when an edit happened meanwhile', () => {
    const load = body('load');
    expect(load).toMatch(/const seq = editSeq/);
    expect(load).toMatch(/seq !== editSeq/);
  });

  it('every write path bumps editSeq', () => {
    for (const fn of ['writeGems', 'ongroup', 'onnewgroup', 'ondeletegroup', 'ondeleteall']) {
      expect(body(fn), fn).toContain('editSeq++');
    }
  });

  it('every gem change goes out through writeGems, which writes the whole list then Ticks', () => {
    expect(code.match(/SetSocketGroupGems/g) ?? []).toHaveLength(1);
    const write = body('writeGems');
    expect(write).toMatch(/SetSocketGroupGems\([^)]*\$state\.snapshot\(/);
    expect(write.indexOf('SetSocketGroupGems')).toBeLessThan(write.indexOf("UITick('currentGemGroup')"));
  });
});

describe('the gem options are written only after the build’s own values were read', () => {
  it('sets gemOptionsLoaded last in the read-back', () => {
    expect(code).toMatch(/gemOptionsLoaded = true;\s*\}\)/);
  });

  it('refuses every option write before that', () => {
    for (const fn of ['onsort', 'onoption']) {
      expect(body(fn), fn).toMatch(/if \(!gemOptionsLoaded/);
    }
    const setters = [
      'SetSortGemsByDPS',
      'SetSortGemsByDPSField',
      'SetDefaultGemLevelMode',
      'SetDefaultGemQuality',
      'SetShowSupportGemTypes',
      'SetShowLegacyGems',
      'SetShowAltQualityGems'
    ];
    const gated = body('onsort') + body('onoption');
    for (const s of setters) {
      expect(code.match(new RegExp(`\\.${s}\\(`, 'g')) ?? [], s).toHaveLength(1);
      expect(gated, s).toContain(`.${s}(`);
    }
  });

  // SkillsTab.lua:130-131
  it("writes the default gem level as the select's mode", () => {
    expect(body('onoption')).toMatch(/\.SetDefaultGemLevelMode\(patch\.defaultLevel\)/);
    expect(code).not.toMatch(/\.SetDefaultGemLevel\(/);
  });
});
