export type SetListOp =
  | { op: 'new'; title: string }
  | { op: 'copy' | 'rename'; index: number; title: string }
  | { op: 'delete'; index: number }
  | { op: 'move'; from: number; to: number };
