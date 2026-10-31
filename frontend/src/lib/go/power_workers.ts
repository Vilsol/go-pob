import { browser } from '$app/environment';
import { writable } from 'svelte/store';
import { followOtherTabs } from '$lib/storage/tabs';

export const POWER_WORKER_CHOICES = [1, 2, 3, 4];

const load = () => {
  const stored = browser ? Number(localStorage.getItem('options:powerWorkers')) : 1;
  return POWER_WORKER_CHOICES.includes(stored) ? stored : 1;
};
export const powerWorkers = writable(load());
powerWorkers.subscribe((v) => browser && localStorage.setItem('options:powerWorkers', v.toString(10)));
followOtherTabs('options:powerWorkers', powerWorkers, load);
