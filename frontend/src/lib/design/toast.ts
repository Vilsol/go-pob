import { get, writable } from 'svelte/store';

// ToastNotification.lua:56-153
export type Toast = { id: string; message: string };

export const toasts = writable<Toast[]>([]);
let nextId = 1;
let dismissed = new Set<string>();

export const addToast = (message: string): string => {
  const id = `toast_${nextId++}`;
  toasts.update((l) => [...l, { id, message }]);
  return id;
};

export const toastExists = (id: string): boolean => get(toasts).some((t) => t.id === id);

export const updateToast = (id: string, message: string): boolean => {
  if (!toastExists(id)) return false;
  toasts.update((l) => l.map((t) => (t.id === id ? { id, message } : t)));
  return true;
};

export const removeToast = (id: string): boolean => {
  if (!toastExists(id)) return false;
  toasts.update((l) => l.filter((t) => t.id !== id));
  return true;
};

export const clearToasts = () => toasts.set([]);

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
