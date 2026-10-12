// @vitest-environment jsdom
import { afterEach, describe, expect, it } from 'vitest';
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte';
import { get } from 'svelte/store';
import Toasts from './Toasts.svelte';
import { addToast, clearDismissed, clearToasts, removeToast, toastExists, toasts, updateToast, wasDismissed } from './toast';

afterEach(() => {
  cleanup();
  clearToasts();
  clearDismissed();
});

// ToastNotification.lua:56-153
describe('toasts', () => {
  it('adds, updates and removes by id', () => {
    const a = addToast('Title\nbody');
    const b = addToast('Other');
    expect(a).not.toBe(b);
    expect(get(toasts).map((t) => t.message)).toEqual(['Title\nbody', 'Other']);
    expect(updateToast(a, 'Changed')).toBe(true);
    expect(get(toasts)[0].message).toBe('Changed');
    expect(removeToast(a)).toBe(true);
    expect(toastExists(a)).toBe(false);
    expect(toastExists(b)).toBe(true);
    expect(updateToast(a, 'x')).toBe(false);
    clearToasts();
    expect(get(toasts)).toEqual([]);
  });

  it('draws each toast with its title and Dismiss, which removes it and is remembered', async () => {
    render(Toasts);
    const id = addToast('Power builder\n10% done');
    const note = await screen.findByRole('status', { name: 'Power builder' });
    expect(note.textContent).toContain('10% done');
    await fireEvent.click(screen.getByRole('button', { name: 'Dismiss' }));
    expect(toastExists(id)).toBe(false);
    expect(wasDismissed(id)).toBe(true);
  });
});
