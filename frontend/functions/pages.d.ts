interface PagesContext<P extends string = never> {
  request: Request;
  params: Record<P, string>;
  waitUntil(promise: Promise<unknown>): void;
}

interface CacheStorage {
  readonly default: Cache;
}
