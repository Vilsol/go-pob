export const WASM_FALLBACK = '/go-pob.wasm.gz';

export const WASM_FILE =
  import.meta.env.MODE === 'development' ? '/go-pob.wasm' : `/${(import.meta.env.VITE_WASM_FILE as string | undefined) || 'go-pob.wasm.gz'}`;

export const WASM2_FALLBACK = '/go-pob2.wasm.gz';

export const WASM2_FILE =
  import.meta.env.MODE === 'development' ? '/go-pob2.wasm' : `/${(import.meta.env.VITE_WASM2_FILE as string | undefined) || 'go-pob2.wasm.gz'}`;
