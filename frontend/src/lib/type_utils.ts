/* eslint-disable */
export type DeepPromise<T> = T extends Array<infer U>
  ? Array<DeepPromise<U>> & Promise<Array<DeepPromise<U>>>
  : T extends ReadonlyArray<infer U>
  ? ReadonlyArray<DeepPromise<U>> & Promise<ReadonlyArray<DeepPromise<U>>>
  : T extends Record<never, never>
  ? { [K in keyof T]?: T[K] extends Function ? T[K] : DeepPromise<T[K]> } & Promise<{
      [K in keyof T]?: T[K] extends Function ? T[K] : DeepPromise<T[K]>;
    }>
  : Promise<T>;

export const dump = (obj: any) => {
  if (typeof obj !== 'object' || obj === null) {
    return obj;
  }

  const out = {} as any;
  for (const name of Object.getOwnPropertyNames(obj)) {
    let val = (obj as any)[name];
    if (typeof val === 'object') {
      val = dump(val);
    } else if (typeof val === 'function' || typeof val === 'symbol') {
      continue
    }
    out[name] = val;
  }
  return out;
};
