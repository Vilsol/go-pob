(() => {
  if (window.__releaseStarted) return;
  window.__releaseStarted = true;
  const cfg = window.__RELEASE ?? {};
  const say = (m) => fetch('/desktop/probe', { method: 'POST', body: JSON.stringify(m) }).catch(() => {});
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
  const json = async (url, init) => {
    const r = await fetch(url, init);
    return { status: r.status, body: await r.json().catch(() => null) };
  };

  const run = async () => {
    await until('pob:data-loaded', () => mark('pob:data-loaded'), 90000);
    await say({ step: 'boot', booted: Math.round(mark('pob:wasm-booted')), dataLoaded: Math.round(mark('pob:data-loaded')) });
    [...document.querySelectorAll('button')].find((b) => b.textContent.trim() === 'New build').click();
    await until('the new build', () => /^Unnamed build/.test(document.title));
    await until('its stats', () => /Life/.test(document.body.innerText));
    await say({ step: 'new build', title: document.title });
    if (cfg.chooser) {
      await fetch('/desktop/open', { method: 'POST' });
      await sleep(3000);
      await say({ step: 'file chooser open, process alive' });
    }
    if (cfg.update) {
      const check = await json('/desktop/update/check');
      await say({ step: 'update check', ...check });
      if (cfg.update === 'check') return;
      const install = await json('/desktop/update/install', { method: 'POST' });
      await say({ step: 'update install', ...install });
      if (cfg.update === 'restart' && install.status === 200) {
        await fetch('/desktop/update/restart', { method: 'POST' });
        await sleep(30000);
      }
    }
  };
  run()
    .catch((e) => say({ fail: String(e) }))
    .finally(quit);
})();
