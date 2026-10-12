// @vitest-environment jsdom
// Build.lua:1325-1350
import { describe, expect, it, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import GameVersionDialog from './GameVersionDialog.svelte';

afterEach(cleanup);

if (!HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = function () {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function () {
    this.open = false;
    this.dispatchEvent(new Event('close'));
  };
}

describe('GameVersionDialog', () => {
  it('converts on its button, once, without cancelling', async () => {
    const onconvert = vi.fn();
    const oncancel = vi.fn();
    const { getByRole, getByText } = render(GameVersionDialog, { open: true, version: '3.29', onconvert, oncancel });
    expect(getByText(/not supported by us/)).toBeTruthy();
    expect(getByText(/may have side effects/)).toBeTruthy();
    await fireEvent.click(getByRole('button', { name: 'Convert to 3.29' }));
    expect(onconvert).toHaveBeenCalledTimes(1);
    expect(oncancel).not.toHaveBeenCalled();
  });

  it('Cancel and closing the dialog both cancel', async () => {
    const onconvert = vi.fn();
    const oncancel = vi.fn();
    const { getByRole } = render(GameVersionDialog, { open: true, version: '3.29', onconvert, oncancel });
    await fireEvent.click(getByRole('button', { name: 'Cancel' }));
    expect(oncancel).toHaveBeenCalledTimes(1);
    cleanup();
    const again = vi.fn();
    const view = render(GameVersionDialog, { open: true, version: '3.29', onconvert, oncancel: again });
    view.container.ownerDocument.querySelector('dialog')!.close();
    expect(again).toHaveBeenCalledTimes(1);
    expect(onconvert).not.toHaveBeenCalled();
  });
});
