export const logError = (err: Error) => {
  console.error(err);
};

export const epochNow = () => performance.timeOrigin + performance.now();

export class Emitter<T> {
  receiver: ((value: T) => void) | undefined;

  emit(value: T) {
    this.receiver?.(value);
  }

  on(receiver: (value: T) => void) {
    this.receiver = receiver;
  }
}
