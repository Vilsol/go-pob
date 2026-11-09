// a response's body read as it arrives, reporting the bytes so far; the total only when the body arrives as the
// Content-Length counts it (no Content-Encoding the browser undoes)
export const readBody = async (res: Response, onbytes: (done: number, total: number | undefined) => void): Promise<ArrayBuffer> => {
  const length = Number(res.headers.get('content-length')) || undefined;
  const total = res.headers.get('content-encoding') ? undefined : length;
  if (!res.body) return res.arrayBuffer();
  const reader = res.body.getReader();
  const chunks: Uint8Array[] = [];
  let done = 0;
  for (;;) {
    const r = await reader.read();
    if (r.done) break;
    chunks.push(r.value);
    done += r.value.byteLength;
    onbytes(done, total);
  }
  const out = new Uint8Array(done);
  let at = 0;
  for (const c of chunks) {
    out.set(c, at);
    at += c.byteLength;
  }
  return out.buffer;
};
