// Build.lua:60-77
// ImportTab.lua:663-666
export const unsavedName = $state({ imported: false });

export const buildName = (path: string | undefined) => path?.split('/').at(-1) || (unsavedName.imported ? 'Imported build' : 'Unnamed build');
