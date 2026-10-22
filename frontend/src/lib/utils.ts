import { inWorker, log } from './app_log';

// a worker forwards its console errors and warnings itself (console_hook)
export const logError = (err: Error) => {
  console.error(err);
  if (!inWorker()) log('error', 'app', err);
};

export const logInfo = (src: string, ...parts: unknown[]) => log('info', src, ...parts);

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
