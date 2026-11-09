import { addToast, removeToast, toastExists, updateToast, type ToastOptions } from '$lib/design/toast';
import { formatProgress } from './format';
import { activityPanelOpen, jobs, type Job } from './jobs';

const text = (j: Job) => {
  if (j.state === 'error') return `${j.label} failed\n${j.error ?? ''}`;
  const phase = j.phases.at(-1)?.name;
  const count = j.done !== undefined ? formatProgress(j.done, j.total, j.unit) : '';
  return [j.label, [phase, count].filter(Boolean).join(' · ')].filter(Boolean).join('\n');
};

// a long job (boot, import, data download) also shows as a toast while it runs, which goes when it finishes and stays
// up if it fails, so nothing is left over the page after a success; one that fails before it is drawn says so where it
// was started; dismissing the toast, or Details, leaves the job in the
// activity panel alone
export const followLongJobs = () => {
  const toastOf = new Map<string, string>();
  const collapsed = new Set<string>();
  return jobs.subscribe((l) => {
    for (const j of l) {
      if (!j.long || collapsed.has(j.id)) continue;
      const id = toastOf.get(j.id);
      if (id !== undefined && !toastExists(id)) {
        collapsed.add(j.id);
        continue;
      }
      const opts: ToastOptions =
        j.state === 'running'
          ? {
              spinner: true,
              progress: j.total ? (j.done ?? 0) / j.total : undefined,
              action: {
                label: 'Details',
                run: () => {
                  collapsed.add(j.id);
                  const shown = toastOf.get(j.id);
                  if (shown !== undefined) removeToast(shown);
                  activityPanelOpen.set(true);
                }
              }
            }
          : {};
      if (id === undefined) {
        if (j.state === 'running') toastOf.set(j.id, addToast(text(j), opts));
      } else if (j.state === 'running' || j.state === 'error') updateToast(id, text(j), opts);
      else removeToast(id);
      if (j.state !== 'running') collapsed.add(j.id);
    }
  });
};
