import { writable } from 'svelte/store';
import type { Outputs } from './custom_types';
import type { pob } from './types';
import type { DeepPromise } from './type_utils';

export const outputs = writable<Outputs | undefined>();

export const currentBuild = writable<DeepPromise<pob.PathOfBuilding> | undefined>();
