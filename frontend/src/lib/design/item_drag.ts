import { dragLabel } from './drag_label';

export type ItemDrag = { id?: number; db?: string; name?: string; shared?: number };

export const ITEM_DRAG_TYPE = 'application/x-gopob-item';

export const startItemDrag = (e: DragEvent, drag: ItemDrag, label = '') => {
  e.dataTransfer?.setData(ITEM_DRAG_TYPE, JSON.stringify(drag));
  if (e.dataTransfer) e.dataTransfer.effectAllowed = 'copyMove';
  dragLabel(e, label);
};

export const acceptItemDrag = (e: DragEvent) => {
  if (!e.dataTransfer?.types.includes(ITEM_DRAG_TYPE)) return false;
  e.preventDefault();
  return true;
};

export const readItemDrag = (e: DragEvent): ItemDrag | undefined => {
  const raw = e.dataTransfer?.getData(ITEM_DRAG_TYPE);
  if (!raw) return undefined;
  e.preventDefault();
  try {
    return JSON.parse(raw) as ItemDrag;
  } catch {
    return undefined;
  }
};

export const MINION_ITEM_DROP = 'gopob-minion-item-drop';
export type MinionItemDrop = { setId: number; drag: ItemDrag; second: boolean; uses: string[] };
