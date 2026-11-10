// a build code's game, read in the main thread so a PoE2 build need not wait for the PoE1 engine

const BASE64 = /^[A-Za-z0-9+/]*={0,2}$/;

// pob.DecodeDecompress: base64url (or standard base64), then zlib; undefined where it fails, and on input past the
// end of the zlib stream, which it ignores
export const decodeCode = async (code: string): Promise<string | undefined> => {
  const b64 = code
    .replace(/[\r\n]/g, '')
    .replaceAll('-', '+')
    .replaceAll('_', '/');
  if (b64.length % 4 !== 0 || !BASE64.test(b64)) return undefined;
  try {
    const bin = atob(b64);
    // a plain loop: Uint8Array.from with a mapper takes about 25 times as long on a large build's code
    const bytes = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
    return await new Response(new Response(bytes).body!.pipeThrough(new DecompressionStream('deflate'))).text();
  } catch {
    return undefined;
  }
};

const ROOT = /^(?:\s|<\?[\s\S]*?\?>|<!--[\s\S]*?-->|<!DOCTYPE[^>]*>)*<(?:[\w.-]+:)?([A-Za-z_][\w.-]*)/;

// builds.DetectGame's first start element
export const rootElement = (xml: string) => ROOT.exec(xml)?.[1];

// the XML of a code that is a PoE2 build; any other code, or one that does not decode, is undefined
export const poe2Xml = async (code: string) => {
  const xml = await decodeCode(code);
  return xml !== undefined && rootElement(xml) === 'PathOfBuilding2' ? xml : undefined;
};
