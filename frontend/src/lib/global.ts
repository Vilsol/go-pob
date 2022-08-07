import { writable } from 'svelte/store';
import type { Outputs } from './custom_types';

export const outputs = writable<Outputs | undefined>();
