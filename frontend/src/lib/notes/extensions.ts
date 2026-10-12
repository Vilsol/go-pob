import { Extension, Mark, mergeAttributes } from '@tiptap/core';
import { TaskItem, TaskList } from '@tiptap/extension-list';
import StarterKit from '@tiptap/starter-kit';

import { COLOR_MARK, codeToCss, isCode } from './pob_text';

export const PobColor = Mark.create({
  name: COLOR_MARK,
  addAttributes() {
    return {
      code: {
        default: null,
        parseHTML: (el) => {
          const c = el.getAttribute('data-pob-color');
          return isCode(c) ? c : null;
        },
        renderHTML: () => ({})
      },
      lead: { default: null, parseHTML: () => null, renderHTML: () => ({}) },
      run: { default: 0, parseHTML: () => 0, renderHTML: () => ({}) }
    };
  },
  parseHTML() {
    return [{ tag: 'span[data-pob-color]' }];
  },
  renderHTML({ mark, HTMLAttributes }) {
    const code = isCode(mark.attrs.code) ? mark.attrs.code : null;
    const css = code ? codeToCss(code) : null;
    return ['span', mergeAttributes(HTMLAttributes, code ? { 'data-pob-color': code } : {}, css ? { style: `color: ${css}` } : {}), 0];
  }
});

const LineTrail = Extension.create({
  name: 'pobLineTrail',
  addGlobalAttributes() {
    return [
      {
        types: ['paragraph', 'heading'],
        attributes: { trail: { default: null, parseHTML: () => null, renderHTML: () => ({}), keepOnSplit: false } }
      }
    ];
  }
});

export const notesExtensions = () => [
  StarterKit.configure({
    trailingNode: false,
    heading: { levels: [1, 2, 3] },
    link: { openOnClick: false, autolink: true, defaultProtocol: 'https' }
  }),
  TaskList,
  TaskItem.configure({ nested: true }),
  PobColor,
  LineTrail
];
