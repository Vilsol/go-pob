import { expect, describe, it } from 'vitest';
import { formatColors } from './display/colors';

describe('formatColors and PoB colour codes', () => {
  /* NotesTab.lua:65-66 */
  it("does not yet understand PoB's ^x hex form", () => {
    expect(formatColors('^x7F7F7Fgrey')).toBe('<span>^x7F7F7Fgrey</span>');
    expect(formatColors('^#7F7F7Fgrey')).toBe("<span><span style='color: #7F7F7F'>grey</span></span>");
  });
});
