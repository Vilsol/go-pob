import { inWorker } from '$lib/app_log';
import { startJob, type JobHandle, type JobResult, type JobUnit } from './jobs';

// one message a progress source posts about one job. The first message for an id starts it (kind and label then
// required); later ones move it to a phase, report progress, and finally end it. Fields combine: a message with a phase
// and done/total enters the phase and reports its progress.
export type JobMessage = {
  id: string;
  kind?: string;
  label?: string;
  long?: boolean;
  phase?: string;
  done?: number;
  total?: number;
  unit?: JobUnit;
  end?: JobResult;
  error?: string;
};
export type JobPost = { pobJob: JobMessage };

export const isJobPost = (data: unknown): data is JobPost =>
  typeof data === 'object' && data !== null && 'pobJob' in data && typeof (data as JobPost).pobJob?.id === 'string';

const open = new Map<string, JobHandle>();

// applies one message from source (a worker's name, so two workers' ids never meet)
export const applyJobMessage = (m: JobMessage, source = 'page') => {
  const key = `${source}:${m.id}`;
  let job = open.get(key);
  if (!job) {
    if (m.end || !m.kind) return;
    job = startJob(m.kind, m.label ?? m.kind, { long: m.long });
    open.set(key, job);
  }
  if (m.phase) job.phase(m.phase);
  if (m.done !== undefined) job.progress(m.done, m.total, m.unit);
  if (m.end) {
    job.end(m.end, m.error);
    open.delete(key);
  }
};

// a worker that stops ends its jobs with it
export const endSourceJobs = (source: string, error: string) => {
  for (const [key, job] of open)
    if (key.startsWith(`${source}:`)) {
      job.end('error', error);
      open.delete(key);
    }
};

// a worker posts to whoever created it; on the page the message applies at once
export const postJob = (m: JobMessage) => {
  if (inWorker()) (self as unknown as { postMessage: (p: JobPost) => void }).postMessage({ pobJob: m });
  else applyJobMessage(m);
};

// the page side: every job message the worker posts lands in the activity store
export const forwardWorkerJobs = (worker: Pick<Worker, 'addEventListener'>, source: string) =>
  worker.addEventListener('message', (e: MessageEvent<unknown>) => {
    if (isJobPost(e.data)) applyJobMessage(e.data.pobJob, source);
  });
