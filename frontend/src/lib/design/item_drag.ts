import type { DragSource } from './reorder';

export type ItemDrag = { id?: number; db?: string; name?: string; shared?: number };

export const itemSource = (drag: ItemDrag, label = ''): DragSource => ({ data: () => ({ item: drag }), label });

export const draggedItem = (data: unknown): ItemDrag | undefined =>
  data && typeof data === 'object' && 'item' in data ? (data as { item: ItemDrag }).item : undefined;

export const MINION_ITEM_DROP = 'gopob-minion-item-drop';
export type MinionItemDrop = { setId: number; drag: ItemDrag; second: boolean; uses: string[] };
