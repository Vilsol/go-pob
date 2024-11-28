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

export const dump = <T>(obj: T): T => {
  if (typeof obj !== 'object' || obj === null) {
    return obj;
  }

  const out = structuredClone(obj);

  for (const prop of Object.getOwnPropertyNames(obj)) {
    // @ts-expect-error TS7053
    // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
    let val = obj[prop];

    if (typeof val === 'object') {
      val = dump(val as object);
    } else if (typeof val === 'function' || typeof val === 'symbol') {
      continue;
    }

    // @ts-expect-error TS7053
    // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
    out[prop] = val;
  }

  return out;
};

export const send = <T>(value: T): Promisify<T> => value as Promisify<T>;
