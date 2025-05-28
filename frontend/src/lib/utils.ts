export const logError = (err: Error) => {
  console.error(err);
  // TODO Push to some viewable error log via UI
};

export class Emitter<T> {
  receiver: ((value: T) => void) | undefined;

  emit(value: T) {
    this.receiver?.(value);
  }

  on(receiver: (value: T) => void) {
    this.receiver = receiver;
  }
}
