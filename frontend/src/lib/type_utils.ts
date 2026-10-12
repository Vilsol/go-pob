import type { ProxyMarked, Remote } from 'comlink';

type Promisify<T> = T extends Promise<unknown> ? T : Promise<T>;

// eslint-disable-next-line no-use-before-define
type ProxiedRemoteProp<T> = T extends object | ProxyMarked ? ProxiedRemote<T> : Promisify<T>;

type ProxiedRemoteObject<T> = Remote<T> & {
  [P in keyof T]: ProxiedRemoteProp<T[P]>;
};

type ProxiedRemoteArray<T extends ArrayLike<unknown>> = Remote<Omit<T, 'length'>> & {
  [P in keyof Omit<T, 'length'>]: ProxiedRemoteProp<T[P]>;
} & {
  length: Promise<number>;
};

export type ProxiedRemote<T> = T extends ArrayLike<unknown> ? ProxiedRemoteArray<T> : ProxiedRemoteObject<T>;

export const releaseWrapper = (obj: unknown) => {
  const release = (obj as { release?: unknown } | null | undefined)?.release;
  if (typeof release === 'function') {
    (release as () => void).call(obj);
  }
};

export const dump = <T>(obj: T, release = false): T => {
  if (typeof obj !== 'object' || obj === null) {
    return obj;
  }

  let out: T;
  try {
    out = release ? ((Array.isArray(obj) ? [] : {}) as T) : structuredClone(obj);
  } catch (error) {
    if (!(error instanceof DOMException) || error.name !== 'DataCloneError') {
      console.error(error);
    }

    out = {} as T;
  }

  for (const prop of Object.getOwnPropertyNames(obj)) {
    if (prop === '__crystallineHandle') continue;
    // @ts-expect-error TS7053
    // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
    let val = obj[prop];

    if (typeof val === 'object') {
      val = dump(val as object, release);
    } else if (typeof val === 'function' || typeof val === 'symbol') {
      continue;
    }

    // @ts-expect-error TS7053
    // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
    out[prop] = val;
  }

  if (release) {
    releaseWrapper(obj);
  }

  return out;
};

export const send = <T>(value: T): Promisify<T> => value as Promisify<T>;
