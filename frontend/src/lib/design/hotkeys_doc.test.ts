import { describe, expect, it } from 'vitest';
import { readFileSync, readdirSync } from 'node:fs';
import { join, relative, resolve } from 'node:path';
import { parseHelp } from './help';

const ROOT = resolve(__dirname, '../../../..');
const FRONTEND = join(ROOT, 'frontend');

const walk = (dir: string): string[] =>
  readdirSync(dir, { withFileTypes: true }).flatMap((d) => (d.isDirectory() ? walk(join(dir, d.name)) : [join(dir, d.name)]));
const SOURCES = [
  ...walk(join(FRONTEND, 'src')).filter((f) => /\.(svelte|ts)$/.test(f) && !/\.test\.ts$|\.stories\.svelte$/.test(f) && !f.includes('/lib/types/')),
  join(ROOT, 'desktop/main.go')
];

const indent = (l: string) => l.length - l.trimStart().length;
// the lines after `start` until the code dedents out of its block
const block = (lines: string[], start: number, inside: (l: string) => boolean) => {
  const out: string[] = [];
  for (let i = start + 1; i < lines.length && (lines[i].trim() === '' || inside(lines[i])); i++) out.push(lines[i]);
  return out.join('\n');
};
const all = (re: RegExp, text: string) => [...text.matchAll(re)].map((m) => m[1]);
const quoted = (s: string) => all(/'([^']*)'/g, s);

/** Every key literal a keyboard handler in `src` compares against. */
const scan = (src: string): string[] => {
  const code = src.replace(/\/\*[\s\S]*?\*\//g, '').replace(/(^|[^:'"`])\/\/.*$/gm, '$1');
  const lines = code.split('\n');
  const keys = [
    ...all(/\b(?:e|ev|event)\.key(?:\.toLowerCase\(\))?\s*[!=]==?\s*'([^']*)'/g, code),
    ...(/\b(?:e|ev)\.key\s*[!=]==?\s*WIKI_KEY/.test(code) ? ['F1'] : []),
    ...[...code.matchAll(/\[([^\]]*)\]\.includes\((?:e|ev)\.key\)/g)].flatMap((m) => quoted(m[1])),
    ...[...code.matchAll(/\{([^{}]*)\}\[(?:e|ev)\.key\]/g)].flatMap((m) => all(/(\w+)\s*:/g, m[1])),
    ...all(/SetAccelerator\("([^"]+)"\)/g, code)
  ];
  lines.forEach((l, i) => {
    if (/const k = (?:e|ev)\.key\.toLowerCase\(\);/.test(l))
      keys.push(
        ...all(
          /(?<![\w.])k\s*[!=]==?\s*'([^']*)'/g,
          block(lines, i - 1, (x) => indent(x) >= indent(l))
        )
      );
    if (/switch \((?:e|ev)\.key\) \{/.test(l))
      keys.push(
        ...all(
          /case '([^']*)':/g,
          block(lines, i, (x) => indent(x) > indent(l))
        )
      );
  });
  return [...new Set(keys)];
};

type Row = {
  /* path from the repository root */
  file: string;
  /* key literals the scan finds in the file */
  keys?: string[];
  /* where the scan cannot see the handler: text the file must contain */
  evidence?: RegExp;
  /* the "<sub-heading>: <key>" lines under Hotkeys this handler documents */
  help?: string[];
  /* why the handler is left out of the help */
  why?: string;
};

const GENERIC = 'standard widget keys, covered by the generic "Arrow keys, Home, End", "Enter" and "Esc" lines';
const A11Y = 'keyboard access to a pointer control; not a shortcut';

const ALL = 'Everywhere in an open build';
const TREE = 'Passive tree';
const NOTES = 'Notes (while typing)';
const LIST = 'Build list';
const FIELDS = 'Lists, menus and fields';
const D = 'frontend/src/lib/design/';
const SH = 'frontend/src/lib/shell/';

const INVENTORY: Row[] = [
  { file: `${SH}AppTopBar.svelte`, keys: ['F1'], help: [`${ALL}: F1`] },
  { file: `${D}TopBar.svelte`, keys: ['s', 'i', 'z', 'y'], help: [`${ALL}: Ctrl+S`, `${ALL}: Ctrl+I`, `${ALL}: Ctrl+Z`, `${ALL}: Ctrl+Y or Ctrl+Shift+Z`] },
  { file: `${D}TopBar.svelte`, keys: ['Escape'], help: [`${ALL}: Esc`] },
  { file: 'frontend/src/routes/+layout.svelte', keys: ['u'], help: [`${ALL}: Ctrl+U`] },
  { file: `${D}NavTabs.svelte`, evidence: /\(\?:Digit\|Numpad\)\(\[0-9\]\)/, help: [`${ALL}: 1 - 8`] },
  { file: `${D}Dialog.svelte`, keys: ['Escape'], help: [`${ALL}: Esc`] },
  { file: `${D}Dialog.svelte`, keys: ['Tab'], why: 'keeps focus inside the open dialog' },
  { file: `${D}Popover.svelte`, keys: ['Escape'], help: [`${ALL}: Esc`] },
  { file: `${D}Menu.svelte`, keys: ['Escape', 'ArrowUp', 'ArrowDown'], help: [`${ALL}: Esc`, `${FIELDS}: Arrow keys, Home, End`] },
  { file: `${D}Menu.svelte`, keys: ['Tab'], why: 'closes the menu as focus leaves it' },
  { file: `${D}keys.ts`, keys: ['Home', 'End'], help: [`${FIELDS}: Arrow keys, Home, End`] },

  { file: `${D}TreePage.svelte`, keys: ['F1'], help: [`${TREE}: F1`] },
  { file: `${D}TreePage.svelte`, keys: ['PageUp', 'PageDown'], help: [`${TREE}: PgUp / PgDn`] },
  {
    file: `${D}TreePage.svelte`,
    keys: ['d', 'c', 'm', 'f', 'p'],
    help: [`${TREE}: Ctrl+D`, `${TREE}: Ctrl+C`, `${TREE}: Ctrl+M`, `${TREE}: Ctrl+F`, `${TREE}: P`]
  },
  { file: `${D}TreePage.svelte`, keys: ['ArrowUp', 'ArrowDown'], help: [`${TREE}: Up / Down`] },
  { file: `${D}TreePage.svelte`, keys: ['Escape'], help: [`${ALL}: Esc`] },
  { file: `${D}TreePage.svelte`, keys: ['Tab'], why: 'closes the node menu as focus leaves it' },
  { file: `${D}TreePage.svelte`, keys: ['ArrowLeft', 'ArrowRight'], why: A11Y },
  { file: 'frontend/src/lib/components/skill-tree/SkillTree.svelte', keys: ['Shift'], help: [`${TREE}: Hold Shift`] },
  { file: `${D}TreeView.svelte`, keys: ['ContextMenu', 'F10'], help: [`${FIELDS}: Shift+F10 or the Menu key`] },

  { file: `${D}ItemsPage.svelte`, keys: ['e'], help: ['Items: E'] },
  { file: `${SH}AppItems.svelte`, keys: ['d', 'f'], help: ['Items: Ctrl+D', 'Items: Ctrl+F'] },
  { file: `${D}ItemsPool.svelte`, keys: ['c', 'Delete', 'ArrowUp', 'ArrowDown'], help: ['Items: Ctrl+C', 'Items: Delete', 'Items: Up / Down'] },
  { file: `${D}ItemDB.svelte`, keys: ['c'], help: ['Items: Ctrl+C'] },
  { file: `${D}ItemEditor.svelte`, keys: ['z', 'y'], help: ['Items: Ctrl+Z / Ctrl+Y'] },
  { file: `${D}FloatingItemTip.svelte`, evidence: /shift = e\.shiftKey/, help: ['Items: Hold Shift'] },
  { file: `${D}ItemsBoard.svelte`, keys: ['ArrowLeft', 'ArrowUp', 'ArrowRight', 'ArrowDown'], why: A11Y },
  { file: `${D}ItemSockets.svelte`, keys: ['Escape', 'Enter', ' '], why: A11Y },

  { file: `${D}SkillsPage.svelte`, keys: ['c'], help: ['Skills: Ctrl+C'] },
  { file: `${D}SkillsPage.svelte`, evidence: /onpaste=/, help: ['Skills: Ctrl+V'] },
  { file: `${D}reorder.ts`, keys: [' ', 'ArrowUp', 'ArrowDown', 'Escape'], help: ['Skills: Space', 'Skills: Alt+Up / Alt+Down'] },
  { file: `${D}reorder.ts`, keys: ['Enter'], help: [`${FIELDS}: Enter`] },
  { file: `${D}SkillGroupEditor.svelte`, keys: ['Escape', 'ArrowDown', 'ArrowUp', 'Enter'], why: GENERIC },

  { file: `${D}ConfigPage.svelte`, keys: ['f'], help: ['Config: Ctrl+F'] },

  // TipTap's StarterKit binds these (notes/extensions.ts); the toolbar labels name them
  { file: `${D}NotesPage.svelte`, evidence: /keys: 'Ctrl\+B'[\s\S]*keys: 'Ctrl\+I'[\s\S]*keys: 'Ctrl\+U'/, help: [`${NOTES}: Ctrl+B / Ctrl+I / Ctrl+U`] },
  { file: `${D}NotesPage.svelte`, evidence: /keys: 'Ctrl\+Shift\+S'/, help: [`${NOTES}: Ctrl+Shift+S`] },
  { file: `${D}NotesPage.svelte`, evidence: /keys: 'Ctrl\+E'/, help: [`${NOTES}: Ctrl+E`] },
  { file: `${D}NotesPage.svelte`, evidence: /keys: `Ctrl\+Alt\+\$\{level\}`/, help: [`${NOTES}: Ctrl+Alt+1 - 3`] },
  {
    file: `${D}NotesPage.svelte`,
    evidence: /keys: 'Ctrl\+Shift\+8'[\s\S]*keys: 'Ctrl\+Shift\+7'[\s\S]*keys: 'Ctrl\+Shift\+9'/,
    help: [`${NOTES}: Ctrl+Shift+7 / 8 / 9`]
  },
  { file: `${D}NotesPage.svelte`, evidence: /keys: 'Ctrl\+Shift\+B'/, help: [`${NOTES}: Ctrl+Shift+B`] },
  { file: `${D}NotesPage.svelte`, evidence: /keys: 'Ctrl\+Alt\+C'/, help: [`${NOTES}: Ctrl+Alt+C`] },
  { file: `${D}NotesPage.svelte`, evidence: /keys: 'Ctrl\+Z'[\s\S]*keys: 'Ctrl\+Shift\+Z'/, help: [`${NOTES}: Ctrl+Z / Ctrl+Shift+Z`] },
  { file: `${D}NotesPage.svelte`, keys: ['k'], help: [`${NOTES}: Ctrl+K`] },
  { file: `${D}NotesPage.svelte`, keys: ['Escape'], help: [`${ALL}: Esc`] },

  {
    file: `${D}BuildTable.svelte`,
    keys: ['Enter', 'F2', 'Delete', 'c', 'x', 'v'],
    help: [`${LIST}: Enter`, `${LIST}: F2`, `${LIST}: Delete`, `${LIST}: Ctrl+C / Ctrl+X`, `${LIST}: Ctrl+V`]
  },
  { file: `${D}BuildTable.svelte`, keys: ['Escape'], help: [`${ALL}: Esc`] },
  { file: `${D}BuildTable.svelte`, keys: [' '], why: 'selects the focused row, as a click does' },
  { file: `${D}SetManager.svelte`, keys: ['Delete', 'F2'], help: [`${FIELDS}: F2 / Delete in a set list`] },

  { file: `${D}NumberInput.svelte`, keys: ['ArrowUp', 'ArrowDown'], help: [`${FIELDS}: Up / Down in a number box`] },
  { file: `${D}NumberInput.svelte`, keys: ['PageUp', 'PageDown'], help: [`${FIELDS}: PgUp / PgDn in a number box`] },
  { file: `${D}NumberInput.svelte`, keys: ['Enter'], help: [`${FIELDS}: Enter`] },
  { file: `${D}NumberInput.svelte`, keys: ['Home', 'End', 'Escape'], why: 'minimum / maximum and cancel typing; standard spin-button keys' },
  { file: `${D}ImportExportPage.svelte`, keys: ['Enter'], help: [`${FIELDS}: Enter`] },
  { file: `${D}CharacterImport.svelte`, keys: ['Enter'], help: [`${FIELDS}: Enter`] },
  { file: `${SH}SaveAsDialog.svelte`, keys: ['Enter'], help: [`${FIELDS}: Enter`] },
  { file: `${SH}PromptDialog.svelte`, keys: ['Enter'], help: [`${FIELDS}: Enter`] },
  { file: `${D}ModCell.svelte`, keys: ['Enter'], help: [`${FIELDS}: Enter`] },
  { file: `${D}Listbox.svelte`, keys: ['Enter'], help: [`${FIELDS}: Enter`] },
  { file: `${D}Dropdown.svelte`, keys: [' ', 'Backspace', 'Enter', 'Escape', 'Tab', 'ArrowDown', 'ArrowUp', 'Home', 'End'], why: GENERIC },
  { file: `${D}StatPicker.svelte`, keys: ['ArrowDown', 'ArrowUp', 'Enter', 'Escape'], why: GENERIC },
  { file: `${D}BuildPicker.svelte`, keys: ['ArrowDown', 'ArrowUp', 'Home', 'End', 'Enter'], why: GENERIC },
  { file: `${D}BuildPicker.svelte`, keys: ['Backspace'], why: 'goes up a folder in the Compare build picker; minor' },
  { file: `${D}TimelessPage.svelte`, keys: ['Enter', 'ArrowDown', 'ArrowUp'], why: GENERIC },
  { file: `${D}PowerReport.svelte`, keys: ['Enter', ' '], why: GENERIC },
  { file: `${D}Sidebar.svelte`, keys: ['ArrowLeft', 'ArrowRight'], why: A11Y },
  { file: `${D}CalcsBreakdown.svelte`, keys: ['ArrowLeft', 'ArrowRight'], why: A11Y },

  { file: 'desktop/main.go', keys: ['CmdOrCtrl+O', 'CmdOrCtrl+R'], help: ['Desktop app only: Ctrl+O', 'Desktop app only: Ctrl+R'] }
];

const helpText = readFileSync(join(FRONTEND, 'static/help.txt'), 'utf8');
/** "<sub-heading>: <key>" for every key line under Hotkeys. */
const documented = (text: string) => {
  const hotkeys = parseHelp(text, false).find((s) => s.title === 'Hotkeys');
  expect(hotkeys, 'help.txt has no Hotkeys section').toBeTruthy();
  let sub = '';
  const out: string[] = [];
  for (const l of hotkeys!.lines) {
    if (l.key === undefined && l.text.trim().endsWith(':')) sub = l.text.trim().slice(0, -1);
    else if (l.key !== undefined) out.push(`${sub}: ${l.key}`);
  }
  return out;
};

const problems = (inventory: Row[], docs: string[], read: (file: string) => string) => {
  const out: string[] = [];
  const found = new Map(SOURCES.map((f) => [relative(ROOT, f), scan(read(relative(ROOT, f)))]));
  for (const r of inventory) {
    if (!r.help === !r.why) out.push(`${r.file}: give a row either help lines or a reason`);
    for (const h of r.help ?? []) if (!docs.includes(h)) out.push(`${r.file}: "${h}" is not a key line under Hotkeys`);
    const keys = found.get(r.file);
    if (!keys) {
      out.push(`${r.file}: not a scanned source`);
      continue;
    }
    for (const k of r.keys ?? []) if (!keys.includes(k)) out.push(`${r.file}: no handler for '${k}' any more`);
    if (r.evidence && !r.evidence.test(read(r.file))) out.push(`${r.file}: ${r.evidence} no longer matches`);
  }
  for (const [file, keys] of found)
    for (const k of keys)
      if (!inventory.some((r) => r.file === file && r.keys?.includes(k))) out.push(`${file}: '${k}' has a handler but is neither documented nor allowed`);
  for (const h of docs) if (!inventory.some((r) => r.help?.includes(h))) out.push(`help.txt: "${h}" has no handler`);
  return out;
};
const disk = (file: string) => readFileSync(join(ROOT, file), 'utf8');

describe('Hotkeys in help.txt', () => {
  it('match the keyboard handlers in the code', () => {
    expect(problems(INVENTORY, documented(helpText), disk)).toEqual([]);
  });

  const edit = (file: string, from: string, to: string) => (f: string) => {
    const text = disk(f);
    if (f !== file) return text;
    expect(text, `${file} no longer holds the text the check edits`).toContain(from);
    return text.replace(from, to);
  };

  it('fail on a documented key with no handler', () => {
    const docs = documented(helpText);
    expect(problems(INVENTORY, [...docs, `${TREE}: Ctrl+Q`], disk)).toEqual([`help.txt: "${TREE}: Ctrl+Q" has no handler`]);
    const gone = edit(`${SH}AppItems.svelte`, "k === 'd'", "k === 'dd'");
    expect(problems(INVENTORY, docs, gone)).toEqual([
      `${SH}AppItems.svelte: no handler for 'd' any more`,
      `${SH}AppItems.svelte: 'dd' has a handler but is neither documented nor allowed`
    ]);
    const label = edit(`${D}NotesPage.svelte`, "keys: 'Ctrl+E'", "keys: 'Ctrl+Shift+E'");
    expect(problems(INVENTORY, docs, label)).toEqual([`${D}NotesPage.svelte: /keys: 'Ctrl\\+E'/ no longer matches`]);
  });

  it('fail on a new handler that is neither documented nor allowed', () => {
    const docs = documented(helpText);
    const local = edit(`${D}TreePage.svelte`, 'const k = e.key.toLowerCase();', "const k = e.key.toLowerCase();\n    if (k === 'q') return;");
    expect(problems(INVENTORY, docs, local)).toEqual([`${D}TreePage.svelte: 'q' has a handler but is neither documented nor allowed`]);
    const fresh = edit(`${D}Sidebar.svelte`, '<script lang="ts">', '<script lang="ts">\n  const f9 = (e: KeyboardEvent) => e.key === \'F9\';');
    expect(problems(INVENTORY, docs, fresh)).toEqual([`${D}Sidebar.svelte: 'F9' has a handler but is neither documented nor allowed`]);
  });
});
