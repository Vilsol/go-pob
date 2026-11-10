// A sprite sheet decoded off the main thread: an <img> decodes on the main thread when it is first drawn or uploaded,
// which blocks the page for tens to hundreds of milliseconds per sheet
export const loadBitmap = async (url: string): Promise<ImageBitmap> => {
  const res = await fetch(url, { mode: 'cors', credentials: 'same-origin' });
  if (!res.ok) throw new Error(`${url}: HTTP ${res.status}`);
  return createImageBitmap(await res.blob(), { premultiplyAlpha: 'premultiply' });
};
