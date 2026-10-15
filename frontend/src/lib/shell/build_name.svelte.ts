// Build.lua:60-77
// ImportTab.lua:663-666
export const unsavedName = $state<{ imported: boolean; predefined?: string }>({ imported: false });

// Build.lua:89-100
export const opening = { modFlag: false };

export const buildName = (path: string | undefined) => path?.split('/').at(-1) || (unsavedName.imported ? 'Imported build' : 'Unnamed build');

// ImportTab.lua:1925-1931
export const predefinedName = (account: string, character: string) => `${account.replace(/\s+/g, '').replace(/#\d+/g, '')} - ${character}`;

// Build.lua:1400
export const saveAsName = (path: string | undefined) => (!path && unsavedName.predefined) || buildName(path).replace(/[\\/:*?"<>|\p{Cc}]/gu, '-');
