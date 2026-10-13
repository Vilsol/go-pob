import { afterEach, describe, expect, it, vi } from 'vitest';
import { canStart, characterList, leaguesOf, missingDiscriminator, NEGATIVE, normaliseAccount, REALM_LIST, type AccountCharacter } from '$lib/design/account';
import { downloadCharacter, fetchCharacterList, saveAccount, removeAccount } from './account_import';

const [PC, XBOX] = REALM_LIST;

afterEach(() => vi.unstubAllGlobals());

const serve = (answers: Record<string, () => Response>) => {
  const f = vi.fn((url: string) => {
    const endpoint = /\/api\/poe\/(\w+)\?/.exec(url)?.[1] ?? '';
    return Promise.resolve(answers[endpoint]?.() ?? new Response('', { status: 500 }));
  });
  vi.stubGlobal('fetch', f);
  return f;
};

const chars: AccountCharacter[] = [
  { name: 'beta', realm: 'pc', class: 'Necromancer', league: 'Standard', level: 90 },
  { name: 'Alpha', realm: 'pc', class: 'Deadeye', league: 'Mercenaries', level: 95 },
  { name: 'Gamma', realm: 'xbox', class: 'Slayer', league: 'Standard', level: 80 }
];

describe('the account name (ImportTab.lua:368-404, 906-913)', () => {
  it('needs a discriminator to start, and says so', () => {
    expect(canStart('Vilsol#3184')).toBe(true);
    expect(canStart('Vilsol-3184')).toBe(true);
    expect(canStart('#3184')).toBe(false);
    expect(canStart('Vilsol')).toBe(false);
    expect(missingDiscriminator('Vilsol')).toBe(true);
    expect(missingDiscriminator('')).toBe(false);
    expect(missingDiscriminator('Vilsol-3184')).toBe(false);
  });

  it('loses spaces on PC, keeps them as + on consoles, and makes the last - a #', () => {
    expect(normaliseAccount(PC, ' Vil sol-3184 ')).toBe('Vilsol#3184');
    expect(normaliseAccount(XBOX, ' ?Some Name-1234? ')).toBe('Some+Name#1234');
    expect(normaliseAccount(PC, 'a-b-1234')).toBe('a-b#1234');
  });
});

describe('BuildCharacterList (ImportTab.lua:973-990, 1026-1094)', () => {
  it("lists a realm's characters in a league by name, and every league's with its league named under All", () => {
    expect(leaguesOf(chars)).toEqual(['Mercenaries', 'Standard']);
    expect(characterList('pc', 'Standard', chars).map((e) => [e.label, e.detail])).toEqual([['beta', 'Necromancer lvl 90']]);
    expect(characterList('pc', undefined, chars).map((e) => [e.label, e.detail])).toEqual([
      ['Alpha', 'Deadeye lvl 95 in Mercenaries'],
      ['beta', 'Necromancer lvl 90 in Standard']
    ]);
  });
});

describe('DownloadSiteCharacterList (ImportTab.lua:886-1020)', () => {
  it("asks for the normalised name's characters, then takes the profile page's spelling", async () => {
    const f = serve({
      characters: () => Response.json(chars),
      profile: () => Response.json({ accountName: 'Vilsol-3184' })
    });
    const r = await fetchCharacterList(PC, 'vilsol-3184');
    expect(f.mock.calls[0][0]).toBe('/api/poe/characters?realm=pc&account=vilsol%233184');
    expect(f.mock.calls[1][0]).toBe('/api/poe/profile?realm=pc&account=vilsol%233184');
    expect(r).toEqual({ ok: true, status: 'Character list successfully retrieved.', accountName: 'Vilsol#3184', characters: chars });
  });

  it('keeps the typed name when pathofexile.com challenges the profile page, and fails on any other profile error', async () => {
    serve({ characters: () => Response.json(chars), profile: () => new Response('pathofexile.com answered 403 (challenge)', { status: 403 }) });
    expect(await fetchCharacterList(PC, 'vilsol#3184')).toMatchObject({ ok: true, accountName: 'vilsol#3184' });
    serve({ characters: () => Response.json(chars), profile: () => new Response('pathofexile.com answered 500', { status: 500 }) });
    expect(await fetchCharacterList(PC, 'vilsol#3184')).toEqual({
      ok: false,
      status: `${NEGATIVE}Error retrieving character list, try again (Response code: 500)`
    });
  });

  it.each([
    [401, 'Account profile is private or does not exist.'],
    [403, 'Account profile is private or does not exist.'],
    [404, 'Account name is incorrect.'],
    [429, 'Error retrieving character list, try again (Response code: 429)']
  ])("says PoB's message for %s", async (status, text) => {
    serve({ characters: () => new Response('', { status }) });
    expect(await fetchCharacterList(PC, 'a#1234')).toEqual({ ok: false, status: NEGATIVE + text });
  });

  it('says when the list is not JSON, or empty', async () => {
    serve({ characters: () => new Response('<html>') });
    expect((await fetchCharacterList(PC, 'a#1234')).status).toBe(`${NEGATIVE}Error processing character list, try again later`);
    serve({ characters: () => Response.json([]) });
    expect((await fetchCharacterList(PC, 'a#1234')).status).toBe(`${NEGATIVE}The account has no characters to import.`);
  });
});

describe('DownloadItems / DownloadPassiveTree (ImportTab.lua:813-885)', () => {
  it("asks for the character's answer, and says PoB's message on an error or a false", async () => {
    const f = serve({ passives: () => new Response('{"hashes":[]}'), items: () => new Response('false') });
    expect(await downloadCharacter('passives', PC, 'Vilsol#3184', chars[0])).toEqual({ ok: true, body: '{"hashes":[]}' });
    expect(f.mock.calls[0][0]).toBe('/api/poe/passives?realm=pc&account=Vilsol%233184&character=beta');
    expect(await downloadCharacter('items', PC, 'Vilsol#3184', chars[0])).toEqual({
      ok: false,
      status: `${NEGATIVE}Failed to retrieve character data, try again.`
    });
    serve({});
    expect(await downloadCharacter('items', PC, 'Vilsol#3184', chars[0])).toEqual({
      ok: false,
      status: `${NEGATIVE}Error importing character data, try again (Response code: 500)`
    });
  });
});

describe('the account history (ImportTab.lua:385-393, 801-811)', () => {
  it('adds an account once, sorted ignoring case, and removes it', () => {
    const h = saveAccount(saveAccount(['b#1'], 'A#2', PC), 'b#1', PC);
    expect(h).toEqual(['A#2', 'b#1']);
    expect(removeAccount(h, 'A#2')).toEqual(['b#1']);
  });
});
