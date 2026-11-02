// the Go runtime shim a worker instantiates its wasm with: standard Go's loads with the worker, TinyGo's only when
// chosen, as a file of its own
import '../../wasm_exec.js';
import tinyShimUrl from '../../wasm_exec_tinygo.js?url';
import { ENGINE_CRASH, type EngineCrash, type WasmEngine } from './wasm_file';

const shim = globalThis as { Go?: unknown };
const goShim = shim.Go;
let tinyShim: unknown;
let active: WasmEngine = 'go';

export const useEngine = async (engine: WasmEngine) => {
  if (engine === 'tinygo' && tinyShim === undefined) {
    // by URL: a built worker is one script, which cannot split off a chunk
    await import(/* @vite-ignore */ tinyShimUrl);
    tinyShim = shim.Go;
  }
  shim.Go = engine === 'tinygo' ? tinyShim : goShim;
  active = engine;
};

export const activeEngine = () => active;

// a trap inside the wasm (a TinyGo panic, a stack overflow, running out of memory) leaves the module unusable; the
// worker tells the page, which switches back to standard Go
export const watchTraps = (instance: WebAssembly.Instance): WebAssembly.Instance => {
  const exports: Record<string, unknown> = {};
  let reported = false;
  for (const [name, value] of Object.entries(instance.exports)) {
    exports[name] =
      typeof value !== 'function'
        ? value
        : (...args: unknown[]) => {
            try {
              return (value as (...a: unknown[]) => unknown)(...args);
            } catch (e) {
              if (e instanceof WebAssembly.RuntimeError && !reported) {
                reported = true;
                postMessage({ type: ENGINE_CRASH, engine: active, message: e.message } satisfies EngineCrash);
              }
              throw e;
            }
          };
  }
  return { exports } as WebAssembly.Instance;
};
