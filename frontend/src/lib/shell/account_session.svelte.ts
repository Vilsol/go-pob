import { get } from 'svelte/store';
import { outputs } from '$lib/global.js';
import { syncWrap } from '$lib/go/worker';
import { logError } from '$lib/utils';
import { NEGATIVE, POSITIVE, type AccountCharacter, type ImportStage } from '$lib/design/account';
import { downloadCharacter, fetchCharacterList, lastAccount, lastRealm, loadHistory, saveAccount } from './account_import';
import { predefinedName, unsavedName } from './build_name.svelte';

// ImportTab.lua:321-472, 813-1020
type ItemOpts = { clearItems: boolean; clearSkills: boolean; ignoreWeaponSwap: boolean };
export const accountSession = $state({
  realm: lastRealm(),
  account: lastAccount(),
  accountName: '',
  history: loadHistory(),
  status: 'Idle',
  stage: 'GETACCOUNTNAME' as ImportStage,
  characters: [] as AccountCharacter[],
  lastLeague: undefined as string | undefined,
  selected: undefined as AccountCharacter | undefined,
  ignoreWeaponSwap: false
});

export const resetAccountSession = () => {
  Object.assign(accountSession, { status: 'Idle', stage: 'GETACCOUNTNAME', characters: [], selected: undefined, accountName: '' });
};

export const startAccount = async () => {
  const s = accountSession;
  s.stage = 'DOWNLOADCHARLIST';
  s.status = 'Retrieving character list...';
  const r = await fetchCharacterList(s.realm, s.account);
  s.status = r.status;
  if (!r.ok) {
    s.stage = 'GETACCOUNTNAME';
    return;
  }
  s.account = s.accountName = r.accountName;
  s.characters = r.characters;
  s.stage = 'SELECTCHAR';
  s.history = saveAccount(s.history, s.account, s.realm);
};

const importCharacter = async (
  c: AccountCharacter,
  endpoint: 'passives' | 'items',
  apply: (body: string) => Promise<{ ok: boolean; text: string } | undefined>
) => {
  const s = accountSession;
  s.stage = 'IMPORTING';
  s.status = endpoint === 'passives' ? 'Retrieving character passive tree...' : 'Retrieving character items...';
  const d = await downloadCharacter(endpoint, s.realm, s.accountName, c);
  if (!d.ok) {
    s.stage = 'SELECTCHAR';
    s.status = d.status;
    return;
  }
  s.lastLeague ??= c.league;
  const r = await apply(d.body);
  s.stage = 'SELECTCHAR';
  s.status = r?.ok ? `${POSITIVE}${r.text}` : `${NEGATIVE}${r?.text ?? 'no build'}`;
};

/* ImportTab.lua:430-442 */
export const importTree = (c: AccountCharacter, deleteJewels: boolean, confirm: (run: () => void) => void) => {
  const run = () =>
    void importCharacter(c, 'passives', async (body) => syncWrap?.ImportAccountPassiveTree(JSON.stringify(c), body, deleteJewels)).catch(logError);
  if ((get(outputs)?.PointsUsed?.[0] ?? 0) > 0) confirm(run);
  else run();
  unsavedName.predefined = predefinedName(accountSession.account, c.name);
};

/* ImportTab.lua:448-453 */
export const importItems = (c: AccountCharacter, o: ItemOpts) => {
  accountSession.ignoreWeaponSwap = o.ignoreWeaponSwap;
  void importCharacter(c, 'items', async (body) =>
    syncWrap?.ImportAccountItems(JSON.stringify(c), body, o.clearItems, o.clearSkills, o.ignoreWeaponSwap)
  ).catch(logError);
  unsavedName.predefined = predefinedName(accountSession.account, c.name);
};

/* CompareTab.lua:266-271 */
export const canReimport = () => accountSession.stage === 'SELECTCHAR' && !!accountSession.selected;

/* CompareTab.lua:1288-1308 */
export const reimportPrimary = (confirm: (run: () => void) => void) => {
  const c = accountSession.selected;
  if (!canReimport() || !c) return;
  importTree(c, true, confirm);
  importItems(c, { clearItems: true, clearSkills: true, ignoreWeaponSwap: accountSession.ignoreWeaponSwap });
};
