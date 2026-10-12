// @vitest-environment jsdom
// Main.lua:408-417

import { describe, expect, it, vi } from 'vitest';
import { dragLabel } from './drag_label';

describe('dragLabel', () => {
  it("makes the row's text the drag image, the cursor at its left edge", () => {
    const setDragImage = vi.fn();
    dragLabel({ dataTransfer: { setDragImage } } as unknown as DragEvent, 'Kaom’s Heart');
    expect(setDragImage).toHaveBeenCalledTimes(1);
    const [el, x, y] = setDragImage.mock.calls[0] as [HTMLElement, number, number];
    expect([el.textContent, x, y]).toEqual(['Kaom’s Heart', 0, 8]);
    expect(el.isConnected).toBe(true);
  });
});
