import { startJob, type JobHandle } from '$lib/activity/jobs';

// the images the tree has asked for and how many have arrived; one asked for while the job runs raises its total
let job: JobHandle | undefined;
let asked = 0;
let arrived = 0;
let failed = 0;

const settle = () => {
  arrived++;
  if (arrived < asked) return void job?.progress(arrived, asked);
  job?.progress(arrived, asked).end(failed ? 'error' : 'ok', failed ? `${failed} of ${asked} tree images did not load` : undefined);
  job = undefined;
  asked = arrived = failed = 0;
};

// A sprite sheet decoded off the main thread: an <img> decodes on the main thread when it is first drawn or uploaded,
// which blocks the page for tens to hundreds of milliseconds per sheet
export const loadBitmap = async (url: string): Promise<ImageBitmap> => {
  job ??= startJob('tree-images', "Loading the tree's images").phase('Sprite sheets and art');
  job.progress(arrived, ++asked);
  try {
    const res = await fetch(url, { mode: 'cors', credentials: 'same-origin' });
    if (!res.ok) throw new Error(`${url}: HTTP ${res.status}`);
    return await createImageBitmap(await res.blob(), { premultiplyAlpha: 'premultiply' });
  } catch (e) {
    failed++;
    throw e;
  } finally {
    settle();
  }
};
