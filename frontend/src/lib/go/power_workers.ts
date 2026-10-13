import { browser } from '$app/environment';
import { writable } from 'svelte/store';

export const POWER_WORKER_CHOICES = [1, 2, 3, 4];

const stored = browser ? Number(localStorage.getItem('options:powerWorkers')) : 1;
export const powerWorkers = writable(POWER_WORKER_CHOICES.includes(stored) ? stored : 1);
powerWorkers.subscribe((v) => browser && localStorage.setItem('options:powerWorkers', v.toString(10)));
