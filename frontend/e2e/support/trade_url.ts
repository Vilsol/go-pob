import { gunzipSync } from 'node:zlib';

// A trade search URL's path and its query, which the last path segment carries gzipped and Base64URL encoded.
export function tradeUrlQuery(url: string): { base: string; query: unknown } {
  const i = url.lastIndexOf('/');
  return { base: url.slice(0, i), query: JSON.parse(gunzipSync(Buffer.from(url.slice(i + 1), 'base64url')).toString()) as unknown };
}
