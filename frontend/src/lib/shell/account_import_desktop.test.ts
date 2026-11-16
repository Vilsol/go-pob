import { afterEach, expect, it, vi } from 'vitest';
import { NEGATIVE, REALM_LIST } from '$lib/design/account';

vi.stubEnv('VITE_DESKTOP', '1');
const { fetchCharacterList } = await import('./account_import');

afterEach(() => vi.unstubAllGlobals());

it("names this computer, not pob.gg, when pathofexile.com limits the desktop app's proxy", async () => {
  vi.stubGlobal('fetch', () => Promise.resolve(new Response('rate limited by pathofexile.com', { status: 429, headers: { 'Retry-After': '90' } })));
  expect(await fetchCharacterList(REALM_LIST[0], 'Vilsol#3184')).toEqual({
    ok: false,
    status:
      `${NEGATIVE}pathofexile.com is limiting import requests from this computer; try again in about 2 minutes. ` +
      'Meanwhile, import a build code or link (pobb.in, poe.ninja, pastebin).'
  });
});
