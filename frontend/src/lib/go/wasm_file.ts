export const WASM_FALLBACK = '/go-pob.wasm.gz';

export const WASM_FILE =
  import.meta.env.MODE === 'development' ? '/go-pob.wasm' : `/${(import.meta.env.VITE_WASM_FILE as string | undefined) || 'go-pob.wasm.gz'}`;

export const WASM2_FALLBACK = '/go-pob2.wasm.gz';

export const WASM2_FILE =
  import.meta.env.MODE === 'development' ? '/go-pob2.wasm' : `/${(import.meta.env.VITE_WASM2_FILE as string | undefined) || 'go-pob2.wasm.gz'}`;

// the engine Settings offers: standard Go, or TinyGo's smaller build of the same code
export type WasmEngine = 'go' | 'tinygo';

export const ENGINE_KEY = 'options:engine';

// each wasm file above has a TinyGo twin beside it: go-pob.wasm.gz -> go-pob-tiny.wasm.gz
export const engineFile = (path: string, engine: WasmEngine) => (engine === 'tinygo' ? path.replace(/(go-pob2?)\.wasm/, '$1-tiny.wasm') : path);

// what a worker posts the page when its wasm traps
export const ENGINE_CRASH = 'pob:engine-crash';
export type EngineCrash = { type: typeof ENGINE_CRASH; engine: WasmEngine; message: string };
