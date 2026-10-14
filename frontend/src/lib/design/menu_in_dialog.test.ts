// @vitest-environment jsdom

import { afterEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte';
import { createRawSnippet } from 'svelte';
import Menu from './Menu.svelte';

afterEach(cleanup);

const trigger = createRawSnippet(() => ({ render: () => '<button>Actions</button>' }));
const items = [{ label: 'Copy' }, { label: 'Delete' }];

describe('Menu', () => {
  it('Escape on the open menu does not reach its dialog', async () => {
    const outer = vi.fn();
    document.body.addEventListener('keydown', outer);
    try {
      render(Menu, { label: 'Actions', items, open: true, trigger });
      await fireEvent.keyDown(screen.getByRole('menu'), { key: 'Escape' });
      expect(screen.queryByRole('menu')).toBeNull();
      expect(outer).not.toHaveBeenCalled();
    } finally {
      document.body.removeEventListener('keydown', outer);
    }
  });

  it('places the open menu fixed to the viewport', () => {
    render(Menu, { label: 'Actions', items, open: true, trigger });
    expect(screen.getByRole('menu').style.position).toBe('fixed');
  });
});
