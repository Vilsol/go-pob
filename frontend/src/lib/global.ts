import { writable } from 'svelte/store';
import type { Outputs } from './custom_types';
import type { pob } from './types';
import type { DeepPromise } from './type_utils';
import { syncWrap } from './go/worker';
import { browser } from '$app/env';

export const outputs = writable<Outputs | undefined>();

export const currentBuild = writable<DeepPromise<pob.PathOfBuilding> | undefined>();

let uiTickLock = false;
let uiTickAfter = false;
let uiTickAfterSource = '';
export const UITick = (source: string) => {
  if (!syncWrap) {
    return;
  }

  if (uiTickLock) {
    uiTickAfter = true;
    uiTickAfterSource = source;
    return;
  }
  uiTickLock = true;

  syncWrap
    .Tick(source)
    .catch((err) => {
      console.error(err);
    })
    .then(() => {
      uiTickLock = false;
      if (uiTickAfter) {
        uiTickAfter = false;
        UITick(uiTickAfterSource);
      }
    });
};

// Options

export const fontScaling = writable(parseFloat(browser ? localStorage.getItem('options:fontScaling') || '12' : '12'));
fontScaling.subscribe((v) => browser && localStorage.setItem('options:fontScaling', v.toString(10)));
