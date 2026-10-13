import { browser } from '$app/environment';
import { writable, type Writable } from 'svelte/store';

export const THEMES = ['Dark', 'Light', 'High contrast', 'Follow system'] as const;
export type Theme = (typeof THEMES)[number];

const persisted = <T>(key: string, fallback: T, read: (s: string) => T | undefined): Writable<T> => {
  let start = fallback;
  try {
    const s = browser ? localStorage.getItem(key) : null;
    if (s !== null) start = read(s) ?? fallback;
  } catch {
    // empty
  }
  const store = writable(start);
  store.subscribe((v) => {
    try {
      if (browser) localStorage.setItem(key, String(v));
    } catch {
      // empty
    }
  });
  return store;
};

export const fontScaling = persisted('options:fontScaling', 12, (s) => parseFloat(s) || undefined);
export const theme = persisted<Theme>('options:theme', 'Dark', (s) => THEMES.find((t) => t === s));
/* Main.lua:1027-1029, PassiveTreeView.lua:1251 */
export const showAnimations = persisted('options:showAnimations', true, (s) => s !== 'false');

export const themeAttr = (t: Theme, systemLight: boolean) =>
  t === 'Light' || (t === 'Follow system' && systemLight) ? 'light' : t === 'High contrast' ? 'contrast' : 'dark';

export const rootFontSize = (pt: number) => `${(16 * Math.min(Math.max(pt || 12, 6), 18)) / 12}px`;

if (browser) {
  const root = document.documentElement;
  const light = window.matchMedia?.('(prefers-color-scheme: light)');
  let chosen: Theme = 'Dark';
  const applyTheme = () => {
    root.dataset.theme = themeAttr(chosen, !!light?.matches);
    root.style.colorScheme = root.dataset.theme === 'light' ? 'light' : 'dark';
  };
  theme.subscribe((t) => {
    chosen = t;
    applyTheme();
  });
  light?.addEventListener?.('change', applyTheme);
  showAnimations.subscribe((on) => {
    if (on) delete root.dataset.motion;
    else root.dataset.motion = 'off';
  });
  fontScaling.subscribe((pt) => (root.style.fontSize = rootFontSize(pt)));
}

export const ITEMS_LAYOUTS = ['board', 'list'] as const;
export type ItemsLayout = (typeof ITEMS_LAYOUTS)[number];
export const itemsLayout = persisted<ItemsLayout>('options:itemsLayout', 'board', (s) => ITEMS_LAYOUTS.find((l) => l === s));
