import { writable } from 'svelte/store';

// a game data load that failed, as its notice says it
export const dataErrorText = (message: string) => `${navigator.onLine ? '' : 'This browser is offline and has not kept a copy of it. '}${message}`;

// why the PoE2 engine's game data did not load; the layout shows it with a retry
export const poe2DataError = writable<string | undefined>();
