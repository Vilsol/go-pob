export type HelpLine = { key?: string; text: string; dev?: boolean };
export type HelpSection = { title: string; lines: HelpLine[] };

/* Main.lua:1359-1397 */
export const parseHelp = (text: string, devMode: boolean): HelpSection[] => {
  const out: HelpSection[] = [];
  for (const raw of text.replace(/\r/g, '').split('\n')) {
    const title = /^---\[(.+)\]$/.exec(raw)?.[1];
    if (title) {
      out.push({ title, lines: [] });
      continue;
    }
    const section = out.at(-1);
    if (!section) continue;
    const dev = /^DEV\[(.+)\]$/.exec(raw)?.[1];
    if (dev !== undefined && !devMode) continue;
    const line = dev ?? raw;
    const tab = /^(.*)\t+(.*)$/.exec(line);
    const entry: HelpLine = tab ? { key: tab[1], text: tab[2] } : { text: line };
    if (dev !== undefined) entry.dev = true;
    section.lines.push(entry);
  }
  // Main.lua:1398-1411
  const k = out.findIndex((s) => s.title === 'Contents');
  if (k >= 0) out[k].lines.splice(Math.max(0, k - 1), 0, { text: ' ' }, ...out.map((s, i) => ({ text: `${i + 1}. ${s.title}` })));
  return out;
};

export type ChangeVersion = { version: string; date: string; lines: { text: string; sub?: boolean }[] };

/* Main.lua:1338-1358 */
export const parseChangelog = (text: string): ChangeVersion[] => {
  const out: ChangeVersion[] = [];
  for (const line of text.replace(/\r/g, '').replace(/\n$/, '').split('\n')) {
    const head = /^VERSION\[(.+)\]\[(.+)\]$/.exec(line);
    if (head) out.push({ version: head[1], date: head[2], lines: [] });
    else out.at(-1)?.lines.push(line.startsWith('---') ? { text: line, sub: true } : { text: line });
  }
  return out;
};

/* Main.lua:1306-1319 */
export const changesSince = (text: string, running: string): ChangeVersion[] => {
  const all = parseChangelog(text);
  const i = all.findIndex((v) => v.version === running);
  return i < 0 ? all : all.slice(0, i);
};
