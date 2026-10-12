import { vi } from 'vitest';

vi.mock('@zenfs/core', () => {
  const unavailable = (name: string) => () => {
    throw new Error(
      `ZenFS is stubbed in component tests and \`fs.${name}\` was called. Storage is deliberately ` +
        `out of scope here; wire a real in-memory backend if a test needs it.`
    );
  };

  return {
    configure: () => Promise.resolve(undefined),
    fs: new Proxy({}, { get: (_t, prop) => unavailable(String(prop)) })
  };
});

vi.mock('@zenfs/dom', () => ({ IndexedDB: { name: 'IndexedDB-stub', create: () => Promise.resolve({}) } }));
