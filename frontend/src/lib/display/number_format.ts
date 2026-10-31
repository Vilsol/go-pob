import { writable } from 'svelte/store';
import { followOtherTabs } from '$lib/storage/tabs';

// Main.lua:1056-1079, Main.lua:483-500
export type NumberFormat = {
  showThousandsSeparators: boolean;
  thousandsSeparator: string;
  decimalSeparator: string;
  useCompactValues: boolean;
};

/* Main.lua:100-110 */
export const DEFAULT_NUMBER_FORMAT: NumberFormat = { showThousandsSeparators: true, thousandsSeparator: ',', decimalSeparator: '.', useCompactValues: false };

export const NUMBER_FORMAT_KEY = 'options:numberFormat';
const KEY = NUMBER_FORMAT_KEY;

const load = (): NumberFormat => {
  try {
    const raw = globalThis.localStorage?.getItem(KEY);
    if (raw) return { ...DEFAULT_NUMBER_FORMAT, ...(JSON.parse(raw) as Partial<NumberFormat>) };
  } catch {
    // empty
  }
  return { ...DEFAULT_NUMBER_FORMAT };
};

let current: NumberFormat = load();
export const numberFormat = writable<NumberFormat>(current);
numberFormat.subscribe((v) => {
  current = v;
  try {
    globalThis.localStorage?.setItem(KEY, JSON.stringify(v));
  } catch {
    // empty
  }
});
followOtherTabs(KEY, numberFormat, load);

export const currentNumberFormat = () => current;
