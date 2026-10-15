import { addToast } from '$lib/design/toast';

interface UpdateCheck {
  enabled: boolean;
  current?: string;
  version?: string;
  page?: string;
  install: boolean;
  error?: string;
}

const failure = async (r: Response) => ((await r.json().catch(() => null)) as { error?: string } | null)?.error ?? `HTTP ${r.status}`;

export const checkDesktopUpdate = async () => {
  const r = await fetch('/desktop/update/check');
  const c = (await r.json()) as UpdateCheck;
  if (!c.enabled) {
    addToast('Update check failed!\nThis build has no update source; updates come with the next release.');
    return false;
  }
  if (c.error) {
    addToast(`Update check failed!\n${c.error}`);
    return false;
  }
  if (!c.version) {
    addToast('No update available\nYou are running the latest version.');
    return false;
  }
  if (!c.install) {
    addToast(`Update Available\ngo-pob ${c.version} is out (running ${c.current ?? '?'}).${c.page ? `\n${c.page}` : ''}`);
    return false;
  }
  const i = await fetch('/desktop/update/install', { method: 'POST' });
  if (!i.ok) {
    addToast(`Update failed!\n${await failure(i)}`);
    return false;
  }
  addToast('Update Available\nAn update has been downloaded and is ready\nto be applied.');
  return true;
};

export const applyDesktopUpdate = async () => {
  const r = await fetch('/desktop/update/restart', { method: 'POST' });
  if (!r.ok) addToast(`Update failed!\n${await failure(r)}`);
};
