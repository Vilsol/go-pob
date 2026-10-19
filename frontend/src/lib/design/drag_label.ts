// Main.lua:408-417, ListControl.lua:314
export const DRAG_LABEL_Y = 8;

export const dragLabel = (text: string) => {
  const el = document.createElement('div');
  el.textContent = text;
  el.dataset.dragLabel = '';
  el.setAttribute('aria-hidden', 'true');
  Object.assign(el.style, {
    position: 'fixed',
    top: '0',
    left: '0',
    padding: '0 1px',
    font: '16px sans-serif',
    lineHeight: '18px',
    color: '#fff',
    background: 'rgba(38, 38, 38, 0.75)',
    whiteSpace: 'nowrap',
    pointerEvents: 'none',
    zIndex: 'var(--z-index-drag)'
  });
  return el;
};
