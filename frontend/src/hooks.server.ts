import { dev } from '$app/environment';
import { assets } from '$app/paths';
import type { Handle } from '@sveltejs/kit';
import { WASM_FILE } from '$lib/go/wasm_file';

const preload = `<link rel="preload" href="${assets}${WASM_FILE}" as="fetch" crossorigin="anonymous" />`;

export const handle: Handle = ({ event, resolve }) =>
  resolve(event, {
    transformPageChunk: ({ html }) => (dev ? html : html.replace('</head>', `${preload}\n</head>`))
  });
