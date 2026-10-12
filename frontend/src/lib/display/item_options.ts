import { writable } from 'svelte/store';

const KEY = 'options:slotOnlyTooltips';
const load = () => {
  try {
    return globalThis.localStorage?.getItem(KEY) !== 'false';
  } catch {
    return true;
  }
};

/* Main.lua:1115-1118, Main.lua:114, ItemsTab.lua:5081-5088 */
export const slotOnlyTooltips = writable<boolean>(load());
slotOnlyTooltips.subscribe((v) => {
  try {
    globalThis.localStorage?.setItem(KEY, String(v));
  } catch {
    // empty
  }
});

const option = (key: string) => {
  let on = true;
  try {
    on = globalThis.localStorage?.getItem(key) !== 'false';
  } catch {
    // empty
  }
  const s = writable<boolean>(on);
  s.subscribe((v) => {
    try {
      globalThis.localStorage?.setItem(key, String(v));
    } catch {
      // empty
    }
  });
  return s;
};

/* Main.lua:1120-1125 */
export const migrateEldritchImplicits = option('options:migrateEldritchImplicits');
/* Main.lua:1127-1131 */
export const notSupportedModTooltips = option('options:notSupportedModTooltips');
/* Main.lua:1020-1024 */
export const showFlavourText = option('options:showFlavourText');
/* Main.lua:1109-1113, Build.lua:566 */
export const showWarnings = option('options:showWarnings');
/* Main.lua:121, 1032-1035 */
export const showAllItemAffixes = option('options:showAllItemAffixes');

const numberOption = (key: string, def: number, lo: number, hi: number) => {
  let v = def;
  try {
    const raw = globalThis.localStorage?.getItem(key);
    if (raw !== null && raw !== undefined && Number.isFinite(Number(raw))) v = Number(raw);
  } catch {
    // empty
  }
  const s = writable<number>(Math.min(Math.max(v, lo), hi));
  s.subscribe((n) => {
    try {
      globalThis.localStorage?.setItem(key, String(n));
    } catch {
      // empty
    }
  });
  return s;
};

/* Main.lua:1092-1096 */
export const defaultCharLevel = numberOption('options:defaultCharLevel', 1, 1, 100);
/* Main.lua:1085-1089 */
export const defaultGemQuality = numberOption('options:defaultGemQuality', 0, 0, 23);
/* Main.lua:1099-1106 */
export const defaultItemAffixQuality = numberOption('options:defaultItemAffixQuality', 0.5, 0, 1);
