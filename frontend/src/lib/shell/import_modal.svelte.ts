// the Import / Export dialog, drawn over whatever page is open; every entry point opens it through here
export type ImportTab = 'import' | 'export';

export const importDialog = $state({ open: false, tab: 'import' as ImportTab, focus: false });

// the dialog's code field takes focus as it opens, before the dialog itself could note what had it
let opener: HTMLElement | null = null;
export const returnFocus = () => {
  if (opener?.isConnected) opener.focus();
  opener = null;
};

export const openImport = (tab: ImportTab = 'import') => {
  if (!importDialog.open) opener = typeof document === 'undefined' ? null : (document.activeElement as HTMLElement | null);
  importDialog.tab = tab;
  importDialog.focus = tab === 'import';
  importDialog.open = true;
};

// a comparison the dialog loaded while the Compare page is already open; goto to the same page does not remount it
export const compareLoaded = $state({ seq: 0 });
