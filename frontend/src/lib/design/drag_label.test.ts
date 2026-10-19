// @vitest-environment jsdom
// Main.lua:408-417

import { describe, expect, it } from 'vitest';
import { DRAG_LABEL_Y, dragLabel } from './drag_label';

describe('dragLabel', () => {
  it("is the row's text, held with the cursor at its left edge", () => {
    const el = dragLabel('Kaom’s Heart');
    expect([el.textContent, el.style.position, el.style.pointerEvents, DRAG_LABEL_Y]).toEqual(['Kaom’s Heart', 'fixed', 'none', 8]);
  });
});
