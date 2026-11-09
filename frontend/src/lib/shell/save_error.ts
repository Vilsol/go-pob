import { WRITER_SILENT } from '$lib/storage/tabs';

// Build.lua:2069-2072; a save another tab has not written yet says that instead
export const saveErrorText = (path: string, e?: unknown) => {
  const message = e instanceof Error ? e.message : '';
  const silent = message.indexOf(WRITER_SILENT);
  if (silent >= 0) return message.slice(silent);
  return `Couldn't save the build file:\n${path}\nMake sure the save folder exists and is writable.`;
};
