// Main.lua:408-417, ListControl.lua:314
export const dragLabel = (e: DragEvent, text: string) => {
  const dt = e.dataTransfer;
  if (!dt?.setDragImage || !text) return;
  const el = document.createElement('div');
  el.textContent = text;
  el.dataset.dragLabel = '';
  Object.assign(el.style, {
    position: 'fixed',
    top: '-1000px',
    left: '0',
    padding: '0 1px',
    font: '16px sans-serif',
    lineHeight: '18px',
    color: '#fff',
    background: 'rgba(38, 38, 38, 0.75)',
    whiteSpace: 'nowrap',
    pointerEvents: 'none'
  });
  document.body.append(el);
  dt.setDragImage(el, 0, 8);
  setTimeout(() => el.remove());
};
