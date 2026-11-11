import { get, writable } from 'svelte/store';

// ToastNotification.lua:56-153; spinner, progress, actions, keys and auto-dismiss are go-pob's, for activity toasts
export type ToastAction = { label: string; run: () => void; primary?: boolean };
export type ToastLine = { text: string; head?: boolean; sub?: boolean };
export type ToastOptions = {
  spinner?: boolean;
  // 0..1, or undefined for none
  progress?: number;
  actions?: ToastAction[];
  // removes the toast this many ms after it was last set
  timeout?: number;
  // the toast's id: raising a key that is shown replaces that toast instead of adding a second
  key?: string;
  role?: 'status' | 'alert';
  tone?: 'error';
  // the Dismiss button's label, or false for a notice that stays until its cause goes
  dismiss?: string | false;
  // lines a toggle button shows inside the toast, e.g. an update's changes
  more?: { label: string; name: string; lines: ToastLine[] };
};
export type Toast = { id: string; message: string } & ToastOptions;

export const toasts = writable<Toast[]>([]);
let nextId = 1;
let dismissed = new Set<string>();
const timers = new Map<string, ReturnType<typeof setTimeout>>();

const stopTimer = (id: string) => {
  const t = timers.get(id);
  if (t !== undefined) clearTimeout(t);
  timers.delete(id);
};

export const toastExists = (id: string): boolean => get(toasts).some((t) => t.id === id);

export const removeToast = (id: string): boolean => {
  stopTimer(id);
  if (!toastExists(id)) return false;
  toasts.update((l) => l.filter((t) => t.id !== id));
  return true;
};

const schedule = (id: string, timeout: number | undefined) => {
  stopTimer(id);
  if (timeout !== undefined)
    timers.set(
      id,
      setTimeout(() => {
        timers.delete(id);
        removeToast(id);
      }, timeout)
    );
};

export const updateToast = (id: string, message: string, opts: ToastOptions = {}): boolean => {
  if (!toastExists(id)) return false;
  toasts.update((l) => l.map((t) => (t.id === id ? { id, message, ...opts } : t)));
  schedule(id, opts.timeout);
  return true;
};

export const addToast = (message: string, opts: ToastOptions = {}): string => {
  if (opts.key !== undefined && updateToast(opts.key, message, opts)) return opts.key;
  const id = opts.key ?? `toast_${nextId++}`;
  toasts.update((l) => [...l, { id, message, ...opts }]);
  schedule(id, opts.timeout);
  return id;
};

export const clearToasts = () => {
  for (const t of timers.values()) clearTimeout(t);
  timers.clear();
  toasts.set([]);
};

/* ToastNotification.lua:214-222 */
export const dismissToast = (id: string) => {
  dismissed.add(id);
  removeToast(id);
};

export const wasDismissed = (id: string): boolean => dismissed.has(id);

export const clearDismissed = (id?: string) => {
  if (id) dismissed.delete(id);
  else dismissed = new Set();
};
