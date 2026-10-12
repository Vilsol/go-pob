import { browser } from '$app/environment';
import { tick } from 'svelte';
import { derived, writable } from 'svelte/store';
import type { TickWarnings } from './custom_types';
import { buildSidebar } from './display/sidebar';
import { outputs } from './global';
import { syncWrap } from './go/worker';
import { epochNow, logError } from './utils';

export const tickWarnings = writable<TickWarnings | undefined>();

let latest = 0;
function fetchWarnings(id: number) {
  if (!syncWrap) {
    return;
  }
  syncWrap
    .GetSidebarWarnings()
    .then((w) => {
      if (id !== latest || !w) {
        return;
      }
      tickWarnings.set(w);
      void tick().then(() => {
        document.documentElement.dataset.warningsMs = w.ms.toFixed(1);
        document.documentElement.dataset.warningsAt = String(epochNow());
        document.documentElement.dataset.warningsSeq = String(w.seq);
      });
    })
    .catch(logError);
}

if (browser) {
  outputs.subscribe((out) => {
    const id = ++latest;
    if (out) {
      fetchWarnings(id);
    }
  });
}

export const sidebar = derived([outputs, tickWarnings], ([$outputs, $tickWarnings]) => ($outputs ? buildSidebar($outputs, $tickWarnings) : undefined));
