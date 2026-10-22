import { writable } from 'svelte/store';

export type LogLevel = 'info' | 'warn' | 'error';
export type LogEntry = { at: number; level: LogLevel; src: string; msg: string };
export type LogMessage = { pobLog: LogEntry };

export const LOG_CAP = 1000;

export const createLog = (cap = LOG_CAP) => {
  const { subscribe, update, set } = writable<LogEntry[]>([]);
  return {
    subscribe,
    clear: () => set([]),
    push: (e: LogEntry) =>
      update((lines) => {
        const next = lines.length >= cap ? lines.slice(lines.length - cap + 1) : lines.slice();
        next.push(e);
        return next;
      })
  };
};

export const appLog = createLog();

export const isLogMessage = (data: unknown): data is LogMessage =>
  typeof data === 'object' && data !== null && 'pobLog' in data && typeof (data as LogMessage).pobLog?.msg === 'string';

export const inWorker = () =>
  typeof window === 'undefined' && typeof self !== 'undefined' && typeof (self as { postMessage?: unknown }).postMessage === 'function';

// a worker posts its lines to whoever created it; the page keeps them
export const emitLog = (e: LogEntry) => {
  if (inWorker()) {
    (self as unknown as { postMessage: (m: LogMessage) => void }).postMessage({ pobLog: e });
  } else {
    appLog.push(e);
  }
};

export const logText = (v: unknown): string => {
  if (v instanceof Error) return v.message ? `${v.name}: ${v.message}` : v.name;
  if (typeof v === 'string') return v;
  if (v === undefined) return 'undefined';
  try {
    return JSON.stringify(v) ?? typeof v;
  } catch {
    return Object.prototype.toString.call(v);
  }
};

export const log = (level: LogLevel, src: string, ...parts: unknown[]) => emitLog({ at: Date.now(), level, src, msg: parts.map(logText).join(' ') });

/* eslint-disable no-control-regex */
const ANSI = /\u001b\[[\d;]*m/g;
const GO_LEVEL = /^(?:\S+\s+)?(TRC|DBG|INF|WRN|ERR)\s+(.*)$/;
/* eslint-enable no-control-regex */
const GO_LEVELS: Record<string, LogLevel> = { TRC: 'info', DBG: 'info', INF: 'info', WRN: 'warn', ERR: 'error' };

// a line of the Go engine's slog output (tint): an optional time, the level, the source, the message and attributes
export const goLogLine = (line: string): { level: LogLevel; msg: string } | undefined => {
  const m = GO_LEVEL.exec(line.replace(ANSI, '').trim());
  return m ? { level: GO_LEVELS[m[1]], msg: m[2] } : undefined;
};

export const forwardWorkerLog = (worker: Pick<Worker, 'addEventListener'>) =>
  worker.addEventListener('message', (e: MessageEvent<unknown>) => {
    if (isLogMessage(e.data)) emitLog(e.data.pobLog);
  });

export const captureUncaught = (target: Pick<WindowEventHandlers & EventTarget, 'addEventListener'>, src: string) => {
  target.addEventListener('error', (e: Event) => {
    const ev = e as ErrorEvent;
    log('error', src, ev.error ?? ev.message ?? 'uncaught error');
  });
  target.addEventListener('unhandledrejection', (e: Event) => log('error', src, 'unhandled rejection:', (e as PromiseRejectionEvent).reason));
};

const pad = (n: number) => String(n).padStart(2, '0');
export const logTime = (at: number) => {
  const d = new Date(at);
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
};

export type ReportEnv = { version?: string; commit?: string; built?: string; gameData?: string; userAgent?: string; platform?: string; language?: string };

export const bugReport = (lines: LogEntry[], env: ReportEnv) => {
  const head = [
    `go-pob ${env.version || '(unknown version)'}`,
    env.commit && `commit: ${env.commit}`,
    env.built && `built: ${env.built}`,
    env.gameData && `game data: ${env.gameData}`,
    env.platform && `platform: ${env.platform}`,
    env.language && `language: ${env.language}`,
    env.userAgent && `browser: ${env.userAgent}`,
    `log: ${lines.length} lines`
  ].filter(Boolean);
  const body = lines.map((l) => `${new Date(l.at).toISOString()} ${l.level.toUpperCase().padEnd(5)} [${l.src}] ${l.msg}`);
  return [...head, '', ...body].join('\n');
};
