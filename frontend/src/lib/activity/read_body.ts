// a response's body read as it arrives, reporting the bytes so far; the total only when the body arrives as the
// Content-Length counts it (no Content-Encoding the browser undoes)
export const readBody = async (res: Response, onbytes: (done: number, total: number | undefined) => void): Promise<ArrayBuffer> => {
  const length = Number(res.headers.get('content-length')) || undefined;
  const total = res.headers.get('content-encoding') ? undefined : length;
  if (!res.body) return res.arrayBuffer();
  const reader = res.body.getReader();
  // with the total known each chunk lands in place as it arrives, so no one task copies the whole body at the end
  let into = total ? new Uint8Array(total) : undefined;
  const chunks: Uint8Array[] = [];
  let done = 0;
  for (;;) {
    const r = await reader.read();
    if (r.done) break;
    if (into && done + r.value.byteLength <= into.byteLength) into.set(r.value, done);
    else {
      if (into) chunks.push(into.subarray(0, done));
      into = undefined;
      chunks.push(r.value);
    }
    done += r.value.byteLength;
    onbytes(done, total);
  }
  if (into) return done === into.byteLength ? into.buffer : into.buffer.slice(0, done);
  const out = new Uint8Array(done);
  let at = 0;
  for (const c of chunks) {
    out.set(c, at);
    at += c.byteLength;
  }
  return out.buffer;
};

// what a wasm download reports to the page while a worker fetches it: the boot job's phase, its bytes so far, and the
// URL whose bytes the worker kept
export type WasmReport = {
  phase: (name: string) => void;
  bytes: (done: number, total: number | undefined) => void;
  fetched: (url: string) => void;
};

// a wasm module's bytes: a .gz the host serves as stored is decompressed here, and a file still encoded some other way
// (a .br the host did not decode) is fetched again from fallback
export const fetchWasm = async (url: string, fallback: string, what: string, report: WasmReport): Promise<ArrayBuffer> => {
  // a .gz arrives as it is stored and is decompressed here, so its bytes are the compressed ones
  report.phase(url.endsWith('.gz') ? 'Downloading the engine (compressed)' : 'Downloading the engine');
  const res = await fetch(url).catch((e: unknown) => {
    throw new Error(`Failed to load ${what}: ${e instanceof Error ? e.message : String(e)} ${url}`);
  });
  if (!res.ok) throw new Error(`Failed to load ${what}: ${res.status} ${url}`);
  const data = await readBody(res, report.bytes);
  performance.mark('pob:wasm-fetched');
  const head = new Uint8Array(data, 0, Math.min(4, data.byteLength));
  if (head[0] === 0x00 && head[1] === 0x61 && head[2] === 0x73 && head[3] === 0x6d) {
    report.fetched(url);
    return data;
  }
  if (head[0] === 0x1f && head[1] === 0x8b) {
    report.fetched(url);
    report.phase('Decompressing the engine');
    return new Response(new Response(data).body!.pipeThrough(new DecompressionStream('gzip'))).arrayBuffer();
  }
  if (url !== fallback) {
    console.warn(`${url} arrived still encoded; falling back to ${fallback}`);
    return fetchWasm(fallback, fallback, what, report);
  }
  throw new Error(`Failed to load ${what}: not a wasm module`);
};
