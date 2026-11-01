/* eslint-disable */

const wrap = (name, fn) => {
  return (...args) => {
    const result = fn.call(undefined, ...args);
    if (globalThis.goInternalError) {
      const error = new Error(globalThis.goInternalError);
      globalThis.goInternalError = undefined;
      throw error;
    }
    return result;
  }
};

let initialized = false;

const pending = (name) => new Proxy({}, {
  get(target, property) {
    if (typeof property === 'symbol' || property === 'then') {
      return undefined;
    }
    if (initialized) {
      throw new Error('crystalline: this ' + name + ' was captured before initializeCrystalline() ran, so it is a stale copy. Read it from the module instead of destructuring it earlier, or move the import after initialisation.');
    }
    throw new Error('crystalline: ' + name + '.' + String(property) + ' was read before initializeCrystalline() ran. Start the Go wasm module, then call initializeCrystalline().');
  }
});

export let config = pending('config');
export let exposition = pending('exposition');
export let storage = pending('storage');

export const initializeCrystalline = () => {
  if (globalThis['go']?.['go-pob2'] === undefined) {
    throw new Error('crystalline: globalThis.go.go-pob2 is not set. Start the Go wasm module before calling initializeCrystalline().');
  }

  const failedImports = globalThis['go']['go-pob2']['__crystalline']?.['importFailures'];
  if (failedImports?.length) {
    throw new Error('crystalline: Go could not reach what it imported: ' + failedImports.join('; '));
  }

  config = {
    InitLogging: wrap('config.InitLogging', globalThis['go']['go-pob2']['config']['InitLogging'])
  };
  exposition = {
    ParseBuild: wrap('exposition.ParseBuild', globalThis['go']['go-pob2']['exposition']['ParseBuild']),
    PassiveTreeJSON: wrap('exposition.PassiveTreeJSON', globalThis['go']['go-pob2']['exposition']['PassiveTreeJSON']),
    Preload: wrap('exposition.Preload', globalThis['go']['go-pob2']['exposition']['Preload']),
    TreeClasses: wrap('exposition.TreeClasses', globalThis['go']['go-pob2']['exposition']['TreeClasses']),
    TreeVersion: globalThis['go']['go-pob2']['exposition']['TreeVersion']
  };
  storage = {
    DirEntryType: globalThis['go']['go-pob2']['storage']['DirEntryType'],
    InitializeStorage: wrap('storage.InitializeStorage', globalThis['go']['go-pob2']['storage']['InitializeStorage'])
  };

  initialized = true;
};

export const boot = async (wasm) => {
  if (globalThis['Go'] === undefined) {
    throw new Error('crystalline: the Go runtime shim is missing. Load wasm_exec.js from your Go toolchain before calling boot().');
  }

  const runtime = new globalThis['Go']();

  let source = wasm;

  if (!(wasm instanceof ArrayBuffer) && !ArrayBuffer.isView(wasm)) {
    const response = await fetch(wasm);

    if (!response.ok) {
      throw new Error('crystalline: could not fetch ' + wasm + ': ' + response.status);
    }

    source = await response.arrayBuffer();
  }

  const { instance } = await WebAssembly.instantiate(source, runtime.importObject);

  // Not awaited: the Go program parks, so this never settles.
  runtime.run(instance);

  initializeCrystalline();

  return { config, exposition, storage };
};