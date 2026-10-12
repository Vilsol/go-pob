import type { ProxiedRemote } from '$lib/type_utils';

type Thunk = () => Promise<unknown>;

const PROMISE_KEYS = new Set(['then', 'catch', 'finally']);

const PROTOCOL_KEYS = new Set(['toJSON', 'toString', 'valueOf', 'constructor', 'inspect', 'nodeType', '$$typeof', 'asymmetricMatch']);

export const readGate: { hold?: (path: string) => Promise<void> | undefined } = {};

const makeRemote = (thunk: Thunk, path: string): never => {
  const target = function () {} as unknown as object;

  let cached: Promise<unknown> | undefined;
  const value = () => (cached ??= thunk());

  return new Proxy(target, {
    get(_t, prop) {
      if (PROMISE_KEYS.has(prop as string)) {
        const p = value();
        return (p[prop as 'then' | 'catch' | 'finally'] as (...a: unknown[]) => unknown).bind(p);
      }

      if (typeof prop === 'symbol' || PROTOCOL_KEYS.has(prop)) {
        return undefined;
      }

      return makeRemote(async () => {
        const v = await value();
        if (v === null || v === undefined) {
          return undefined;
        }
        if (readGate.hold) {
          await readGate.hold(`${path}.${prop}`);
        }
        const child = (v as Record<string, unknown>)[prop];
        return typeof child === 'function' ? (child as (...a: unknown[]) => unknown).bind(v) : child;
      }, `${path}.${prop}`);
    },

    set(_t, prop, incoming: unknown) {
      if (typeof prop === 'symbol') {
        return true;
      }
      void value().then(async (owner) => {
        if (owner === null || owner === undefined) {
          return;
        }
        (owner as Record<string, unknown>)[prop] = await incoming;
      });
      return true;
    },

    apply(_t, _thisArg, args: unknown[]) {
      const p = value().then((f) => {
        if (typeof f !== 'function') {
          throw new TypeError(`remote call on a value that is not a function: ${path}() (got ${typeof f})`);
        }
        return (f as (...a: unknown[]) => unknown)(...args);
      });
      return makeRemote(() => p, `${path}()`);
    }
  }) as never;
};

export const asRemote = <T>(obj: T, label = 'remote'): ProxiedRemote<T> => makeRemote(() => Promise.resolve(obj), label) as ProxiedRemote<T>;
