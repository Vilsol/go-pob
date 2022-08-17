export type DeepPromise<T> = T extends Array<infer U>
  ? Array<DeepPromise<U>>
  : T extends ReadonlyArray<infer U>
  ? ReadonlyArray<DeepPromise<U>>
  : T extends Record<string, unknown>
  ? { [K in keyof T]?: DeepPromise<T[K]> }
  : Promise<T>;
