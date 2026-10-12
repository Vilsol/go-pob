export type TreeView = { offsetX: number; offsetY: number; scaling: number; width: number; height: number };

export type TreeFrame = {
  view: TreeView;
  ms: number;
  at: number;
  hovered: number | undefined;
  hoverPath: number;
  active: number;
  className: string | undefined;
  version?: string;
  renderer?: 'webgl' | 'canvas';
  complete?: boolean;
};

export type TreeProbe = {
  frame(f: TreeFrame): void;
  drawList?: () => { complete: boolean; items: unknown[] };
  selfCheck?: () => string[];
  renderer?: 'webgl' | 'canvas';
};

export const treeProbe = (): TreeProbe | undefined => (globalThis as { __treeProbe?: TreeProbe }).__treeProbe;
