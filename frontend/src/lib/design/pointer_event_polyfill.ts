// jsdom has no PointerEvent; tests that drive pointer engines import this first
globalThis.PointerEvent ??= class extends MouseEvent {
  pointerId: number;
  pointerType: string;
  isPrimary: boolean;
  constructor(type: string, init: PointerEventInit = {}) {
    super(type, init);
    this.pointerId = init.pointerId ?? 0;
    this.pointerType = init.pointerType ?? '';
    this.isPrimary = init.isPrimary ?? false;
  }
} as unknown as typeof PointerEvent;
