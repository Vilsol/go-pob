import { describe, expect, it } from 'vitest';
import { get } from 'svelte/store';
import { appLog, bugReport, captureUncaught, createLog, forwardWorkerLog, goLogLine, type LogEntry } from './app_log';
import { logError } from './utils';

const entry = (i: number, level: LogEntry['level'] = 'info'): LogEntry => ({ at: i, level, src: 'test', msg: `line ${i}` });

describe('the log store', () => {
  it('keeps the newest lines up to its cap, dropping the oldest', () => {
    const log = createLog(3);
    for (let i = 1; i <= 5; i++) log.push(entry(i));
    expect(get(log).map((l) => l.msg)).toEqual(['line 3', 'line 4', 'line 5']);
  });

  it('keeps each line at its level', () => {
    const log = createLog();
    log.push(entry(1, 'info'));
    log.push(entry(2, 'warn'));
    log.push(entry(3, 'error'));
    expect(get(log).map((l) => l.level)).toEqual(['info', 'warn', 'error']);
  });

  it('logError on the page lands as an error line', () => {
    appLog.clear();
    const quiet = console.error;
    console.error = () => {};
    try {
      logError(new TypeError('boom'));
    } finally {
      console.error = quiet;
    }
    expect(get(appLog)).toMatchObject([{ level: 'error', src: 'app', msg: 'TypeError: boom' }]);
  });
});

describe('forwarding', () => {
  // the real Go output of slog.Error("engine broke", "key", 3) and slog.Warn("careful") through config.InitLogging
  const ERR = '\u001b[91mERR\u001b[0m \u001b[2mcalculator/calc.go:13\u001b[0m \u001b[1m\u001b[38mengine broke\u001b[0m \u001b[2mkey=\u001b[0m3';
  const WRN = '\u001b[93mWRN\u001b[0m \u001b[2mcalculator/calc.go:14\u001b[0m \u001b[1m\u001b[38mcareful\u001b[0m';

  it('reads the level of a Go slog line', () => {
    expect(goLogLine(ERR)).toEqual({ level: 'error', msg: 'calculator/calc.go:13 engine broke key=3' });
    expect(goLogLine(WRN)).toEqual({ level: 'warn', msg: 'calculator/calc.go:14 careful' });
    expect(goLogLine('DBG with a time 12:00 stamp')).toEqual({ level: 'info', msg: 'with a time 12:00 stamp' });
    expect(goLogLine('panic: not a slog line')).toBeUndefined();
  });

  it("keeps a worker's posted lines and ignores its other messages", () => {
    appLog.clear();
    const target = new EventTarget();
    forwardWorkerLog(target);
    target.dispatchEvent(new MessageEvent('message', { data: { pobLog: entry(1, 'warn') } }));
    target.dispatchEvent(new MessageEvent('message', { data: { id: 'comlink', type: 'APPLY' } }));
    target.dispatchEvent(new MessageEvent('message', { data: { type: 'progress', seq: 1 } }));
    expect(get(appLog)).toEqual([entry(1, 'warn')]);
  });

  it('logs uncaught errors and unhandled rejections', () => {
    appLog.clear();
    const target = new EventTarget();
    captureUncaught(target, 'page');
    target.dispatchEvent(Object.assign(new Event('error'), { error: new RangeError('out of range') }));
    target.dispatchEvent(Object.assign(new Event('unhandledrejection'), { reason: new Error('nobody caught me') }));
    expect(get(appLog).map(({ level, src, msg }) => ({ level, src, msg }))).toEqual([
      { level: 'error', src: 'page', msg: 'RangeError: out of range' },
      { level: 'error', src: 'page', msg: 'unhandled rejection: Error: nobody caught me' }
    ]);
  });
});

describe('the bug report', () => {
  it('carries the version, commit, browser and every line', () => {
    const text = bugReport([entry(0, 'error'), entry(1000)], { version: 'v1.2.3', commit: 'abc123', userAgent: 'TestBrowser/1', platform: 'Linux' });
    expect(text.split('\n')).toEqual([
      'go-pob v1.2.3',
      'commit: abc123',
      'platform: Linux',
      'browser: TestBrowser/1',
      'log: 2 lines',
      '',
      '1970-01-01T00:00:00.000Z ERROR [test] line 0',
      '1970-01-01T00:00:01.000Z INFO  [test] line 1000'
    ]);
  });
});
