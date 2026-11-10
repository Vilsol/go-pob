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
