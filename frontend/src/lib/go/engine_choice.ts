import { browser } from '$app/environment';
import { writable } from 'svelte/store';
import { followOtherTabs } from '$lib/storage/tabs';
import { ENGINE_CRASH, ENGINE_KEY, type EngineCrash, type WasmEngine } from './wasm_file';

const FALLBACK_KEY = 'options:engineFallback';

const loadEngine = (): WasmEngine => {
  try {
    return browser && localStorage.getItem(ENGINE_KEY) === 'tinygo' ? 'tinygo' : 'go';
  } catch {
    return 'go';
  }
};

// the engine this page booted; Settings' choice takes effect on the reload it starts
export const runningEngine: WasmEngine = loadEngine();

export const wasmEngine = writable<WasmEngine>(runningEngine);
wasmEngine.subscribe((v) => browser && localStorage.setItem(ENGINE_KEY, v));
followOtherTabs(ENGINE_KEY, wasmEngine, loadEngine);

// a TinyGo engine that fails to boot or traps switches the app back to standard Go and reloads; the reloaded page
// says why
export const fallBackToGo = (reason: string) => {
  if (runningEngine !== 'tinygo') return false;
  console.error(`TinyGo engine: ${reason}; switching back to Go`);
  localStorage.setItem(ENGINE_KEY, 'go');
  localStorage.setItem(FALLBACK_KEY, reason);
  location.reload();
  return true;
};

export const takeFallbackNotice = (): string | undefined => {
  if (!browser) return undefined;
  const reason = localStorage.getItem(FALLBACK_KEY);
  if (reason === null) return undefined;
  localStorage.removeItem(FALLBACK_KEY);
  return reason;
};

export const watchEngineCrash = (worker: Worker) =>
  worker.addEventListener('message', (e: MessageEvent<EngineCrash | undefined>) => {
    if (e.data?.type === ENGINE_CRASH) fallBackToGo(`it crashed (${e.data.message})`);
  });
