(() => {
  if (window.__smokeStarted) return;
  window.__smokeStarted = true;
  const cfg = window.__SMOKE ?? {};
  const log = [];
  const say = (m) => {
    log.push(m);
    return fetch('/desktop/probe', { method: 'POST', body: JSON.stringify(m) }).catch(() => {});
  };
  const quit = () => fetch('/desktop/probe/quit', { method: 'POST' });
  const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
  const until = async (what, pred, timeout = 60000) => {
    const end = performance.now() + timeout;
    for (;;) {
      const v = await pred();
      if (v) return v;
      if (performance.now() > end) throw new Error(`timed out waiting for ${what}`);
      await sleep(25);
    }
  };
  const mark = (n) => performance.getEntriesByName(n)[0]?.startTime;
  const step = async (name, fn) => {
    const t = performance.now();
    const detail = await fn();
    await say({ step: name, ms: Math.round(performance.now() - t), ...(detail ?? {}) });
  };
  const errors = [];
  addEventListener('error', (e) => errors.push(String(e.message)));
  addEventListener('unhandledrejection', (e) => errors.push(String(e.reason)));
  const imported = async () => {
    const title = await until('the imported build', () => /^Imported build \(/.test(document.title) && document.title);
    await until('its stats', () => /Life/.test(document.body.innerText));
    return { title };
  };
  // Build.lua:1421-1448
  const replaced = async (open) => {
    const before = document.body.innerText;
    await open();
    await until('the replacing build', () => {
      const prompt = [...document.querySelectorAll('dialog[open]')].find((d) => /unsaved changes/.test(d.textContent));
      [...(prompt?.querySelectorAll('button') ?? [])].find((b) => b.textContent.trim() === "Don't Save")?.click();
      return /^Imported build \(/.test(document.title) && /Life/.test(document.body.innerText) && document.body.innerText !== before;
    });
    for (let text = ''; text !== document.body.innerText; await sleep(500)) text = document.body.innerText;
    return { title: document.title };
  };

  const main = async () => {
    await step('New build → stats', async () => {
      [...document.querySelectorAll('button')].find((b) => b.textContent.trim() === 'New build').click();
      await until('the new build', () => /^Unnamed build/.test(document.title));
      await until('its stats', () => /Life/.test(document.body.innerText));
    });
    if (cfg.link)
      await step(`import ${cfg.link} (/api proxy) → stats`, () =>
        replaced(() => dispatchEvent(new CustomEvent('pob:desktop-import', { detail: cfg.link })))
      );
    await step('open build file → stats', () =>
      replaced(async () => {
        const got = new Promise((r) => addEventListener('pob:desktop-import', r, { once: true }));
        await fetch('/desktop/open', { method: 'POST' });
        await got;
      })
    );
    await step('power report Life depth 2, 1 worker', async () => {
      const r = await globalThis.cachedWorker.syncWrap.BuildPowerReportLive('Life', 2, 1);
      if (!r?.rows?.length) throw new Error('power report returned no rows');
      return { rows: r.rows.length, reportMs: Math.round(r.ms) };
    });
  };

  (async () => {
    try {
      await until('pob:data-loaded', () => mark('pob:data-loaded'), 120000);
      await say({
        step: 'boot',
        booted: Math.round(mark('pob:wasm-booted')),
        dataLoaded: Math.round(mark('pob:data-loaded')),
        agent: navigator.userAgent,
        cores: navigator.hardwareConcurrency
      });
      if (cfg.phase === 'link') await step('launched by link → stats', imported);
      else await main();
      await say({ done: cfg.phase, errors });
    } catch (e) {
      await say({ fail: String(e), errors, title: document.title, path: location.pathname });
    }
    await fetch('/desktop/fs/settings/probe-log.json', { method: 'PUT', body: JSON.stringify(log) }).catch(() => {});
    quit();
  })();
})();
