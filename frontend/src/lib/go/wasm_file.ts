export const WASM_FALLBACK = '/go-pob.wasm.gz';

export const WASM_FILE =
  import.meta.env.MODE === 'development' ? '/go-pob.wasm' : `/${(import.meta.env.VITE_WASM_FILE as string | undefined) || 'go-pob.wasm.gz'}`;
