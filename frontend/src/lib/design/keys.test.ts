// @vitest-environment jsdom
import { describe, expect, it } from 'vitest';
import { isTypingTarget, rovingIndex } from './keys';

const key = (k: string) => new KeyboardEvent('keydown', { key: k });

describe('isTypingTarget', () => {
  it('text fields, selects and contenteditable are typing targets', () => {
    for (const tag of ['input', 'textarea', 'select']) expect(isTypingTarget(document.createElement(tag))).toBe(true);
    const div = document.createElement('div');
    div.contentEditable = 'true';
    // jsdom does not compute isContentEditable
    Object.defineProperty(div, 'isContentEditable', { value: true });
    expect(isTypingTarget(div)).toBe(true);
  });

  it('buttons, the window and null are not', () => {
    expect(isTypingTarget(document.createElement('button'))).toBe(false);
    expect(isTypingTarget(window)).toBe(false);
    expect(isTypingTarget(null)).toBe(false);
  });

  it('focus inside a menu counts only with widgets', () => {
    const menu = document.createElement('div');
    menu.setAttribute('role', 'menu');
    const item = menu.appendChild(document.createElement('button'));
    expect(isTypingTarget(item)).toBe(false);
    expect(isTypingTarget(item, { widgets: true })).toBe(true);
  });
});

describe('rovingIndex', () => {
  it('moves and wraps horizontally', () => {
    expect(rovingIndex(key('ArrowRight'), 0, 3)).toBe(1);
    expect(rovingIndex(key('ArrowRight'), 2, 3)).toBe(0);
    expect(rovingIndex(key('ArrowLeft'), 0, 3)).toBe(2);
    expect(rovingIndex(key('Home'), 2, 3)).toBe(0);
    expect(rovingIndex(key('End'), 0, 3)).toBe(2);
    expect(rovingIndex(key('ArrowDown'), 0, 3)).toBeUndefined();
    expect(rovingIndex(key('a'), 0, 3)).toBeUndefined();
  });

  it('vertical uses up and down', () => {
    expect(rovingIndex(key('ArrowDown'), 0, 3, { orientation: 'vertical' })).toBe(1);
    expect(rovingIndex(key('ArrowUp'), 0, 3, { orientation: 'vertical' })).toBe(2);
    expect(rovingIndex(key('ArrowRight'), 0, 3, { orientation: 'vertical' })).toBeUndefined();
  });

  it('clamps without wrap', () => {
    expect(rovingIndex(key('ArrowRight'), 2, 3, { wrap: false })).toBe(2);
    expect(rovingIndex(key('ArrowLeft'), 0, 3, { wrap: false })).toBe(0);
  });

  it('enters from no focus at the first or last item, and an empty list never moves', () => {
    expect(rovingIndex(key('ArrowDown'), -1, 4, { orientation: 'vertical' })).toBe(0);
    expect(rovingIndex(key('ArrowUp'), -1, 4, { orientation: 'vertical' })).toBe(3);
    expect(rovingIndex(key('ArrowDown'), -1, 0, { orientation: 'vertical' })).toBeUndefined();
    expect(rovingIndex(key('Home'), 0, 0)).toBeUndefined();
  });
});
