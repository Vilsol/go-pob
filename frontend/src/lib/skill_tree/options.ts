import { browser } from '$app/environment';
import { writable, type Writable } from 'svelte/store';

// Main.lua:969-1013

export const hexColour = (v: string) => (/^0x[0-9a-fA-F]{6}$/.test(v) ? `#${v.slice(2)}` : undefined);

const persisted = <T>(key: string, fallback: T, read: (s: string) => T): Writable<T> => {
  let start = fallback;
  try {
    const s = browser ? localStorage.getItem(key) : null;
    if (s !== null) start = read(s);
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

export const colorPositive = persisted('options:colorPositive', '', String);
export const colorNegative = persisted('options:colorNegative', '', String);
export const colorHighlight = persisted('options:colorHighlight', '', String);
export const edgeSearchHighlight = persisted('options:edgeSearchHighlight', true, (s) => s !== 'false');

// Global.lua:79-90
if (browser) {
  const apply = (name: string) => (v: string) => {
    const c = hexColour(v);
    if (c) document.documentElement.style.setProperty(name, c);
    else document.documentElement.style.removeProperty(name);
  };
  colorPositive.subscribe(apply('--color-better'));
  colorNegative.subscribe(apply('--color-worse'));
}
