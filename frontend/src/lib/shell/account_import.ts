// ImportTab.lua:321-472, 813-1094

import { NEGATIVE, normaliseAccount, REALM_LIST, type AccountCharacter, type Realm } from '$lib/design/account';

const api = (endpoint: string, params: Record<string, string>) => fetch(`/api/poe/${endpoint}?${new URLSearchParams(params)}`);

const errMsg = (res: Response) => `Response code: ${res.status}`;

export type CharacterListResult = { ok: true; status: string; accountName: string; characters: AccountCharacter[] } | { ok: false; status: string };

/* ImportTab.lua:886-1020 */
export const fetchCharacterList = async (realm: Realm, buf: string): Promise<CharacterListResult> => {
  const accountName = normaliseAccount(realm, buf);
  const params = { realm: realm.realmCode, account: accountName };
  let res: Response;
  try {
    res = await api('characters', params);
  } catch (e) {
    return { ok: false, status: `${NEGATIVE}Error retrieving character list, try again (${String(e)})` };
  }
  if (res.status === 401 || res.status === 403) return { ok: false, status: `${NEGATIVE}Account profile is private or does not exist.` };
  if (res.status === 404) return { ok: false, status: `${NEGATIVE}Account name is incorrect.` };
  if (!res.ok) return { ok: false, status: `${NEGATIVE}Error retrieving character list, try again (${errMsg(res)})` };
  let characters: AccountCharacter[];
  try {
    characters = (await res.json()) as AccountCharacter[];
    if (!Array.isArray(characters)) throw new Error('Return type is not a table');
  } catch {
    return { ok: false, status: `${NEGATIVE}Error processing character list, try again later` };
  }
  if (characters.length === 0) return { ok: false, status: `${NEGATIVE}The account has no characters to import.` };

  let realName = accountName;
  try {
    const profile = await api('profile', params);
    if (profile.ok) realName = ((await profile.json()) as { accountName: string }).accountName.replace(/^(.*)[#-]/s, '$1#');
    else if (!(profile.status === 403 && (await profile.text()).includes('(challenge)')))
      return { ok: false, status: `${NEGATIVE}Error retrieving character list, try again (${errMsg(profile)})` };
  } catch (e) {
    return { ok: false, status: `${NEGATIVE}Error retrieving character list, try again (${String(e)})` };
  }
  return { ok: true, status: 'Character list successfully retrieved.', accountName: realName, characters };
};

/* ImportTab.lua:813-885 */
export const downloadCharacter = async (
  endpoint: 'items' | 'passives',
  realm: Realm,
  accountName: string,
  character: AccountCharacter
): Promise<{ ok: true; body: string } | { ok: false; status: string }> => {
  let res: Response;
  try {
    res = await api(endpoint, { realm: realm.realmCode, account: accountName, character: character.name });
  } catch (e) {
    return { ok: false, status: `${NEGATIVE}Error importing character data, try again (${String(e)})` };
  }
  if (!res.ok) return { ok: false, status: `${NEGATIVE}Error importing character data, try again (${errMsg(res)})` };
  const body = await res.text();
  if (body === 'false') return { ok: false, status: `${NEGATIVE}Failed to retrieve character data, try again.` };
  return { ok: true, body };
};

const HISTORY_KEY = 'import:accounts';
const LAST_ACCOUNT_KEY = 'import:lastAccount';
const LAST_REALM_KEY = 'import:lastRealm';

const read = (key: string) => {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
};
const write = (key: string, value: string) => {
  try {
    localStorage.setItem(key, value);
  } catch {
    // empty
  }
};

export const loadHistory = (): string[] => {
  try {
    const list = JSON.parse(read(HISTORY_KEY) ?? '[]') as unknown;
    return Array.isArray(list) ? list.filter((s): s is string => typeof s === 'string') : [];
  } catch {
    return [];
  }
};
export const lastAccount = () => read(LAST_ACCOUNT_KEY) ?? '';
export const lastRealm = () => REALM_LIST.find((r) => r.id === read(LAST_REALM_KEY)) ?? REALM_LIST[0];

/* ImportTab.lua:801-811 */
export const saveAccount = (history: string[], accountName: string, realm: Realm) => {
  write(LAST_ACCOUNT_KEY, accountName);
  write(LAST_REALM_KEY, realm.id);
  if (history.includes(accountName)) return history;
  const next = [...history, accountName].sort((a, b) => (a.toLowerCase() < b.toLowerCase() ? -1 : a.toLowerCase() > b.toLowerCase() ? 1 : 0));
  write(HISTORY_KEY, JSON.stringify(next));
  return next;
};

/* ImportTab.lua:385-393 */
export const removeAccount = (history: string[], accountName: string) => {
  const next = history.filter((a) => a !== accountName);
  write(HISTORY_KEY, JSON.stringify(next));
  return next;
};
