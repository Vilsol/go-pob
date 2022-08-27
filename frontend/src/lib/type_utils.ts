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
